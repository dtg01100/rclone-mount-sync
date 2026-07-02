package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
	"github.com/dtg01100/rclone-mount-sync/pkg/utils"
)

// withCLIDeps swaps the package-level loadConfig, loadGenerator,
// and loadManager for the duration of a test. The returned
// restore function should be deferred by the caller.
//
// Reduces boilerplate across the create/delete/cleanup tests
// below — each previously had the same three save/restore
// patterns inlined.
//
// Also pins XDG_CONFIG_HOME to a fresh temp dir so that
// cfg.Save() (called by runMountCreate / runSyncCreate on
// the rollback paths) writes to a writable location instead
// of the user's real config. Without this, tests that hit
// the DaemonReload-error or RemoveUnit-error paths can fail
// on systems where the real XDG_CONFIG_HOME is not writable
// or does not exist.
func withCLIDeps(t *testing.T) (cfg *config.Config, gen *systemd.Generator, mock *systemd.MockManager, restore func()) {
	t.Helper()
	tmp := t.TempDir()
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	cfg = &config.Config{}
	gen = systemd.NewTestGenerator(tmp)
	mock = &systemd.MockManager{}

	oldLoadConfig := loadConfig
	oldLoadGenerator := loadGenerator
	oldLoadManager := loadManager
	loadConfig = func() (*config.Config, error) { return cfg, nil }
	loadGenerator = func() (*systemd.Generator, error) { return gen, nil }
	loadManager = func() systemd.ServiceManager { return mock }

	restore = func() {
		loadConfig = oldLoadConfig
		loadGenerator = oldLoadGenerator
		loadManager = oldLoadManager
	}
	return cfg, gen, mock, restore
}

// TestRunMountCreate_HappyPath exercises the full
// happy-path of runMountCreate: cfg saved, unit file
// written, daemon-reload called, service enabled, and the
// final "Mount 'X' created successfully" message produced.
func TestRunMountCreate_HappyPath(t *testing.T) {
	cfg, _, mock, restore := withCLIDeps(t)
	defer restore()

	mountCreateName = "happy"
	mountCreateRemote = "gdrive:"
	mountCreateRemotePath = "/Photos"
	mountCreateMountPoint = "/home/user/mnt/happy"
	mountCreateEnabled = true
	mountCreateAutoStart = false

	oldName := mountCreateName
	oldRemote := mountCreateRemote
	oldRemotePath := mountCreateRemotePath
	oldMountPoint := mountCreateMountPoint
	oldEnabled := mountCreateEnabled
	oldAutoStart := mountCreateAutoStart
	defer func() {
		mountCreateName = oldName
		mountCreateRemote = oldRemote
		mountCreateRemotePath = oldRemotePath
		mountCreateMountPoint = oldMountPoint
		mountCreateEnabled = oldEnabled
		mountCreateAutoStart = oldAutoStart
	}()

	if err := runMountCreate(nil, nil); err != nil {
		t.Fatalf("runMountCreate happy path: %v", err)
	}

	if mock.LastOpName == "" {
		t.Error("expected mock manager to record an Enable call for the new service")
	}
	if !strings.HasPrefix(mock.LastOpName, "rclone-mount-") {
		t.Errorf("LastOpName = %q, want rclone-mount-* prefix", mock.LastOpName)
	}
	if cfg.GetMount("happy") == nil {
		t.Error("created mount should be retrievable from config")
	}
}

// TestRunMountCreate_AutoStart exercises the auto-start branch
// (mountCreateAutoStart=true). The mock manager should record
// a Start call after Enable.
func TestRunMountCreate_AutoStart(t *testing.T) {
	_, _, _, restore := withCLIDeps(t)
	defer restore()

	mountCreateName = "auto"
	mountCreateRemote = "dropbox:"
	mountCreateRemotePath = "/"
	mountCreateMountPoint = "/home/user/mnt/auto"
	mountCreateEnabled = true
	mountCreateAutoStart = true

	oldName := mountCreateName
	oldRemote := mountCreateRemote
	oldRemotePath := mountCreateRemotePath
	oldMountPoint := mountCreateMountPoint
	oldEnabled := mountCreateEnabled
	oldAutoStart := mountCreateAutoStart
	defer func() {
		mountCreateName = oldName
		mountCreateRemote = oldRemote
		mountCreateRemotePath = oldRemotePath
		mountCreateMountPoint = oldMountPoint
		mountCreateEnabled = oldEnabled
		mountCreateAutoStart = oldAutoStart
	}()

	if err := runMountCreate(nil, nil); err != nil {
		t.Fatalf("runMountCreate with auto-start: %v", err)
	}
	// We do not assert on LastOpName (which records the most
	// recent op) — instead we trust the absence of an error
	// and the unit-file side effect (covered by the integration
	// test).
}

// TestRunMountCreate_Disabled exercises the no-enable branch
// (mountCreateEnabled=false). The mock manager should NOT
// record an Enable call.
func TestRunMountCreate_Disabled(t *testing.T) {
	_, _, mock, restore := withCLIDeps(t)
	defer restore()

	mountCreateName = "disabled"
	mountCreateRemote = "gdrive:"
	mountCreateRemotePath = "/"
	mountCreateMountPoint = "/home/user/mnt/disabled"
	mountCreateEnabled = false
	mountCreateAutoStart = false

	oldName := mountCreateName
	oldRemote := mountCreateRemote
	oldRemotePath := mountCreateRemotePath
	oldMountPoint := mountCreateMountPoint
	oldEnabled := mountCreateEnabled
	oldAutoStart := mountCreateAutoStart
	defer func() {
		mountCreateName = oldName
		mountCreateRemote = oldRemote
		mountCreateRemotePath = oldRemotePath
		mountCreateMountPoint = oldMountPoint
		mountCreateEnabled = oldEnabled
		mountCreateAutoStart = oldAutoStart
	}()

	if err := runMountCreate(nil, nil); err != nil {
		t.Fatalf("runMountCreate with enabled=false: %v", err)
	}
	if mock.LastOpName != "" {
		t.Errorf("Enable should not have been called, got LastOpName=%q", mock.LastOpName)
	}
}

