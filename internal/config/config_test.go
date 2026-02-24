package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dtg01100/rclone-mount-sync/internal/models"
)

func TestNewConfigWithDefaults(t *testing.T) {
	cfg := newConfigWithDefaults()

	if cfg.Version != "1.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1.0")
	}

	if cfg.Settings.DefaultMountDir != "~/mnt" {
		t.Errorf("DefaultMountDir = %q, want %q", cfg.Settings.DefaultMountDir, "~/mnt")
	}

	if cfg.Defaults.Mount.LogLevel != "INFO" {
		t.Errorf("Mount.LogLevel = %q, want %q", cfg.Defaults.Mount.LogLevel, "INFO")
	}

	if cfg.Defaults.Sync.Transfers != 4 {
		t.Errorf("Sync.Transfers = %d, want %d", cfg.Defaults.Sync.Transfers, 4)
	}

	if len(cfg.Mounts) != 0 {
		t.Errorf("Mounts length = %d, want 0", len(cfg.Mounts))
	}

	if len(cfg.SyncJobs) != 0 {
		t.Errorf("SyncJobs length = %d, want 0", len(cfg.SyncJobs))
	}
}

func TestConfigAddMount(t *testing.T) {
	cfg := newConfigWithDefaults()

	mount := models.MountConfig{
		Name:        "test-mount",
		Remote:      "gdrive:",
		RemotePath:  "/",
		MountPoint:  "/mnt/test",
		Description: "Test mount",
	}

	if err := cfg.AddMount(mount); err != nil {
		t.Errorf("AddMount() error = %v", err)
	}

	if len(cfg.Mounts) != 1 {
		t.Fatalf("Mounts length = %d, want 1", len(cfg.Mounts))
	}

	if cfg.Mounts[0].Name != "test-mount" {
		t.Errorf("Mount.Name = %q, want %q", cfg.Mounts[0].Name, "test-mount")
	}

	if cfg.Mounts[0].ID == "" {
		t.Error("Mount.ID should be generated")
	}

	if cfg.Mounts[0].CreatedAt.IsZero() {
		t.Error("Mount.CreatedAt should be set")
	}
}

func TestConfigAddMountDuplicate(t *testing.T) {
	cfg := newConfigWithDefaults()

	mount := models.MountConfig{
		Name:       "test-mount",
		Remote:     "gdrive:",
		MountPoint: "/mnt/test",
	}

	if err := cfg.AddMount(mount); err != nil {
		t.Errorf("AddMount() first call error = %v", err)
	}

	if err := cfg.AddMount(mount); err == nil {
		t.Error("AddMount() should return error for duplicate name")
	}
}

func TestConfigRemoveMount(t *testing.T) {
	cfg := newConfigWithDefaults()

	mount := models.MountConfig{
		Name:       "test-mount",
		Remote:     "gdrive:",
		MountPoint: "/mnt/test",
	}

	cfg.AddMount(mount)

	if err := cfg.RemoveMount("test-mount"); err != nil {
		t.Errorf("RemoveMount() error = %v", err)
	}

	if len(cfg.Mounts) != 0 {
		t.Errorf("Mounts length = %d, want 0", len(cfg.Mounts))
	}
}

func TestConfigRemoveMountNotFound(t *testing.T) {
	cfg := newConfigWithDefaults()

	if err := cfg.RemoveMount("nonexistent"); err == nil {
		t.Error("RemoveMount() should return error for nonexistent mount")
	}
}

func TestConfigGetMount(t *testing.T) {
	cfg := newConfigWithDefaults()

	mount := models.MountConfig{
		Name:       "test-mount",
		Remote:     "gdrive:",
		MountPoint: "/mnt/test",
	}

	cfg.AddMount(mount)

	result := cfg.GetMount("test-mount")
	if result == nil {
		t.Fatal("GetMount() returned nil")
	}

	if result.Name != "test-mount" {
		t.Errorf("GetMount().Name = %q, want %q", result.Name, "test-mount")
	}

	if result.Remote != "gdrive:" {
		t.Errorf("GetMount().Remote = %q, want %q", result.Remote, "gdrive:")
	}
}

func TestConfigGetMountNotFound(t *testing.T) {
	cfg := newConfigWithDefaults()

	result := cfg.GetMount("nonexistent")
	if result != nil {
		t.Errorf("GetMount() = %v, want nil", result)
	}
}

func TestConfigAddSyncJob(t *testing.T) {
	cfg := newConfigWithDefaults()

	job := models.SyncJobConfig{
		Name:        "test-sync",
		Source:      "gdrive:/Photos",
		Destination: "/home/user/Backup",
		Description: "Test sync job",
	}

	if err := cfg.AddSyncJob(job); err != nil {
		t.Errorf("AddSyncJob() error = %v", err)
	}

	if len(cfg.SyncJobs) != 1 {
		t.Fatalf("SyncJobs length = %d, want 1", len(cfg.SyncJobs))
	}

	if cfg.SyncJobs[0].Name != "test-sync" {
		t.Errorf("SyncJob.Name = %q, want %q", cfg.SyncJobs[0].Name, "test-sync")
	}

	if cfg.SyncJobs[0].ID == "" {
		t.Error("SyncJob.ID should be generated")
	}
}

func TestConfigAddSyncJobDuplicate(t *testing.T) {
	cfg := newConfigWithDefaults()

	job := models.SyncJobConfig{
		Name:        "test-sync",
		Source:      "gdrive:/Photos",
		Destination: "/home/user/Backup",
	}

	if err := cfg.AddSyncJob(job); err != nil {
		t.Errorf("AddSyncJob() first call error = %v", err)
	}

	if err := cfg.AddSyncJob(job); err == nil {
		t.Error("AddSyncJob() should return error for duplicate name")
	}
}

func TestConfigRemoveSyncJob(t *testing.T) {
	cfg := newConfigWithDefaults()

	job := models.SyncJobConfig{
		Name:        "test-sync",
		Source:      "gdrive:/Photos",
		Destination: "/home/user/Backup",
	}

	cfg.AddSyncJob(job)

	if err := cfg.RemoveSyncJob("test-sync"); err != nil {
		t.Errorf("RemoveSyncJob() error = %v", err)
	}

	if len(cfg.SyncJobs) != 0 {
		t.Errorf("SyncJobs length = %d, want 0", len(cfg.SyncJobs))
	}
}

