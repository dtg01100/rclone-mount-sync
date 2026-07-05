package screens

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
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
	defer func() { _ = os.RemoveAll(tmpDir) }()

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
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yaml")
	backupPath := configPath + ".bak"

	originalContent := `version: "1.0"
mounts:
  - id: original123
    name: OriginalMount
`
	if err := os.WriteFile(configPath, []byte(originalContent), 0644); err != nil { //nolint:gosec
		t.Fatalf("failed to write config: %v", err)
	}
	if err := os.WriteFile(backupPath, []byte(originalContent), 0644); err != nil { //nolint:gosec
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
		t.Errorf("RollbackMount failed: %v", err)
	}

	if len(cfg.Mounts) != 1 {
		t.Errorf("after rollback, Mounts length = %d, want 1", len(cfg.Mounts))
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
		t.Errorf("RollbackSyncJob failed: %v", err)
	}

	// Verify jobs were restored
	if len(cfg.SyncJobs) != 1 {
		t.Errorf("after rollback, SyncJobs length = %d, want 1", len(cfg.SyncJobs))
	}
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

// Tests for RollbackSyncJob with systemdFailed=true

func TestRollbackSyncJob_SystemdFailed(t *testing.T) {
	origJobs := []models.SyncJobConfig{
		{ID: "abc12345", Name: "Job1"},
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

	err := mgr.RollbackSyncJob(data, true)
	if err != nil {
		t.Errorf("RollbackSyncJob failed: %v", err)
	}

	// Verify jobs were restored
	if len(cfg.SyncJobs) != 1 {
		t.Errorf("after rollback, SyncJobs length = %d, want 1", len(cfg.SyncJobs))
	}
}

func TestRollbackSyncJob_SystemdFailed_DeleteOperation(t *testing.T) {
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

	// For delete operation with systemdFailed=true, systemd cleanup should be skipped
	err := mgr.RollbackSyncJob(data, true)
	if err != nil {
		t.Errorf("RollbackSyncJob() error = %v", err)
	}

	if len(cfg.SyncJobs) != 1 {
		t.Errorf("after rollback, SyncJobs length = %d, want 1", len(cfg.SyncJobs))
	}
}

// Tests for RollbackMount with systemdFailed=true

func TestRollbackMount_SystemdFailed(t *testing.T) {
	origMounts := []models.MountConfig{
		{ID: "abc12345", Name: "Mount1"},
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

	err := mgr.RollbackMount(data, true)
	if err != nil {
		t.Errorf("RollbackMount failed: %v", err)
	}

	// Verify mounts were restored
	if len(cfg.Mounts) != 1 {
		t.Errorf("after rollback, Mounts length = %d, want 1", len(cfg.Mounts))
	}
}

func TestRollbackMount_SystemdFailed_DeleteOperation(t *testing.T) {
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

	// For delete operation with systemdFailed=true, systemd cleanup should be skipped
	err := mgr.RollbackMount(data, true)
	if err != nil {
		t.Errorf("RollbackMount() error = %v", err)
	}

	if len(cfg.Mounts) != 1 {
		t.Errorf("after rollback, Mounts length = %d, want 1", len(cfg.Mounts))
	}
}

// Tests for RollbackManager with all nil services

// TestRollbackManager_CleanupSystemd is a sub-tested no-panic guard
// for the four Cleanup*Systemd code paths. It ensures the methods
// accept both with-manager and nil-manager wiring without panicking.
// It does not assert on error values because the underlying
// systemd interactions are environment-dependent.
func TestRollbackManager_CleanupSystemd(t *testing.T) {
	tests := []struct {
		name   string
		mgr    systemd.ServiceManager
		method func(*RollbackManager, string)
		id     string
	}{
		{
			name:   "mount cleanup with manager",
			mgr:    &systemd.Manager{},
			method: (*RollbackManager).CleanupMountSystemd,
			id:     "test1234",
		},
		{
			name:   "mount cleanup with nil manager",
			mgr:    nil,
			method: (*RollbackManager).CleanupMountSystemd,
			id:     "test1234",
		},
		{
			name:   "sync job cleanup with manager",
			mgr:    &systemd.Manager{},
			method: (*RollbackManager).CleanupSyncJobSystemd,
			id:     "test1234",
		},
		{
			name:   "sync job cleanup with nil manager",
			mgr:    nil,
			method: (*RollbackManager).CleanupSyncJobSystemd,
			id:     "test1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig()
			gen := systemd.NewTestGenerator(t.TempDir())
			rollbackMgr := NewRollbackManager(cfg, gen, tt.mgr)
			tt.method(rollbackMgr, tt.id)
		})
	}
}

func TestRollbackManager_NilConfig(t *testing.T) {
	// NewRollbackManager should not panic with nil config
	mgr := NewRollbackManager(nil, nil, nil)

	if mgr == nil {
		t.Fatal("NewRollbackManager() returned nil")
	}

	// Verify the manager was created with nil config
	if mgr.config != nil {
		t.Error("expected config to be nil")
	}
}

func TestRollbackManager_PrepareMountRollback_NilConfig(t *testing.T) {
	mgr := NewRollbackManager(nil, nil, nil)

	// PrepareMountRollback panics with nil config - this is expected behavior
	// In real code, config should never be nil when calling these methods
	defer func() {
		if r := recover(); r == nil {
			t.Error("PrepareMountRollback should panic with nil config")
		}
	}()

	_ = mgr.PrepareMountRollback("test123", "TestMount", OperationCreate)
}

// Tests for MountRollbackData and SyncJobRollbackData structs

func TestMountRollbackData_Fields(t *testing.T) {
	mounts := []models.MountConfig{
		{ID: "test1234", Name: "TestMount"},
	}

	data := MountRollbackData{
		OriginalMounts: mounts,
		Operation:      OperationCreate,
		MountID:        "new1234",
		MountName:      "NewMount",
	}

	if len(data.OriginalMounts) != 1 {
		t.Errorf("OriginalMounts length = %d, want 1", len(data.OriginalMounts))
	}
	if data.Operation != OperationCreate {
		t.Errorf("Operation = %v, want %v", data.Operation, OperationCreate)
	}
	if data.MountID != "new1234" {
		t.Errorf("MountID = %q, want 'new1234'", data.MountID)
	}
	if data.MountName != "NewMount" {
		t.Errorf("MountName = %q, want 'NewMount'", data.MountName)
	}
}

func TestSyncJobRollbackData_Fields(t *testing.T) {
	jobs := []models.SyncJobConfig{
		{ID: "test1234", Name: "TestJob"},
	}

	data := SyncJobRollbackData{
		OriginalJobs: jobs,
		Operation:    OperationUpdate,
		JobID:        "new1234",
		JobName:      "NewJob",
	}

	if len(data.OriginalJobs) != 1 {
		t.Errorf("OriginalJobs length = %d, want 1", len(data.OriginalJobs))
	}
	if data.Operation != OperationUpdate {
		t.Errorf("Operation = %v, want %v", data.Operation, OperationUpdate)
	}
	if data.JobID != "new1234" {
		t.Errorf("JobID = %q, want 'new1234'", data.JobID)
	}
	if data.JobName != "NewJob" {
		t.Errorf("JobName = %q, want 'NewJob'", data.JobName)
	}
}

// Tests for OperationType values

func TestOperationType_Values(t *testing.T) {
	if OperationCreate != 0 {
		t.Errorf("OperationCreate = %d, want 0", OperationCreate)
	}
	if OperationUpdate != 1 {
		t.Errorf("OperationUpdate = %d, want 1", OperationUpdate)
	}
	if OperationDelete != 2 {
		t.Errorf("OperationDelete = %d, want 2", OperationDelete)
	}
}

// Tests for PrepareMountRollback with various operations

func TestPrepareMountRollback_CreateOperation(t *testing.T) {
	cfg := createTestConfig()
	cfg.Mounts = []models.MountConfig{
		{ID: "existing1", Name: "ExistingMount1"},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := mgr.PrepareMountRollback("new12345", "NewMount", OperationCreate)

	if data.Operation != OperationCreate {
		t.Errorf("Operation = %v, want %v", data.Operation, OperationCreate)
	}
	if len(data.OriginalMounts) != 1 {
		t.Errorf("OriginalMounts length = %d, want 1", len(data.OriginalMounts))
	}
}

func TestPrepareMountRollback_UpdateOperation(t *testing.T) {
	cfg := createTestConfig()
	cfg.Mounts = []models.MountConfig{
		{ID: "existing1", Name: "ExistingMount1"},
		{ID: "existing2", Name: "ExistingMount2"},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := mgr.PrepareMountRollback("existing1", "ExistingMount1", OperationUpdate)

	if data.Operation != OperationUpdate {
		t.Errorf("Operation = %v, want %v", data.Operation, OperationUpdate)
	}
	if len(data.OriginalMounts) != 2 {
		t.Errorf("OriginalMounts length = %d, want 2", len(data.OriginalMounts))
	}
}

func TestPrepareMountRollback_DeleteOperation(t *testing.T) {
	cfg := createTestConfig()
	cfg.Mounts = []models.MountConfig{
		{ID: "existing1", Name: "ExistingMount1"},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := mgr.PrepareMountRollback("existing1", "ExistingMount1", OperationDelete)

	if data.Operation != OperationDelete {
		t.Errorf("Operation = %v, want %v", data.Operation, OperationDelete)
	}
	if len(data.OriginalMounts) != 1 {
		t.Errorf("OriginalMounts length = %d, want 1", len(data.OriginalMounts))
	}
}

// Tests for PrepareSyncJobRollback with various operations

func TestPrepareSyncJobRollback_CreateOperation(t *testing.T) {
	cfg := createSyncTestConfig()
	cfg.SyncJobs = []models.SyncJobConfig{
		{ID: "existing1", Name: "ExistingJob1"},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := mgr.PrepareSyncJobRollback("new12345", "NewJob", OperationCreate)

	if data.Operation != OperationCreate {
		t.Errorf("Operation = %v, want %v", data.Operation, OperationCreate)
	}
	if len(data.OriginalJobs) != 1 {
		t.Errorf("OriginalJobs length = %d, want 1", len(data.OriginalJobs))
	}
}

func TestPrepareSyncJobRollback_UpdateOperation(t *testing.T) {
	cfg := createSyncTestConfig()
	cfg.SyncJobs = []models.SyncJobConfig{
		{ID: "existing1", Name: "ExistingJob1"},
		{ID: "existing2", Name: "ExistingJob2"},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := mgr.PrepareSyncJobRollback("existing1", "ExistingJob1", OperationUpdate)

	if data.Operation != OperationUpdate {
		t.Errorf("Operation = %v, want %v", data.Operation, OperationUpdate)
	}
	if len(data.OriginalJobs) != 2 {
		t.Errorf("OriginalJobs length = %d, want 2", len(data.OriginalJobs))
	}
}

func TestPrepareSyncJobRollback_DeleteOperation(t *testing.T) {
	cfg := createSyncTestConfig()
	cfg.SyncJobs = []models.SyncJobConfig{
		{ID: "existing1", Name: "ExistingJob1"},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := mgr.PrepareSyncJobRollback("existing1", "ExistingJob1", OperationDelete)

	if data.Operation != OperationDelete {
		t.Errorf("Operation = %v, want %v", data.Operation, OperationDelete)
	}
	if len(data.OriginalJobs) != 1 {
		t.Errorf("OriginalJobs length = %d, want 1", len(data.OriginalJobs))
	}
}

// Tests for RollbackMount with Update operation

func TestRollbackMount_UpdateOperation(t *testing.T) {
	origMounts := []models.MountConfig{
		{ID: "abc12345", Name: "Mount1", MountPoint: "/mnt/original"},
	}

	cfg := &config.Config{
		Version: "1.0",
		Mounts: []models.MountConfig{
			{ID: "abc12345", Name: "Mount1", MountPoint: "/mnt/updated"},
		},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := MountRollbackData{
		OriginalMounts: origMounts,
		Operation:      OperationUpdate,
		MountID:        "abc12345",
		MountName:      "Mount1",
	}

	err := mgr.RollbackMount(data, false)
	if err != nil {
		t.Errorf("RollbackMount() error = %v", err)
	}

	if cfg.Mounts[0].MountPoint != "/mnt/original" {
		t.Errorf("MountPoint = %q, want '/mnt/original'", cfg.Mounts[0].MountPoint)
	}
}

// Tests for RollbackSyncJob with Update operation

func TestRollbackSyncJob_UpdateOperation(t *testing.T) {
	origJobs := []models.SyncJobConfig{
		{ID: "abc12345", Name: "Job1", Destination: "/mnt/original"},
	}

	cfg := &config.Config{
		Version: "1.0",
		SyncJobs: []models.SyncJobConfig{
			{ID: "abc12345", Name: "Job1", Destination: "/mnt/updated"},
		},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := SyncJobRollbackData{
		OriginalJobs: origJobs,
		Operation:    OperationUpdate,
		JobID:        "abc12345",
		JobName:      "Job1",
	}

	err := mgr.RollbackSyncJob(data, false)
	if err != nil {
		t.Errorf("RollbackSyncJob() error = %v", err)
	}

	if cfg.SyncJobs[0].Destination != "/mnt/original" {
		t.Errorf("Destination = %q, want '/mnt/original'", cfg.SyncJobs[0].Destination)
	}
}

// TestRollbackMount_SystemdCleanupErrorsSurfaces verifies that errors
// from the systemd cleanup branch (Stop/Disable/RemoveUnit/DaemonReload)
// are returned to the caller, even when config.RestoreFromBackup()
// succeeds. Previously these were silently dropped on the success
// path, leaving callers unaware of partial-cleanup failures.
func TestRollbackMount_SystemdCleanupErrorsSurfaces(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	appDir := filepath.Join(configDir, "rclone-mount-sync")
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	backupContent := `version: "1.0"
mounts:
  - id: original01
    name: OriginalMount
`
	if err := os.WriteFile(filepath.Join(appDir, "config.yaml.bak"), []byte(backupContent), 0o600); err != nil {
		t.Fatalf("WriteFile backup: %v", err)
	}

	origMounts := []models.MountConfig{
		{ID: "original01", Name: "OriginalMount"},
	}
	cfg := &config.Config{
		Version: "1.0",
		Mounts: []models.MountConfig{
			{ID: "modified01", Name: "ModifiedMount"},
		},
	}

	gen := systemd.NewTestGenerator(tmpDir)
	mgr := &systemd.MockManager{
		StopErr:    errors.New("synthetic stop failure"),
		DisableErr: errors.New("synthetic disable failure"),
	}

	rm := NewRollbackManager(cfg, gen, mgr)
	data := MountRollbackData{
		OriginalMounts: origMounts,
		Operation:      OperationCreate,
		MountID:        "failed01",
		MountName:      "FailedMount",
	}

	err := rm.RollbackMount(data, true)
	if err == nil {
		t.Fatal("RollbackMount returned nil error; want error from Stop/Disable cleanup")
	}
	if !strings.Contains(err.Error(), "synthetic stop failure") {
		t.Errorf("error = %q, want to contain 'synthetic stop failure'", err.Error())
	}
	if !strings.Contains(err.Error(), "synthetic disable failure") {
		t.Errorf("error = %q, want to contain 'synthetic disable failure'", err.Error())
	}
	// Config must still have been restored from the backup.
	if len(cfg.Mounts) != 1 || cfg.Mounts[0].Name != "OriginalMount" {
		t.Errorf("Mounts not restored: got %+v", cfg.Mounts)
	}
}

// TestRollbackSyncJob_SystemdCleanupErrorsSurfaces mirrors the mount
// version for sync job rollback.
func TestRollbackSyncJob_SystemdCleanupErrorsSurfaces(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	appDir := filepath.Join(configDir, "rclone-mount-sync")
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	backupContent := `version: "1.0"
sync_jobs:
  - id: original01
    name: OriginalJob
`
	if err := os.WriteFile(filepath.Join(appDir, "config.yaml.bak"), []byte(backupContent), 0o600); err != nil {
		t.Fatalf("WriteFile backup: %v", err)
	}

	origJobs := []models.SyncJobConfig{
		{ID: "original01", Name: "OriginalJob"},
	}
	cfg := &config.Config{
		Version: "1.0",
		SyncJobs: []models.SyncJobConfig{
			{ID: "modified01", Name: "ModifiedJob"},
		},
	}

	gen := systemd.NewTestGenerator(tmpDir)
	mgr := &systemd.MockManager{
		StopErr: errors.New("synthetic stop failure"),
	}

	rm := NewRollbackManager(cfg, gen, mgr)
	data := SyncJobRollbackData{
		OriginalJobs: origJobs,
		Operation:    OperationUpdate,
		JobID:        "failed01",
		JobName:      "FailedJob",
	}

	err := rm.RollbackSyncJob(data, true)
	if err == nil {
		t.Fatal("RollbackSyncJob returned nil error; want error from Stop cleanup")
	}
	if !strings.Contains(err.Error(), "synthetic stop failure") {
		t.Errorf("error = %q, want to contain 'synthetic stop failure'", err.Error())
	}
	if len(cfg.SyncJobs) != 1 || cfg.SyncJobs[0].Name != "OriginalJob" {
		t.Errorf("SyncJobs not restored: got %+v", cfg.SyncJobs)
	}
}
