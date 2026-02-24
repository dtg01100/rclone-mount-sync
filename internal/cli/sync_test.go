package cli

import (
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
		Defaults: config.DefaultConfig(),
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

	files, _ := os.ReadDir(tmp)
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
	_ = os.WriteFile(filepath.Join(tmp, serviceName), []byte("[Unit]\n"), 0644)
	_ = os.WriteFile(filepath.Join(tmp, timerName), []byte("[Unit]\n"), 0644)

	if err := runSyncDelete(nil, []string{job.Name}); err != nil {
		t.Fatalf("runSyncDelete failed: %v", err)
	}
}

func TestSyncList(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig(),
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

	if _, _, err := runCmd(t, syncListCmd); err != nil {
		t.Fatalf("sync list failed: %v", err)
	}
}

func TestSyncRun(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig(),
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
		RunSyncNowResult: nil,
	}
	loadManager = func() systemd.ServiceManager { return mock }

	if err := runSyncRun(nil, []string{"test-sync-run"}); err != nil {
		t.Fatalf("runSyncRun failed: %v", err)
	}
}

func TestSyncDeleteNotFound(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig(),
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
		Defaults: config.DefaultConfig(),
	}

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func() (*config.Config, error) { return cfg, nil }

	err := runSyncRun(nil, []string{"nonexistent-job"})
	if err == nil {
		t.Fatalf("expected error when running non-existent sync job")
	}
}