func TestConfigRemoveSyncJobNotFound(t *testing.T) {
	cfg := newConfigWithDefaults()

	if err := cfg.RemoveSyncJob("nonexistent"); err == nil {
		t.Error("RemoveSyncJob() should return error for nonexistent job")
	}
}

func TestConfigGetSyncJob(t *testing.T) {
	cfg := newConfigWithDefaults()

	job := models.SyncJobConfig{
		Name:        "test-sync",
		Source:      "gdrive:/Photos",
		Destination: "/home/user/Backup",
	}

	cfg.AddSyncJob(job)

	result := cfg.GetSyncJob("test-sync")
	if result == nil {
		t.Fatal("GetSyncJob() returned nil")
	}

	if result.Name != "test-sync" {
		t.Errorf("GetSyncJob().Name = %q, want %q", result.Name, "test-sync")
	}

	if result.Source != "gdrive:/Photos" {
		t.Errorf("GetSyncJob().Source = %q, want %q", result.Source, "gdrive:/Photos")
	}
}

func TestConfigGetSyncJobNotFound(t *testing.T) {
	cfg := newConfigWithDefaults()

	result := cfg.GetSyncJob("nonexistent")
	if result != nil {
		t.Errorf("GetSyncJob() = %v, want nil", result)
	}
}

func TestConfigIDGeneration(t *testing.T) {
	cfg := newConfigWithDefaults()

	mount1 := models.MountConfig{
		Name:       "mount1",
		Remote:     "gdrive:",
		MountPoint: "/mnt/1",
	}
	mount2 := models.MountConfig{
		Name:       "mount2",
		Remote:     "gdrive:",
		MountPoint: "/mnt/2",
	}

	cfg.AddMount(mount1)
	cfg.AddMount(mount2)

	if cfg.Mounts[0].ID == cfg.Mounts[1].ID {
		t.Error("Mount IDs should be unique")
	}
}

func TestConfigSyncJobIDGeneration(t *testing.T) {
	cfg := newConfigWithDefaults()

	job1 := models.SyncJobConfig{
		Name:        "job1",
		Source:      "gdrive:/a",
		Destination: "/backup/a",
	}
	job2 := models.SyncJobConfig{
		Name:        "job2",
		Source:      "gdrive:/b",
		Destination: "/backup/b",
	}

	cfg.AddSyncJob(job1)
	cfg.AddSyncJob(job2)

	if cfg.SyncJobs[0].ID == cfg.SyncJobs[1].ID {
		t.Error("SyncJob IDs should be unique")
	}
}

func TestConfigPreservesExistingID(t *testing.T) {
	cfg := newConfigWithDefaults()

	mount := models.MountConfig{
		Name:       "test-mount",
		Remote:     "gdrive:",
		MountPoint: "/mnt/test",
		ID:         "custom-id-123",
	}

	cfg.AddMount(mount)

	if cfg.Mounts[0].ID != "custom-id-123" {
		t.Errorf("Mount.ID = %q, want %q", cfg.Mounts[0].ID, "custom-id-123")
	}
}

func TestAddRecentPath(t *testing.T) {
	tests := []struct {
		name          string
		initialPaths  []string
		addPath       string
		expectedPaths []string
		expectedCount int
	}{
		{
			name:          "add to empty list",
			initialPaths:  []string{},
			addPath:       "/home/user/docs",
			expectedPaths: []string{"/home/user/docs"},
			expectedCount: 1,
		},
		{
			name:          "add to existing list",
			initialPaths:  []string{"/home/user/a", "/home/user/b"},
			addPath:       "/home/user/c",
			expectedPaths: []string{"/home/user/c", "/home/user/a", "/home/user/b"},
			expectedCount: 3,
		},
		{
			name:          "duplicate moves to front",
			initialPaths:  []string{"/home/user/a", "/home/user/b", "/home/user/c"},
			addPath:       "/home/user/b",
			expectedPaths: []string{"/home/user/b", "/home/user/a", "/home/user/c"},
			expectedCount: 3,
		},
		{
			name:          "duplicate at front stays at front",
			initialPaths:  []string{"/home/user/a", "/home/user/b"},
			addPath:       "/home/user/a",
			expectedPaths: []string{"/home/user/a", "/home/user/b"},
			expectedCount: 2,
		},
		{
			name:          "max 10 items - truncates oldest",
			initialPaths:  []string{"/p1", "/p2", "/p3", "/p4", "/p5", "/p6", "/p7", "/p8", "/p9", "/p10"},
			addPath:       "/p11",
			expectedPaths: []string{"/p11", "/p1", "/p2", "/p3", "/p4", "/p5", "/p6", "/p7", "/p8", "/p9"},
			expectedCount: 10,
		},
		{
			name:          "max 10 items - existing at 10",
			initialPaths:  []string{"/p1", "/p2", "/p3", "/p4", "/p5", "/p6", "/p7", "/p8", "/p9", "/p10"},
			addPath:       "/p5",
			expectedPaths: []string{"/p5", "/p1", "/p2", "/p3", "/p4", "/p6", "/p7", "/p8", "/p9", "/p10"},
			expectedCount: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newConfigWithDefaults()
			cfg.Settings.RecentPaths = tt.initialPaths

			cfg.AddRecentPath(tt.addPath)

			if len(cfg.Settings.RecentPaths) != tt.expectedCount {
				t.Errorf("RecentPaths count = %d, want %d", len(cfg.Settings.RecentPaths), tt.expectedCount)
			}

			for i, expected := range tt.expectedPaths {
				if i >= len(cfg.Settings.RecentPaths) {
					t.Errorf("RecentPaths[%d] missing, want %q", i, expected)
					continue
				}
				if cfg.Settings.RecentPaths[i] != expected {
					t.Errorf("RecentPaths[%d] = %q, want %q", i, cfg.Settings.RecentPaths[i], expected)
				}
			}
		})
	}
}

