package config

import (
	"testing"

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
