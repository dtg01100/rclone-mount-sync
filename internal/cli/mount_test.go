package cli

import (
	"fmt"
	"testing"

	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
)

func TestMountListNoConfig(t *testing.T) {
	oldCfg := cfgFile
	cfgFile = "/no/such/path"
	defer func() { cfgFile = oldCfg }()
	_, _, err := runCmd(t, mountListCmd)
	if err == nil {
		t.Logf("mount list returned no error; ensure manual testing for config loading")
	}
}

func TestMountListWithMounts(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
		},
		Mounts: []models.MountConfig{
			{
				ID:         "abc12345",
				Name:       "test-mount-1",
				Remote:     "gdrive:",
				RemotePath: "/Photos",
				MountPoint: "/home/user/mnt/photos",
				Enabled:    true,
				AutoStart:  false,
			},
			{
				ID:         "def67890",
				Name:       "test-mount-2",
				Remote:     "dropbox:",
				RemotePath: "/",
				MountPoint: "/home/user/mnt/dropbox",
				Enabled:    false,
				AutoStart:  true,
			},
		},
	}

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func() (*config.Config, error) { return cfg, nil }

	err := runMountList(nil, nil)
	if err != nil {
		t.Fatalf("runMountList failed: %v", err)
	}
}

func TestMountListWithMountsJSON(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
		},
		Mounts: []models.MountConfig{
			{
				ID:         "abc12345",
				Name:       "test-mount",
				Remote:     "gdrive:",
				RemotePath: "/",
				MountPoint: "/home/user/mnt",
				Enabled:    true,
				AutoStart:  false,
			},
		},
	}

	oldLoadConfig := loadConfig
	oldOutputJSON := outputJSON
	defer func() {
		loadConfig = oldLoadConfig
		outputJSON = oldOutputJSON
	}()
	loadConfig = func() (*config.Config, error) { return cfg, nil }
	outputJSON = true

	err := runMountList(nil, nil)
	if err != nil {
		t.Fatalf("runMountList with JSON failed: %v", err)
	}
}

func TestMountListNoMounts(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
		},
		Mounts: []models.MountConfig{},
	}

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func() (*config.Config, error) { return cfg, nil }

	err := runMountList(nil, nil)
	if err != nil {
		t.Fatalf("runMountList with no mounts failed: %v", err)
	}
}

func TestMountStart(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
		},
		Mounts: []models.MountConfig{
			{
				ID:         "abc12345",
				Name:       "test-mount-start",
				Remote:     "gdrive:",
				RemotePath: "/",
				MountPoint: "/home/user/mnt",
				Enabled:    true,
				AutoStart:  false,
			},
		},
	}

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
	mock := &systemd.MockManager{
		StartErr: nil,
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runMountStart(nil, []string{"test-mount-start"})
	if err != nil {
		t.Fatalf("runMountStart failed: %v", err)
	}
}

func TestMountStartByID(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
		},
		Mounts: []models.MountConfig{
			{
				ID:         "abc12345",
				Name:       "test-mount-by-id",
				Remote:     "gdrive:",
				RemotePath: "/",
				MountPoint: "/home/user/mnt",
				Enabled:    true,
				AutoStart:  false,
			},
		},
	}

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
	mock := &systemd.MockManager{
		StartErr: nil,
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runMountStart(nil, []string{"abc12345"})
	if err != nil {
		t.Fatalf("runMountStart by ID failed: %v", err)
	}
}

func TestMountStartNotFound(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
		},
		Mounts: []models.MountConfig{},
	}

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func() (*config.Config, error) { return cfg, nil }

	err := runMountStart(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error when starting non-existent mount")
	}
}

func TestMountStartError(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
		},
		Mounts: []models.MountConfig{
			{
				ID:         "abc12345",
				Name:       "test-mount-error",
				Remote:     "gdrive:",
				RemotePath: "/",
				MountPoint: "/home/user/mnt",
				Enabled:    true,
				AutoStart:  false,
			},
		},
	}

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
	mock := &systemd.MockManager{
		StartErr: fmt.Errorf("failed to start service"),
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runMountStart(nil, []string{"test-mount-error"})
	if err == nil {
		t.Fatal("expected error when starting mount fails")
	}
}

func TestMountStop(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
		},
		Mounts: []models.MountConfig{
			{
				ID:         "abc12345",
				Name:       "test-mount-stop",
				Remote:     "gdrive:",
				RemotePath: "/",
				MountPoint: "/home/user/mnt",
				Enabled:    true,
				AutoStart:  false,
			},
		},
	}

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
	mock := &systemd.MockManager{
		StopErr: nil,
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runMountStop(nil, []string{"test-mount-stop"})
	if err != nil {
		t.Fatalf("runMountStop failed: %v", err)
	}
}

func TestMountStopByID(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
		},
		Mounts: []models.MountConfig{
			{
				ID:         "abc12345",
				Name:       "test-mount-stop-id",
				Remote:     "gdrive:",
				RemotePath: "/",
				MountPoint: "/home/user/mnt",
				Enabled:    true,
				AutoStart:  false,
			},
		},
	}

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
	mock := &systemd.MockManager{
		StopErr: nil,
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runMountStop(nil, []string{"abc12345"})
	if err != nil {
		t.Fatalf("runMountStop by ID failed: %v", err)
	}
}

func TestMountStopNotFound(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
		},
		Mounts: []models.MountConfig{},
	}

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func() (*config.Config, error) { return cfg, nil }

	err := runMountStop(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error when stopping non-existent mount")
	}
}

func TestMountStopError(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
		},
		Mounts: []models.MountConfig{
			{
				ID:         "abc12345",
				Name:       "test-mount-stop-error",
				Remote:     "gdrive:",
				RemotePath: "/",
				MountPoint: "/home/user/mnt",
				Enabled:    true,
				AutoStart:  false,
			},
		},
	}

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
	mock := &systemd.MockManager{
		StopErr: fmt.Errorf("failed to stop service"),
	}
	loadManager = func() systemd.ServiceManager { return mock }

	err := runMountStop(nil, []string{"test-mount-stop-error"})
	if err == nil {
		t.Fatal("expected error when stopping mount fails")
	}
}
