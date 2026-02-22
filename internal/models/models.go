// Package models defines the core data structures for the rclone-mount-sync application.
package models

import (
	"time"
)

// MountConfig represents the configuration for an rclone mount.
type MountConfig struct {
	// Identification
	ID          string `json:"id" yaml:"id" mapstructure:"id"`
	Name        string `json:"name" yaml:"name" mapstructure:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description,omitempty"`

	// Rclone Configuration
	Remote     string `json:"remote" yaml:"remote" mapstructure:"remote"`                // e.g., "gdrive:"
	RemotePath string `json:"remote_path" yaml:"remote_path" mapstructure:"remote_path"` // e.g., "/" or "/Music"
	MountPoint string `json:"mount_point" yaml:"mount_point" mapstructure:"mount_point"` // Local mount path

	// Mount Options
	MountOptions MountOptions `json:"mount_options" yaml:"mount_options" mapstructure:"mount_options"`

	// Service Configuration
	AutoStart bool `json:"auto_start" yaml:"auto_start" mapstructure:"auto_start"`
	Enabled   bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// Metadata
	CreatedAt  time.Time `json:"created_at" yaml:"created_at" mapstructure:"created_at"`
	ModifiedAt time.Time `json:"modified_at" yaml:"modified_at" mapstructure:"modified_at"`
}

// MountOptions contains all configurable options for an rclone mount.
type MountOptions struct {
	// FUSE Options
	AllowOther bool   `json:"allow_other,omitempty" yaml:"allow_other,omitempty" mapstructure:"allow_other,omitempty"`
	AllowRoot  bool   `json:"allow_root,omitempty" yaml:"allow_root,omitempty" mapstructure:"allow_root,omitempty"`
	Umask      string `json:"umask,omitempty" yaml:"umask,omitempty" mapstructure:"umask,omitempty"` // e.g., "002"
	UID        int    `json:"uid,omitempty" yaml:"uid,omitempty" mapstructure:"uid,omitempty"`
	GID        int    `json:"gid,omitempty" yaml:"gid,omitempty" mapstructure:"gid,omitempty"`

	// Performance Options
	BufferSize       string `json:"buffer_size,omitempty" yaml:"buffer_size,omitempty" mapstructure:"buffer_size,omitempty"` // e.g., "16M"
	DirCacheTime     string `json:"dir_cache_time,omitempty" yaml:"dir_cache_time,omitempty" mapstructure:"dir_cache_time,omitempty"`
	VFSReadChunkSize string `json:"vfs_read_chunk_size,omitempty" yaml:"vfs_read_chunk_size,omitempty" mapstructure:"vfs_read_chunk_size,omitempty"`
	VFSCacheMode     string `json:"vfs_cache_mode,omitempty" yaml:"vfs_cache_mode,omitempty" mapstructure:"vfs_cache_mode,omitempty"`          // off, full, writes
	VFSCacheMaxAge   string `json:"vfs_cache_max_age,omitempty" yaml:"vfs_cache_max_age,omitempty" mapstructure:"vfs_cache_max_age,omitempty"` // e.g., "24h"
	VFSCacheMaxSize  string `json:"vfs_cache_max_size,omitempty" yaml:"vfs_cache_max_size,omitempty" mapstructure:"vfs_cache_max_size,omitempty"`
	VFSWriteBack     string `json:"vfs_write_back,omitempty" yaml:"vfs_write_back,omitempty" mapstructure:"vfs_write_back,omitempty"` // e.g., "5s"

	// Behavior Options
	NoModTime  bool `json:"no_modtime,omitempty" yaml:"no_modtime,omitempty" mapstructure:"no_modtime,omitempty"`
	NoChecksum bool `json:"no_checksum,omitempty" yaml:"no_checksum,omitempty" mapstructure:"no_checksum,omitempty"`
	ReadOnly   bool `json:"read_only,omitempty" yaml:"read_only,omitempty" mapstructure:"read_only,omitempty"`

	// Network Options
	ConnectTimeout string `json:"connect_timeout,omitempty" yaml:"connect_timeout,omitempty" mapstructure:"connect_timeout,omitempty"`
	Timeout        string `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout,omitempty"`

	// Logging Options
	LogLevel string `json:"log_level,omitempty" yaml:"log_level,omitempty" mapstructure:"log_level,omitempty"` // ERROR, NOTICE, INFO, DEBUG

	// Advanced
	Config    string `json:"config,omitempty" yaml:"config,omitempty" mapstructure:"config,omitempty"`             // Custom rclone config file
	ExtraArgs string `json:"extra_args,omitempty" yaml:"extra_args,omitempty" mapstructure:"extra_args,omitempty"` // Additional CLI args
}

