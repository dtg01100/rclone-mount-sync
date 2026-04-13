package cli

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
)

func TestServicesListNoSystemd(t *testing.T) {
	// Test that services list properly handles when systemd is unavailable
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	// Mock manager that indicates systemd is not available
	mock := &systemd.MockManager{
		IsSystemdAvailableResult: false,
		ListServicesErr:         fmt.Errorf("systemd user session not available"),
	}
	loadManager = func() systemd.ServiceManager { return mock }

	// Call runServicesList directly to bypass cobra command routing
	err := runServicesList(nil, nil)
	// Should return error when systemd is unavailable
	if err == nil {
		t.Error("expected error when systemd is unavailable")
	}
	if err != nil && !strings.Contains(err.Error(), "systemd") {
		t.Errorf("expected systemd-related error, got: %v", err)
	}
}

func TestServicesListSystemdAvailable(t *testing.T) {
	// Test that services list succeeds when systemd is available
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		IsSystemdAvailableResult: true,
		ListServicesResult:       []systemd.UnitStatus{},
	}
	loadManager = func() systemd.ServiceManager { return mock }

	// Call runServicesList directly
	err := runServicesList(nil, nil)
	if err != nil {
		t.Fatalf("expected no error when systemd is available: %v", err)
	}
}

func TestServicesListWithServices(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		ListServicesResult: []systemd.UnitStatus{
			{Name: "rclone-mount-abc123.service", Enabled: true, Active: true, State: "active"},
			{Name: "rclone-mount-def456.service", Enabled: false, Active: false, State: "inactive"},
			{Name: "rclone-sync-xyz789.service", Enabled: true, Active: true, State: "active"},
		},
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runServicesList(nil, nil)
	if err != nil {
		t.Fatalf("runServicesList failed: %v", err)
	}
}

func TestServicesListWithServicesJSON(t *testing.T) {
	oldLoadManager := loadManager
	oldOutputJSON := outputJSON
	defer func() {
		loadManager = oldLoadManager
		outputJSON = oldOutputJSON
	}()

	mock := &systemd.MockManager{
		ListServicesResult: []systemd.UnitStatus{
			{Name: "rclone-mount-abc.service", Enabled: true, Active: true, State: "active"},
		},
	}
	loadManager = func() systemd.ServiceManager { return mock }
	outputJSON = true

	err := runServicesList(nil, nil)
	if err != nil {
		t.Fatalf("runServicesList JSON failed: %v", err)
	}
}

func TestServicesListNoServices(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		ListServicesResult: []systemd.UnitStatus{},
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runServicesList(nil, nil)
	if err != nil {
		t.Fatalf("runServicesList with no services failed: %v", err)
	}
}

func TestServicesListError(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		ListServicesErr: fmt.Errorf("failed to list services"),
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runServicesList(nil, nil)
	if err == nil {
		t.Fatal("expected error when listing services fails")
	}
}

func TestServicesStatus(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		GetDetailedStatusResult: &models.ServiceStatus{
			Name:        "rclone-mount-abc123.service",
			Type:        "mount",
			LoadState:   "loaded",
			ActiveState: "active",
			SubState:    "running",
			Enabled:     true,
			MainPID:     12345,
			MountPoint:  "/home/user/mnt",
			IsMounted:   true,
		},
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runServicesStatus(nil, []string{"rclone-mount-abc123"})
	if err != nil {
		t.Fatalf("runServicesStatus failed: %v", err)
	}
}

func TestServicesStatusWithServiceSuffix(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		GetDetailedStatusResult: &models.ServiceStatus{
			Name:        "rclone-mount-abc123.service",
			Type:        "mount",
			LoadState:   "loaded",
			ActiveState: "active",
			SubState:    "running",
			Enabled:     true,
		},
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runServicesStatus(nil, []string{"rclone-mount-abc123.service"})
	if err != nil {
		t.Fatalf("runServicesStatus with .service suffix failed: %v", err)
	}
}

