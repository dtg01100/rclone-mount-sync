// Package systemd provides functionality for generating systemd unit files.
package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/dtg01100/rclone-mount-sync/internal/models"
)

const (
	DefaultMemoryMax = "1G"
	DefaultCPUQuota  = "50%"
)

// Generator generates systemd unit files.
type Generator struct {
	systemdDir     string // Full path to user systemd directory
	rclonePath     string // Path to rclone binary
	configPath     string // Path to rclone config file
	logDir         string // Directory for log files
	fusermountPath string // Path to fusermount binary (prefers fusermount3)
}

// NewGenerator creates a new unit file generator.
func NewGenerator() (*Generator, error) {
	systemdDir, err := GetUserSystemdPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get systemd path: %w", err)
	}

	// Find rclone binary - check environment variable first, then PATH
	rclonePath := os.Getenv("RCLONE_BINARY_PATH")
	if rclonePath == "" {
		rclonePath, err = exec.LookPath("rclone")
		if err != nil {
			rclonePath = "/usr/bin/rclone" // Default fallback
		}
	}

	// Get rclone config path
	configPath := getRcloneConfigPath()

	// Get log directory
	logDir, err := getLogDir()
	if err != nil {
		logDir = "/tmp" // Fallback
	}

	// Find fusermount - prefer fusermount3 (newer) over fusermount
	fusermountPath := findFusermount()

	return &Generator{
		systemdDir:     systemdDir,
		rclonePath:     rclonePath,
		configPath:     configPath,
		logDir:         logDir,
		fusermountPath: fusermountPath,
	}, nil
}

// GetSystemdDir returns the systemd user directory path.
func (g *Generator) GetSystemdDir() string {
	return g.systemdDir
}

// findFusermount locates fusermount3 or fusermount binary, preferring fusermount3.
func findFusermount() string {
	if path, err := exec.LookPath("fusermount3"); err == nil {
		return path
	}
	if path, err := exec.LookPath("fusermount"); err == nil {
		return path
	}
	return "/bin/fusermount" // Default fallback
}

// GenerateMountService generates a systemd service unit for an rclone mount.
func (g *Generator) GenerateMountService(mount *models.MountConfig) (string, error) {
	mountPoint := expandPath(mount.MountPoint)
	mountOptions := g.buildMountOptions(&mount.MountOptions)

	safeName, err := sanitizeIniValue(mount.Name, "Name")
	if err != nil {
		return "", err
	}
	safeRemote, err := sanitizeIniValue(mount.Remote, "Remote")
	if err != nil {
		return "", err
	}
	safeRemotePath, err := sanitizeIniValue(mount.RemotePath, "RemotePath")
	if err != nil {
		return "", err
	}
	safeMountPoint, err := sanitizeShellValue(mountPoint, "MountPoint")
	if err != nil {
		return "", err
	}

	data := MountUnitData{
		Name:           safeName,
		Remote:         safeRemote,
		RemotePath:     safeRemotePath,
		MountPoint:     safeMountPoint,
		MountOptions:   mountOptions,
		RclonePath:     g.rclonePath,
		FusermountPath: g.fusermountPath,
		MemoryMax:      DefaultMemoryMax,
		CPUQuota:       DefaultCPUQuota,
	}

	tmpl, err := template.New("mount-service").Parse(MountServiceTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse mount service template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute mount service template: %w", err)
	}

	return buf.String(), nil
}

// WriteMountService generates and writes a systemd service unit for an rclone mount.
func (g *Generator) WriteMountService(mount *models.MountConfig) (string, error) {
	content, err := g.GenerateMountService(mount)
	if err != nil {
		return "", err
	}

	filename := g.ServiceName(mount.ID, "mount") + ".service"
	if err := g.WriteUnitFile(filename, content); err != nil {
		return "", fmt.Errorf("failed to write mount service file: %w", err)
	}

	return filepath.Join(g.systemdDir, filename), nil
}

// GenerateSyncService generates a systemd service unit for an rclone sync job.
func (g *Generator) GenerateSyncService(job *models.SyncJobConfig) (string, error) {
	syncOptions := g.buildSyncOptions(&job.SyncOptions)

	direction := job.SyncOptions.Direction
	if direction == "" {
		direction = "sync"
	}

	execCondition := UnmeteredNetworkExecCondition
	if !job.Schedule.RequireUnmetered {
		execCondition = ""
	}

	safeName, err := sanitizeIniValue(job.Name, "Name")
	if err != nil {
		return "", err
	}
	safeSource, err := sanitizeShellValue(job.Source, "Source")
	if err != nil {
		return "", err
	}
	safeDest, err := sanitizeShellValue(expandPath(job.Destination), "Destination")
	if err != nil {
		return "", err
	}

	data := SyncUnitData{
		Name:             safeName,
		Source:           safeSource,
		Destination:      safeDest,
		Direction:        direction,
		SyncOptions:      syncOptions,
		RclonePath:       g.rclonePath,
		RequireACPower:   job.Schedule.RequireACPower,
		RequireUnmetered: job.Schedule.RequireUnmetered,
		ExecCondition:    execCondition,
		MemoryMax:        DefaultMemoryMax,
		CPUQuota:         DefaultCPUQuota,
	}

	tmpl, err := template.New("sync-service").Parse(SyncServiceTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse sync service template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute sync service template: %w", err)
	}

	return buf.String(), nil
}