// SyncJobConfig represents the configuration for an rclone sync job.
type SyncJobConfig struct {
	// Identification
	ID          string `json:"id" yaml:"id" mapstructure:"id"`
	Name        string `json:"name" yaml:"name" mapstructure:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description,omitempty"`

	// Rclone Configuration
	Source      string `json:"source" yaml:"source" mapstructure:"source"`                // e.g., "gdrive:/Photos"
	Destination string `json:"destination" yaml:"destination" mapstructure:"destination"` // e.g., "/home/user/Backup/Photos"

	// Sync Options
	SyncOptions SyncOptions `json:"sync_options" yaml:"sync_options" mapstructure:"sync_options"`

	// Schedule Configuration
	Schedule ScheduleConfig `json:"schedule" yaml:"schedule" mapstructure:"schedule"`

	// Service Configuration
	AutoStart bool `json:"auto_start" yaml:"auto_start" mapstructure:"auto_start"` // Start timer on boot
	Enabled   bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// Metadata
	CreatedAt  time.Time `json:"created_at" yaml:"created_at" mapstructure:"created_at"`
	ModifiedAt time.Time `json:"modified_at" yaml:"modified_at" mapstructure:"modified_at"`
	LastRun    time.Time `json:"last_run,omitempty" yaml:"last_run,omitempty" mapstructure:"last_run,omitempty"`
}

// SyncOptions contains all configurable options for an rclone sync job.
type SyncOptions struct {
	// Sync Direction & Behavior
	Direction string `json:"direction" yaml:"direction" mapstructure:"direction"` // "sync", "copy", "move"

	// Conflict Resolution
	ConflictResolution string `json:"conflict_resolution,omitempty" yaml:"conflict_resolution,omitempty" mapstructure:"conflict_resolution,omitempty"`
	// Options: "newer", "larger", "none"

	// Deletion Handling
	DeleteExtraneous bool `json:"delete_extraneous,omitempty" yaml:"delete_extraneous,omitempty" mapstructure:"delete_extraneous,omitempty"`
	DeleteAfter      bool `json:"delete_after,omitempty" yaml:"delete_after,omitempty" mapstructure:"delete_after,omitempty"`

	// Filtering
	IncludePattern string `json:"include_pattern,omitempty" yaml:"include_pattern,omitempty" mapstructure:"include_pattern,omitempty"`
	ExcludePattern string `json:"exclude_pattern,omitempty" yaml:"exclude_pattern,omitempty" mapstructure:"exclude_pattern,omitempty"`
	MaxAge         string `json:"max_age,omitempty" yaml:"max_age,omitempty" mapstructure:"max_age,omitempty"` // e.g., "30d"
	MinAge         string `json:"min_age,omitempty" yaml:"min_age,omitempty" mapstructure:"min_age,omitempty"`

	// Performance
	Transfers      int    `json:"transfers,omitempty" yaml:"transfers,omitempty" mapstructure:"transfers,omitempty"` // Parallel transfers
	Checkers       int    `json:"checkers,omitempty" yaml:"checkers,omitempty" mapstructure:"checkers,omitempty"`
	BandwidthLimit string `json:"bandwidth_limit,omitempty" yaml:"bandwidth_limit,omitempty" mapstructure:"bandwidth_limit,omitempty"` // e.g., "10M"

	// Verification
	CheckSum bool `json:"checksum,omitempty" yaml:"checksum,omitempty" mapstructure:"checksum,omitempty"`
	DryRun   bool `json:"dry_run,omitempty" yaml:"dry_run,omitempty" mapstructure:"dry_run,omitempty"`

	// Logging Options
	LogLevel string `json:"log_level,omitempty" yaml:"log_level,omitempty" mapstructure:"log_level,omitempty"` // ERROR, NOTICE, INFO, DEBUG

	// Advanced
	Config    string `json:"config,omitempty" yaml:"config,omitempty" mapstructure:"config,omitempty"`
	ExtraArgs string `json:"extra_args,omitempty" yaml:"extra_args,omitempty" mapstructure:"extra_args,omitempty"`
}

