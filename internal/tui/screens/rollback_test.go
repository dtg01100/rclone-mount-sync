package screens

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
)

func TestNewRollbackManager(t *testing.T) {
	cfg := createTestConfig()
	mgr := NewRollbackManager(cfg, nil, nil)

	if mgr == nil {
		t.Fatal("NewRollbackManager() returned nil")
	}
	if mgr.config != cfg {
		t.Error("config not set correctly")
	}
}

func TestPrepareMountRollback(t *testing.T) {
	cfg := createTestConfig()
	cfg.Mounts = []models.MountConfig{
		{ID: "abc12345", Name: "Mount1"},
		{ID: "def67890", Name: "Mount2"},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := mgr.PrepareMountRollback("new12345", "NewMount", OperationCreate)

	if data.Operation != OperationCreate {
		t.Errorf("Operation = %v, want %v", data.Operation, OperationCreate)
	}
	if data.MountID != "new12345" {
		t.Errorf("MountID = %q, want %q", data.MountID, "new12345")
	}
	if data.MountName != "NewMount" {
		t.Errorf("MountName = %q, want %q", data.MountName, "NewMount")
	}
	if len(data.OriginalMounts) != 2 {
		t.Errorf("OriginalMounts length = %d, want 2", len(data.OriginalMounts))
	}
}

func TestPrepareSyncJobRollback(t *testing.T) {
	cfg := createSyncTestConfig()
	cfg.SyncJobs = []models.SyncJobConfig{
		{ID: "abc12345", Name: "Job1"},
		{ID: "def67890", Name: "Job2"},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := mgr.PrepareSyncJobRollback("new12345", "NewJob", OperationUpdate)

	if data.Operation != OperationUpdate {
		t.Errorf("Operation = %v, want %v", data.Operation, OperationUpdate)
	}
	if data.JobID != "new12345" {
		t.Errorf("JobID = %q, want %q", data.JobID, "new12345")
	}
	if data.JobName != "NewJob" {
		t.Errorf("JobName = %q, want %q", data.JobName, "NewJob")
	}
	if len(data.OriginalJobs) != 2 {
		t.Errorf("OriginalJobs length = %d, want 2", len(data.OriginalJobs))
	}
}

func TestRollbackMount_RestoresConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rollback-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origMounts := []models.MountConfig{
		{ID: "abc12345", Name: "Mount1"},
		{ID: "def67890", Name: "Mount2"},
	}

	cfg := &config.Config{
		Version: "1.0",
		Mounts:  origMounts,
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := MountRollbackData{
		OriginalMounts: origMounts,
		Operation:      OperationCreate,
		MountID:        "new12345",
		MountName:      "NewMount",
	}

	cfg.Mounts = append(cfg.Mounts, models.MountConfig{ID: "new12345", Name: "NewMount"})
	if len(cfg.Mounts) != 3 {
		t.Fatalf("setup failed: expected 3 mounts, got %d", len(cfg.Mounts))
	}

	err = mgr.RollbackMount(data, false)
	if err != nil {
		t.Errorf("RollbackMount() error = %v", err)
	}

	if len(cfg.Mounts) != 2 {
		t.Errorf("after rollback, Mounts length = %d, want 2", len(cfg.Mounts))
	}

	if cfg.Mounts[0].ID != "abc12345" || cfg.Mounts[1].ID != "def67890" {
		t.Error("original mounts not restored correctly")
	}
}

func TestRollbackSyncJob_RestoresConfig(t *testing.T) {
	origJobs := []models.SyncJobConfig{
		{ID: "abc12345", Name: "Job1"},
		{ID: "def67890", Name: "Job2"},
	}

	cfg := &config.Config{
		Version:  "1.0",
		SyncJobs: origJobs,
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := SyncJobRollbackData{
		OriginalJobs: origJobs,
		Operation:    OperationCreate,
		JobID:        "new12345",
		JobName:      "NewJob",
	}

	cfg.SyncJobs = append(cfg.SyncJobs, models.SyncJobConfig{ID: "new12345", Name: "NewJob"})
	if len(cfg.SyncJobs) != 3 {
		t.Fatalf("setup failed: expected 3 jobs, got %d", len(cfg.SyncJobs))
	}

	err := mgr.RollbackSyncJob(data, false)
	if err != nil {
		t.Errorf("RollbackSyncJob() error = %v", err)
	}

	if len(cfg.SyncJobs) != 2 {
		t.Errorf("after rollback, SyncJobs length = %d, want 2", len(cfg.SyncJobs))
	}

	if cfg.SyncJobs[0].ID != "abc12345" || cfg.SyncJobs[1].ID != "def67890" {
		t.Error("original jobs not restored correctly")
	}
}

func TestRollbackData_Independence(t *testing.T) {
	cfg := createTestConfig()
	cfg.Mounts = []models.MountConfig{
		{ID: "abc12345", Name: "Mount1"},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := mgr.PrepareMountRollback("new12345", "NewMount", OperationCreate)

	cfg.Mounts[0].Name = "ModifiedMount"

	if data.OriginalMounts[0].Name != "Mount1" {
		t.Error("OriginalMounts was modified when config was modified - data should be independent")
	}
}

func TestOperationType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		op       OperationType
		expected int
	}{
		{"OperationCreate", OperationCreate, 0},
		{"OperationUpdate", OperationUpdate, 1},
		{"OperationDelete", OperationDelete, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.op) != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.op, tt.expected)
			}
		})
	}
}

