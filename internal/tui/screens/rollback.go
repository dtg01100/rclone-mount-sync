package screens

import (
	"fmt"

	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
)

type OperationType int

const (
	OperationCreate OperationType = iota
	OperationUpdate
	OperationDelete
)

type MountRollbackData struct {
	OriginalMounts []models.MountConfig
	Operation      OperationType
	MountID        string
	MountName      string
}

type SyncJobRollbackData struct {
	OriginalJobs []models.SyncJobConfig
	Operation    OperationType
	JobID        string
	JobName      string
}

type RollbackManager struct {
	config    *config.Config
	generator *systemd.Generator
	manager   *systemd.Manager
}

func NewRollbackManager(cfg *config.Config, gen *systemd.Generator, mgr *systemd.Manager) *RollbackManager {
	return &RollbackManager{
		config:    cfg,
		generator: gen,
		manager:   mgr,
	}
}

func (r *RollbackManager) PrepareMountRollback(mountID, mountName string, op OperationType) MountRollbackData {
	originalMounts := make([]models.MountConfig, len(r.config.Mounts))
	copy(originalMounts, r.config.Mounts)
	return MountRollbackData{
		OriginalMounts: originalMounts,
		Operation:      op,
		MountID:        mountID,
		MountName:      mountName,
	}
}

func (r *RollbackManager) PrepareSyncJobRollback(jobID, jobName string, op OperationType) SyncJobRollbackData {
	originalJobs := make([]models.SyncJobConfig, len(r.config.SyncJobs))
	copy(originalJobs, r.config.SyncJobs)
	return SyncJobRollbackData{
		OriginalJobs: originalJobs,
		Operation:    op,
		JobID:        jobID,
		JobName:      jobName,
	}
}

func (r *RollbackManager) RollbackMount(data MountRollbackData, systemdFailed bool) error {
	var errs []error

	if systemdFailed && data.Operation != OperationDelete {
		if r.generator != nil {
			serviceName := r.generator.ServiceName(data.MountID, "mount") + ".service"
			_ = r.manager.Stop(serviceName)
			_ = r.manager.Disable(serviceName)
			_ = r.generator.RemoveUnit(serviceName)
			_ = r.manager.DaemonReload()
		}
	}

	if err := config.RestoreFromBackup(); err == nil {
		r.config.Mounts = data.OriginalMounts
		return nil
	}

	r.config.Mounts = data.OriginalMounts
	if err := r.config.Save(); err != nil {
		errs = append(errs, fmt.Errorf("failed to restore config: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("rollback encountered errors: %v", errs)
	}
	return nil
}

func (r *RollbackManager) RollbackSyncJob(data SyncJobRollbackData, systemdFailed bool) error {
	var errs []error

	if systemdFailed && data.Operation != OperationDelete {
		if r.generator != nil {
			serviceName := r.generator.ServiceName(data.JobID, "sync") + ".service"
			timerName := r.generator.ServiceName(data.JobID, "sync") + ".timer"
			_ = r.manager.Stop(serviceName)
			_ = r.manager.StopTimer(timerName)
			_ = r.manager.Disable(serviceName)
			_ = r.manager.DisableTimer(timerName)
			_ = r.generator.RemoveUnit(serviceName)
			_ = r.generator.RemoveUnit(timerName)
			_ = r.manager.DaemonReload()
		}
	}

	if err := config.RestoreFromBackup(); err == nil {
		r.config.SyncJobs = data.OriginalJobs
		return nil
	}

	r.config.SyncJobs = data.OriginalJobs
	if err := r.config.Save(); err != nil {
		errs = append(errs, fmt.Errorf("failed to restore config: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("rollback encountered errors: %v", errs)
	}
	return nil
}

func (r *RollbackManager) CleanupMountSystemd(mountID string) {
	if r.generator == nil || r.manager == nil {
		return
	}
	serviceName := r.generator.ServiceName(mountID, "mount") + ".service"
	_ = r.manager.Stop(serviceName)
	_ = r.manager.Disable(serviceName)
	_ = r.generator.RemoveUnit(serviceName)
	_ = r.manager.DaemonReload()
}

func (r *RollbackManager) CleanupSyncJobSystemd(jobID string) {
	if r.generator == nil || r.manager == nil {
		return
	}
	serviceName := r.generator.ServiceName(jobID, "sync") + ".service"
	timerName := r.generator.ServiceName(jobID, "sync") + ".timer"
	_ = r.manager.Stop(serviceName)
	_ = r.manager.StopTimer(timerName)
	_ = r.manager.Disable(serviceName)
	_ = r.manager.DisableTimer(timerName)
	_ = r.generator.RemoveUnit(serviceName)
	_ = r.generator.RemoveUnit(timerName)
	_ = r.manager.DaemonReload()
}