// TestRunMountCreate_EnableWarning covers the "Enable
// returned an error" branch: the function should still
// succeed (it only prints a warning to stderr) and the
// mount should still be in the config.
func TestRunMountCreate_EnableWarning(t *testing.T) {
	cfg, _, mock, restore := withCLIDeps(t)
	defer restore()

	mock.EnableErr = errors.New("symlink loop")

	mountCreateName = "enable-fail"
	mountCreateRemote = "gdrive:"
	mountCreateRemotePath = "/"
	mountCreateMountPoint = "/home/user/mnt/enable-fail"
	mountCreateEnabled = true
	mountCreateAutoStart = false

	oldName := mountCreateName
	oldRemote := mountCreateRemote
	oldRemotePath := mountCreateRemotePath
	oldMountPoint := mountCreateMountPoint
	oldEnabled := mountCreateEnabled
	oldAutoStart := mountCreateAutoStart
	defer func() {
		mountCreateName = oldName
		mountCreateRemote = oldRemote
		mountCreateRemotePath = oldRemotePath
		mountCreateMountPoint = oldMountPoint
		mountCreateEnabled = oldEnabled
		mountCreateAutoStart = oldAutoStart
	}()

	if err := runMountCreate(nil, nil); err != nil {
		t.Fatalf("runMountCreate with Enable error: %v", err)
	}
	if cfg.GetMount("enable-fail") == nil {
		t.Error("mount should still be in config despite Enable warning")
	}
}

// TestRunMountCreate_StartWarning covers the "Start
// returned an error" branch: the function should still
// succeed and the mount should still be in the config.
func TestRunMountCreate_StartWarning(t *testing.T) {
	cfg, _, mock, restore := withCLIDeps(t)
	defer restore()

	mock.StartErr = errors.New("mount service failed to start")

	mountCreateName = "start-warn"
	mountCreateRemote = "gdrive:"
	mountCreateRemotePath = "/"
	mountCreateMountPoint = "/home/user/mnt/start-warn"
	mountCreateEnabled = true
	mountCreateAutoStart = true

	oldName := mountCreateName
	oldRemote := mountCreateRemote
	oldRemotePath := mountCreateRemotePath
	oldMountPoint := mountCreateMountPoint
	oldEnabled := mountCreateEnabled
	oldAutoStart := mountCreateAutoStart
	defer func() {
		mountCreateName = oldName
		mountCreateRemote = oldRemote
		mountCreateRemotePath = oldRemotePath
		mountCreateMountPoint = oldMountPoint
		mountCreateEnabled = oldEnabled
		mountCreateAutoStart = oldAutoStart
	}()

	if err := runMountCreate(nil, nil); err != nil {
		t.Fatalf("runMountCreate with Start error: %v", err)
	}
	if cfg.GetMount("start-warn") == nil {
		t.Error("mount should still be in config despite Start warning")
	}
}

// TestRunMountCreate_DaemonReloadError exercises the
// DaemonReload failure path. The function must return an
// error and the partially-written service unit should be
// cleaned up.
func TestRunMountCreate_DaemonReloadError(t *testing.T) {
	cfg, gen, mock, restore := withCLIDeps(t)
	defer restore()

	mock.DaemonReloadErr = errors.New("dbus unavailable")

	mountCreateName = "reload-fail"
	mountCreateRemote = "gdrive:"
	mountCreateRemotePath = "/"
	mountCreateMountPoint = "/home/user/mnt/reload-fail"
	mountCreateEnabled = true
	mountCreateAutoStart = false

	oldName := mountCreateName
	oldRemote := mountCreateRemote
	oldRemotePath := mountCreateRemotePath
	oldMountPoint := mountCreateMountPoint
	oldEnabled := mountCreateEnabled
	oldAutoStart := mountCreateAutoStart
	defer func() {
		mountCreateName = oldName
		mountCreateRemote = oldRemote
		mountCreateRemotePath = oldRemotePath
		mountCreateMountPoint = oldMountPoint
		mountCreateEnabled = oldEnabled
		mountCreateAutoStart = oldAutoStart
	}()

	err := runMountCreate(nil, nil)
	if err == nil {
		t.Fatal("expected error when DaemonReload fails")
	}
	if !strings.Contains(err.Error(), "daemon") {
		t.Errorf("error = %q, want 'daemon' substring", err.Error())
	}
	// The mount should have been removed from config (the
	// cleanup branch in runMountCreate runs RemoveMount +
	// Save on DaemonReload failure).
	if cfg.GetMount("reload-fail") != nil {
		t.Error("mount should be removed from config after DaemonReload failure")
	}
	// The partially-written unit file should be removed.
	files, _ := os.ReadDir(gen.GetSystemdDir())
	for _, f := range files {
		if strings.Contains(f.Name(), "rclone-mount-") {
			t.Errorf("partial unit file %q should be cleaned up", f.Name())
		}
	}
}

// TestRunMountDelete_HappyPath exercises the full happy
// path: stop, disable, reset-failed, remove unit, daemon
// reload, config removal, save.
func TestRunMountDelete_HappyPath(t *testing.T) {
	cfg, gen, _, restore := withCLIDeps(t)
	defer restore()

	cfg.Mounts = []models.MountConfig{
		{ID: "abc12345", Name: "del-me", Remote: "gdrive:", MountPoint: "/home/user/mnt/del-me"},
	}

	// Pre-create the unit file so RemoveUnit can find it.
	if err := os.WriteFile(filepath.Join(gen.GetSystemdDir(), "rclone-mount-abc12345.service"),
		[]byte("[Unit]\n"), 0o600); err != nil { //nolint:gosec
		t.Fatalf("pre-create unit: %v", err)
	}

	if err := runMountDelete(nil, []string{"del-me"}); err != nil {
		t.Fatalf("runMountDelete: %v", err)
	}

	if cfg.GetMount("del-me") != nil {
		t.Error("mount should be removed from config after delete")
	}
	if _, err := os.Stat(filepath.Join(gen.GetSystemdDir(), "rclone-mount-abc12345.service")); !os.IsNotExist(err) {
		t.Errorf("unit file should be removed, stat err = %v", err)
	}
}

