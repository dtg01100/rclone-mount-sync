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

func TestSyncCreateAndDeleteFlow(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
			Sync: config.SyncDefaults{
				LogLevel:  "INFO",
				Transfers: 4,
				Checkers:  8,
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
	mock := &systemd.MockManager{}
	loadManager = func() systemd.ServiceManager { return mock }

	syncCreateName = "test-sync"
	syncCreateSource = "gdrive:/Photos"
	syncCreateDestination = "/home/user/Backup/Photos"
	syncCreateSchedule = "daily"
	syncCreateEnabled = true

	if err := runSyncCreate(nil, nil); err != nil {
		t.Fatalf("runSyncCreate failed: %v", err)
	}

	files, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatalf("failed to read dir %q: %v", tmp, err)
	}
	found := false
	for _, f := range files {
		if f.Type().IsRegular() && (filepath.Ext(f.Name()) == ".service" || filepath.Ext(f.Name()) == ".timer") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected service/timer unit written in %s", tmp)
	}

	job := cfg.GetSyncJob(syncCreateName)
	if job == nil {
		t.Fatalf("sync job not found in config")
	}

	serviceName := "rclone-sync-" + job.ID + ".service"
	timerName := "rclone-sync-" + job.ID + ".timer"
	if err := os.WriteFile(filepath.Join(tmp, serviceName), []byte("[Unit]\n"), 0600); err != nil { //nolint:gosec
		t.Fatalf("failed to write service file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, timerName), []byte("[Unit]\n"), 0600); err != nil { //nolint:gosec
		t.Fatalf("failed to write timer file: %v", err)
	}

	if err := runSyncDelete(nil, []string{job.Name}); err != nil {
		t.Fatalf("runSyncDelete failed: %v", err)
	}
}

func TestSyncList(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
			Sync: config.SyncDefaults{
				LogLevel:  "INFO",
				Transfers: 4,
				Checkers:  8,
			},
		},
		SyncJobs: []models.SyncJobConfig{
			{
				ID:          "12345",
				Name:        "test-sync-1",
				Source:      "gdrive:/Docs",
				Destination: "/home/user/Docs",
				Enabled:     true,
				Schedule: models.ScheduleConfig{
					Type:       "timer",
					OnCalendar: "daily",
				},
			},
		},
	}

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func() (*config.Config, error) { return cfg, nil }

	err := runSyncList(nil, nil)
	if err != nil {
		t.Fatalf("runSyncList failed: %v", err)
	}
}

func TestSyncRun(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
			Sync: config.SyncDefaults{
				LogLevel:  "INFO",
				Transfers: 4,
				Checkers:  8,
			},
		},
		SyncJobs: []models.SyncJobConfig{
			{
				ID:          "abc123",
				Name:        "test-sync-run",
				Source:      "gdrive:/Photos",
				Destination: "/home/user/Backup/Photos",
				Enabled:     true,
				Schedule: models.ScheduleConfig{
					Type:       "timer",
					OnCalendar: "daily",
				},
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
	loadGenerator = func() (*systemd.Generator, error) { return systemd.NewTestGenerator(t.TempDir()), nil }

	mock := &systemd.MockManager{
		RunSyncNowErr: nil,
	}
	loadManager = func() systemd.ServiceManager { return mock }

	if err := runSyncRun(nil, []string{"test-sync-run"}); err != nil {
		t.Fatalf("runSyncRun failed: %v", err)
	}
}

func TestSyncDeleteNotFound(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
			Sync: config.SyncDefaults{
				LogLevel:  "INFO",
				Transfers: 4,
				Checkers:  8,
			},
		},
	}

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func() (*config.Config, error) { return cfg, nil }

	err := runSyncDelete(nil, []string{"nonexistent-job"})
	if err == nil {
		t.Fatalf("expected error when deleting non-existent sync job")
	}
}

func TestSyncRunNotFound(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
			Sync: config.SyncDefaults{
				LogLevel:  "INFO",
				Transfers: 4,
				Checkers:  8,
			},
		},
	}

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func() (*config.Config, error) { return cfg, nil }

	err := runSyncRun(nil, []string{"nonexistent-job"})
	if err == nil {
		t.Fatalf("expected error when running non-existent sync job")
	}
}

