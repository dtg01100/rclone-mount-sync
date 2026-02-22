package systemd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/google/uuid"
)

// OrphanedUnit represents a unit file that exists in systemd but has no corresponding config entry.
type OrphanedUnit struct {
	Name     string // Unit filename (e.g., "rclone-mount-mydrive.service")
	Type     string // "mount" or "sync"
	ID       string // Extracted ID (may be a name for legacy units)
	IsLegacy bool   // True if this is a name-based (legacy) unit
	Path     string // Full path to the unit file
	Imported bool   // True if this unit can be imported
}

// ImportedConfig represents configuration recovered from an orphaned unit file.
type ImportedConfig struct {
	Mount   *models.MountConfig
	SyncJob *models.SyncJobConfig
	Unit    OrphanedUnit
}

// ReconciliationResult contains the results of a reconciliation scan.
type ReconciliationResult struct {
	OrphanedUnits []OrphanedUnit
	Errors        []error
}

// Reconciler detects orphaned and legacy unit files.
type Reconciler struct {
	generator *Generator
	manager   *Manager
}

// NewReconciler creates a new reconciler.
func NewReconciler(generator *Generator, manager *Manager) *Reconciler {
	return &Reconciler{
		generator: generator,
		manager:   manager,
	}
}

// ScanForOrphans scans the systemd directory for orphaned unit files.
// validIDs should contain all known mount and sync job IDs from the config.
func (r *Reconciler) ScanForOrphans(validMountIDs, validSyncIDs map[string]bool) (*ReconciliationResult, error) {
	result := &ReconciliationResult{}

	systemdDir := r.generator.GetSystemdDir()
	entries, err := os.ReadDir(systemdDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil // No systemd directory, no orphans
		}
		return nil, fmt.Errorf("failed to read systemd directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Only process rclone unit files
		if !strings.HasPrefix(name, "rclone-") {
			continue
		}

		// Skip timer files - we handle them with their service files
		if strings.HasSuffix(name, ".timer") {
			continue
		}

		// Parse the unit name
		id, unitType, isLegacy := r.parseUnitFile(name)

		// Check if this ID exists in our valid IDs
		var isValid bool
		switch unitType {
		case "mount":
			isValid = validMountIDs[id]
		case "sync":
			isValid = validSyncIDs[id]
		default:
			continue // Unknown type, skip
		}

		if !isValid {
			result.OrphanedUnits = append(result.OrphanedUnits, OrphanedUnit{
				Name:     name,
				Type:     unitType,
				ID:       id,
				IsLegacy: isLegacy,
				Path:     filepath.Join(systemdDir, name),
			})
		}
	}

	return result, nil
}

// parseUnitFile extracts the ID and type from a unit filename.
// Returns (id, type, isLegacy).
// Legacy units have name-based IDs (sanitized names), new units have 8-char UUIDs.
func (r *Reconciler) parseUnitFile(filename string) (id string, unitType string, isLegacy bool) {
	// Remove .service suffix
	name := strings.TrimSuffix(filename, ".service")

	// Parse rclone-{type}-{id}
	if strings.HasPrefix(name, "rclone-mount-") {
		id = strings.TrimPrefix(name, "rclone-mount-")
		unitType = "mount"
	} else if strings.HasPrefix(name, "rclone-sync-") {
		id = strings.TrimPrefix(name, "rclone-sync-")
		unitType = "sync"
	}

	// Check if this looks like a legacy name-based unit
	// ID-based units are 8-character alphanumeric strings
	isLegacy = !isValidID(id)

	return id, unitType, isLegacy
}