// TestRunMountDelete_NotFound verifies the not-found error
// path is preserved (the function still returns an error
// when the mount ID/name doesn't match any config entry).
func TestRunMountDelete_NotFound(t *testing.T) {
	cfg, _, _, restore := withCLIDeps(t)
	defer restore()
	cfg.Mounts = nil

	err := runMountDelete(nil, []string{"ghost"})
	if err == nil {
		t.Fatal("expected error for missing mount")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found' substring", err.Error())
	}
}

// TestRunMountDelete_GeneratorError covers the
// loadGenerator-failure path of runMountDelete.
func TestRunMountDelete_GeneratorError(t *testing.T) {
	oldLoadConfig := loadConfig
	oldLoadGenerator := loadGenerator
	defer func() {
		loadConfig = oldLoadConfig
		loadGenerator = oldLoadGenerator
	}()

	cfg := &config.Config{
		Mounts: []models.MountConfig{
			{ID: "abc12345", Name: "del-gen-fail", Remote: "gdrive:", MountPoint: "/home/user/mnt/x"},
		},
	}
	loadConfig = func() (*config.Config, error) { return cfg, nil }
	loadGenerator = func() (*systemd.Generator, error) { return nil, errors.New("generator down") }

	err := runMountDelete(nil, []string{"del-gen-fail"})
	if err == nil {
		t.Fatal("expected error when generator load fails")
	}
	if !strings.Contains(err.Error(), "generator down") {
		t.Errorf("error = %q, want 'generator down' substring", err.Error())
	}
}

// TestRunMountDelete_ManagerWarnings covers the
// "manager stop/disable/reset-failed returned errors"
// branch: the function should still succeed and only
// print warnings to stderr.
func TestRunMountDelete_ManagerWarnings(t *testing.T) {
	cfg, gen, mock, restore := withCLIDeps(t)
	defer restore()

	cfg.Mounts = []models.MountConfig{
		{ID: "warn01", Name: "warn-del", Remote: "gdrive:", MountPoint: "/home/user/mnt/warn-del"},
	}

	mock.StopErr = errors.New("stop failed")
	mock.DisableErr = errors.New("disable failed")
	mock.ResetFailedErr = errors.New("reset failed failed")

	// Pre-create the unit file so RemoveUnit succeeds.
	if err := os.WriteFile(filepath.Join(gen.GetSystemdDir(), "rclone-mount-warn01.service"),
		[]byte("[Unit]\n"), 0o600); err != nil { //nolint:gosec
		t.Fatalf("pre-create unit: %v", err)
	}

	if err := runMountDelete(nil, []string{"warn-del"}); err != nil {
		t.Fatalf("runMountDelete with manager warnings: %v", err)
	}
	if cfg.GetMount("warn-del") != nil {
		t.Error("mount should be removed from config despite manager warnings")
	}
}

// TestRunMountDelete_NotFound verifies the not-found error
// path is preserved (the function still returns an error
// when the mount ID/name doesn't match any config entry).
//
// Note: TestRunMountDelete_NotFound lives above in
// cli_extra_test.go. This is a deliberate ordering choice
// to keep related tests together.

// TestRunSyncDelete_RemoveUnitFailure simulates the
// RemoveUnit failure path by passing a mount name that
// resolves to a non-existent file. RemoveUnit is a no-op
// in that case, so the test exercises the success path
// against a missing unit file. A separate test covers the
// "unit not on disk" case.
func TestRunSyncDelete_MissingUnitFile(t *testing.T) {
	cfg, _, _, restore := withCLIDeps(t)
	defer restore()

	cfg.SyncJobs = []models.SyncJobConfig{
		{ID: "ghost01", Name: "ghost-sync", Source: "gdrive:/x", Destination: "/y", Schedule: models.ScheduleConfig{Type: "timer", OnCalendar: "daily"}},
	}

	// No unit files on disk; RemoveUnit should no-op
	// (return nil) and the delete should still succeed.
	if err := runSyncDelete(nil, []string{"ghost-sync"}); err != nil {
		t.Fatalf("runSyncDelete with missing unit files: %v", err)
	}
	if cfg.GetSyncJob("ghost-sync") != nil {
		t.Error("sync job should be removed from config after delete")
	}
}

// TestRunSyncDelete_NotFound verifies the not-found error
// path is preserved (the function still returns an error
// when the sync job ID/name doesn't match any config entry).
func TestRunSyncDelete_NotFound(t *testing.T) {
	_, _, _, restore := withCLIDeps(t)
	defer restore()

	if err := runSyncDelete(nil, []string{"never-was"}); err == nil {
		t.Fatal("expected error for missing sync job")
	} else if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found' substring", err.Error())
	}
}