func TestAddRecentPathMostRecentFirst(t *testing.T) {
	cfg := newConfigWithDefaults()

	paths := []string{"/a", "/b", "/c", "/d", "/e"}
	for _, p := range paths {
		cfg.AddRecentPath(p)
	}

	for i := 0; i < len(paths); i++ {
		expectedIdx := len(paths) - 1 - i
		if cfg.Settings.RecentPaths[i] != paths[expectedIdx] {
			t.Errorf("RecentPaths[%d] = %q, want %q", i, cfg.Settings.RecentPaths[i], paths[expectedIdx])
		}
	}
}

func TestAddRecentPathRemovesDuplicates(t *testing.T) {
	cfg := newConfigWithDefaults()

	cfg.AddRecentPath("/a")
	cfg.AddRecentPath("/b")
	cfg.AddRecentPath("/a")
	cfg.AddRecentPath("/c")
	cfg.AddRecentPath("/b")

	count := make(map[string]int)
	for _, p := range cfg.Settings.RecentPaths {
		count[p]++
		if count[p] > 1 {
			t.Errorf("Duplicate path %q found in RecentPaths", p)
		}
	}

	if len(cfg.Settings.RecentPaths) != 3 {
		t.Errorf("RecentPaths count = %d, want 3", len(cfg.Settings.RecentPaths))
	}

	if cfg.Settings.RecentPaths[0] != "/b" {
		t.Errorf("Most recent path = %q, want %q", cfg.Settings.RecentPaths[0], "/b")
	}
}

func TestAddMountTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		existing    []models.MountConfig
		add         models.MountConfig
		wantErr     bool
		errContains string
	}{
		{
			name:     "add new mount",
			existing: nil,
			add: models.MountConfig{
				Name:       "new-mount",
				Remote:     "gdrive:",
				MountPoint: "/mnt/new",
			},
			wantErr: false,
		},
		{
			name: "duplicate name",
			existing: []models.MountConfig{
				{Name: "existing-mount", Remote: "gdrive:", MountPoint: "/mnt/existing"},
			},
			add: models.MountConfig{
				Name:       "existing-mount",
				Remote:     "dropbox:",
				MountPoint: "/mnt/other",
			},
			wantErr:     true,
			errContains: "already exists",
		},
		{
			name:     "mount with existing ID preserved",
			existing: nil,
			add: models.MountConfig{
				Name:       "custom-id-mount",
				Remote:     "gdrive:",
				MountPoint: "/mnt/custom",
				ID:         "my-custom-id",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newConfigWithDefaults()
			for _, m := range tt.existing {
				cfg.Mounts = append(cfg.Mounts, m)
			}

			err := cfg.AddMount(tt.add)

			if tt.wantErr {
				if err == nil {
					t.Error("AddMount() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("AddMount() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("AddMount() unexpected error = %v", err)
				return
			}

			found := false
			for _, m := range cfg.Mounts {
				if m.Name == tt.add.Name {
					found = true
					if tt.add.ID != "" && m.ID != tt.add.ID {
						t.Errorf("Mount.ID = %q, want %q", m.ID, tt.add.ID)
					}
					break
				}
			}
			if !found {
				t.Error("Mount not found after AddMount()")
			}
		})
	}
}

func TestRemoveMountTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		existing    []models.MountConfig
		removeName  string
		wantErr     bool
		errContains string
	}{
		{
			name: "remove existing mount",
			existing: []models.MountConfig{
				{Name: "mount1"},
				{Name: "mount2"},
				{Name: "mount3"},
			},
			removeName: "mount2",
			wantErr:    false,
		},
		{
			name: "remove non-existent mount",
			existing: []models.MountConfig{
				{Name: "mount1"},
			},
			removeName:  "nonexistent",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:        "remove from empty list",
			existing:    nil,
			removeName:  "anything",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "remove first mount",
			existing: []models.MountConfig{
				{Name: "first"},
				{Name: "second"},
			},
			removeName: "first",
			wantErr:    false,
		},
		{
			name: "remove last mount",
			existing: []models.MountConfig{
				{Name: "first"},
				{Name: "last"},
			},
			removeName: "last",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newConfigWithDefaults()
			for _, m := range tt.existing {
				cfg.Mounts = append(cfg.Mounts, m)
			}

			initialCount := len(cfg.Mounts)
			err := cfg.RemoveMount(tt.removeName)

			if tt.wantErr {
				if err == nil {
					t.Error("RemoveMount() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("RemoveMount() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("RemoveMount() unexpected error = %v", err)
				return
			}

			if len(cfg.Mounts) != initialCount-1 {
				t.Errorf("Mounts count = %d, want %d", len(cfg.Mounts), initialCount-1)
			}

			for _, m := range cfg.Mounts {
				if m.Name == tt.removeName {
					t.Errorf("Mount %q still exists after removal", tt.removeName)
				}
			}
		})
	}
}