func TestServicesStatusJSON(t *testing.T) {
	oldLoadManager := loadManager
	oldOutputJSON := outputJSON
	defer func() {
		loadManager = oldLoadManager
		outputJSON = oldOutputJSON
	}()

	mock := &systemd.MockManager{
		GetDetailedStatusResult: &models.ServiceStatus{
			Name:        "rclone-mount-abc123.service",
			Type:        "mount",
			LoadState:   "loaded",
			ActiveState: "active",
			SubState:    "running",
			Enabled:     true,
		},
	}
	loadManager = func() systemd.ServiceManager { return mock }
	outputJSON = true

	err := runServicesStatus(nil, []string{"rclone-mount-abc123"})
	if err != nil {
		t.Fatalf("runServicesStatus JSON failed: %v", err)
	}
}

func TestServicesStatusError(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		GetDetailedStatusErr: fmt.Errorf("service not found"),
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runServicesStatus(nil, []string{"nonexistent-service"})
	if err == nil {
		t.Fatal("expected error when getting status fails")
	}
}

func TestServicesStatusSyncJob(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		GetDetailedStatusResult: &models.ServiceStatus{
			Name:        "rclone-sync-abc123.service",
			Type:        "sync",
			LoadState:   "loaded",
			ActiveState: "active",
			SubState:    "running",
			Enabled:     true,
			TimerActive: true,
			LastRun:     time.Now().Add(-1 * time.Hour),
			NextRun:     time.Now().Add(23 * time.Hour),
			ActivatedAt: time.Now().Add(-2 * time.Hour),
		},
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runServicesStatus(nil, []string{"rclone-sync-abc123"})
	if err != nil {
		t.Fatalf("runServicesStatus for sync job failed: %v", err)
	}
}

func TestServicesStatusWithExitCode(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		GetDetailedStatusResult: &models.ServiceStatus{
			Name:        "rclone-mount-failed.service",
			Type:        "mount",
			LoadState:   "loaded",
			ActiveState: "failed",
			SubState:    "failed",
			Enabled:     false,
			ExitCode:    1,
			InactiveAt:  time.Now(),
		},
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runServicesStatus(nil, []string{"rclone-mount-failed"})
	if err != nil {
		t.Fatalf("runServicesStatus with exit code failed: %v", err)
	}
}

func TestServicesLogs(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		GetLogsResult: "Jan 01 12:00:00 host systemd[1]: Started rclone mount.\nJan 01 12:01:00 host rclone[123]: Mounting...\n",
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runServicesLogs(nil, []string{"rclone-mount-abc123"})
	if err != nil {
		t.Fatalf("runServicesLogs failed: %v", err)
	}
}

func TestServicesLogsWithServiceSuffix(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		GetLogsResult: "log line 1\nlog line 2\n",
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runServicesLogs(nil, []string{"rclone-mount-abc123.service"})
	if err != nil {
		t.Fatalf("runServicesLogs with .service suffix failed: %v", err)
	}
}

func TestServicesLogsError(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		GetLogsErr: fmt.Errorf("failed to get logs"),
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runServicesLogs(nil, []string{"nonexistent-service"})
	if err == nil {
		t.Fatal("expected error when getting logs fails")
	}
}

func TestServicesLogsFollow(t *testing.T) {
	oldLoadManager := loadManager
	oldLogsFollow := logsFollow
	defer func() {
		loadManager = oldLoadManager
		logsFollow = oldLogsFollow
	}()

	logsFollow = true

	err := runServicesLogs(nil, []string{"rclone-mount-abc123"})
	if err != nil {
		t.Fatalf("runServicesLogs with follow flag failed: %v", err)
	}
}

func TestServicesLogsCustomLines(t *testing.T) {
	oldLoadManager := loadManager
	oldLogsLines := logsLines
	defer func() {
		loadManager = oldLoadManager
		logsLines = oldLogsLines
	}()

	mock := &systemd.MockManager{}
	loadManager = func() systemd.ServiceManager { return mock }
	logsLines = 100

	err := runServicesLogs(nil, []string{"rclone-mount-abc123"})
	if err != nil {
		t.Fatalf("runServicesLogs with custom lines failed: %v", err)
	}
}
