package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMountConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   MountConfig
		expected MountConfig
	}{
		{
			name: "full config",
			config: MountConfig{
				ID:          "test-id-1",
				Name:        "test-mount",
				Description: "Test mount description",
				Remote:      "gdrive:",
				RemotePath:  "/Music",
				MountPoint:  "/mnt/gdrive",
				MountOptions: MountOptions{
					AllowOther:   true,
					VFSCacheMode: "full",
					BufferSize:   "16M",
					LogLevel:     "INFO",
				},
				AutoStart:  true,
				Enabled:    true,
				CreatedAt:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				ModifiedAt: time.Date(2024, 1, 16, 14, 45, 0, 0, time.UTC),
			},
			expected: MountConfig{
				ID:          "test-id-1",
				Name:        "test-mount",
				Description: "Test mount description",
				Remote:      "gdrive:",
				RemotePath:  "/Music",
				MountPoint:  "/mnt/gdrive",
				AutoStart:   true,
				Enabled:     true,
			},
		},
		{
			name: "minimal config",
			config: MountConfig{
				ID:         "test-id-2",
				Name:       "minimal-mount",
				Remote:     "dropbox:",
				RemotePath: "/",
				MountPoint: "/mnt/dropbox",
			},
			expected: MountConfig{
				ID:         "test-id-2",
				Name:       "minimal-mount",
				Remote:     "dropbox:",
				RemotePath: "/",
				MountPoint: "/mnt/dropbox",
			},
		},
		{
			name: "empty values",
			config: MountConfig{
				ID:         "",
				Name:       "",
				Remote:     "",
				RemotePath: "",
				MountPoint: "",
			},
			expected: MountConfig{
				ID:         "",
				Name:       "",
				Remote:     "",
				RemotePath: "",
				MountPoint: "",
			},
		},
		{
			name: "special characters in name",
			config: MountConfig{
				ID:         "test-id-3",
				Name:       "test-mount-特殊字符-émojis-🎉",
				Remote:     "s3:bucket",
				RemotePath: "/path/with spaces",
				MountPoint: "/mnt/path with spaces",
			},
			expected: MountConfig{
				ID:         "test-id-3",
				Name:       "test-mount-特殊字符-émojis-🎉",
				Remote:     "s3:bucket",
				RemotePath: "/path/with spaces",
				MountPoint: "/mnt/path with spaces",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.ID != tt.expected.ID {
				t.Errorf("ID = %q, want %q", tt.config.ID, tt.expected.ID)
			}
			if tt.config.Name != tt.expected.Name {
				t.Errorf("Name = %q, want %q", tt.config.Name, tt.expected.Name)
			}
			if tt.config.Remote != tt.expected.Remote {
				t.Errorf("Remote = %q, want %q", tt.config.Remote, tt.expected.Remote)
			}
			if tt.config.RemotePath != tt.expected.RemotePath {
				t.Errorf("RemotePath = %q, want %q", tt.config.RemotePath, tt.expected.RemotePath)
			}
			if tt.config.MountPoint != tt.expected.MountPoint {
				t.Errorf("MountPoint = %q, want %q", tt.config.MountPoint, tt.expected.MountPoint)
			}
			if tt.config.AutoStart != tt.expected.AutoStart {
				t.Errorf("AutoStart = %v, want %v", tt.config.AutoStart, tt.expected.AutoStart)
			}
			if tt.config.Enabled != tt.expected.Enabled {
				t.Errorf("Enabled = %v, want %v", tt.config.Enabled, tt.expected.Enabled)
			}
		})
	}
}