// GenerateSyncTimer generates a systemd timer unit for an rclone sync job.
func (g *Generator) GenerateSyncTimer(job *models.SyncJobConfig) (string, error) {
	timerDirectives := g.buildTimerDirectives(&job.Schedule)

	safeName, err := sanitizeIniValue(job.Name, "Name")
	if err != nil {
		return "", err
	}

	data := TimerUnitData{
		Name:            safeName,
		TimerDirectives: timerDirectives,
	}

	tmpl, err := template.New("sync-timer").Parse(SyncTimerTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse sync timer template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute sync timer template: %w", err)
	}

	return buf.String(), nil
}

// WriteSyncUnits generates and writes both service and timer units for a sync job.
func (g *Generator) WriteSyncUnits(job *models.SyncJobConfig) (servicePath, timerPath string, err error) {
	// Generate and write service
	serviceContent, err := g.GenerateSyncService(job)
	if err != nil {
		return "", "", err
	}

	serviceFilename := g.ServiceName(job.ID, "sync") + ".service"
	if err := g.WriteUnitFile(serviceFilename, serviceContent); err != nil {
		return "", "", fmt.Errorf("failed to write sync service file: %w", err)
	}
	servicePath = filepath.Join(g.systemdDir, serviceFilename)

	// Generate and write timer (only if schedule type is not manual)
	if job.Schedule.Type != "manual" {
		timerContent, err := g.GenerateSyncTimer(job)
		if err != nil {
			return servicePath, "", err
		}

		timerFilename := g.ServiceName(job.ID, "sync") + ".timer"
		if err := g.WriteUnitFile(timerFilename, timerContent); err != nil {
			return servicePath, "", fmt.Errorf("failed to write sync timer file: %w", err)
		}
		timerPath = filepath.Join(g.systemdDir, timerFilename)
	}

	return servicePath, timerPath, nil
}

// ServiceName generates a systemd unit name from the ID.
// Format: rclone-{type}-{id}
// IDs are 8-character alphanumeric strings (truncated UUIDs), so no sanitization needed.
func (g *Generator) ServiceName(id, unitType string) string {
	return fmt.Sprintf("rclone-%s-%s", unitType, id)
}

// RemoveUnit removes a unit file from the systemd directory.
// The name must not contain path separators or ".." to prevent path traversal.
func (g *Generator) RemoveUnit(name string) error {
	if err := validateUnitFilename(name); err != nil {
		return fmt.Errorf("invalid unit filename: %w", err)
	}
	path := filepath.Join(g.systemdDir, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to remove
	}
	return os.Remove(path)
}

// WriteUnitFile writes a unit file to the systemd user directory.
// The filename must not contain path separators or ".." to prevent path traversal.
// The write is atomic: the content is written to a temp file in the same
// directory, fsync'd, and then renamed to the final name. This prevents
// systemd from reading a truncated/partial unit file if the process is
// killed mid-write.
func (g *Generator) WriteUnitFile(filename, content string) error {
	if err := validateUnitFilename(filename); err != nil {
		return fmt.Errorf("invalid unit filename: %w", err)
	}
	// Ensure directory exists with restrictive permissions: the systemd
	// user directory may hold service files whose ExecStart lines embed
	// user-controlled paths.
	if err := os.MkdirAll(g.systemdDir, 0o700); err != nil {
		return fmt.Errorf("failed to create systemd directory: %w", err)
	}

	finalPath := filepath.Join(g.systemdDir, filename)
	tmp, err := os.CreateTemp(g.systemdDir, filename+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp unit file: %w", err)
	}
	tmpName := tmp.Name()
	// Ensure the temp file is removed if we fail before the rename.
	defer func() {
		if _, statErr := os.Stat(tmpName); statErr == nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write([]byte(content)); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write unit file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to sync unit file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close unit file: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("failed to chmod unit file: %w", err)
	}
	if err := os.Rename(tmpName, finalPath); err != nil {
		return fmt.Errorf("failed to rename unit file: %w", err)
	}
	return nil
}

// validateUnitFilename ensures a unit filename doesn't contain path traversal.
func validateUnitFilename(name string) error {
	if strings.Contains(name, "..") {
		return fmt.Errorf("filename must not contain '..'")
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("filename must not contain path separators")
	}
	return nil
}

