package config

import (
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