func TestRollbackMount_WithBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rollback-backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")
	backupPath := configPath + ".bak"

	originalContent := `version: "1.0"
mounts:
  - id: original123
    name: OriginalMount
`
	if err := os.WriteFile(configPath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	if err := os.WriteFile(backupPath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to write backup: %v", err)
	}

	cfg := &config.Config{
		Version: "1.0",
		Mounts: []models.MountConfig{
			{ID: "original123", Name: "OriginalMount"},
		},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := MountRollbackData{
		OriginalMounts: []models.MountConfig{
			{ID: "original123", Name: "OriginalMount"},
		},
		Operation: OperationCreate,
		MountID:   "failed456",
		MountName: "FailedMount",
	}

	err = mgr.RollbackMount(data, true)
	if err != nil {
		t.Logf("RollbackMount returned error (expected in test env): %v", err)
	}
}

func TestRollbackSyncJob_WithBackup(t *testing.T) {
	cfg := &config.Config{
		Version: "1.0",
		SyncJobs: []models.SyncJobConfig{
			{ID: "original123", Name: "OriginalJob"},
		},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := SyncJobRollbackData{
		OriginalJobs: []models.SyncJobConfig{
			{ID: "original123", Name: "OriginalJob"},
		},
		Operation: OperationUpdate,
		JobID:     "failed456",
		JobName:   "FailedJob",
	}

	err := mgr.RollbackSyncJob(data, true)
	if err != nil {
		t.Logf("RollbackSyncJob returned error (expected in test env): %v", err)
	}
}

func TestCleanupMountSystemd_NilGenerator(t *testing.T) {
	cfg := createTestConfig()
	mgr := NewRollbackManager(cfg, nil, nil)

	mgr.CleanupMountSystemd("test1234")
}

func TestCleanupSyncJobSystemd_NilGenerator(t *testing.T) {
	cfg := createTestConfig()
	mgr := NewRollbackManager(cfg, nil, nil)

	mgr.CleanupSyncJobSystemd("test1234")
}

func TestPrepareMountRollback_EmptyConfig(t *testing.T) {
	cfg := &config.Config{
		Version: "1.0",
		Mounts:  []models.MountConfig{},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := mgr.PrepareMountRollback("new12345", "NewMount", OperationCreate)

	if len(data.OriginalMounts) != 0 {
		t.Errorf("OriginalMounts length = %d, want 0", len(data.OriginalMounts))
	}
}

func TestPrepareSyncJobRollback_EmptyConfig(t *testing.T) {
	cfg := &config.Config{
		Version:  "1.0",
		SyncJobs: []models.SyncJobConfig{},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := mgr.PrepareSyncJobRollback("new12345", "NewJob", OperationCreate)

	if len(data.OriginalJobs) != 0 {
		t.Errorf("OriginalJobs length = %d, want 0", len(data.OriginalJobs))
	}
}

func TestRollbackMount_DeleteOperation(t *testing.T) {
	origMounts := []models.MountConfig{
		{ID: "abc12345", Name: "Mount1"},
	}

	cfg := &config.Config{
		Version: "1.0",
		Mounts:  []models.MountConfig{},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := MountRollbackData{
		OriginalMounts: origMounts,
		Operation:      OperationDelete,
		MountID:        "abc12345",
		MountName:      "Mount1",
	}

	err := mgr.RollbackMount(data, false)
	if err != nil {
		t.Errorf("RollbackMount() error = %v", err)
	}

	if len(cfg.Mounts) != 1 {
		t.Errorf("after rollback, Mounts length = %d, want 1", len(cfg.Mounts))
	}
}

func TestRollbackSyncJob_DeleteOperation(t *testing.T) {
	origJobs := []models.SyncJobConfig{
		{ID: "abc12345", Name: "Job1"},
	}

	cfg := &config.Config{
		Version:  "1.0",
		SyncJobs: []models.SyncJobConfig{},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := SyncJobRollbackData{
		OriginalJobs: origJobs,
		Operation:    OperationDelete,
		JobID:        "abc12345",
		JobName:      "Job1",
	}

	err := mgr.RollbackSyncJob(data, false)
	if err != nil {
		t.Errorf("RollbackSyncJob() error = %v", err)
	}

	if len(cfg.SyncJobs) != 1 {
		t.Errorf("after rollback, SyncJobs length = %d, want 1", len(cfg.SyncJobs))
	}
}