// buildMountOptions builds the mount options string for rclone.
func (g *Generator) buildMountOptions(opts *models.MountOptions) string {
	var args []string

	// Config path
	configPath := opts.Config
	if configPath == "" {
		configPath = g.configPath
	}
	if configPath != "" {
		args = append(args, fmt.Sprintf("--config=%s", configPath))
	}

	// VFS options
	if opts.VFSCacheMode != "" {
		args = append(args, fmt.Sprintf("--vfs-cache-mode=%s", opts.VFSCacheMode))
	}
	if opts.VFSCacheMaxAge != "" {
		args = append(args, fmt.Sprintf("--vfs-cache-max-age=%s", opts.VFSCacheMaxAge))
	}
	if opts.VFSCacheMaxSize != "" {
		args = append(args, fmt.Sprintf("--vfs-cache-max-size=%s", opts.VFSCacheMaxSize))
	}
	if opts.VFSReadChunkSize != "" {
		args = append(args, fmt.Sprintf("--vfs-read-chunk-size=%s", opts.VFSReadChunkSize))
	}
	if opts.VFSWriteBack != "" {
		args = append(args, fmt.Sprintf("--vfs-write-back=%s", opts.VFSWriteBack))
	}

	// Buffer size
	if opts.BufferSize != "" {
		args = append(args, fmt.Sprintf("--buffer-size=%s", opts.BufferSize))
	}

	// Dir cache time
	if opts.DirCacheTime != "" {
		args = append(args, fmt.Sprintf("--dir-cache-time=%s", opts.DirCacheTime))
	}

	// FUSE options
	if opts.AllowOther {
		args = append(args, "--allow-other")
	}
	if opts.AllowRoot {
		args = append(args, "--allow-root")
	}
	if opts.Umask != "" {
		args = append(args, fmt.Sprintf("--umask=%s", opts.Umask))
	}
	if opts.UID > 0 {
		args = append(args, fmt.Sprintf("--uid=%d", opts.UID))
	}
	if opts.GID > 0 {
		args = append(args, fmt.Sprintf("--gid=%d", opts.GID))
	}

	// Behavior options
	if opts.NoModTime {
		args = append(args, "--no-modtime")
	}
	if opts.NoChecksum {
		args = append(args, "--no-checksum")
	}
	if opts.ReadOnly {
		args = append(args, "--read-only")
	}

	// Network options
	if opts.ConnectTimeout != "" {
		args = append(args, fmt.Sprintf("--connect-timeout=%s", opts.ConnectTimeout))
	}
	if opts.Timeout != "" {
		args = append(args, fmt.Sprintf("--timeout=%s", opts.Timeout))
	}

	// Logging options
	if opts.LogLevel != "" {
		args = append(args, fmt.Sprintf("--log-level=%s", opts.LogLevel))
	}

	// Extra arguments (sanitized to prevent systemd directive injection)
	if opts.ExtraArgs != "" {
		args = append(args, sanitizeExtraArgs(opts.ExtraArgs))
	}

	return strings.Join(args, " \\\n ")
}

// buildSyncOptions builds the sync options string for rclone.
func (g *Generator) buildSyncOptions(opts *models.SyncOptions) string {
	var args []string

	// Config path
	configPath := opts.Config
	if configPath == "" {
		configPath = g.configPath
	}
	if configPath != "" {
		args = append(args, fmt.Sprintf("--config=%s", configPath))
	}

	// Deletion handling
	if opts.DeleteExtraneous {
		args = append(args, "--delete-extraneous")
	}
	if opts.DeleteAfter {
		args = append(args, "--delete-after")
	}

	// Filtering
	if opts.IncludePattern != "" {
		args = append(args, fmt.Sprintf("--include=%s", opts.IncludePattern))
	}
	if opts.ExcludePattern != "" {
		args = append(args, fmt.Sprintf("--exclude=%s", opts.ExcludePattern))
	}
	if opts.MaxAge != "" {
		args = append(args, fmt.Sprintf("--max-age=%s", opts.MaxAge))
	}
	if opts.MinAge != "" {
		args = append(args, fmt.Sprintf("--min-age=%s", opts.MinAge))
	}

	// Performance
	if opts.Transfers > 0 {
		args = append(args, fmt.Sprintf("--transfers=%d", opts.Transfers))
	}
	if opts.Checkers > 0 {
		args = append(args, fmt.Sprintf("--checkers=%d", opts.Checkers))
	}
	if opts.BandwidthLimit != "" {
		args = append(args, fmt.Sprintf("--bwlimit=%s", opts.BandwidthLimit))
	}

	// Verification
	if opts.CheckSum {
		args = append(args, "--checksum")
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}

	// Logging options
	if opts.LogLevel != "" {
		args = append(args, fmt.Sprintf("--log-level=%s", opts.LogLevel))
	}

	// Create empty source dirs
	args = append(args, "--create-empty-src-dirs")

	// Extra arguments (sanitized to prevent systemd directive injection)
	if opts.ExtraArgs != "" {
		args = append(args, sanitizeExtraArgs(opts.ExtraArgs))
	}

	return strings.Join(args, " \\\n ")
}

