package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
)

func TestMountListNoConfig(t *testing.T) {
	// Test that mount list handles gracefully when config loading fails
	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()

	// Mock loadConfig to return an error
	loadConfig = func() (*config.Config, error) {
		return nil, fmt.Errorf("failed to load config: config directory not found")
	}

	err := runMountList(nil, nil)
	if err == nil {
		t.Error("mount list should return error when config loading fails")
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

func TestMountCreateValidationMissingFields(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.Config{Defaults: config.DefaultConfig{Mount: config.MountDefaults{LogLevel: "INFO", VFSCacheMode: "full", BufferSize: "16M"}}}

	oldLoadConfig := loadConfig
	oldLoadGenerator := loadGenerator
	oldLoadManager := loadManager
	oldMountCreateName := mountCreateName
	oldMountCreateRemote := mountCreateRemote
	oldMountCreateMountPoint := mountCreateMountPoint
	defer func() {
		loadConfig = oldLoadConfig
		loadGenerator = oldLoadGenerator
		loadManager = oldLoadManager
		mountCreateName = oldMountCreateName
		mountCreateRemote = oldMountCreateRemote
		mountCreateMountPoint = oldMountCreateMountPoint
	}()

	loadConfig = func() (*config.Config, error) { return cfg, nil }
	loadGenerator = func() (*systemd.Generator, error) { return systemd.NewTestGenerator(tmp), nil }
	loadManager = func() systemd.ServiceManager { return &systemd.MockManager{} }

	mountCreateName = ""
	mountCreateRemote = ""
	mountCreateMountPoint = ""

	if err := runMountCreate(nil, nil); err == nil {
		t.Fatal("expected runMountCreate to fail when required fields are missing")
	}

	mountCreateName = "test-mount"
	mountCreateRemote = "gdrive:"
	mountCreateMountPoint = ""

	if err := runMountCreate(nil, nil); err == nil {
		t.Fatal("expected runMountCreate to fail when mount point is missing")
	}

	mountCreateName = "test-mount"
	mountCreateRemote = ""
	mountCreateMountPoint = "/home/user/mnt"

	if err := runMountCreate(nil, nil); err == nil {
		t.Fatal("expected runMountCreate to fail when remote is missing")
	}
}

// TestMountCreateLoadConfigError tests the early-return path when loadConfig
// itself fails. The previous "TestMountCreateSaveConfigError" mock returned
// an error from loadConfig which is checked before cfg.Save() is ever
// reached — it therefore exercised the wrong seam. Renamed and pointed at
// the correct path; the actual cfg.Save() failure is exercised by
// TestMountCreateSaveConfigError_RealSaveFailure via a read-only tmpdir.
func TestMountCreateLoadConfigError(t *testing.T) {
	oldLoadConfig := loadConfig
	t.Cleanup(func() { loadConfig = oldLoadConfig })

	loadConfig = func() (*config.Config, error) {
		return nil, fmt.Errorf("simulated config load failure")
	}

	if err := runMountCreate(nil, nil); err == nil {
		t.Fatal("expected runMountCreate to fail when loadConfig fails")
	}
}

// TestMountCreateSaveConfigError_RealSaveFailure makes Save() fail by
// pointing XDG_CONFIG_HOME at a read-only directory so config.Save()
// fails when it tries to write config.yaml. The previous version of
// this test set cfgFile, but cfgFile is only consulted during
// PersistentPreRun and runMountCreate calls loadConfig() directly, so
// the override was inert. Set XDG_CONFIG_HOME directly instead.
func TestMountCreateSaveConfigError_RealSaveFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("read-only directory check is bypassed when running as root")
	}
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0o500); err != nil {
		t.Fatalf("create read-only dir: %v", err)
	}

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Cleanup(func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) })
	_ = os.Setenv("XDG_CONFIG_HOME", readOnlyDir)

	if err := runMountCreate(nil, nil); err == nil {
		t.Fatal("expected runMountCreate to fail when cfg.Save() fails")
	}
}

func TestMountCreateGeneratorError(t *testing.T) {
	cfg := &config.Config{Defaults: config.DefaultConfig{Mount: config.MountDefaults{LogLevel: "INFO", VFSCacheMode: "full", BufferSize: "16M"}}}

	oldLoadConfig := loadConfig
	oldLoadGenerator := loadGenerator
	oldMountCreateName := mountCreateName
	oldMountCreateRemote := mountCreateRemote
	oldMountCreateMountPoint := mountCreateMountPoint
	defer func() {
		loadConfig = oldLoadConfig
		loadGenerator = oldLoadGenerator
		mountCreateName = oldMountCreateName
		mountCreateRemote = oldMountCreateRemote
		mountCreateMountPoint = oldMountCreateMountPoint
	}()

	loadConfig = func() (*config.Config, error) { return cfg, nil }
	loadGenerator = func() (*systemd.Generator, error) { return nil, fmt.Errorf("generator init failed") }
	mountCreateName = "test-mount"
	mountCreateRemote = "gdrive:"
	mountCreateMountPoint = "/home/user/mnt"

	if err := runMountCreate(nil, nil); err == nil {
		t.Fatal("expected runMountCreate to fail when generator init fails")
	}
}

func TestMountStartGeneratorError(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{Mount: config.MountDefaults{LogLevel: "INFO", VFSCacheMode: "full", BufferSize: "16M"}},
		Mounts: []models.MountConfig{
			{ID: "abc12345", Name: "test-mount-gen", Remote: "gdrive:", RemotePath: "/", MountPoint: "/home/user/mnt", Enabled: true},
		},
	}

	oldLoadConfig := loadConfig
	oldLoadGenerator := loadGenerator
	defer func() {
		loadConfig = oldLoadConfig
		loadGenerator = oldLoadGenerator
	}()

	loadConfig = func() (*config.Config, error) { return cfg, nil }
	loadGenerator = func() (*systemd.Generator, error) { return nil, fmt.Errorf("generator unavailable") }

	err := runMountStart(nil, []string{"test-mount-gen"})
	if err == nil {
		t.Fatal("expected error when generator load fails")
	}
}

func TestMountStopGeneratorError(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{Mount: config.MountDefaults{LogLevel: "INFO", VFSCacheMode: "full", BufferSize: "16M"}},
		Mounts: []models.MountConfig{
			{ID: "abc12345", Name: "test-mount-stop-gen", Remote: "gdrive:", RemotePath: "/", MountPoint: "/home/user/mnt", Enabled: true},
		},
	}

	oldLoadConfig := loadConfig
	oldLoadGenerator := loadGenerator
	defer func() {
		loadConfig = oldLoadConfig
		loadGenerator = oldLoadGenerator
	}()

	loadConfig = func() (*config.Config, error) { return cfg, nil }
	loadGenerator = func() (*systemd.Generator, error) { return nil, fmt.Errorf("generator unavailable") }

	err := runMountStop(nil, []string{"test-mount-stop-gen"})
	if err == nil {
		t.Fatal("expected error when generator load fails")
	}
}