func TestGetMountTableDriven(t *testing.T) {
	tests := []struct {
		name     string
		existing []models.MountConfig
		getName  string
		wantNil  bool
		wantName string
	}{
		{
			name: "get existing mount",
			existing: []models.MountConfig{
				{Name: "mount1", Remote: "gdrive:"},
				{Name: "mount2", Remote: "dropbox:"},
			},
			getName:  "mount2",
			wantNil:  false,
			wantName: "mount2",
		},
		{
			name: "get non-existent mount",
			existing: []models.MountConfig{
				{Name: "mount1"},
			},
			getName: "nonexistent",
			wantNil: true,
		},
		{
			name:     "get from empty list",
			existing: nil,
			getName:  "anything",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newConfigWithDefaults()
			for _, m := range tt.existing {
				cfg.Mounts = append(cfg.Mounts, m)
			}

			result := cfg.GetMount(tt.getName)

			if tt.wantNil {
				if result != nil {
					t.Errorf("GetMount() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("GetMount() returned nil, expected non-nil")
			}

			if result.Name != tt.wantName {
				t.Errorf("GetMount().Name = %q, want %q", result.Name, tt.wantName)
			}
		})
	}
}

func TestAddSyncJobTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		existing    []models.SyncJobConfig
		add         models.SyncJobConfig
		wantErr     bool
		errContains string
	}{
		{
			name:     "add new sync job",
			existing: nil,
			add: models.SyncJobConfig{
				Name:        "new-job",
				Source:      "gdrive:/Photos",
				Destination: "/backup/photos",
			},
			wantErr: false,
		},
		{
			name: "duplicate name",
			existing: []models.SyncJobConfig{
				{Name: "existing-job", Source: "gdrive:/a", Destination: "/backup/a"},
			},
			add: models.SyncJobConfig{
				Name:        "existing-job",
				Source:      "dropbox:/b",
				Destination: "/backup/b",
			},
			wantErr:     true,
			errContains: "already exists",
		},
		{
			name:     "sync job with existing ID preserved",
			existing: nil,
			add: models.SyncJobConfig{
				Name:        "custom-id-job",
				Source:      "gdrive:/data",
				Destination: "/backup/data",
				ID:          "my-sync-id",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newConfigWithDefaults()
			for _, j := range tt.existing {
				cfg.SyncJobs = append(cfg.SyncJobs, j)
			}

			err := cfg.AddSyncJob(tt.add)

			if tt.wantErr {
				if err == nil {
					t.Error("AddSyncJob() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("AddSyncJob() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("AddSyncJob() unexpected error = %v", err)
				return
			}

			found := false
			for _, j := range cfg.SyncJobs {
				if j.Name == tt.add.Name {
					found = true
					if tt.add.ID != "" && j.ID != tt.add.ID {
						t.Errorf("SyncJob.ID = %q, want %q", j.ID, tt.add.ID)
					}
					break
				}
			}
			if !found {
				t.Error("SyncJob not found after AddSyncJob()")
			}
		})
	}
}

func TestRemoveSyncJobTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		existing    []models.SyncJobConfig
		removeName  string
		wantErr     bool
		errContains string
	}{
		{
			name: "remove existing sync job",
			existing: []models.SyncJobConfig{
				{Name: "job1"},
				{Name: "job2"},
				{Name: "job3"},
			},
			removeName: "job2",
			wantErr:    false,
		},
		{
			name: "remove non-existent sync job",
			existing: []models.SyncJobConfig{
				{Name: "job1"},
			},
			removeName:  "nonexistent",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:        "remove from empty list",
			existing:    nil,
			removeName:  "anything",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "remove first sync job",
			existing: []models.SyncJobConfig{
				{Name: "first"},
				{Name: "second"},
			},
			removeName: "first",
			wantErr:    false,
		},
		{
			name: "remove last sync job",
			existing: []models.SyncJobConfig{
				{Name: "first"},
				{Name: "last"},
			},
			removeName: "last",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newConfigWithDefaults()
			for _, j := range tt.existing {
				cfg.SyncJobs = append(cfg.SyncJobs, j)
			}

			initialCount := len(cfg.SyncJobs)
			err := cfg.RemoveSyncJob(tt.removeName)

			if tt.wantErr {
				if err == nil {
					t.Error("RemoveSyncJob() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("RemoveSyncJob() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("RemoveSyncJob() unexpected error = %v", err)
				return
			}

			if len(cfg.SyncJobs) != initialCount-1 {
				t.Errorf("SyncJobs count = %d, want %d", len(cfg.SyncJobs), initialCount-1)
			}

			for _, j := range cfg.SyncJobs {
				if j.Name == tt.removeName {
					t.Errorf("SyncJob %q still exists after removal", tt.removeName)
				}
			}
		})
	}
}

func TestGetSyncJobTableDriven(t *testing.T) {
	tests := []struct {
		name     string
		existing []models.SyncJobConfig
		getName  string
		wantNil  bool
		wantName string
	}{
		{
			name: "get existing sync job",
			existing: []models.SyncJobConfig{
				{Name: "job1", Source: "gdrive:/a"},
				{Name: "job2", Source: "dropbox:/b"},
			},
			getName:  "job2",
			wantNil:  false,
			wantName: "job2",
		},
		{
			name: "get non-existent sync job",
			existing: []models.SyncJobConfig{
				{Name: "job1"},
			},
			getName: "nonexistent",
			wantNil: true,
		},
		{
			name:     "get from empty list",
			existing: nil,
			getName:  "anything",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newConfigWithDefaults()
			for _, j := range tt.existing {
				cfg.SyncJobs = append(cfg.SyncJobs, j)
			}

			result := cfg.GetSyncJob(tt.getName)

			if tt.wantNil {
				if result != nil {
					t.Errorf("GetSyncJob() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("GetSyncJob() returned nil, expected non-nil")
			}

			if result.Name != tt.wantName {
				t.Errorf("GetSyncJob().Name = %q, want %q", result.Name, tt.wantName)
			}
		})
	}
}

func TestSaveAndLoadWithRecentPaths(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDir = origGetConfigDir }()

	cfg := newConfigWithDefaults()
	cfg.Settings.RecentPaths = []string{"/path/a", "/path/b", "/path/c"}
	cfg.Settings.DefaultMountDir = "/custom/mnt"
	cfg.Settings.RcloneBinaryPath = "/usr/local/bin/rclone"
	cfg.Settings.Editor = "vim"

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	configPath := filepath.Join(tmpDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded.Settings.RecentPaths) != 3 {
		t.Errorf("RecentPaths count = %d, want 3", len(loaded.Settings.RecentPaths))
	}

	for i, expected := range []string{"/path/a", "/path/b", "/path/c"} {
		if i >= len(loaded.Settings.RecentPaths) {
			t.Errorf("RecentPaths[%d] missing", i)
			continue
		}
		if loaded.Settings.RecentPaths[i] != expected {
			t.Errorf("RecentPaths[%d] = %q, want %q", i, loaded.Settings.RecentPaths[i], expected)
		}
	}

	if loaded.Settings.DefaultMountDir != "/custom/mnt" {
		t.Errorf("DefaultMountDir = %q, want %q", loaded.Settings.DefaultMountDir, "/custom/mnt")
	}

	if loaded.Settings.RcloneBinaryPath != "/usr/local/bin/rclone" {
		t.Errorf("RcloneBinaryPath = %q, want %q", loaded.Settings.RcloneBinaryPath, "/usr/local/bin/rclone")
	}

	if loaded.Settings.Editor != "vim" {
		t.Errorf("Editor = %q, want %q", loaded.Settings.Editor, "vim")
	}
}

