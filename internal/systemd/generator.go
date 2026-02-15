// Package systemd provides functionality for generating systemd unit files.
package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/dtg01100/rclone-mount-sync/internal/models"
)

// UserSystemdDir is the relative path to the user systemd directory.
const UserSystemdDir = ".config/systemd/user"

// MountServiceTemplate is the systemd service unit template for mounts.
const MountServiceTemplate = `[Unit]
Description=Rclone mount: {{.Name}}
Documentation=man:rclone(1)
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=30
StartLimitBurst=5

[Service]
Type=notify
ExecStartPre=/bin/mkdir -p {{.MountPoint}}
ExecStart={{.RclonePath}} mount \
    {{.Remote}}{{.RemotePath}} \
    {{.MountPoint}} \
    {{.MountOptions}}
ExecStop=/bin/fusermount -u {{.MountPoint}}
ExecStopPost=/bin/rmdir {{.MountPoint}}
Restart=on-failure
RestartSec=5s
Environment="PATH=/usr/local/bin:/usr/bin:/bin"
NoNewPrivileges=true

[Install]
WantedBy=default.target
`

// SyncServiceTemplate is the systemd service unit template for sync jobs.
const SyncServiceTemplate = `[Unit]
Description=Rclone sync: {{.Name}}
Documentation=man:rclone(1)
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart={{.RclonePath}} {{.Direction}} \
    {{.Source}} \
    {{.Destination}} \
    {{.SyncOptions}}
Environment="PATH=/usr/local/bin:/usr/bin:/bin"
MemoryMax=1G
CPUQuota=50%

[Install]
WantedBy=default.target
`

// SyncTimerTemplate is the systemd timer unit template for sync jobs.
const SyncTimerTemplate = `[Unit]
Description=Timer for rclone sync: {{.Name}}
Documentation=man:rclone(1)

[Timer]
{{.TimerDirectives}}

[Install]
WantedBy=timers.target
`

// MountUnitData contains data for mount service unit generation.
type MountUnitData struct {
	Name         string
	Remote       string
	RemotePath   string
	MountPoint   string
	ConfigPath   string
	MountOptions string
	LogLevel     string
	LogPath      string
	RclonePath   string
}

// SyncUnitData contains data for sync service unit generation.
type SyncUnitData struct {
	Name        string
	Source      string
	Destination string
	Direction   string
	ConfigPath  string
	SyncOptions string
	LogLevel    string
	LogPath     string
	RclonePath  string
}

// TimerUnitData contains data for timer unit generation.
type TimerUnitData struct {
	Name            string
	TimerDirectives string
}

// Generator generates systemd unit files.
type Generator struct {
	systemdDir  string // Full path to user systemd directory
	rclonePath  string // Path to rclone binary
	configPath  string // Path to rclone config file
	logDir      string // Directory for log files
}

// NewGenerator creates a new unit file generator.
func NewGenerator() (*Generator, error) {
	systemdDir, err := GetUserSystemdPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get systemd path: %w", err)
	}

	// Find rclone binary
	rclonePath, err := exec.LookPath("rclone")
	if err != nil {
		rclonePath = "/usr/bin/rclone" // Default fallback
	}

	// Get rclone config path
	configPath := getRcloneConfigPath()

	// Get log directory
	logDir, err := getLogDir()
	if err != nil {
		logDir = "/tmp" // Fallback
	}

	return &Generator{
		systemdDir: systemdDir,
		rclonePath: rclonePath,
		configPath: configPath,
		logDir:     logDir,
	}, nil
}

// GetSystemdDir returns the systemd user directory path.
func (g *Generator) GetSystemdDir() string {
	return g.systemdDir
}