// ScheduleConfig defines the schedule for a sync job.
type ScheduleConfig struct {
	// Schedule Type
	Type string `json:"type" yaml:"type" mapstructure:"type"` // "timer", "onboot", "manual"

	// Timer Configuration (systemd timer syntax)
	OnCalendar         string `json:"on_calendar,omitempty" yaml:"on_calendar,omitempty" mapstructure:"on_calendar,omitempty"` // e.g., "daily", "*-*-* 02:00:00"
	OnBootSec          string `json:"on_boot_sec,omitempty" yaml:"on_boot_sec,omitempty" mapstructure:"on_boot_sec,omitempty"` // e.g., "5min"
	OnActiveSec        string `json:"on_active_sec,omitempty" yaml:"on_active_sec,omitempty" mapstructure:"on_active_sec,omitempty"`
	RandomizedDelaySec string `json:"randomized_delay_sec,omitempty" yaml:"randomized_delay_sec,omitempty" mapstructure:"randomized_delay_sec,omitempty"`
	Persistent         bool   `json:"persistent,omitempty" yaml:"persistent,omitempty" mapstructure:"persistent,omitempty"` // Catch up missed runs

	// Run Conditions
	RequireACPower   bool `json:"require_ac_power,omitempty" yaml:"require_ac_power,omitempty" mapstructure:"require_ac_power,omitempty"`    // Only run when on AC power
	RequireUnmetered bool `json:"require_unmetered,omitempty" yaml:"require_unmetered,omitempty" mapstructure:"require_unmetered,omitempty"` // Only run on non-metered connection
}

// ServiceStatus represents the status of a systemd service.
type ServiceStatus struct {
	Name     string `json:"name" mapstructure:"name"`
	Type     string `json:"type" mapstructure:"type"` // "mount" or "sync"
	UnitFile string `json:"unit_file" mapstructure:"unit_file"`

	// Systemd Status
	LoadState   string `json:"load_state" mapstructure:"load_state"`     // "loaded", "not-found", etc.
	ActiveState string `json:"active_state" mapstructure:"active_state"` // "active", "inactive", "failed"
	SubState    string `json:"sub_state" mapstructure:"sub_state"`       // "running", "exited", "dead", etc.

	// Service Details
	Enabled  bool `json:"enabled" mapstructure:"enabled"`
	MainPID  int  `json:"main_pid,omitempty" mapstructure:"main_pid,omitempty"`
	ExitCode int  `json:"exit_code,omitempty" mapstructure:"exit_code,omitempty"`

	// Timestamps
	ActivatedAt time.Time `json:"activated_at,omitempty" mapstructure:"activated_at,omitempty"`
	InactiveAt  time.Time `json:"inactive_at,omitempty" mapstructure:"inactive_at,omitempty"`

	// For mounts
	MountPoint string `json:"mount_point,omitempty" mapstructure:"mount_point,omitempty"`
	IsMounted  bool   `json:"is_mounted,omitempty" mapstructure:"is_mounted,omitempty"`

	// For sync jobs
	LastRun     time.Time `json:"last_run,omitempty" mapstructure:"last_run,omitempty"`
	NextRun     time.Time `json:"next_run,omitempty" mapstructure:"next_run,omitempty"`
	TimerActive bool      `json:"timer_active,omitempty" mapstructure:"timer_active,omitempty"`
}
