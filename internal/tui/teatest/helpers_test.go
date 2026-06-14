// Package teatest contains program-test (full PTY) coverage for the
// TUI. Each test spins up a real tea.Program via teatest and drives
// it with tea.KeyMsg, asserting on the rendered output.
//
// The tests in this package rely on AppDeps (defined in app.go) to
// inject mocks in place of the real config/rclone/systemd stack. The
// production code path (NewApp + initializeServices) is unchanged.
package teatest

import (
	"io"
	"testing"

	"github.com/charmbracelet/x/exp/teatest"
	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/rclone"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
	"github.com/dtg01100/rclone-mount-sync/internal/tui"
)

// testTermSize is the fixed terminal size used by every teatest
// case. Keeping this constant across the suite makes golden files
// stable; if a test needs a different size, it should pass
// teatest.WithInitialTermSize explicitly.
const (
	testTermWidth  = 100
	testTermHeight = 30
)

// newTestProgram returns a *teatest.TestModel running an App built
// from the supplied deps. The model uses a fixed 100x30 terminal and
// a 2-second default program timeout; tests that need longer (e.g.
// the 5s live tick) should pass teatest.WithProgramTimeout to the
// returned TestModel via tm options or wrap the call.
//
// The deps.Config is saved to a temp XDG_CONFIG_HOME so the
// screens' loadMounts/loadSyncJobs Reload() calls find it on
// disk and don't wipe the in-memory list.
//
// The deps.Config is used as-is; callers typically pass a
// testConfig() so the screens see a known shape.
func newTestProgram(t *testing.T, deps tui.AppDeps) *teatest.TestModel {
	t.Helper()

	// Point XDG at a temp dir so config.Load / config.Reload
	// reads from a known location.
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Persist the test config so the screens' Reload() picks
	// up the same mounts/sync_jobs we set in memory.
	if deps.Config != nil {
		if err := deps.Config.Save(); err != nil {
			t.Fatalf("failed to save test config: %v", err)
		}
	}

	app := tui.NewAppWithDeps("dev", deps)
	tm := teatest.NewTestModel(
		t,
		app,
		teatest.WithInitialTermSize(testTermWidth, testTermHeight),
	)
	// Default 2s program timeout is enough for most tests; tests
	// that send a key that should make the program exit (q / ctrl+c
	// from main) use teatest.WithFinalTimeout via FinalOutput.
	return tm
}

// defaultDeps returns a baseline AppDeps suitable for most tests:
// empty config, no rclone binary, no generator, no manager. Screens
// that need these will surface a friendly error rather than
// panicking, which is exactly what the existing direct-model tests
// rely on.
func defaultDeps() tui.AppDeps {
	return tui.AppDeps{
		Config: testConfig(),
	}
}

// testConfig returns a fresh, valid config with no mounts or sync
// jobs. Tests that need entries should mutate the returned Config
// before constructing the program.
func testConfig() *config.Config {
	return &config.Config{
		Version: "1.0",
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
		Mounts:   []models.MountConfig{},
		SyncJobs: []models.SyncJobConfig{},
	}
}

// testConfigWithMounts returns a config pre-populated with a few
// mount entries. Useful for tests that need to exercise list-mode
// behavior on the Mounts screen.
func testConfigWithMounts(mounts ...models.MountConfig) *config.Config {
	cfg := testConfig()
	cfg.Mounts = append(cfg.Mounts, mounts...)
	return cfg
}

// testConfigWithSyncJobs returns a config pre-populated with a few
// sync job entries.
func testConfigWithSyncJobs(jobs ...models.SyncJobConfig) *config.Config {
	cfg := testConfig()
	cfg.SyncJobs = append(cfg.SyncJobs, jobs...)
	return cfg
}

// sampleMount returns a minimal-valid MountConfig with the given
// name; ID is derived from the name.
func sampleMount(name string) models.MountConfig {
	return models.MountConfig{
		ID:          "id-" + name,
		Name:        name,
		Remote:      "gdrive:",
		RemotePath:  "/" + name,
		MountPoint:  "/home/user/mnt/" + name,
		Description: "test mount " + name,
	}
}

// sampleSyncJob returns a minimal-valid SyncJobConfig with the
// given name.
func sampleSyncJob(name string) models.SyncJobConfig {
	return models.SyncJobConfig{
		ID:          "id-" + name,
		Name:        name,
		Source:      "gdrive:/" + name,
		Destination: "/home/user/Backup/" + name,
		Description: "test sync " + name,
	}
}

// depsWithMockManager returns an AppDeps bundle with a fresh
// *systemd.MockManager pre-attached. The mock records the unit name
// of the most recent operation in LastOpName, so tests can assert
// the start/stop/toggle paths actually invoked the manager with the
// expected name.
func depsWithMockManager() tui.AppDeps {
	deps := defaultDeps()
	deps.Manager = &systemd.MockManager{
		IsSystemdAvailableResult: true,
		// StatusResult: a default-inactive unit. Tests that
		// need a specific status can mutate it after
		// constructing the program.
		StatusResult: &systemd.UnitStatus{
			Active: false,
			State:  "inactive",
		},
	}
	// rclone: use the empty-struct pattern that the existing
	// direct-model tests rely on. IsInstalled() returns false
	// for a zero-value client, so screens that need a remote
	// listing surface a friendly error rather than panicking.
	deps.Rclone = &rclone.Client{}
	return deps
}

// readAll reads all bytes from an io.Reader (typically
// teatest.TestModel.Output) into a string. Returns "" on error so
// callers can use strings.Contains without an extra nil check.
func readAll(r io.Reader) string {
	if r == nil {
		return ""
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return ""
	}
	return string(b)
}