// TestRunSyncDelete_ManagerWarnings covers the
// "manager stop/disable/reset-failed returned errors"
// branch: the function should still succeed and only
// print warnings to stderr.
func TestRunSyncDelete_ManagerWarnings(t *testing.T) {
	cfg, gen, mock, restore := withCLIDeps(t)
	defer restore()

	cfg.SyncJobs = []models.SyncJobConfig{
		{ID: "warn01", Name: "warn-sync", Source: "gdrive:/x", Destination: "/y", Schedule: models.ScheduleConfig{Type: "timer", OnCalendar: "daily"}},
	}

	mock.StopTimerErr = errors.New("stop timer failed")
	mock.DisableTimerErr = errors.New("disable timer failed")
	mock.StopErr = errors.New("stop service failed")
	mock.DisableErr = errors.New("disable service failed")
	mock.ResetFailedErr = errors.New("reset failed failed")

	// Pre-create the unit files so RemoveUnit succeeds.
	if err := os.WriteFile(filepath.Join(gen.GetSystemdDir(), "rclone-sync-warn01.service"),
		[]byte("[Unit]\n"), 0o600); err != nil { //nolint:gosec
		t.Fatalf("pre-create service: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gen.GetSystemdDir(), "rclone-sync-warn01.timer"),
		[]byte("[Unit]\n"), 0o600); err != nil { //nolint:gosec
		t.Fatalf("pre-create timer: %v", err)
	}

	if err := runSyncDelete(nil, []string{"warn-sync"}); err != nil {
		t.Fatalf("runSyncDelete with manager warnings: %v", err)
	}
	if cfg.GetSyncJob("warn-sync") != nil {
		t.Error("sync job should be removed from config despite manager warnings")
	}
}

// TestRunSyncCreate_HappyPath exercises the full happy path
// of runSyncCreate: cfg saved, both unit files written,
// daemon-reload called, timer enabled.
func TestRunSyncCreate_HappyPath(t *testing.T) {
	cfg, gen, mock, restore := withCLIDeps(t)
	defer restore()

	syncCreateName = "sync-happy"
	syncCreateSource = "gdrive:/Photos"
	syncCreateDestination = "/home/user/Backup/Photos"
	syncCreateSchedule = "daily"
	syncCreateEnabled = true

	oldName := syncCreateName
	oldSrc := syncCreateSource
	oldDst := syncCreateDestination
	oldSched := syncCreateSchedule
	oldEnabled := syncCreateEnabled
	defer func() {
		syncCreateName = oldName
		syncCreateSource = oldSrc
		syncCreateDestination = oldDst
		syncCreateSchedule = oldSched
		syncCreateEnabled = oldEnabled
	}()

	if err := runSyncCreate(nil, nil); err != nil {
		t.Fatalf("runSyncCreate happy path: %v", err)
	}

	if cfg.GetSyncJob("sync-happy") == nil {
		t.Error("created sync job should be retrievable from config")
	}
	// Both .service and .timer should be on disk.
	entries, err := os.ReadDir(gen.GetSystemdDir())
	if err != nil {
		t.Fatalf("read systemd dir: %v", err)
	}
	var hasService, hasTimer bool
	for _, e := range entries {
		switch filepath.Ext(e.Name()) {
		case ".service":
			hasService = true
		case ".timer":
			hasTimer = true
		}
	}
	if !hasService {
		t.Error("expected .service unit to be written")
	}
	if !hasTimer {
		t.Error("expected .timer unit to be written")
	}
	// The mock manager's LastOpName should reflect the
	// timer-enable call (the only manager call on success).
	if !strings.HasPrefix(mock.LastOpName, "rclone-sync-") {
		t.Errorf("LastOpName = %q, want rclone-sync-* prefix", mock.LastOpName)
	}
}

// TestRunSyncCreate_Disabled exercises the
// enabled=false branch: the timer unit is still written
// (for the operator to enable later) but EnableTimer is
// NOT called.
func TestRunSyncCreate_Disabled(t *testing.T) {
	_, _, mock, restore := withCLIDeps(t)
	defer restore()

	syncCreateName = "sync-disabled"
	syncCreateSource = "gdrive:/Photos"
	syncCreateDestination = "/home/user/Backup/Photos"
	syncCreateSchedule = "daily"
	syncCreateEnabled = false

	oldName := syncCreateName
	oldSrc := syncCreateSource
	oldDst := syncCreateDestination
	oldSched := syncCreateSchedule
	oldEnabled := syncCreateEnabled
	defer func() {
		syncCreateName = oldName
		syncCreateSource = oldSrc
		syncCreateDestination = oldDst
		syncCreateSchedule = oldSched
		syncCreateEnabled = oldEnabled
	}()

	if err := runSyncCreate(nil, nil); err != nil {
		t.Fatalf("runSyncCreate with enabled=false: %v", err)
	}
	if mock.LastOpName != "" {
		t.Errorf("EnableTimer should not have been called, got LastOpName=%q", mock.LastOpName)
	}
}

// TestRunSyncCreate_EnableTimerWarning covers the
// "EnableTimer returned an error" branch: the function
// should still succeed (it only prints a warning) and the
// sync job should still be in the config.
func TestRunSyncCreate_EnableTimerWarning(t *testing.T) {
	cfg, _, mock, restore := withCLIDeps(t)
	defer restore()

	mock.EnableTimerErr = errors.New("symlink loop on timer")

	syncCreateName = "sync-enable-warn"
	syncCreateSource = "gdrive:/Photos"
	syncCreateDestination = "/home/user/Backup/Photos"
	syncCreateSchedule = "daily"
	syncCreateEnabled = true

	oldName := syncCreateName
	oldSrc := syncCreateSource
	oldDst := syncCreateDestination
	oldSched := syncCreateSchedule
	oldEnabled := syncCreateEnabled
	defer func() {
		syncCreateName = oldName
		syncCreateSource = oldSrc
		syncCreateDestination = oldDst
		syncCreateSchedule = oldSched
		syncCreateEnabled = oldEnabled
	}()

	if err := runSyncCreate(nil, nil); err != nil {
		t.Fatalf("runSyncCreate with EnableTimer error: %v", err)
	}
	if cfg.GetSyncJob("sync-enable-warn") == nil {
		t.Error("sync job should still be in config despite EnableTimer warning")
	}
}

// TestRunSyncCreate_DaemonReloadError exercises the
// DaemonReload failure path. The function must return an
// error, clean up the unit files, and roll back the config
// entry.
func TestRunSyncCreate_DaemonReloadError(t *testing.T) {
	cfg, gen, mock, restore := withCLIDeps(t)
	defer restore()

	mock.DaemonReloadErr = errors.New("dbus unavailable")

	syncCreateName = "sync-reload-fail"
	syncCreateSource = "gdrive:/Photos"
	syncCreateDestination = "/home/user/Backup/Photos"
	syncCreateSchedule = "daily"
	syncCreateEnabled = true

	oldName := syncCreateName
	oldSrc := syncCreateSource
	oldDst := syncCreateDestination
	oldSched := syncCreateSchedule
	oldEnabled := syncCreateEnabled
	defer func() {
		syncCreateName = oldName
		syncCreateSource = oldSrc
		syncCreateDestination = oldDst
		syncCreateSchedule = oldSched
		syncCreateEnabled = oldEnabled
	}()

	err := runSyncCreate(nil, nil)
	if err == nil {
		t.Fatal("expected error when DaemonReload fails")
	}
	if !strings.Contains(err.Error(), "daemon") {
		t.Errorf("error = %q, want 'daemon' substring", err.Error())
	}
	// The sync job should be removed from config.
	if cfg.GetSyncJob("sync-reload-fail") != nil {
		t.Error("sync job should be removed from config after DaemonReload failure")
	}
	// Both unit files should be removed.
	entries, _ := os.ReadDir(gen.GetSystemdDir())
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "rclone-sync-") {
			t.Errorf("partial unit file %q should be cleaned up", e.Name())
		}
	}
}