// buildTimerDirectives builds timer directives from schedule configuration.
func (g *Generator) buildTimerDirectives(schedule *models.ScheduleConfig) string {
	var directives []string

	switch schedule.Type {
	case "timer":
		if schedule.OnCalendar != "" {
			directives = append(directives, fmt.Sprintf("OnCalendar=%s", schedule.OnCalendar))
		}
	case "onboot":
		if schedule.OnBootSec != "" {
			directives = append(directives, fmt.Sprintf("OnBootSec=%s", schedule.OnBootSec))
		}
	}

	// Interval-based scheduling
	if schedule.OnActiveSec != "" {
		directives = append(directives, fmt.Sprintf("OnUnitActiveSec=%s", schedule.OnActiveSec))
	}

	// Randomized delay
	if schedule.RandomizedDelaySec != "" {
		directives = append(directives, fmt.Sprintf("RandomizedDelaySec=%s", schedule.RandomizedDelaySec))
	}

	// Persistent to catch missed runs
	if schedule.Persistent {
		directives = append(directives, "Persistent=true")
	}

	// Default if no directives
	if len(directives) == 0 {
		directives = append(directives, "OnCalendar=daily")
	}

	return strings.Join(directives, "\n")
}

// sanitizeExtraArgs removes newlines and escapes percent signs to prevent
// systemd unit file directive injection. It returns an empty string with
// a logged warning if the args contain systemd directive patterns.
func sanitizeExtraArgs(args string) string {
	if err := models.ValidateExtraArgs(args); err != nil {
		// Strip dangerous content rather than silently passing it through
		cleaned := strings.ReplaceAll(args, "\r", " ")
		cleaned = strings.ReplaceAll(cleaned, "\n", " ")
		// Remove fields that look like systemd directives (Key=Value)
		var safe []string
		for _, field := range strings.Fields(cleaned) {
			idx := strings.IndexByte(field, '=')
			if idx > 0 && idx < len(field)-1 {
				key := field[:idx]
				if models.IsAlpha(key) {
					continue
				}
			}
			safe = append(safe, field)
		}
		args = strings.Join(safe, " ")
	}
	args = strings.ReplaceAll(args, "%", "%%")
	return strings.TrimSpace(args)
}

// sanitizeShellValue rejects values containing characters that would be
// interpreted by a shell (or that could split systemd ExecStart arguments
// unexpectedly). It is used for values that flow into shell-evaluated
// directives (ExecStartPre/ExecStopPost) and into positional ExecStart args
// where word-splitting would be dangerous.
//
// The allowed set is intentionally narrow: alphanumerics, the path-relevant
// punctuation (/ _ - .), and '='. Anything else (spaces, quotes, semicolons,
// backticks, $ ( ) { } | & < > * ? ! \n \r \t, etc.) is rejected with an
// error so the caller can surface the problem instead of silently writing a
// unit file that, when started, executes unintended commands.
func sanitizeShellValue(value, field string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", field)
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '/' || r == '_' || r == '-' || r == '.' || r == '=' || r == ':' || r == '@':
		default:
			return "", fmt.Errorf("%s contains illegal character %q: only alphanumerics and / _ - . = : @ are allowed (systemd passes this value to a shell)", field, r)
		}
	}
	return value, nil
}

// sanitizeIniValue strips characters that would break out of a systemd
// unit-file directive: newlines, carriage returns, and NUL bytes. These are
// the characters that would allow a value to start a new directive
// (e.g. "innocent\nExecStart=/bin/sh -c 'rm -rf /'"). Any other character
// is allowed, including spaces, because many systemd directives are
// space-separated value lists and some values (like Description) may
// legitimately contain spaces.
func sanitizeIniValue(value, field string) (string, error) {
	if strings.ContainsAny(value, "\r\n\x00") {
		return "", fmt.Errorf("%s must not contain newline, carriage return, or NUL characters (systemd unit-file injection risk)", field)
	}
	return value, nil
}

// NewTestGenerator creates a generator for use in tests.
// It uses the provided temp directory for all output.
func NewTestGenerator(tmpDir string) *Generator {
	return &Generator{
		systemdDir:     tmpDir,
		rclonePath:     "/usr/bin/rclone",
		configPath:     "/tmp/rclone.conf",
		logDir:         tmpDir,
		fusermountPath: "/bin/fusermount",
	}
}
