package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
)

func TestMountCreateAndDeleteFlow(t *testing.T) {
	tmp := t.TempDir()
	// Prepare a minimal config
	cfg := &config.Config{}

	// Override loaders
	oldLoadConfig := loadConfig
	oldLoadGenerator := loadGenerator
	oldLoadManager := loadManager
	defer func() {
		loadConfig = oldLoadConfig
		loadGenerator = oldLoadGenerator
		loadManager = oldLoadManager
	}()

	loadConfig = func() (*config.Config, error) { return cfg, nil }
	loadGenerator = func() (*systemd.Generator, error) { return systemd.NewTestGenerator(tmp), nil }
	mock := &systemd.MockManager{}
	loadManager = func() systemd.ServiceManager { return mock }

	// Set create flags
	mountCreateName = "test-mount"
	mountCreateRemote = "remote:"
	mountCreateRemotePath = "/"
	mountCreateMountPoint = filepath.Join(tmp, "mnt")
	mountCreateEnabled = true
	mountCreateAutoStart = true

	// Run create
	if err := runMountCreate(nil, nil); err != nil {
		t.Fatalf("runMountCreate failed: %v", err)
	}

	// Verify unit file exists
	// Generator.ServiceName uses random ID; find created file in tmp dir
	files, _ := os.ReadDir(tmp)
	found := false
	for _, f := range files {
		if f.Type().IsRegular() && (filepath.Ext(f.Name()) == ".service" || filepath.Ext(f.Name()) == ".timer") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected service unit written in %s", tmp)
	}

	// Add mount to config so delete can find it
	m := models.MountConfig{Name: mountCreateName, ID: "abc12345", MountPoint: mountCreateMountPoint}
	cfg.Mounts = append(cfg.Mounts, m)

	// Create a fake unit file matching generator.ServiceName
	_ = os.WriteFile(filepath.Join(tmp, "rclone-mount-abc12345.service"), []byte("[Unit]\n"), 0644)

	// Run delete
	if err := runMountDelete(nil, []string{mountCreateName}); err != nil {
		t.Fatalf("runMountDelete failed: %v", err)
	}
}

func TestServicesCommandsWithMockManager(t *testing.T) {
	// Setup mock manager
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		ListServicesResult: []systemd.ServiceStatus{
			{Name: "rclone-mount-abc.service", Enabled: true, Active: true, State: "active"},
		},
		GetDetailedStatusResult: &models.ServiceStatus{
			Name:        "rclone-mount-abc.service",
			Type:        "mount",
			LoadState:   "loaded",
			ActiveState: "active",
			SubState:    "running",
		},
		GetLogsResult: "log1\nlog2\n",
	}

	loadManager = func() systemd.ServiceManager { return mock }

	// Test list
	if _, _, err := runCmd(t, servicesListCmd); err != nil {
		t.Fatalf("services list failed: %v", err)
	}

	// Test status
	if _, _, err := runCmd(t, servicesStatusCmd, "rclone-mount-abc"); err != nil {
		t.Fatalf("services status failed: %v", err)
	}

	// Test logs
	if _, _, err := runCmd(t, servicesLogsCmd, "rclone-mount-abc"); err != nil {
		t.Fatalf("services logs failed: %v", err)
	}
}