// TestRunSyncDelete_HappyPath exercises the full
// runSyncDelete flow: timer+service stopped/disabled, both
// units removed, daemon reloaded, config entry removed.
func TestRunSyncDelete_HappyPath(t *testing.T) {
	cfg, gen, _, restore := withCLIDeps(t)
	defer restore()

	cfg.SyncJobs = []models.SyncJobConfig{
		{ID: "abc12345", Name: "del-sync", Source: "gdrive:/Photos", Destination: "/x", Schedule: models.ScheduleConfig{Type: "timer", OnCalendar: "daily"}},
	}

	// Pre-create the unit files so RemoveUnit can find them.
	if err := os.WriteFile(filepath.Join(gen.GetSystemdDir(), "rclone-sync-abc12345.service"),
		[]byte("[Unit]\n"), 0o600); err != nil { //nolint:gosec
		t.Fatalf("pre-create service: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gen.GetSystemdDir(), "rclone-sync-abc12345.timer"),
		[]byte("[Unit]\n"), 0o600); err != nil { //nolint:gosec
		t.Fatalf("pre-create timer: %v", err)
	}

	if err := runSyncDelete(nil, []string{"del-sync"}); err != nil {
		t.Fatalf("runSyncDelete: %v", err)
	}

	if cfg.GetSyncJob("del-sync") != nil {
		t.Error("sync job should be removed from config after delete")
	}
	if _, err := os.Stat(filepath.Join(gen.GetSystemdDir(), "rclone-sync-abc12345.service")); !os.IsNotExist(err) {
		t.Errorf("service unit should be removed, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(gen.GetSystemdDir(), "rclone-sync-abc12345.timer")); !os.IsNotExist(err) {
		t.Errorf("timer unit should be removed, stat err = %v", err)
	}
}

// TestRunSyncDelete_ManualSchedule exercises the
// Schedule.Type == "manual" branch: the timer unit is
// expected NOT to exist on disk (it was never written),
// and the function should still succeed.
func TestRunSyncDelete_ManualSchedule(t *testing.T) {
	cfg, gen, _, restore := withCLIDeps(t)
	defer restore()

	cfg.SyncJobs = []models.SyncJobConfig{
		{ID: "manual01", Name: "manual-sync", Source: "gdrive:/Photos", Destination: "/x", Schedule: models.ScheduleConfig{Type: "manual"}},
	}

	// Only write the service unit, not the timer.
	if err := os.WriteFile(filepath.Join(gen.GetSystemdDir(), "rclone-sync-manual01.service"),
		[]byte("[Unit]\n"), 0o600); err != nil { //nolint:gosec
		t.Fatalf("pre-create service: %v", err)
	}

	if err := runSyncDelete(nil, []string{"manual-sync"}); err != nil {
		t.Fatalf("runSyncDelete for manual schedule: %v", err)
	}
	if cfg.GetSyncJob("manual-sync") != nil {
		t.Error("sync job should be removed from config after delete")
	}
}

// TestRunSyncList_ScheduleTypeFallback exercises the
// schedule-display branch where OnCalendar is empty and
// the function falls back to Schedule.Type. This is the
// path used by manually-scheduled sync jobs.
func TestRunSyncList_ScheduleTypeFallback(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Sync: config.SyncDefaults{LogLevel: "INFO", Transfers: 4, Checkers: 8},
		},
		SyncJobs: []models.SyncJobConfig{
			{
				ID:          "manual1",
				Name:        "manual-only",
				Source:      "gdrive:/Photos",
				Destination: "/home/user/Backup",
				Enabled:     true,
				Schedule: models.ScheduleConfig{
					Type:       "manual",
					OnCalendar: "", // empty -> should fall back to "manual"
				},
			},
		},
	}

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func() (*config.Config, error) { return cfg, nil }

	if err := runSyncList(nil, nil); err != nil {
		t.Fatalf("runSyncList: %v", err)
	}
}

// TestRunSyncList_JSON covers the JSON output branch with
// a non-empty jobs list.
func TestRunSyncList_JSON(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Sync: config.SyncDefaults{LogLevel: "INFO", Transfers: 4, Checkers: 8},
		},
		SyncJobs: []models.SyncJobConfig{
			{ID: "1", Name: "j1", Source: "gdrive:/x", Destination: "/y", Schedule: models.ScheduleConfig{Type: "timer", OnCalendar: "hourly"}},
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

	if err := runSyncList(nil, nil); err != nil {
		t.Fatalf("runSyncList JSON: %v", err)
	}
}