func TestSyncCreateValidationMissingFields(t *testing.T) {
	cfg := &config.Config{Defaults: config.DefaultConfig{Sync: config.SyncDefaults{LogLevel: "INFO", Transfers: 4, Checkers: 8}}}
	tmpDir := t.TempDir()

	oldLoadConfig := loadConfig
	oldLoadGenerator := loadGenerator
	oldLoadManager := loadManager
	oldSyncCreateName := syncCreateName
	oldSyncCreateSource := syncCreateSource
	oldSyncCreateDestination := syncCreateDestination
	defer func() {
		loadConfig = oldLoadConfig
		loadGenerator = oldLoadGenerator
		loadManager = oldLoadManager
		syncCreateName = oldSyncCreateName
		syncCreateSource = oldSyncCreateSource
		syncCreateDestination = oldSyncCreateDestination
	}()

	loadConfig = func() (*config.Config, error) { return cfg, nil }
	loadGenerator = func() (*systemd.Generator, error) { return systemd.NewTestGenerator(tmpDir), nil }
	loadManager = func() systemd.ServiceManager { return &systemd.MockManager{} }

	syncCreateName = ""
	syncCreateSource = ""
	syncCreateDestination = ""

	if err := runSyncCreate(nil, nil); err == nil {
		t.Fatal("expected runSyncCreate to fail when required fields are missing")
	}

	syncCreateName = "test-sync"
	syncCreateSource = "gdrive:/Photos"
	syncCreateDestination = ""

	if err := runSyncCreate(nil, nil); err == nil {
		t.Fatal("expected runSyncCreate to fail when destination is missing")
	}
}

func TestSyncCreateSaveConfigError(t *testing.T) {
	oldLoadConfig := loadConfig
	oldSyncCreateName := syncCreateName
	oldSyncCreateSource := syncCreateSource
	oldSyncCreateDestination := syncCreateDestination
	defer func() {
		loadConfig = oldLoadConfig
		syncCreateName = oldSyncCreateName
		syncCreateSource = oldSyncCreateSource
		syncCreateDestination = oldSyncCreateDestination
	}()

	loadConfig = func() (*config.Config, error) {
		return &config.Config{}, fmt.Errorf("config save failed")
	}
	syncCreateName = "test-sync"
	syncCreateSource = "gdrive:/Photos"
	syncCreateDestination = "/home/user/Backup"

	if err := runSyncCreate(nil, nil); err == nil {
		t.Fatal("expected runSyncCreate to fail when config save fails")
	}
}

func TestSyncCreateGeneratorError(t *testing.T) {
	cfg := &config.Config{Defaults: config.DefaultConfig{Sync: config.SyncDefaults{LogLevel: "INFO", Transfers: 4, Checkers: 8}}}

	oldLoadConfig := loadConfig
	oldLoadGenerator := loadGenerator
	oldSyncCreateName := syncCreateName
	oldSyncCreateSource := syncCreateSource
	oldSyncCreateDestination := syncCreateDestination
	defer func() {
		loadConfig = oldLoadConfig
		loadGenerator = oldLoadGenerator
		syncCreateName = oldSyncCreateName
		syncCreateSource = oldSyncCreateSource
		syncCreateDestination = oldSyncCreateDestination
	}()

	loadConfig = func() (*config.Config, error) { return cfg, nil }
	loadGenerator = func() (*systemd.Generator, error) { return nil, fmt.Errorf("generator init failed") }
	syncCreateName = "test-sync"
	syncCreateSource = "gdrive:/Photos"
	syncCreateDestination = "/home/user/Backup"

	if err := runSyncCreate(nil, nil); err == nil {
		t.Fatal("expected runSyncCreate to fail when generator init fails")
	}
}

func TestSyncRunGeneratorError(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{Sync: config.SyncDefaults{LogLevel: "INFO", Transfers: 4, Checkers: 8}},
		SyncJobs: []models.SyncJobConfig{
			{ID: "abc123", Name: "test-sync-gen", Source: "gdrive:/Photos", Destination: "/home/user/Backup/Photos", Enabled: true, Schedule: models.ScheduleConfig{Type: "timer", OnCalendar: "daily"}},
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

	err := runSyncRun(nil, []string{"test-sync-gen"})
	if err == nil {
		t.Fatal("expected error when generator load fails")
	}
}

func TestSyncDeleteGeneratorError(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{Sync: config.SyncDefaults{LogLevel: "INFO", Transfers: 4, Checkers: 8}},
		SyncJobs: []models.SyncJobConfig{
			{ID: "abc123", Name: "test-sync-del", Source: "gdrive:/Photos", Destination: "/home/user/Backup/Photos", Enabled: true, Schedule: models.ScheduleConfig{Type: "timer", OnCalendar: "daily"}},
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

	err := runSyncDelete(nil, []string{"test-sync-del"})
	if err == nil {
		t.Fatal("expected error when generator load fails")
	}
}