func TestSaveWithMountsAndSyncJobs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDir = origGetConfigDir }()

	cfg := newConfigWithDefaults()

	cfg.AddMount(models.MountConfig{
		Name:       "test-mount",
		Remote:     "gdrive:",
		MountPoint: "/mnt/gdrive",
	})

	cfg.AddSyncJob(models.SyncJobConfig{
		Name:        "test-sync",
		Source:      "gdrive:/Photos",
		Destination: "/backup/photos",
	})

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded.Mounts) != 1 {
		t.Errorf("Mounts count = %d, want 1", len(loaded.Mounts))
	} else if loaded.Mounts[0].Name != "test-mount" {
		t.Errorf("Mounts[0].Name = %q, want %q", loaded.Mounts[0].Name, "test-mount")
	}

	if len(loaded.SyncJobs) != 1 {
		t.Errorf("SyncJobs count = %d, want 1", len(loaded.SyncJobs))
	} else if loaded.SyncJobs[0].Name != "test-sync" {
		t.Errorf("SyncJobs[0].Name = %q, want %q", loaded.SyncJobs[0].Name, "test-sync")
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configDir := filepath.Join(tmpDir, "nested", "config", "dir")

	origGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) { return configDir, nil }
	defer func() { getConfigDir = origGetConfigDir }()

	cfg := newConfigWithDefaults()
	cfg.Settings.DefaultMountDir = "~/test"

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("Config directory was not created")
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestLoadNonExistentConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDir = origGetConfigDir }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Version != "1.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1.0")
	}

	if len(cfg.Mounts) != 0 {
		t.Errorf("Mounts count = %d, want 0", len(cfg.Mounts))
	}

	if len(cfg.SyncJobs) != 0 {
		t.Errorf("SyncJobs count = %d, want 0", len(cfg.SyncJobs))
	}
}

func TestLoadExistingConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDir = origGetConfigDir }()

	cfg := newConfigWithDefaults()
	cfg.Settings.DefaultMountDir = "/existing/mnt"
	cfg.Settings.RecentPaths = []string{"/recent/1", "/recent/2"}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Settings.DefaultMountDir != "/existing/mnt" {
		t.Errorf("DefaultMountDir = %q, want %q", loaded.Settings.DefaultMountDir, "/existing/mnt")
	}

	if len(loaded.Settings.RecentPaths) != 2 {
		t.Errorf("RecentPaths count = %d, want 2", len(loaded.Settings.RecentPaths))
	}
}

func TestTimestampsSetOnAdd(t *testing.T) {
	cfg := newConfigWithDefaults()

	beforeAdd := time.Now()
	cfg.AddMount(models.MountConfig{
		Name:       "test",
		Remote:     "gdrive:",
		MountPoint: "/mnt/test",
	})

	mount := cfg.GetMount("test")
	if mount.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if mount.ModifiedAt.IsZero() {
		t.Error("ModifiedAt should be set")
	}
	if mount.CreatedAt != mount.ModifiedAt {
		t.Error("CreatedAt and ModifiedAt should be equal on add")
	}
	if mount.CreatedAt.Before(beforeAdd) {
		t.Error("CreatedAt should be after or equal to time before add")
	}
}

func TestSyncJobTimestampsSetOnAdd(t *testing.T) {
	cfg := newConfigWithDefaults()

	beforeAdd := time.Now()
	cfg.AddSyncJob(models.SyncJobConfig{
		Name:        "test",
		Source:      "gdrive:/src",
		Destination: "/dst",
	})

	job := cfg.GetSyncJob("test")
	if job.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if job.ModifiedAt.IsZero() {
		t.Error("ModifiedAt should be set")
	}
	if job.CreatedAt != job.ModifiedAt {
		t.Error("CreatedAt and ModifiedAt should be equal on add")
	}
	if job.CreatedAt.Before(beforeAdd) {
		t.Error("CreatedAt should be after or equal to time before add")
	}
}

func TestSaveCreatesBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDir = origGetConfigDir }()

	cfg := newConfigWithDefaults()
	cfg.Settings.DefaultMountDir = "/first/mnt"
	if err := cfg.Save(); err != nil {
		t.Fatalf("First Save() error = %v", err)
	}

	backupPath := filepath.Join(tmpDir, "config.yaml.bak")
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("Backup should not exist after first save (no existing config)")
	}

	cfg.Settings.DefaultMountDir = "/second/mnt"
	if err := cfg.Save(); err != nil {
		t.Fatalf("Second Save() error = %v", err)
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("Backup file should exist after second save")
	}

	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if !strings.Contains(string(backupContent), "/first/mnt") {
		t.Error("Backup should contain the first config")
	}

	configContent, err := os.ReadFile(filepath.Join(tmpDir, "config.yaml"))
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	if !strings.Contains(string(configContent), "/second/mnt") {
		t.Error("Config should contain the second value")
	}
}

func TestAtomicWriteTempFileCleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDir = origGetConfigDir }()

	cfg := newConfigWithDefaults()
	cfg.Settings.DefaultMountDir = "/original/mnt"
	if err := cfg.Save(); err != nil {
		t.Fatalf("First Save() error = %v", err)
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read config dir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp.yaml") {
			t.Error("Temp file should not remain after successful save")
		}
	}

	cfg.Settings.DefaultMountDir = "/new/mnt"
	if err := cfg.Save(); err != nil {
		t.Fatalf("Second Save() error = %v", err)
	}

	entries, err = os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read config dir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp.yaml") {
			t.Error("Temp file should not remain after successful save")
		}
	}

	configContent, err := os.ReadFile(filepath.Join(tmpDir, "config.yaml"))
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	if !strings.Contains(string(configContent), "/new/mnt") {
		t.Error("Config should contain the new value")
	}
}

func TestRestoreFromBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDir = origGetConfigDir }()

	cfg := newConfigWithDefaults()
	cfg.Settings.DefaultMountDir = "/original/mnt"
	if err := cfg.Save(); err != nil {
		t.Fatalf("First Save() error = %v", err)
	}

	cfg.Settings.DefaultMountDir = "/new/mnt"
	if err := cfg.Save(); err != nil {
		t.Fatalf("Second Save() error = %v", err)
	}

	hasBackup, err := HasBackup()
	if err != nil {
		t.Fatalf("HasBackup() error = %v", err)
	}
	if !hasBackup {
		t.Fatal("HasBackup() should return true")
	}

	if err := RestoreFromBackup(); err != nil {
		t.Fatalf("RestoreFromBackup() error = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Settings.DefaultMountDir != "/original/mnt" {
		t.Errorf("DefaultMountDir = %q, want %q", loaded.Settings.DefaultMountDir, "/original/mnt")
	}

	hasBackup, err = HasBackup()
	if err != nil {
		t.Fatalf("HasBackup() after restore error = %v", err)
	}
	if hasBackup {
		t.Error("HasBackup() should return false after restore (backup consumed)")
	}
}

func TestRestoreFromBackupNoBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDir = origGetConfigDir }()

	hasBackup, err := HasBackup()
	if err != nil {
		t.Fatalf("HasBackup() error = %v", err)
	}
	if hasBackup {
		t.Error("HasBackup() should return false when no backup exists")
	}

	err = RestoreFromBackup()
	if err == nil {
		t.Error("RestoreFromBackup() should return error when no backup exists")
	}
}

func TestBackupOnlyKeepsMostRecent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDir = origGetConfigDir }()

	cfg := newConfigWithDefaults()
	for i := 0; i < 5; i++ {
		cfg.Settings.DefaultMountDir = fmt.Sprintf("/mnt/%d", i)
		if err := cfg.Save(); err != nil {
			t.Fatalf("Save() iteration %d error = %v", i, err)
		}
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read config dir: %v", err)
	}

	backupCount := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".bak") {
			backupCount++
		}
	}

	if backupCount != 1 {
		t.Errorf("Expected 1 backup file, got %d", backupCount)
	}

	backupPath := filepath.Join(tmpDir, "config.yaml.bak")
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}

	if !strings.Contains(string(backupContent), "/mnt/3") {
		t.Errorf("Backup should contain the second-to-last config (/mnt/3)")
	}
}

func TestHasBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDir = origGetConfigDir }()

	hasBackup, err := HasBackup()
	if err != nil {
		t.Fatalf("HasBackup() error = %v", err)
	}
	if hasBackup {
		t.Error("HasBackup() should be false initially")
	}

	cfg := newConfigWithDefaults()
	cfg.Settings.DefaultMountDir = "/first"
	if err := cfg.Save(); err != nil {
		t.Fatalf("First Save() error = %v", err)
	}

	hasBackup, _ = HasBackup()
	if hasBackup {
		t.Error("HasBackup() should be false after first save (no prior config)")
	}

	cfg.Settings.DefaultMountDir = "/second"
	if err := cfg.Save(); err != nil {
		t.Fatalf("Second Save() error = %v", err)
	}

	hasBackup, err = HasBackup()
	if err != nil {
		t.Fatalf("HasBackup() after second save error = %v", err)
	}
	if !hasBackup {
		t.Error("HasBackup() should be true after second save")
	}
}

func TestExportConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := newConfigWithDefaults()
	cfg.AddMount(models.MountConfig{
		Name:       "test-mount",
		Remote:     "gdrive:",
		MountPoint: "/mnt/test",
	})
	cfg.AddSyncJob(models.SyncJobConfig{
		Name:        "test-sync",
		Source:      "gdrive:/Photos",
		Destination: "/backup/photos",
	})

	tests := []struct {
		name     string
		filePath string
	}{
		{"export to YAML", filepath.Join(tmpDir, "export.yaml")},
		{"export to JSON", filepath.Join(tmpDir, "export.json")},
		{"export to YML extension", filepath.Join(tmpDir, "export.yml")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := cfg.ExportConfig(tt.filePath); err != nil {
				t.Fatalf("ExportConfig() error = %v", err)
			}

			if _, err := os.Stat(tt.filePath); os.IsNotExist(err) {
				t.Fatal("Export file was not created")
			}

			content, err := os.ReadFile(tt.filePath)
			if err != nil {
				t.Fatalf("Failed to read export file: %v", err)
			}

			if len(content) == 0 {
				t.Error("Export file is empty")
			}
		})
	}
}

func TestExportConfigUnsupportedFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := newConfigWithDefaults()

	err = cfg.ExportConfig(filepath.Join(tmpDir, "export.txt"))
	if err == nil {
		t.Error("ExportConfig() should return error for unsupported format")
	}
}

func TestExportConfigCreatesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := newConfigWithDefaults()

	exportPath := filepath.Join(tmpDir, "nested", "dir", "export.yaml")
	if err := cfg.ExportConfig(exportPath); err != nil {
		t.Fatalf("ExportConfig() error = %v", err)
	}

	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Error("Export file was not created in nested directory")
	}
}

func TestImportConfigYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	exportPath := filepath.Join(tmpDir, "test-export.yaml")
	exportContent := `version: "1.0"
mounts:
  - id: mount1
    name: imported-mount
    remote: "gdrive:"
    remote_path: /
    mount_point: /mnt/imported
    enabled: true
sync_jobs:
  - id: sync1
    name: imported-sync
    source: "gdrive:/Docs"
    destination: /backup/docs
    enabled: true
exported: "2024-01-01T00:00:00Z"
`
	if err := os.WriteFile(exportPath, []byte(exportContent), 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	cfg := newConfigWithDefaults()
	if err := cfg.ImportConfig(exportPath, ImportModeMerge); err != nil {
		t.Fatalf("ImportConfig() error = %v", err)
	}

	if len(cfg.Mounts) != 1 {
		t.Errorf("Mounts count = %d, want 1", len(cfg.Mounts))
	} else if cfg.Mounts[0].Name != "imported-mount" {
		t.Errorf("Mount name = %q, want 'imported-mount'", cfg.Mounts[0].Name)
	}

	if len(cfg.SyncJobs) != 1 {
		t.Errorf("SyncJobs count = %d, want 1", len(cfg.SyncJobs))
	} else if cfg.SyncJobs[0].Name != "imported-sync" {
		t.Errorf("SyncJob name = %q, want 'imported-sync'", cfg.SyncJobs[0].Name)
	}
}

func TestImportConfigJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	exportPath := filepath.Join(tmpDir, "test-export.json")
	exportContent := `{
  "version": "1.0",
  "mounts": [
    {
      "id": "mount1",
      "name": "json-mount",
      "remote": "dropbox:",
      "remote_path": "/",
      "mount_point": "/mnt/json",
      "enabled": true
    }
  ],
  "sync_jobs": [],
  "exported": "2024-01-01T00:00:00Z"
}`
	if err := os.WriteFile(exportPath, []byte(exportContent), 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	cfg := newConfigWithDefaults()
	if err := cfg.ImportConfig(exportPath, ImportModeMerge); err != nil {
		t.Fatalf("ImportConfig() error = %v", err)
	}

	if len(cfg.Mounts) != 1 {
		t.Errorf("Mounts count = %d, want 1", len(cfg.Mounts))
	} else if cfg.Mounts[0].Name != "json-mount" {
		t.Errorf("Mount name = %q, want 'json-mount'", cfg.Mounts[0].Name)
	}
}

func TestImportConfigFileNotExist(t *testing.T) {
	cfg := newConfigWithDefaults()

	err := cfg.ImportConfig("/nonexistent/file.yaml", ImportModeMerge)
	if err == nil {
		t.Error("ImportConfig() should return error for non-existent file")
	}
}

func TestImportConfigInvalidFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	exportPath := filepath.Join(tmpDir, "invalid.txt")
	if err := os.WriteFile(exportPath, []byte("invalid content"), 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	cfg := newConfigWithDefaults()
	err = cfg.ImportConfig(exportPath, ImportModeMerge)
	if err == nil {
		t.Error("ImportConfig() should return error for unsupported format")
	}
}

func TestImportConfigInvalidContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	exportPath := filepath.Join(tmpDir, "empty.yaml")
	if err := os.WriteFile(exportPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	cfg := newConfigWithDefaults()
	err = cfg.ImportConfig(exportPath, ImportModeMerge)
	if err == nil {
		t.Error("ImportConfig() should return error for invalid/empty config")
	}
}

func TestImportConfigMergeMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	exportPath := filepath.Join(tmpDir, "merge-test.yaml")
	exportContent := `version: "1.0"
mounts:
  - id: new-mount
    name: new-mount
    remote: "gdrive:"
    remote_path: /
    mount_point: /mnt/new
    enabled: true
sync_jobs:
  - id: new-sync
    name: new-sync
    source: "gdrive:/New"
    destination: /backup/new
    enabled: true
exported: "2024-01-01T00:00:00Z"
`
	if err := os.WriteFile(exportPath, []byte(exportContent), 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	cfg := newConfigWithDefaults()
	cfg.AddMount(models.MountConfig{
		Name:       "existing-mount",
		Remote:     "dropbox:",
		MountPoint: "/mnt/existing",
	})
	cfg.AddSyncJob(models.SyncJobConfig{
		Name:        "existing-sync",
		Source:      "dropbox:/Docs",
		Destination: "/backup/docs",
	})

	if err := cfg.ImportConfig(exportPath, ImportModeMerge); err != nil {
		t.Fatalf("ImportConfig() error = %v", err)
	}

	if len(cfg.Mounts) != 2 {
		t.Errorf("Mounts count = %d, want 2", len(cfg.Mounts))
	}

	if len(cfg.SyncJobs) != 2 {
		t.Errorf("SyncJobs count = %d, want 2", len(cfg.SyncJobs))
	}

	mountNames := make(map[string]bool)
	for _, m := range cfg.Mounts {
		mountNames[m.Name] = true
	}
	if !mountNames["existing-mount"] || !mountNames["new-mount"] {
		t.Error("Merge should keep existing and add new mounts")
	}
}