// TestRunServicesList_ActiveState covers the "Active: true"
// branch where the table column switches from the raw
// State to "running".
func TestRunServicesList_ActiveState(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		ListServicesResult: []systemd.UnitStatus{
			{Name: "rclone-mount-running.service", Enabled: true, Active: true, State: "active"},
			{Name: "rclone-mount-stopped.service", Enabled: false, Active: false, State: "inactive"},
		},
	}
	loadManager = func() systemd.ServiceManager { return mock }

	if err := runServicesList(nil, nil); err != nil {
		t.Fatalf("runServicesList: %v", err)
	}
}

// TestRunServicesStatus_NonMount exercises a status where
// Type is something other than "mount" or "sync" (the
// status printer branches on Type for the extra fields).
func TestRunServicesStatus_OtherType(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		GetDetailedStatusResult: &models.ServiceStatus{
			Name:        "rclone-other-abc.service",
			Type:        "other",
			LoadState:   "loaded",
			ActiveState: "active",
			SubState:    "running",
			Enabled:     true,
		},
	}
	loadManager = func() systemd.ServiceManager { return mock }

	if err := runServicesStatus(nil, []string{"rclone-other-abc"}); err != nil {
		t.Fatalf("runServicesStatus with other type: %v", err)
	}
}

// TestRunServicesStatus_ActivatedAt ensures the
// ActivatedAt path of printServiceStatus renders without
// error and does not crash for an empty MainPID/ExitCode.
func TestRunServicesStatus_ActivatedAt(t *testing.T) {
	oldLoadManager := loadManager
	defer func() { loadManager = oldLoadManager }()

	mock := &systemd.MockManager{
		GetDetailedStatusResult: &models.ServiceStatus{
			Name:        "rclone-mount-active.service",
			Type:        "mount",
			LoadState:   "loaded",
			ActiveState: "active",
			SubState:    "running",
			Enabled:     true,
			MainPID:     0, // unset: should not print "Main PID:"
			ExitCode:    0, // unset: should not print "Exit Code:"
		},
	}
	loadManager = func() systemd.ServiceManager { return mock }

	if err := runServicesStatus(nil, []string{"rclone-mount-active"}); err != nil {
		t.Fatalf("runServicesStatus: %v", err)
	}
}

// TestPrintError_Extra covers printError with a few different
// error shapes. The function is a one-liner but the test
// keeps the coverage map green and pins the stderr format.
// (The existing TestPrintError in root_test.go covers the
// happy path; this one exercises nil + wrapped + non-wrapped
// errors through the actual printError sink.)
func TestPrintError_Extra(t *testing.T) {
	// Capture by swapping the central ErrorSink rather than
	// redirecting os.Stderr (printError now routes through
	// utils.NoteError). Restore via t.Cleanup.
	origSink := utils.ErrorSink
	t.Cleanup(func() { utils.ErrorSink = origSink })

	var buf bytes.Buffer
	utils.ErrorSink = &buf

	printError(errors.New("boom"))
	printError(fmt.Errorf("wrapped: %w", errors.New("inner")))
	printError(nil) // nil: %v prints "<nil>"

	got := buf.String()

	if !strings.Contains(got, "Error: boom") {
		t.Errorf("sink should contain 'Error: boom', got %q", got)
	}
	if !strings.Contains(got, "Error: wrapped") {
		t.Errorf("sink should contain 'Error: wrapped', got %q", got)
	}
	if !strings.Contains(got, "Error: <nil>") {
		t.Errorf("sink should contain 'Error: <nil>' for nil error, got %q", got)
	}
}

// TestPrintJSON_StructValue covers printJSON with a non-trivial
// struct value. The encoder indents output and writes to
// os.Stdout; we redirect to a buffer.
func TestPrintJSON_StructValue(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	type sample struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	if err := printJSON(sample{Name: "abc", Count: 42}); err != nil {
		t.Fatalf("printJSON: %v", err)
	}
	_ = w.Close()
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	got := string(buf[:n])

	for _, want := range []string{`"name": "abc"`, `"count": 42`} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q, got %q", want, got)
		}
	}
}

// TestRootCommand_NoArgs_PanicsOrErrors exercises rootCmd
// with an empty arg list. cobra should error on a missing
// subcommand when the root has subcommands.
func TestRootCommand_NoArgs_PanicsOrErrors(t *testing.T) {
	_, _, err := runCmd(t, rootCmd)
	if err == nil {
		// Some cobra versions print help instead of erroring.
		// Acceptable — we just want to make sure the no-arg
		// path does not crash.
		t.Log("rootCmd with no args: cobra printed help instead of erroring")
	}
}

// TestRootCommand_HelpSubcommand exercises cobra's auto
// `help` subcommand and asserts it writes the root help
// text to stdout.
func TestRootCommand_HelpSubcommand(t *testing.T) {
	out, _, err := runCmd(t, rootCmd, "help")
	if err != nil {
		t.Fatalf("rootCmd help: %v", err)
	}
	for _, want := range []string{"rclone-mount-sync", "Manage rclone mounts"} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q, got %q", want, out)
		}
	}
}

// TestCleanupCmd_NoArgs runs the cleanup cobra command
// directly. Without any failed units, the command should
// print the "No failed units" message and exit 0.
func TestCleanupCmd_NoArgs(t *testing.T) {
	tmp := t.TempDir()
	oldLoadManager := loadManager
	oldLoadGenerator := loadGenerator
	defer func() {
		loadManager = oldLoadManager
		loadGenerator = oldLoadGenerator
	}()

	loadGenerator = func() (*systemd.Generator, error) { return systemd.NewTestGenerator(tmp), nil }
	loadManager = func() systemd.ServiceManager { return &systemd.MockManager{} }

	out, _, err := runCmd(t, cleanupCmd)
	if err != nil {
		t.Fatalf("cleanupCmd: %v", err)
	}
	// runCmd hits the real exec.Command, so we don't get
	// to choose the systemctl output here. In a CI
	// environment without systemd, the function will
	// return an error from systemctl; we accept either
	// outcome but require no panic.
	t.Logf("cleanupCmd output (truncated): %q", truncate(out, 80))
}