func TestMountConfigJSONSerialization(t *testing.T) {
	tests := []struct {
		name    string
		config  MountConfig
		wantErr bool
	}{
		{
			name: "full config serialization",
			config: MountConfig{
				ID:          "json-test-1",
				Name:        "json-mount",
				Description: "JSON test mount",
				Remote:      "gdrive:",
				RemotePath:  "/Documents",
				MountPoint:  "/mnt/gdrive-docs",
				MountOptions: MountOptions{
					AllowOther:   true,
					VFSCacheMode: "full",
					BufferSize:   "32M",
					LogLevel:     "DEBUG",
				},
				AutoStart:  true,
				Enabled:    true,
				CreatedAt:  time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
				ModifiedAt: time.Date(2024, 6, 2, 13, 30, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "minimal config serialization",
			config: MountConfig{
				ID:         "json-test-2",
				Name:       "minimal-json",
				Remote:     "onedrive:",
				RemotePath: "/",
				MountPoint: "/mnt/onedrive",
			},
			wantErr: false,
		},
		{
			name: "with special characters",
			config: MountConfig{
				ID:          "json-test-3",
				Name:        "test-\"quotes\"-and\\slashes",
				Description: "Description with \n newlines \t tabs",
				Remote:      "remote:",
				RemotePath:  "/path/with/unicode/日本語",
				MountPoint:  "/mnt/unicode",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var unmarshaled MountConfig
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Errorf("json.Unmarshal() error = %v", err)
				return
			}

			if unmarshaled.ID != tt.config.ID {
				t.Errorf("ID = %q, want %q", unmarshaled.ID, tt.config.ID)
			}
			if unmarshaled.Name != tt.config.Name {
				t.Errorf("Name = %q, want %q", unmarshaled.Name, tt.config.Name)
			}
			if unmarshaled.Remote != tt.config.Remote {
				t.Errorf("Remote = %q, want %q", unmarshaled.Remote, tt.config.Remote)
			}
			if unmarshaled.RemotePath != tt.config.RemotePath {
				t.Errorf("RemotePath = %q, want %q", unmarshaled.RemotePath, tt.config.RemotePath)
			}
			if unmarshaled.MountPoint != tt.config.MountPoint {
				t.Errorf("MountPoint = %q, want %q", unmarshaled.MountPoint, tt.config.MountPoint)
			}
			if unmarshaled.AutoStart != tt.config.AutoStart {
				t.Errorf("AutoStart = %v, want %v", unmarshaled.AutoStart, tt.config.AutoStart)
			}
			if unmarshaled.Enabled != tt.config.Enabled {
				t.Errorf("Enabled = %v, want %v", unmarshaled.Enabled, tt.config.Enabled)
			}
			if !unmarshaled.CreatedAt.Equal(tt.config.CreatedAt) {
				t.Errorf("CreatedAt = %v, want %v", unmarshaled.CreatedAt, tt.config.CreatedAt)
			}
			if !unmarshaled.ModifiedAt.Equal(tt.config.ModifiedAt) {
				t.Errorf("ModifiedAt = %v, want %v", unmarshaled.ModifiedAt, tt.config.ModifiedAt)
			}
		})
	}
}

func TestMountOptions(t *testing.T) {
	tests := []struct {
		name    string
		options MountOptions
		check   func(t *testing.T, opts MountOptions)
	}{
		{
			name: "fuse options",
			options: MountOptions{
				AllowOther: true,
				AllowRoot:  true,
				Umask:      "002",
				UID:        1000,
				GID:        1000,
			},
			check: func(t *testing.T, opts MountOptions) {
				if !opts.AllowOther {
					t.Error("AllowOther should be true")
				}
				if !opts.AllowRoot {
					t.Error("AllowRoot should be true")
				}
				if opts.Umask != "002" {
					t.Errorf("Umask = %q, want %q", opts.Umask, "002")
				}
				if opts.UID != 1000 {
					t.Errorf("UID = %d, want %d", opts.UID, 1000)
				}
				if opts.GID != 1000 {
					t.Errorf("GID = %d, want %d", opts.GID, 1000)
				}
			},
		},
		{
			name: "performance options",
			options: MountOptions{
				BufferSize:       "16M",
				DirCacheTime:     "5m",
				VFSReadChunkSize: "64M",
				VFSCacheMode:     "full",
				VFSCacheMaxAge:   "24h",
				VFSCacheMaxSize:  "10G",
				VFSWriteBack:     "5s",
			},
			check: func(t *testing.T, opts MountOptions) {
				if opts.BufferSize != "16M" {
					t.Errorf("BufferSize = %q, want %q", opts.BufferSize, "16M")
				}
				if opts.VFSCacheMode != "full" {
					t.Errorf("VFSCacheMode = %q, want %q", opts.VFSCacheMode, "full")
				}
				if opts.VFSCacheMaxAge != "24h" {
					t.Errorf("VFSCacheMaxAge = %q, want %q", opts.VFSCacheMaxAge, "24h")
				}
				if opts.VFSWriteBack != "5s" {
					t.Errorf("VFSWriteBack = %q, want %q", opts.VFSWriteBack, "5s")
				}
			},
		},
		{
			name: "behavior options",
			options: MountOptions{
				NoModTime:  true,
				NoChecksum: true,
				ReadOnly:   true,
			},
			check: func(t *testing.T, opts MountOptions) {
				if !opts.NoModTime {
					t.Error("NoModTime should be true")
				}
				if !opts.NoChecksum {
					t.Error("NoChecksum should be true")
				}
				if !opts.ReadOnly {
					t.Error("ReadOnly should be true")
				}
			},
		},
		{
			name: "network options",
			options: MountOptions{
				ConnectTimeout: "30s",
				Timeout:        "5m",
			},
			check: func(t *testing.T, opts MountOptions) {
				if opts.ConnectTimeout != "30s" {
					t.Errorf("ConnectTimeout = %q, want %q", opts.ConnectTimeout, "30s")
				}
				if opts.Timeout != "5m" {
					t.Errorf("Timeout = %q, want %q", opts.Timeout, "5m")
				}
			},
		},
		{
			name: "logging and advanced options",
			options: MountOptions{
				LogLevel:  "DEBUG",
				Config:    "/etc/rclone/custom.conf",
				ExtraArgs: "--no-gzip-encoding",
			},
			check: func(t *testing.T, opts MountOptions) {
				if opts.LogLevel != "DEBUG" {
					t.Errorf("LogLevel = %q, want %q", opts.LogLevel, "DEBUG")
				}
				if opts.Config != "/etc/rclone/custom.conf" {
					t.Errorf("Config = %q, want %q", opts.Config, "/etc/rclone/custom.conf")
				}
				if opts.ExtraArgs != "--no-gzip-encoding" {
					t.Errorf("ExtraArgs = %q, want %q", opts.ExtraArgs, "--no-gzip-encoding")
				}
			},
		},
		{
			name:    "empty options",
			options: MountOptions{},
			check: func(t *testing.T, opts MountOptions) {
				if opts.AllowOther {
					t.Error("AllowOther should be false by default")
				}
				if opts.UID != 0 {
					t.Errorf("UID should be 0 by default, got %d", opts.UID)
				}
				if opts.BufferSize != "" {
					t.Errorf("BufferSize should be empty by default, got %q", opts.BufferSize)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.options)
		})
	}
}

func TestMountOptionsJSONSerialization(t *testing.T) {
	opts := MountOptions{
		AllowOther:       true,
		AllowRoot:        false,
		Umask:            "077",
		UID:              1001,
		GID:              1001,
		BufferSize:       "64M",
		DirCacheTime:     "10m",
		VFSReadChunkSize: "128M",
		VFSCacheMode:     "writes",
		VFSCacheMaxAge:   "48h",
		VFSCacheMaxSize:  "20G",
		VFSWriteBack:     "10s",
		NoModTime:        false,
		NoChecksum:       true,
		ReadOnly:         false,
		ConnectTimeout:   "60s",
		Timeout:          "10m",
		LogLevel:         "NOTICE",
		Config:           "/custom/config.conf",
		ExtraArgs:        "--verbose --stats 10s",
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled MountOptions
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.AllowOther != opts.AllowOther {
		t.Errorf("AllowOther = %v, want %v", unmarshaled.AllowOther, opts.AllowOther)
	}
	if unmarshaled.Umask != opts.Umask {
		t.Errorf("Umask = %q, want %q", unmarshaled.Umask, opts.Umask)
	}
	if unmarshaled.UID != opts.UID {
		t.Errorf("UID = %d, want %d", unmarshaled.UID, opts.UID)
	}
	if unmarshaled.BufferSize != opts.BufferSize {
		t.Errorf("BufferSize = %q, want %q", unmarshaled.BufferSize, opts.BufferSize)
	}
	if unmarshaled.VFSCacheMode != opts.VFSCacheMode {
		t.Errorf("VFSCacheMode = %q, want %q", unmarshaled.VFSCacheMode, opts.VFSCacheMode)
	}
	if unmarshaled.LogLevel != opts.LogLevel {
		t.Errorf("LogLevel = %q, want %q", unmarshaled.LogLevel, opts.LogLevel)
	}
	if unmarshaled.Config != opts.Config {
		t.Errorf("Config = %q, want %q", unmarshaled.Config, opts.Config)
	}
	if unmarshaled.ExtraArgs != opts.ExtraArgs {
		t.Errorf("ExtraArgs = %q, want %q", unmarshaled.ExtraArgs, opts.ExtraArgs)
	}
}

func TestSyncJobConfig(t *testing.T) {
	tests := []struct {
		name   string
		config SyncJobConfig
		check  func(t *testing.T, c SyncJobConfig)
	}{
		{
			name: "full sync config",
			config: SyncJobConfig{
				ID:          "sync-1",
				Name:        "photos-backup",
				Description: "Backup photos from Google Drive",
				Source:      "gdrive:/Photos",
				Destination: "/home/user/Backup/Photos",
				SyncOptions: SyncOptions{
					Direction:          "sync",
					ConflictResolution: "newer",
					DeleteExtraneous:   true,
					Transfers:          4,
					Checkers:           8,
					CheckSum:           true,
				},
				Schedule: ScheduleConfig{
					Type:       "timer",
					OnCalendar: "daily",
					Persistent: true,
				},
				AutoStart:  true,
				Enabled:    true,
				CreatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				ModifiedAt: time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
				LastRun:    time.Date(2024, 6, 20, 2, 0, 0, 0, time.UTC),
			},
			check: func(t *testing.T, c SyncJobConfig) {
				if c.ID != "sync-1" {
					t.Errorf("ID = %q, want %q", c.ID, "sync-1")
				}
				if c.Name != "photos-backup" {
					t.Errorf("Name = %q, want %q", c.Name, "photos-backup")
				}
				if c.Source != "gdrive:/Photos" {
					t.Errorf("Source = %q, want %q", c.Source, "gdrive:/Photos")
				}
				if c.Destination != "/home/user/Backup/Photos" {
					t.Errorf("Destination = %q, want %q", c.Destination, "/home/user/Backup/Photos")
				}
				if !c.AutoStart {
					t.Error("AutoStart should be true")
				}
				if !c.Enabled {
					t.Error("Enabled should be true")
				}
			},
		},
		{
			name: "minimal sync config",
			config: SyncJobConfig{
				ID:          "sync-2",
				Name:        "minimal-sync",
				Source:      "dropbox:/Documents",
				Destination: "/backup/docs",
			},
			check: func(t *testing.T, c SyncJobConfig) {
				if c.AutoStart {
					t.Error("AutoStart should be false by default")
				}
				if c.Enabled {
					t.Error("Enabled should be false by default")
				}
			},
		},
		{
			name: "copy direction",
			config: SyncJobConfig{
				ID:          "sync-3",
				Name:        "copy-job",
				Source:      "s3:bucket/data",
				Destination: "/local/backup",
				SyncOptions: SyncOptions{
					Direction: "copy",
				},
			},
			check: func(t *testing.T, c SyncJobConfig) {
				if c.SyncOptions.Direction != "copy" {
					t.Errorf("Direction = %q, want %q", c.SyncOptions.Direction, "copy")
				}
			},
		},
		{
			name: "move direction",
			config: SyncJobConfig{
				ID:          "sync-4",
				Name:        "move-job",
				Source:      "local:/temp/uploads",
				Destination: "gdrive:/Archive",
				SyncOptions: SyncOptions{
					Direction: "move",
				},
			},
			check: func(t *testing.T, c SyncJobConfig) {
				if c.SyncOptions.Direction != "move" {
					t.Errorf("Direction = %q, want %q", c.SyncOptions.Direction, "move")
				}
			},
		},
		{
			name: "special characters",
			config: SyncJobConfig{
				ID:          "sync-5",
				Name:        "特殊字符-sync-🎉",
				Description: "Description with \"quotes\" and 'apostrophes'",
				Source:      "remote:/path/with spaces/and/日本語",
				Destination: "/backup/path with spaces/日本語",
			},
			check: func(t *testing.T, c SyncJobConfig) {
				if c.Name != "特殊字符-sync-🎉" {
					t.Errorf("Name = %q, want %q", c.Name, "特殊字符-sync-🎉")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.config)
		})
	}
}

func TestSyncJobConfigJSONSerialization(t *testing.T) {
	config := SyncJobConfig{
		ID:          "json-sync-1",
		Name:        "json-sync-test",
		Description: "JSON serialization test",
		Source:      "gdrive:/Videos",
		Destination: "/backup/videos",
		SyncOptions: SyncOptions{
			Direction:          "sync",
			ConflictResolution: "newer",
			DeleteExtraneous:   true,
			DeleteAfter:        true,
			IncludePattern:     "*.mp4",
			ExcludePattern:     "*.tmp",
			MaxAge:             "365d",
			MinAge:             "1d",
			Transfers:          8,
			Checkers:           16,
			BandwidthLimit:     "50M",
			CheckSum:           true,
			DryRun:             false,
			LogLevel:           "INFO",
			Config:             "/etc/rclone/rclone.conf",
			ExtraArgs:          "--stats-one-line",
		},
		Schedule: ScheduleConfig{
			Type:               "timer",
			OnCalendar:         "*-*-* 03:00:00",
			OnBootSec:          "5min",
			OnActiveSec:        "1h",
			RandomizedDelaySec: "300",
			Persistent:         true,
		},
		AutoStart:  true,
		Enabled:    true,
		CreatedAt:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		ModifiedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		LastRun:    time.Date(2024, 6, 15, 3, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled SyncJobConfig
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.ID != config.ID {
		t.Errorf("ID = %q, want %q", unmarshaled.ID, config.ID)
	}
	if unmarshaled.Name != config.Name {
		t.Errorf("Name = %q, want %q", unmarshaled.Name, config.Name)
	}
	if unmarshaled.Source != config.Source {
		t.Errorf("Source = %q, want %q", unmarshaled.Source, config.Source)
	}
	if unmarshaled.Destination != config.Destination {
		t.Errorf("Destination = %q, want %q", unmarshaled.Destination, config.Destination)
	}
	if unmarshaled.SyncOptions.Direction != config.SyncOptions.Direction {
		t.Errorf("SyncOptions.Direction = %q, want %q", unmarshaled.SyncOptions.Direction, config.SyncOptions.Direction)
	}
	if unmarshaled.Schedule.Type != config.Schedule.Type {
		t.Errorf("Schedule.Type = %q, want %q", unmarshaled.Schedule.Type, config.Schedule.Type)
	}
	if !unmarshaled.CreatedAt.Equal(config.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", unmarshaled.CreatedAt, config.CreatedAt)
	}
	if !unmarshaled.ModifiedAt.Equal(config.ModifiedAt) {
		t.Errorf("ModifiedAt = %v, want %v", unmarshaled.ModifiedAt, config.ModifiedAt)
	}
	if !unmarshaled.LastRun.Equal(config.LastRun) {
		t.Errorf("LastRun = %v, want %v", unmarshaled.LastRun, config.LastRun)
	}
}

func TestSyncOptions(t *testing.T) {
	tests := []struct {
		name    string
		options SyncOptions
		check   func(t *testing.T, opts SyncOptions)
	}{
		{
			name: "full sync options",
			options: SyncOptions{
				Direction:          "sync",
				ConflictResolution: "newer",
				DeleteExtraneous:   true,
				DeleteAfter:        true,
				IncludePattern:     "*.jpg,*.png",
				ExcludePattern:     "*.tmp,*.bak",
				MaxAge:             "30d",
				MinAge:             "1h",
				Transfers:          10,
				Checkers:           20,
				BandwidthLimit:     "100M",
				CheckSum:           true,
				DryRun:             true,
				LogLevel:           "DEBUG",
				Config:             "/custom/rclone.conf",
				ExtraArgs:          "--verbose",
			},
			check: func(t *testing.T, opts SyncOptions) {
				if opts.Direction != "sync" {
					t.Errorf("Direction = %q, want %q", opts.Direction, "sync")
				}
				if opts.ConflictResolution != "newer" {
					t.Errorf("ConflictResolution = %q, want %q", opts.ConflictResolution, "newer")
				}
				if !opts.DeleteExtraneous {
					t.Error("DeleteExtraneous should be true")
				}
				if !opts.DeleteAfter {
					t.Error("DeleteAfter should be true")
				}
				if opts.Transfers != 10 {
					t.Errorf("Transfers = %d, want %d", opts.Transfers, 10)
				}
				if opts.Checkers != 20 {
					t.Errorf("Checkers = %d, want %d", opts.Checkers, 20)
				}
				if !opts.CheckSum {
					t.Error("CheckSum should be true")
				}
				if !opts.DryRun {
					t.Error("DryRun should be true")
				}
			},
		},
		{
			name: "copy options",
			options: SyncOptions{
				Direction:      "copy",
				BandwidthLimit: "10M",
				Transfers:      4,
			},
			check: func(t *testing.T, opts SyncOptions) {
				if opts.Direction != "copy" {
					t.Errorf("Direction = %q, want %q", opts.Direction, "copy")
				}
				if opts.BandwidthLimit != "10M" {
					t.Errorf("BandwidthLimit = %q, want %q", opts.BandwidthLimit, "10M")
				}
			},
		},
		{
			name:    "empty options",
			options: SyncOptions{},
			check: func(t *testing.T, opts SyncOptions) {
				if opts.Direction != "" {
					t.Errorf("Direction should be empty, got %q", opts.Direction)
				}
				if opts.DeleteExtraneous {
					t.Error("DeleteExtraneous should be false by default")
				}
				if opts.Transfers != 0 {
					t.Errorf("Transfers should be 0 by default, got %d", opts.Transfers)
				}
				if opts.CheckSum {
					t.Error("CheckSum should be false by default")
				}
			},
		},
		{
			name: "conflict resolution options",
			options: SyncOptions{
				ConflictResolution: "larger",
			},
			check: func(t *testing.T, opts SyncOptions) {
				if opts.ConflictResolution != "larger" {
					t.Errorf("ConflictResolution = %q, want %q", opts.ConflictResolution, "larger")
				}
			},
		},
		{
			name: "age filtering options",
			options: SyncOptions{
				MaxAge: "365d",
				MinAge: "7d",
			},
			check: func(t *testing.T, opts SyncOptions) {
				if opts.MaxAge != "365d" {
					t.Errorf("MaxAge = %q, want %q", opts.MaxAge, "365d")
				}
				if opts.MinAge != "7d" {
					t.Errorf("MinAge = %q, want %q", opts.MinAge, "7d")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.options)
		})
	}
}

func TestSyncOptionsJSONSerialization(t *testing.T) {
	opts := SyncOptions{
		Direction:          "sync",
		ConflictResolution: "none",
		DeleteExtraneous:   false,
		DeleteAfter:        true,
		IncludePattern:     "*",
		ExcludePattern:     "*.git/*",
		MaxAge:             "180d",
		MinAge:             "0",
		Transfers:          6,
		Checkers:           12,
		BandwidthLimit:     "25M",
		CheckSum:           false,
		DryRun:             true,
		LogLevel:           "NOTICE",
		Config:             "",
		ExtraArgs:          "--stats 30s",
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled SyncOptions
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.Direction != opts.Direction {
		t.Errorf("Direction = %q, want %q", unmarshaled.Direction, opts.Direction)
	}
	if unmarshaled.ConflictResolution != opts.ConflictResolution {
		t.Errorf("ConflictResolution = %q, want %q", unmarshaled.ConflictResolution, opts.ConflictResolution)
	}
	if unmarshaled.DeleteAfter != opts.DeleteAfter {
		t.Errorf("DeleteAfter = %v, want %v", unmarshaled.DeleteAfter, opts.DeleteAfter)
	}
	if unmarshaled.Transfers != opts.Transfers {
		t.Errorf("Transfers = %d, want %d", unmarshaled.Transfers, opts.Transfers)
	}
	if unmarshaled.DryRun != opts.DryRun {
		t.Errorf("DryRun = %v, want %v", unmarshaled.DryRun, opts.DryRun)
	}
	if unmarshaled.ExtraArgs != opts.ExtraArgs {
		t.Errorf("ExtraArgs = %q, want %q", unmarshaled.ExtraArgs, opts.ExtraArgs)
	}
}

func TestScheduleConfig(t *testing.T) {
	tests := []struct {
		name   string
		config ScheduleConfig
		check  func(t *testing.T, c ScheduleConfig)
	}{
		{
			name: "timer schedule",
			config: ScheduleConfig{
				Type:               "timer",
				OnCalendar:         "daily",
				OnBootSec:          "",
				OnActiveSec:        "",
				RandomizedDelaySec: "300",
				Persistent:         true,
			},
			check: func(t *testing.T, c ScheduleConfig) {
				if c.Type != "timer" {
					t.Errorf("Type = %q, want %q", c.Type, "timer")
				}
				if c.OnCalendar != "daily" {
					t.Errorf("OnCalendar = %q, want %q", c.OnCalendar, "daily")
				}
				if !c.Persistent {
					t.Error("Persistent should be true")
				}
			},
		},
		{
			name: "onboot schedule",
			config: ScheduleConfig{
				Type:       "onboot",
				OnBootSec:  "5min",
				Persistent: false,
			},
			check: func(t *testing.T, c ScheduleConfig) {
				if c.Type != "onboot" {
					t.Errorf("Type = %q, want %q", c.Type, "onboot")
				}
				if c.OnBootSec != "5min" {
					t.Errorf("OnBootSec = %q, want %q", c.OnBootSec, "5min")
				}
				if c.Persistent {
					t.Error("Persistent should be false")
				}
			},
		},
		{
			name: "manual schedule",
			config: ScheduleConfig{
				Type: "manual",
			},
			check: func(t *testing.T, c ScheduleConfig) {
				if c.Type != "manual" {
					t.Errorf("Type = %q, want %q", c.Type, "manual")
				}
			},
		},
		{
			name: "specific time schedule",
			config: ScheduleConfig{
				Type:       "timer",
				OnCalendar: "*-*-* 02:30:00",
				Persistent: true,
			},
			check: func(t *testing.T, c ScheduleConfig) {
				if c.OnCalendar != "*-*-* 02:30:00" {
					t.Errorf("OnCalendar = %q, want %q", c.OnCalendar, "*-*-* 02:30:00")
				}
			},
		},
		{
			name: "weekly schedule",
			config: ScheduleConfig{
				Type:       "timer",
				OnCalendar: "weekly",
				Persistent: true,
			},
			check: func(t *testing.T, c ScheduleConfig) {
				if c.OnCalendar != "weekly" {
					t.Errorf("OnCalendar = %q, want %q", c.OnCalendar, "weekly")
				}
			},
		},
		{
			name: "hourly schedule",
			config: ScheduleConfig{
				Type:       "timer",
				OnCalendar: "hourly",
				Persistent: false,
			},
			check: func(t *testing.T, c ScheduleConfig) {
				if c.OnCalendar != "hourly" {
					t.Errorf("OnCalendar = %q, want %q", c.OnCalendar, "hourly")
				}
			},
		},
		{
			name:   "empty config",
			config: ScheduleConfig{},
			check: func(t *testing.T, c ScheduleConfig) {
				if c.Type != "" {
					t.Errorf("Type should be empty, got %q", c.Type)
				}
				if c.Persistent {
					t.Error("Persistent should be false by default")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.config)
		})
	}
}

func TestScheduleConfigJSONSerialization(t *testing.T) {
	config := ScheduleConfig{
		Type:               "timer",
		OnCalendar:         "*-*-* 04:00:00",
		OnBootSec:          "10min",
		OnActiveSec:        "30min",
		RandomizedDelaySec: "600",
		Persistent:         true,
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled ScheduleConfig
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.Type != config.Type {
		t.Errorf("Type = %q, want %q", unmarshaled.Type, config.Type)
	}
	if unmarshaled.OnCalendar != config.OnCalendar {
		t.Errorf("OnCalendar = %q, want %q", unmarshaled.OnCalendar, config.OnCalendar)
	}
	if unmarshaled.OnBootSec != config.OnBootSec {
		t.Errorf("OnBootSec = %q, want %q", unmarshaled.OnBootSec, config.OnBootSec)
	}
	if unmarshaled.OnActiveSec != config.OnActiveSec {
		t.Errorf("OnActiveSec = %q, want %q", unmarshaled.OnActiveSec, config.OnActiveSec)
	}
	if unmarshaled.RandomizedDelaySec != config.RandomizedDelaySec {
		t.Errorf("RandomizedDelaySec = %q, want %q", unmarshaled.RandomizedDelaySec, config.RandomizedDelaySec)
	}
	if unmarshaled.Persistent != config.Persistent {
		t.Errorf("Persistent = %v, want %v", unmarshaled.Persistent, config.Persistent)
	}
}

func TestServiceStatus(t *testing.T) {
	tests := []struct {
		name   string
		status ServiceStatus
		check  func(t *testing.T, s ServiceStatus)
	}{
		{
			name: "active mount service",
			status: ServiceStatus{
				Name:        "gdrive-mount",
				Type:        "mount",
				UnitFile:    "rclone-mount@gdrive.service",
				LoadState:   "loaded",
				ActiveState: "active",
				SubState:    "running",
				Enabled:     true,
				MainPID:     12345,
				ActivatedAt: time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC),
				MountPoint:  "/mnt/gdrive",
				IsMounted:   true,
			},
			check: func(t *testing.T, s ServiceStatus) {
				if s.Name != "gdrive-mount" {
					t.Errorf("Name = %q, want %q", s.Name, "gdrive-mount")
				}
				if s.Type != "mount" {
					t.Errorf("Type = %q, want %q", s.Type, "mount")
				}
				if s.ActiveState != "active" {
					t.Errorf("ActiveState = %q, want %q", s.ActiveState, "active")
				}
				if !s.Enabled {
					t.Error("Enabled should be true")
				}
				if s.MainPID != 12345 {
					t.Errorf("MainPID = %d, want %d", s.MainPID, 12345)
				}
				if !s.IsMounted {
					t.Error("IsMounted should be true")
				}
			},
		},
		{
			name: "inactive mount service",
			status: ServiceStatus{
				Name:        "dropbox-mount",
				Type:        "mount",
				UnitFile:    "rclone-mount@dropbox.service",
				LoadState:   "loaded",
				ActiveState: "inactive",
				SubState:    "dead",
				Enabled:     false,
				InactiveAt:  time.Date(2024, 6, 2, 15, 30, 0, 0, time.UTC),
				MountPoint:  "/mnt/dropbox",
				IsMounted:   false,
			},
			check: func(t *testing.T, s ServiceStatus) {
				if s.ActiveState != "inactive" {
					t.Errorf("ActiveState = %q, want %q", s.ActiveState, "inactive")
				}
				if s.Enabled {
					t.Error("Enabled should be false")
				}
				if s.IsMounted {
					t.Error("IsMounted should be false")
				}
			},
		},
		{
			name: "failed service",
			status: ServiceStatus{
				Name:        "failed-mount",
				Type:        "mount",
				UnitFile:    "rclone-mount@failed.service",
				LoadState:   "loaded",
				ActiveState: "failed",
				SubState:    "failed",
				Enabled:     true,
				ExitCode:    1,
			},
			check: func(t *testing.T, s ServiceStatus) {
				if s.ActiveState != "failed" {
					t.Errorf("ActiveState = %q, want %q", s.ActiveState, "failed")
				}
				if s.ExitCode != 1 {
					t.Errorf("ExitCode = %d, want %d", s.ExitCode, 1)
				}
			},
		},
		{
			name: "active sync service",
			status: ServiceStatus{
				Name:        "photos-sync",
				Type:        "sync",
				UnitFile:    "rclone-sync@photos.service",
				LoadState:   "loaded",
				ActiveState: "active",
				SubState:    "running",
				Enabled:     true,
				LastRun:     time.Date(2024, 6, 15, 2, 0, 0, 0, time.UTC),
				NextRun:     time.Date(2024, 6, 16, 2, 0, 0, 0, time.UTC),
				TimerActive: true,
			},
			check: func(t *testing.T, s ServiceStatus) {
				if s.Type != "sync" {
					t.Errorf("Type = %q, want %q", s.Type, "sync")
				}
				if !s.TimerActive {
					t.Error("TimerActive should be true")
				}
				if s.NextRun.IsZero() {
					t.Error("NextRun should be set")
				}
			},
		},
		{
			name: "not-found service",
			status: ServiceStatus{
				Name:        "nonexistent",
				Type:        "mount",
				UnitFile:    "rclone-mount@nonexistent.service",
				LoadState:   "not-found",
				ActiveState: "inactive",
				SubState:    "dead",
				Enabled:     false,
			},
			check: func(t *testing.T, s ServiceStatus) {
				if s.LoadState != "not-found" {
					t.Errorf("LoadState = %q, want %q", s.LoadState, "not-found")
				}
			},
		},
		{
			name:   "empty status",
			status: ServiceStatus{},
			check: func(t *testing.T, s ServiceStatus) {
				if s.Name != "" {
					t.Errorf("Name should be empty, got %q", s.Name)
				}
				if s.Enabled {
					t.Error("Enabled should be false by default")
				}
				if s.MainPID != 0 {
					t.Errorf("MainPID should be 0 by default, got %d", s.MainPID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.status)
		})
	}
}

func TestServiceStatusJSONSerialization(t *testing.T) {
	status := ServiceStatus{
		Name:        "test-service",
		Type:        "mount",
		UnitFile:    "rclone-mount@test.service",
		LoadState:   "loaded",
		ActiveState: "active",
		SubState:    "running",
		Enabled:     true,
		MainPID:     54321,
		ExitCode:    0,
		ActivatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		InactiveAt:  time.Time{},
		MountPoint:  "/mnt/test",
		IsMounted:   true,
		LastRun:     time.Date(2024, 6, 20, 3, 0, 0, 0, time.UTC),
		NextRun:     time.Date(2024, 6, 21, 3, 0, 0, 0, time.UTC),
		TimerActive: true,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled ServiceStatus
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.Name != status.Name {
		t.Errorf("Name = %q, want %q", unmarshaled.Name, status.Name)
	}
	if unmarshaled.Type != status.Type {
		t.Errorf("Type = %q, want %q", unmarshaled.Type, status.Type)
	}
	if unmarshaled.ActiveState != status.ActiveState {
		t.Errorf("ActiveState = %q, want %q", unmarshaled.ActiveState, status.ActiveState)
	}
	if unmarshaled.MainPID != status.MainPID {
		t.Errorf("MainPID = %d, want %d", unmarshaled.MainPID, status.MainPID)
	}
	if !unmarshaled.ActivatedAt.Equal(status.ActivatedAt) {
		t.Errorf("ActivatedAt = %v, want %v", unmarshaled.ActivatedAt, status.ActivatedAt)
	}
	if !unmarshaled.LastRun.Equal(status.LastRun) {
		t.Errorf("LastRun = %v, want %v", unmarshaled.LastRun, status.LastRun)
	}
	if !unmarshaled.NextRun.Equal(status.NextRun) {
		t.Errorf("NextRun = %v, want %v", unmarshaled.NextRun, status.NextRun)
	}
}

func TestMountConfigZeroValues(t *testing.T) {
	config := MountConfig{}

	if config.ID != "" {
		t.Errorf("ID should be empty, got %q", config.ID)
	}
	if config.Name != "" {
		t.Errorf("Name should be empty, got %q", config.Name)
	}
	if config.AutoStart {
		t.Error("AutoStart should be false")
	}
	if config.Enabled {
		t.Error("Enabled should be false")
	}
	if !config.CreatedAt.IsZero() {
		t.Error("CreatedAt should be zero")
	}
	if !config.ModifiedAt.IsZero() {
		t.Error("ModifiedAt should be zero")
	}
}

func TestSyncJobConfigZeroValues(t *testing.T) {
	config := SyncJobConfig{}

	if config.ID != "" {
		t.Errorf("ID should be empty, got %q", config.ID)
	}
	if config.Name != "" {
		t.Errorf("Name should be empty, got %q", config.Name)
	}
	if config.AutoStart {
		t.Error("AutoStart should be false")
	}
	if config.Enabled {
		t.Error("Enabled should be false")
	}
	if !config.CreatedAt.IsZero() {
		t.Error("CreatedAt should be zero")
	}
	if !config.ModifiedAt.IsZero() {
		t.Error("ModifiedAt should be zero")
	}
	if !config.LastRun.IsZero() {
		t.Error("LastRun should be zero")
	}
}

func TestMountOptionsZeroValues(t *testing.T) {
	opts := MountOptions{}

	if opts.AllowOther {
		t.Error("AllowOther should be false")
	}
	if opts.AllowRoot {
		t.Error("AllowRoot should be false")
	}
	if opts.UID != 0 {
		t.Errorf("UID should be 0, got %d", opts.UID)
	}
	if opts.GID != 0 {
		t.Errorf("GID should be 0, got %d", opts.GID)
	}
	if opts.BufferSize != "" {
		t.Errorf("BufferSize should be empty, got %q", opts.BufferSize)
	}
	if opts.NoModTime {
		t.Error("NoModTime should be false")
	}
	if opts.ReadOnly {
		t.Error("ReadOnly should be false")
	}
}

func TestSyncOptionsZeroValues(t *testing.T) {
	opts := SyncOptions{}

	if opts.Direction != "" {
		t.Errorf("Direction should be empty, got %q", opts.Direction)
	}
	if opts.DeleteExtraneous {
		t.Error("DeleteExtraneous should be false")
	}
	if opts.Transfers != 0 {
		t.Errorf("Transfers should be 0, got %d", opts.Transfers)
	}
	if opts.CheckSum {
		t.Error("CheckSum should be false")
	}
	if opts.DryRun {
		t.Error("DryRun should be false")
	}
}

func TestScheduleConfigZeroValues(t *testing.T) {
	config := ScheduleConfig{}

	if config.Type != "" {
		t.Errorf("Type should be empty, got %q", config.Type)
	}
	if config.OnCalendar != "" {
		t.Errorf("OnCalendar should be empty, got %q", config.OnCalendar)
	}
	if config.Persistent {
		t.Error("Persistent should be false")
	}
}

func TestServiceStatusZeroValues(t *testing.T) {
	status := ServiceStatus{}

	if status.Name != "" {
		t.Errorf("Name should be empty, got %q", status.Name)
	}
	if status.Enabled {
		t.Error("Enabled should be false")
	}
	if status.MainPID != 0 {
		t.Errorf("MainPID should be 0, got %d", status.MainPID)
	}
	if status.IsMounted {
		t.Error("IsMounted should be false")
	}
	if status.TimerActive {
		t.Error("TimerActive should be false")
	}
}

func TestMountConfigNestedOptions(t *testing.T) {
	config := MountConfig{
		ID:   "nested-test",
		Name: "nested-options",
		MountOptions: MountOptions{
			AllowOther:   true,
			VFSCacheMode: "full",
			LogLevel:     "DEBUG",
		},
	}

	if !config.MountOptions.AllowOther {
		t.Error("Nested MountOptions.AllowOther should be true")
	}
	if config.MountOptions.VFSCacheMode != "full" {
		t.Errorf("Nested MountOptions.VFSCacheMode = %q, want %q", config.MountOptions.VFSCacheMode, "full")
	}
	if config.MountOptions.LogLevel != "DEBUG" {
		t.Errorf("Nested MountOptions.LogLevel = %q, want %q", config.MountOptions.LogLevel, "DEBUG")
	}
}

func TestSyncJobConfigNestedOptions(t *testing.T) {
	config := SyncJobConfig{
		ID:   "nested-sync-test",
		Name: "nested-sync-options",
		SyncOptions: SyncOptions{
			Direction: "sync",
			Transfers: 8,
			DryRun:    true,
		},
		Schedule: ScheduleConfig{
			Type:       "timer",
			OnCalendar: "daily",
			Persistent: true,
		},
	}

	if config.SyncOptions.Direction != "sync" {
		t.Errorf("Nested SyncOptions.Direction = %q, want %q", config.SyncOptions.Direction, "sync")
	}
	if config.SyncOptions.Transfers != 8 {
		t.Errorf("Nested SyncOptions.Transfers = %d, want %d", config.SyncOptions.Transfers, 8)
	}
	if !config.SyncOptions.DryRun {
		t.Error("Nested SyncOptions.DryRun should be true")
	}
	if config.Schedule.Type != "timer" {
		t.Errorf("Nested Schedule.Type = %q, want %q", config.Schedule.Type, "timer")
	}
	if !config.Schedule.Persistent {
		t.Error("Nested Schedule.Persistent should be true")
	}
}

func TestJSONEmptyVsOmitEmpty(t *testing.T) {
	config := MountConfig{
		ID:         "omitempty-test",
		Name:       "test",
		Remote:     "remote:",
		RemotePath: "/",
		MountPoint: "/mnt",
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)

	if config.Description != "" {
		t.Error("Description should be empty for this test")
	}

	requiredFields := []string{`"id"`, `"name"`, `"remote"`, `"remote_path"`, `"mount_point"`}
	for _, field := range requiredFields {
		if !contains(jsonStr, field) {
			t.Errorf("JSON should contain field %q", field)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