func TestImportConfigMergeModeDuplicateNames(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	exportPath := filepath.Join(tmpDir, "dup-test.yaml")
	exportContent := `version: "1.0"
mounts:
  - id: imported-mount
    name: duplicate-name
    remote: "gdrive:"
    remote_path: /
    mount_point: /mnt/imported
    enabled: true
sync_jobs: []
exported: "2024-01-01T00:00:00Z"
`
	if err := os.WriteFile(exportPath, []byte(exportContent), 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	cfg := newConfigWithDefaults()
	cfg.AddMount(models.MountConfig{
		Name:       "duplicate-name",
		Remote:     "dropbox:",
		MountPoint: "/mnt/existing",
	})

	if err := cfg.ImportConfig(exportPath, ImportModeMerge); err != nil {
		t.Fatalf("ImportConfig() error = %v", err)
	}

	if len(cfg.Mounts) != 1 {
		t.Errorf("Mounts count = %d, want 1 (duplicate should be skipped)", len(cfg.Mounts))
	}

	if cfg.Mounts[0].Remote != "dropbox:" {
		t.Error("Existing mount should be kept, not replaced by imported")
	}
}

func TestImportConfigReplaceMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	exportPath := filepath.Join(tmpDir, "replace-test.yaml")
	exportContent := `version: "1.0"
mounts:
  - id: replaced-mount
    name: replaced-mount
    remote: "gdrive:"
    remote_path: /
    mount_point: /mnt/replaced
    enabled: true
sync_jobs:
  - id: replaced-sync
    name: replaced-sync
    source: "gdrive:/Replaced"
    destination: /backup/replaced
    enabled: true
exported: "2024-01-01T00:00:00Z"
`
	if err := os.WriteFile(exportPath, []byte(exportContent), 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	cfg := newConfigWithDefaults()
	cfg.AddMount(models.MountConfig{
		Name:       "old-mount",
		Remote:     "dropbox:",
		MountPoint: "/mnt/old",
	})
	cfg.AddSyncJob(models.SyncJobConfig{
		Name:        "old-sync",
		Source:      "dropbox:/Old",
		Destination: "/backup/old",
	})

	if err := cfg.ImportConfig(exportPath, ImportModeReplace); err != nil {
		t.Fatalf("ImportConfig() error = %v", err)
	}

	if len(cfg.Mounts) != 1 {
		t.Errorf("Mounts count = %d, want 1", len(cfg.Mounts))
	}

	if cfg.Mounts[0].Name != "replaced-mount" {
		t.Errorf("Mount name = %q, want 'replaced-mount'", cfg.Mounts[0].Name)
	}

	if len(cfg.SyncJobs) != 1 {
		t.Errorf("SyncJobs count = %d, want 1", len(cfg.SyncJobs))
	}

	if cfg.SyncJobs[0].Name != "replaced-sync" {
		t.Errorf("SyncJob name = %q, want 'replaced-sync'", cfg.SyncJobs[0].Name)
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origConfig := newConfigWithDefaults()
	origConfig.AddMount(models.MountConfig{
		Name:       "mount1",
		Remote:     "gdrive:",
		MountPoint: "/mnt/gdrive",
	})
	origConfig.AddMount(models.MountConfig{
		Name:       "mount2",
		Remote:     "dropbox:",
		MountPoint: "/mnt/dropbox",
	})
	origConfig.AddSyncJob(models.SyncJobConfig{
		Name:        "sync1",
		Source:      "gdrive:/Photos",
		Destination: "/backup/photos",
	})

	exportPath := filepath.Join(tmpDir, "roundtrip.yaml")
	if err := origConfig.ExportConfig(exportPath); err != nil {
		t.Fatalf("ExportConfig() error = %v", err)
	}

	newConfig := newConfigWithDefaults()
	if err := newConfig.ImportConfig(exportPath, ImportModeReplace); err != nil {
		t.Fatalf("ImportConfig() error = %v", err)
	}

	if len(newConfig.Mounts) != len(origConfig.Mounts) {
		t.Errorf("Mounts count mismatch: got %d, want %d", len(newConfig.Mounts), len(origConfig.Mounts))
	}

	if len(newConfig.SyncJobs) != len(origConfig.SyncJobs) {
		t.Errorf("SyncJobs count mismatch: got %d, want %d", len(newConfig.SyncJobs), len(origConfig.SyncJobs))
	}

	for i := range origConfig.Mounts {
		if newConfig.Mounts[i].Name != origConfig.Mounts[i].Name {
			t.Errorf("Mount[%d].Name = %q, want %q", i, newConfig.Mounts[i].Name, origConfig.Mounts[i].Name)
		}
		if newConfig.Mounts[i].Remote != origConfig.Mounts[i].Remote {
			t.Errorf("Mount[%d].Remote = %q, want %q", i, newConfig.Mounts[i].Remote, origConfig.Mounts[i].Remote)
		}
	}
}

func TestImportConfigGeneratesMissingIDs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	exportPath := filepath.Join(tmpDir, "no-ids.yaml")
	exportContent := `version: "1.0"
mounts:
  - name: no-id-mount
    remote: "gdrive:"
    remote_path: /
    mount_point: /mnt/noid
sync_jobs:
  - name: no-id-sync
    source: "gdrive:/Docs"
    destination: /backup/docs
exported: "2024-01-01T00:00:00Z"
`
	if err := os.WriteFile(exportPath, []byte(exportContent), 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	cfg := newConfigWithDefaults()
	if err := cfg.ImportConfig(exportPath, ImportModeMerge); err != nil {
		t.Fatalf("ImportConfig() error = %v", err)
	}

	if cfg.Mounts[0].ID == "" {
		t.Error("Mount ID should be generated when missing")
	}

	if cfg.SyncJobs[0].ID == "" {
		t.Error("SyncJob ID should be generated when missing")
	}
}

func TestCreateBackupSourceNotExist(t *testing.T) {
	err := createBackup("/nonexistent/path/to/config.yaml", "/tmp/backup.yaml")
	if err == nil {
		t.Error("createBackup() should return error when source file doesn't exist")
	}
	if !strings.Contains(err.Error(), "failed to open config file") {
		t.Errorf("createBackup() error = %v, want error containing 'failed to open config file'", err)
	}
}

func TestCreateBackupSourcePermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Test requires non-root user to test permission denied")
	}

	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(srcPath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	if err := os.Chmod(srcPath, 0000); err != nil {
		t.Fatalf("Failed to chmod source file: %v", err)
	}
	defer os.Chmod(srcPath, 0644)

	backupPath := filepath.Join(tmpDir, "config.yaml.bak")
	err = createBackup(srcPath, backupPath)
	if err == nil {
		t.Error("createBackup() should return error when source file can't be read")
	}
	if !strings.Contains(err.Error(), "failed to open config file") {
		t.Errorf("createBackup() error = %v, want error containing 'failed to open config file'", err)
	}
}

func TestCreateBackupDestPermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Test requires non-root user to test permission denied")
	}

	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(srcPath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	destDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(destDir, 0555); err != nil {
		t.Fatalf("Failed to create readonly dir: %v", err)
	}
	defer os.Chmod(destDir, 0755)

	backupPath := filepath.Join(destDir, "config.yaml.bak")
	err = createBackup(srcPath, backupPath)
	if err == nil {
		t.Error("createBackup() should return error when destination can't be created")
	}
	if !strings.Contains(err.Error(), "failed to create backup file") {
		t.Errorf("createBackup() error = %v, want error containing 'failed to create backup file'", err)
	}
}

func TestCreateBackupSuccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcContent := "test config content\nwith multiple lines\n"
	srcPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(srcPath, []byte(srcContent), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	backupPath := filepath.Join(tmpDir, "config.yaml.bak")
	if err := createBackup(srcPath, backupPath); err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if string(backupContent) != srcContent {
		t.Errorf("Backup content = %q, want %q", string(backupContent), srcContent)
	}

	srcInfo, _ := os.Stat(srcPath)
	backupInfo, _ := os.Stat(backupPath)
	if srcInfo.Mode() != backupInfo.Mode() {
		t.Errorf("Backup mode = %v, want %v", backupInfo.Mode(), srcInfo.Mode())
	}
}