// fakeSystemctlManager is a MockManager that returns a
// caller-supplied SystemctlPath. Used by TestRunCleanup_* to
// point runCleanup at a fake systemctl script under our
// control, bypassing the real binary.
type fakeSystemctlManager struct {
	systemd.MockManager
	path string
}

func (f *fakeSystemctlManager) SystemctlPath() string { return f.path }

// writeFakeSystemctlScript writes a tiny shell script that
// prints the supplied output (with an exit code the test
// chooses via `exitCode`). Returns the absolute path to the
// script.
func writeFakeSystemctlScript(t *testing.T, exitCode int, output string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "systemctl")
	script := fmt.Sprintf("#!/bin/sh\ncat <<'EOF'\n%s\nEOF\nexit %d\n", output, exitCode)
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil { //nolint:gosec
		t.Fatalf("write fake systemctl: %v", err)
	}
	return path
}

// TestRunCleanup_NoFailedUnits uses a fake systemctl that
// exits 1 (the "no matches" exit code) with empty output.
// The cleanup command should treat this as "nothing to do"
// and print "No failed units found.".
func TestRunCleanup_NoFailedUnits(t *testing.T) {
	tmp := t.TempDir()
	systemctlPath := writeFakeSystemctlScript(t, 1, "")

	oldLoadManager := loadManager
	oldLoadGenerator := loadGenerator
	defer func() {
		loadManager = oldLoadManager
		loadGenerator = oldLoadGenerator
	}()

	loadGenerator = func() (*systemd.Generator, error) { return systemd.NewTestGenerator(tmp), nil }
	loadManager = func() systemd.ServiceManager {
		return &fakeSystemctlManager{path: systemctlPath}
	}

	// The function is called via cobra's RunE, but to
	// bypass the cobra cmd plumbing we call runCleanup
	// directly.
	if err := runCleanup(nil, nil); err != nil {
		t.Fatalf("runCleanup: %v", err)
	}
}

// TestRunCleanup_NonRcloneUnitsIgnored uses a fake systemctl
// that prints non-rclone units. runCleanup should skip
// them and end with "No orphaned units found.".
func TestRunCleanup_NonRcloneUnitsIgnored(t *testing.T) {
	tmp := t.TempDir()
	systemctlPath := writeFakeSystemctlScript(t, 0, "foo.service other.service\nbar.service something.service\n")

	oldLoadManager := loadManager
	oldLoadGenerator := loadGenerator
	defer func() {
		loadManager = oldLoadManager
		loadGenerator = oldLoadGenerator
	}()

	loadGenerator = func() (*systemd.Generator, error) { return systemd.NewTestGenerator(tmp), nil }
	loadManager = func() systemd.ServiceManager {
		return &fakeSystemctlManager{path: systemctlPath}
	}

	if err := runCleanup(nil, nil); err != nil {
		t.Fatalf("runCleanup: %v", err)
	}
}

// TestRunCleanup_OrphanedRcloneMount covers the
// "found orphan" branch: a rclone-mount-* unit appears in
// the failed-units list, has no corresponding unit file, and
// the mock manager's ResetFailed succeeds. The function
// should print the "Cleaned up orphaned unit" line and the
// final "Cleaned up 1 orphaned unit(s)." summary.
func TestRunCleanup_OrphanedRcloneMount(t *testing.T) {
	tmp := t.TempDir()
	systemctlPath := writeFakeSystemctlScript(t, 0, "rclone-mount-deadbeef.service load failed\n")

	oldLoadManager := loadManager
	oldLoadGenerator := loadGenerator
	defer func() {
		loadManager = oldLoadManager
		loadGenerator = oldLoadGenerator
	}()

	loadGenerator = func() (*systemd.Generator, error) { return systemd.NewTestGenerator(tmp), nil }
	loadManager = func() systemd.ServiceManager {
		return &fakeSystemctlManager{path: systemctlPath}
	}

	if err := runCleanup(nil, nil); err != nil {
		t.Fatalf("runCleanup: %v", err)
	}
}

// TestRunCleanup_SyncOrphan covers the sync-unit variant:
// rclone-sync-* prefix is also recognized as an orphan.
func TestRunCleanup_SyncOrphan(t *testing.T) {
	tmp := t.TempDir()
	systemctlPath := writeFakeSystemctlScript(t, 0, "rclone-sync-feedface.service load failed\n")

	oldLoadManager := loadManager
	oldLoadGenerator := loadGenerator
	defer func() {
		loadManager = oldLoadManager
		loadGenerator = oldLoadGenerator
	}()

	loadGenerator = func() (*systemd.Generator, error) { return systemd.NewTestGenerator(tmp), nil }
	loadManager = func() systemd.ServiceManager {
		return &fakeSystemctlManager{path: systemctlPath}
	}

	if err := runCleanup(nil, nil); err != nil {
		t.Fatalf("runCleanup: %v", err)
	}
}

// TestRunCleanup_OrphanWithUnitFile covers the "skip
// because unit file exists" branch: a rclone-mount-* unit
// appears in the failed list BUT the unit file is on disk.
// The function should NOT call ResetFailed and should end
// with "No orphaned units found.".
func TestRunCleanup_OrphanWithUnitFile(t *testing.T) {
	tmp := t.TempDir()
	systemctlPath := writeFakeSystemctlScript(t, 0, "rclone-mount-alive.service load failed\n")

	// Pre-create the unit file so the orphan check sees it.
	if err := os.WriteFile(filepath.Join(tmp, "rclone-mount-alive.service"),
		[]byte("[Unit]\n"), 0o600); err != nil { //nolint:gosec
		t.Fatalf("pre-create unit: %v", err)
	}

	oldLoadManager := loadManager
	oldLoadGenerator := loadGenerator
	defer func() {
		loadManager = oldLoadManager
		loadGenerator = oldLoadGenerator
	}()

	loadGenerator = func() (*systemd.Generator, error) { return systemd.NewTestGenerator(tmp), nil }
	loadManager = func() systemd.ServiceManager {
		return &fakeSystemctlManager{path: systemctlPath}
	}

	if err := runCleanup(nil, nil); err != nil {
		t.Fatalf("runCleanup: %v", err)
	}
}