// GenerateMountService generates a systemd service unit for an rclone mount.
func (g *Generator) GenerateMountService(mount *models.MountConfig) (string, error) {
	mountPoint := expandPath(mount.MountPoint)
	mountOptions := g.buildMountOptions(&mount.MountOptions)
	logPath := filepath.Join(g.logDir, fmt.Sprintf("rclone-mount-%s.log", sanitizeName(mount.Name)))

	data := MountUnitData{
		Name:         mount.Name,
		Remote:       mount.Remote,
		RemotePath:   mount.RemotePath,
		MountPoint:   mountPoint,
		MountOptions: mountOptions,
		LogPath:      logPath,
		RclonePath:   g.rclonePath,
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

	filename := g.ServiceName(mount.Name, "mount") + ".service"
	if err := g.WriteUnitFile(filename, content); err != nil {
		return "", fmt.Errorf("failed to write mount service file: %w", err)
	}

	return filepath.Join(g.systemdDir, filename), nil
}

// GenerateSyncService generates a systemd service unit for an rclone sync job.
func (g *Generator) GenerateSyncService(job *models.SyncJobConfig) (string, error) {
	syncOptions := g.buildSyncOptions(&job.SyncOptions)
	logPath := filepath.Join(g.logDir, fmt.Sprintf("rclone-sync-%s.log", sanitizeName(job.Name)))

	direction := job.SyncOptions.Direction
	if direction == "" {
		direction = "sync"
	}

	data := SyncUnitData{
		Name:        job.Name,
		Source:      job.Source,
		Destination: expandPath(job.Destination),
		Direction:   direction,
		SyncOptions: syncOptions,
		LogPath:     logPath,
		RclonePath:  g.rclonePath,
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

	data := TimerUnitData{
		Name:            job.Name,
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

	serviceFilename := g.ServiceName(job.Name, "sync") + ".service"
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

		timerFilename := g.ServiceName(job.Name, "sync") + ".timer"
		if err := g.WriteUnitFile(timerFilename, timerContent); err != nil {
			return servicePath, "", fmt.Errorf("failed to write sync timer file: %w", err)
		}
		timerPath = filepath.Join(g.systemdDir, timerFilename)
	}

	return servicePath, timerPath, nil
}

// ServiceName generates a sanitized systemd unit name.
func (g *Generator) ServiceName(mountOrJobName, unitType string) string {
	name := sanitizeName(mountOrJobName)
	return fmt.Sprintf("rclone-%s-%s", unitType, name)
}

// RemoveUnit removes a unit file from the systemd directory.
func (g *Generator) RemoveUnit(name string) error {
	path := filepath.Join(g.systemdDir, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to remove
	}
	return os.Remove(path)
}

// WriteUnitFile writes a unit file to the systemd user directory.
func (g *Generator) WriteUnitFile(filename, content string) error {
	// Ensure directory exists
	if err := os.MkdirAll(g.systemdDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd directory: %w", err)
	}

	path := filepath.Join(g.systemdDir, filename)
	return os.WriteFile(path, []byte(content), 0644)
}

// buildMountOptions builds the mount options string for rclone.
func (g *Generator) buildMountOptions(opts *models.MountOptions) string {
	var args []string

	// Config path
	configPath := opts.Config
	if configPath == "" {
		configPath = g.configPath
	}
	args = append(args, fmt.Sprintf("--config=%s", configPath))

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

	// Extra arguments
	if opts.ExtraArgs != "" {
		args = append(args, opts.ExtraArgs)
	}

	return strings.Join(args, " \\\n    ")
}

// buildSyncOptions builds the sync options string for rclone.
func (g *Generator) buildSyncOptions(opts *models.SyncOptions) string {
	var args []string

	// Config path
	configPath := opts.Config
	if configPath == "" {
		configPath = g.configPath
	}
	args = append(args, fmt.Sprintf("--config=%s", configPath))

	// Deletion handling
	if opts.DeleteExtraneous {
		if opts.DeleteAfter {
			args = append(args, "--delete-after")
		} else {
			args = append(args, "--delete-after")
		}
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

	// Extra arguments
	if opts.ExtraArgs != "" {
		args = append(args, opts.ExtraArgs)
	}

	return strings.Join(args, " \\\n    ")
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

// GetUserSystemdPath returns the path to the user's systemd unit directory.
func GetUserSystemdPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, "systemd", "user"), nil
}

// sanitizeName sanitizes a name for use in a systemd unit filename.
func sanitizeName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace spaces and special characters with dashes
	reg := regexp.MustCompile(`[^a-z0-9_-]`)
	name = reg.ReplaceAllString(name, "-")

	// Remove consecutive dashes
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	// Trim leading and trailing dashes
	name = strings.Trim(name, "-")

	return name
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// getRcloneConfigPath returns the path to the rclone config file.
func getRcloneConfigPath() string {
	// Check RCLONE_CONFIG environment variable
	if configPath := os.Getenv("RCLONE_CONFIG"); configPath != "" {
		return configPath
	}

	// Default location
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/home", os.Getenv("USER"), ".config", "rclone", "rclone.conf")
	}
	return filepath.Join(home, ".config", "rclone", "rclone.conf")
}

// getLogDir returns the directory for log files.
func getLogDir() (string, error) {
	// Use XDG_STATE_HOME if available, otherwise ~/.local/state
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir != "" {
		logDir := filepath.Join(stateDir, "rclone-mount-sync")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return "", err
		}
		return logDir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	logDir := filepath.Join(home, ".local", "state", "rclone-mount-sync")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", err
	}
	return logDir, nil
}