// isValidID checks if an ID looks like a valid UUID-based ID (8 chars, alphanumeric).
func isValidID(id string) bool {
	if len(id) != 8 {
		return false
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// RemoveOrphan removes an orphaned unit file and its associated timer if any.
func (r *Reconciler) RemoveOrphan(orphan OrphanedUnit) error {
	// Stop and disable the service if running
	serviceName := strings.TrimSuffix(orphan.Name, ".service")
	if isActive, _ := r.manager.IsActive(serviceName); isActive {
		if err := r.manager.Stop(serviceName); err != nil {
			return fmt.Errorf("failed to stop orphan service: %w", err)
		}
	}
	if isEnabled, _ := r.manager.IsEnabled(serviceName); isEnabled {
		if err := r.manager.Disable(serviceName); err != nil {
			return fmt.Errorf("failed to disable orphan service: %w", err)
		}
	}

	// Remove the unit file
	if err := r.generator.RemoveUnit(orphan.Name); err != nil {
		return fmt.Errorf("failed to remove orphan unit file: %w", err)
	}

	// Remove associated timer if it exists
	if orphan.Type == "sync" {
		timerName := strings.Replace(orphan.Name, ".service", ".timer", 1)
		timerPath := filepath.Join(r.generator.GetSystemdDir(), timerName)
		if _, err := os.Stat(timerPath); err == nil {
			timerUnitName := strings.TrimSuffix(timerName, ".timer")
			if isEnabled, _ := r.manager.IsEnabled(timerUnitName); isEnabled {
				r.manager.Disable(timerUnitName)
			}
			r.generator.RemoveUnit(timerName)
		}
	}

	// Reload daemon
	return r.manager.DaemonReload()
}

// Import attempts to parse an orphaned unit file and recover its configuration.
// Returns a partial config with the essential fields populated.
func (r *Reconciler) Import(orphan OrphanedUnit) (*ImportedConfig, error) {
	content, err := os.ReadFile(orphan.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read unit file: %w", err)
	}

	result := &ImportedConfig{Unit: orphan}

	switch orphan.Type {
	case "mount":
		mount, err := r.parseMountUnit(string(content), orphan)
		if err != nil {
			return nil, err
		}
		result.Mount = mount
	case "sync":
		job, err := r.parseSyncUnit(string(content), orphan)
		if err != nil {
			return nil, err
		}
		result.SyncJob = job
	}

	return result, nil
}

// parseMountUnit parses a mount service file and extracts config.
func (r *Reconciler) parseMountUnit(content string, orphan OrphanedUnit) (*models.MountConfig, error) {
	mount := &models.MountConfig{
		ID:         generateNewID(),
		Name:       extractNameFromDescription(content, "mount"),
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
	}

	execStart := extractExecStart(content)
	if execStart != "" {
		parts := parseRcloneMountCommand(execStart)
		if len(parts) >= 2 {
			mount.Remote, mount.RemotePath = parseRemotePath(parts[0])
			mount.MountPoint = parts[1]
		}
	}

	return mount, nil
}

// parseSyncUnit parses a sync service file and extracts config.
func (r *Reconciler) parseSyncUnit(content string, orphan OrphanedUnit) (*models.SyncJobConfig, error) {
	job := &models.SyncJobConfig{
		ID:         generateNewID(),
		Name:       extractNameFromDescription(content, "sync"),
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
		Schedule: models.ScheduleConfig{
			Type: "manual",
		},
	}

	execStart := extractExecStart(content)
	if execStart != "" {
		parts := parseRcloneSyncCommand(execStart)
		if len(parts) >= 3 {
			job.SyncOptions.Direction = parts[0]
			job.Source = parts[1]
			job.Destination = parts[2]
		}
	}

	timerPath := strings.Replace(orphan.Path, ".service", ".timer", 1)
	if timerContent, err := os.ReadFile(timerPath); err == nil {
		job.Schedule = r.parseTimerSchedule(string(timerContent))
	}

	return job, nil
}

func extractNameFromDescription(content, unitType string) string {
	re := regexp.MustCompile(`(?i)^Description=Rclone\s+(?:mount|sync):\s*(.+)$`)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	return "imported-" + unitType
}

func extractExecStart(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var lines []string
	inExecStart := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ExecStart=") {
			inExecStart = true
			lines = append(lines, strings.TrimPrefix(line, "ExecStart="))
		} else if inExecStart {
			if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
				lines = append(lines, strings.TrimSpace(line))
			} else {
				break
			}
		}
	}

	return strings.Join(lines, " ")
}

func parseRcloneMountCommand(execStart string) []string {
	re := regexp.MustCompile(`^\S+\s+mount\s+(\S+)\s+(\S+)`)
	matches := re.FindStringSubmatch(execStart)
	if len(matches) >= 3 {
		return []string{matches[1], matches[2]}
	}
	return nil
}

func parseRcloneSyncCommand(execStart string) []string {
	re := regexp.MustCompile(`^\S+\s+(sync|copy|move)\s+(\S+)\s+(\S+)`)
	matches := re.FindStringSubmatch(execStart)
	if len(matches) >= 4 {
		return []string{matches[1], matches[2], matches[3]}
	}
	return nil
}

func parseRemotePath(remotePath string) (remote, path string) {
	idx := strings.Index(remotePath, ":")
	if idx == -1 {
		return remotePath, "/"
	}
	return remotePath[:idx+1], remotePath[idx+1:]
}

func (r *Reconciler) parseTimerSchedule(content string) models.ScheduleConfig {
	config := models.ScheduleConfig{Type: "timer"}

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "OnCalendar=") {
			config.OnCalendar = strings.TrimPrefix(line, "OnCalendar=")
			config.Type = "timer"
		} else if strings.HasPrefix(line, "OnBootSec=") {
			config.OnBootSec = strings.TrimPrefix(line, "OnBootSec=")
			config.Type = "onboot"
		} else if strings.HasPrefix(line, "OnUnitActiveSec=") {
			config.OnActiveSec = strings.TrimPrefix(line, "OnUnitActiveSec=")
		} else if strings.HasPrefix(line, "RandomizedDelaySec=") {
			config.RandomizedDelaySec = strings.TrimPrefix(line, "RandomizedDelaySec=")
		} else if line == "Persistent=true" {
			config.Persistent = true
		}
	}

	return config
}

func generateNewID() string {
	return uuid.New().String()[:8]
}