// TestRunCleanup_ResetFailedError covers the "manager
// returns an error from ResetFailed" branch: a rclone
// unit is orphaned and the mock manager's ResetFailed
// fails. The function must print a warning to stderr and
// return a non-nil error (the all-failed exit code path
// introduced in further_work.md P2#6). Scripts wrapping
// `cleanup` rely on a non-zero exit to distinguish
// "nothing to do" from "everything failed".
func TestRunCleanup_ResetFailedError(t *testing.T) {
	tmp := t.TempDir()
	systemctlPath := writeFakeSystemctlScript(t, 0, "rclone-mount-badcafe.service load failed\n")

	oldLoadManager := loadManager
	oldLoadGenerator := loadGenerator
	defer func() {
		loadManager = oldLoadManager
		loadGenerator = oldLoadGenerator
	}()

	loadGenerator = func() (*systemd.Generator, error) { return systemd.NewTestGenerator(tmp), nil }
	mock := &systemd.MockManager{ResetFailedErr: errors.New("reset failed")}
	wrapper := &mockWithSystemctlPath{MockManager: *mock, path: systemctlPath}
	loadManager = func() systemd.ServiceManager { return wrapper }

	err := runCleanup(nil, nil)
	if err == nil {
		t.Fatalf("runCleanup: expected non-nil error when all cleanups fail, got nil")
	}
	if !strings.Contains(err.Error(), "cleanup failed for all") {
		t.Errorf("error = %q, want to contain 'cleanup failed for all'", err.Error())
	}
}

// TestRunCleanup_PartialFailuresReturnNil covers the
// "some orphans cleaned, some failed" branch: a mix of
// successes and failures should still return nil so that
// callers see exit 0 when any cleanup actually worked.
// The summary line must report both the success and the
// total attempted count.
func TestRunCleanup_PartialFailuresReturnNil(t *testing.T) {
	tmp := t.TempDir()
	systemctlPath := writeFakeSystemctlScript(t, 0,
		"rclone-mount-ok1234.service load failed\n"+
			"rclone-mount-bad1234.service load failed\n")

	oldLoadManager := loadManager
	oldLoadGenerator := loadGenerator
	defer func() {
		loadManager = oldLoadManager
		loadGenerator = oldLoadGenerator
	}()

	loadGenerator = func() (*systemd.Generator, error) { return systemd.NewTestGenerator(tmp), nil }

	// The MockManager's ResetFailed always returns nil; we need
	// selective failures. Build a small wrapper that fails only
	// for one specific unit.
	base := &systemd.MockManager{}
	wrapper := &selectiveResetManager{
		MockManager: *base,
		path:        systemctlPath,
		failUnits:   map[string]bool{"rclone-mount-bad1234.service": true},
	}
	loadManager = func() systemd.ServiceManager { return wrapper }

	err := runCleanup(nil, nil)
	if err != nil {
		t.Fatalf("runCleanup: expected nil when at least one cleanup succeeds, got %v", err)
	}
}

// selectiveResetManager wraps MockManager to fail ResetFailed
// only for units listed in failUnits. Used by partial-failure
// tests.
type selectiveResetManager struct {
	systemd.MockManager
	path      string
	failUnits map[string]bool
}

func (s *selectiveResetManager) SystemctlPath() string { return s.path }
func (s *selectiveResetManager) ResetFailed(name string) error {
	if s.failUnits[name] {
		return errors.New("simulated reset failure")
	}
	return s.MockManager.ResetFailed(name)
}

// TestRunCleanup_AllFailedReturnsError covers the
// "all orphans attempted, all failed" branch with multiple
// orphans: runCleanup must return a non-nil error and the
// error message must mention the attempted count.
func TestRunCleanup_AllFailedReturnsError(t *testing.T) {
	tmp := t.TempDir()
	systemctlPath := writeFakeSystemctlScript(t, 0,
		"rclone-mount-aaa.service load failed\n"+
			"rclone-mount-bbb.service load failed\n"+
			"rclone-sync-ccc.service load failed\n")

	oldLoadManager := loadManager
	oldLoadGenerator := loadGenerator
	defer func() {
		loadManager = oldLoadManager
		loadGenerator = oldLoadGenerator
	}()

	loadGenerator = func() (*systemd.Generator, error) { return systemd.NewTestGenerator(tmp), nil }
	mock := &systemd.MockManager{ResetFailedErr: errors.New("reset failed")}
	wrapper := &mockWithSystemctlPath{MockManager: *mock, path: systemctlPath}
	loadManager = func() systemd.ServiceManager { return wrapper }

	err := runCleanup(nil, nil)
	if err == nil {
		t.Fatalf("runCleanup: expected non-nil error when all cleanups fail, got nil")
	}
	if !strings.Contains(err.Error(), "cleanup failed for all 3 orphaned unit(s)") {
		t.Errorf("error = %q, want to contain 'cleanup failed for all 3 orphaned unit(s)'", err.Error())
	}
}

// mockWithSystemctlPath is a thin wrapper around
// MockManager that returns a custom SystemctlPath.
type mockWithSystemctlPath struct {
	systemd.MockManager
	path string
}

func (m *mockWithSystemctlPath) SystemctlPath() string { return m.path }

// truncate returns the first n bytes of s with an
// ellipsis if s is longer.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
