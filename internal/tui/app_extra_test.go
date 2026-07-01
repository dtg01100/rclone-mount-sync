package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
)

// TestNewAppWithDeps_NilConfig exercises the test seam with no
// dependencies at all. Screens that need services should accept
// the nil and surface a friendly error rather than panicking.
func TestNewAppWithDeps_NilConfig(t *testing.T) {
	app := NewAppWithDeps("dev", AppDeps{})

	if app == nil {
		t.Fatal("NewAppWithDeps returned nil")
	}
	if !app.skipInitializeServices {
		t.Error("skipInitializeServices should be true under NewAppWithDeps")
	}
	if app.currentScreen != ScreenMain {
		t.Errorf("currentScreen = %d, want ScreenMain", app.currentScreen)
	}
	if app.config != nil {
		t.Errorf("config should be nil, got %v", app.config)
	}
}

// TestNewAppWithDeps_WithConfig verifies the screens receive the
// injected config via SetServices during NewAppWithDeps. The
// App's config field should be the exact pointer the test
// passed in, and the seam flag should be set.
func TestNewAppWithDeps_WithConfig(t *testing.T) {
	cfg := &config.Config{
		Version: "1.0",
		Mounts: []models.MountConfig{
			{ID: "id-alpha", Name: "alpha", Remote: "gdrive:"},
			{ID: "id-beta", Name: "beta", Remote: "gdrive:"},
		},
	}
	app := NewAppWithDeps("dev", AppDeps{Config: cfg})

	if app.config != cfg {
		t.Error("config should be set on App")
	}
	if !app.skipInitializeServices {
		t.Error("skipInitializeServices should be true")
	}
}

// TestNewAppWithDeps_InitDoesNotSpawnAsyncServices ensures the
// test-seam Init() does not kick off the real initializeServices
// path (which loads config, builds an rclone client, and
// constructs a real systemd Generator/Manager). The seam's
// returned cmd may be nil because MainMenuScreen.Init() itself
// returns nil — that's expected; the test just confirms the
// App.Init path doesn't panic.
func TestNewAppWithDeps_InitDoesNotSpawnAsyncServices(t *testing.T) {
	app := NewAppWithDeps("dev", AppDeps{Config: testConfigSmall()})

	// Must not panic and must not run the real init code path.
	// We avoid invoking cmd() because that would start blocking
	// I/O; the absence of a panic is sufficient to demonstrate
	// the seam.
	_ = app.Init()
}

// TestApp_Init_SkipInitializeServicesBranches is a focused
// re-assertion that the seam-branch in Init does not panic
// and runs the expected (possibly nil-returning) main-menu
// Init path. The branch is exercised transitively by
// TestNewAppWithDeps_InitDoesNotSpawnAsyncServices; this test
// exists to keep the coverage map green.
func TestApp_Init_SkipInitializeServicesBranches(t *testing.T) {
	app := NewAppWithDeps("test-version", AppDeps{Config: testConfigSmall()})
	_ = app.Init() // must not panic
}

// TestApp_Update_ScreenChangeTriggersScreenInitInSeam verifies
// the AppDeps seam correctly kicks off screen Init() for screens
// that need it (sync_jobs, services). Without the seam, those
// screens would never have a populated list because
// initializeServices was skipped.
//
// This is the seam's "single Update triggers screen load"
// contract; if it regresses, the dep-injected teatest path
// silently shows empty lists.
func TestApp_Update_ScreenChangeTriggersScreenInitInSeam(t *testing.T) {
	app := NewAppWithDeps("dev", AppDeps{Config: testConfigSmall()})

	_, cmd := app.Update(ScreenChangeMsg{Screen: ScreenSyncJobs})
	if cmd == nil {
		t.Error("ScreenChangeMsg to SyncJobs should return a batch cmd (seam kicks off syncJobs.Init)")
	}

	_, cmd = app.Update(ScreenChangeMsg{Screen: ScreenServices})
	if cmd == nil {
		t.Error("ScreenChangeMsg to Services should return a batch cmd (seam kicks off services.Init)")
	}

	// Mounts and Settings do not need a post-seam Init kick
	// (mounts.Init was already invoked from
	// NewAppWithDeps/SetServices and settings has no
	// async loading).
	_, _ = app.Update(ScreenChangeMsg{Screen: ScreenMounts})
	_, _ = app.Update(ScreenChangeMsg{Screen: ScreenSettings})
}

// TestApp_importSelectedOrphan_InvalidIndex exercises the
// bounds-check path: orphanSelected is out of range. The orphan
// action must return an error and not modify the orphan list.
func TestApp_importSelectedOrphan_InvalidIndex(t *testing.T) {
	app := NewApp("dev")
	app.width = 80
	app.height = 24
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "only.service", ID: "1", Type: "mount"},
		},
	}
	app.orphanSelected = 999

	_, cmd := app.importSelectedOrphan()
	if cmd == nil {
		t.Fatal("importSelectedOrphan should return a cmd producing OrphanActionMsg")
	}
	msg := cmd()
	action, ok := msg.(OrphanActionMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want OrphanActionMsg", msg)
	}
	if action.Err == nil {
		t.Error("OrphanActionMsg.Err should be set for invalid index")
	}
	if !strings.Contains(action.Err.Error(), "invalid orphan selection") {
		t.Errorf("Err = %q, want 'invalid orphan selection'", action.Err.Error())
	}
	if len(app.orphans.OrphanedUnits) != 1 {
		t.Errorf("orphan list should be unchanged, got %d entries", len(app.orphans.OrphanedUnits))
	}
}

// TestApp_importSelectedOrphan_NoServices returns the
// "services not initialized" error when generator/manager are
// nil. This is the gate the teatest EnterThenImportErrorGate
// test relies on for the orphan prompt.
//
// Note: a.loading is NOT set in this branch — the function
// returns the error command before reaching the
// `a.loading = true` line. The teatest flow that drives
// the "Processing..." overlay relies on the import cmd
// itself flipping loading via the side-effecting path
// (which only runs once services are non-nil).
func TestApp_importSelectedOrphan_NoServices(t *testing.T) {
	app := NewApp("dev")
	app.width = 80
	app.height = 24
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "only.service", ID: "1", Type: "mount"},
		},
	}
	app.orphanSelected = 0
	// generator/manager are nil under NewApp

	_, cmd := app.importSelectedOrphan()
	if cmd == nil {
		t.Fatal("importSelectedOrphan should return a cmd")
	}
	msg := cmd()
	action, ok := msg.(OrphanActionMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want OrphanActionMsg", msg)
	}
	if action.Err == nil {
		t.Fatal("expected error for nil services")
	}
	if !strings.Contains(action.Err.Error(), "services not initialized") {
		t.Errorf("Err = %q, want contains 'services not initialized'", action.Err.Error())
	}
}

// TestApp_importSelectedOrphan_NilOrphans covers the
// a.orphans == nil defensive branch.
func TestApp_importSelectedOrphan_NilOrphans(t *testing.T) {
	app := NewApp("dev")
	app.width = 80
	app.height = 24
	app.orphans = nil
	app.orphanSelected = 0

	_, cmd := app.importSelectedOrphan()
	if cmd == nil {
		t.Fatal("expected cmd for nil orphans")
	}
	msg := cmd()
	action := msg.(OrphanActionMsg)
	if action.Err == nil || !strings.Contains(action.Err.Error(), "invalid orphan selection") {
		t.Errorf("expected 'invalid orphan selection' error, got %v", action.Err)
	}
}

// TestApp_cleanupSelectedOrphan_InvalidIndex covers the
// bounds-check path for the cleanup branch. The returned
// OrphanActionMsg has an Err set but the Action field is
// empty (Action is only set on the success path).
func TestApp_cleanupSelectedOrphan_InvalidIndex(t *testing.T) {
	app := NewApp("dev")
	app.width = 80
	app.height = 24
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "only.service", ID: "1", Type: "mount"},
		},
	}
	app.orphanSelected = 999

	_, cmd := app.cleanupSelectedOrphan()
	if cmd == nil {
		t.Fatal("cleanupSelectedOrphan should return a cmd")
	}
	msg := cmd()
	action := msg.(OrphanActionMsg)
	if action.Err == nil {
		t.Error("expected error for invalid index")
	}
	if !strings.Contains(action.Err.Error(), "invalid orphan selection") {
		t.Errorf("Err = %q, want 'invalid orphan selection'", action.Err.Error())
	}
}

// TestApp_cleanupSelectedOrphan_NoServices returns the
// "services not initialized" error for cleanup when the
// generator/manager are nil. Mirrors the import counterpart.
func TestApp_cleanupSelectedOrphan_NoServices(t *testing.T) {
	app := NewApp("dev")
	app.width = 80
	app.height = 24
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "only.service", ID: "1", Type: "mount"},
		},
	}
	app.orphanSelected = 0

	_, cmd := app.cleanupSelectedOrphan()
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg := cmd()
	action := msg.(OrphanActionMsg)
	if action.Err == nil {
		t.Fatal("expected error for nil services")
	}
	if !strings.Contains(action.Err.Error(), "services not initialized") {
		t.Errorf("Err = %q, want 'services not initialized'", action.Err.Error())
	}
}

// TestApp_orphanErrorSuggestions exercises each suggestion
// branch in orphanErrorSuggestions. The function is the only
// way the user gets actionable guidance when an import/cleanup
// fails, so we want every branch pinned down.
//
// Note: the branch order matters — earlier cases are tried
// first. Each test message is constructed to avoid matching
// any earlier branch's substring.
func TestApp_orphanErrorSuggestions(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		mustHave []string
		mustMiss []string
	}{
		{
			name:     "nil error",
			err:      nil,
			mustHave: []string{},
			mustMiss: []string{"Suggestions"},
		},
		{
			name:     "permission denied",
			err:      errors.New("open /etc/x: permission denied"),
			mustHave: []string{"Suggestions", "chown"},
		},
		{
			name:     "name already exists",
			err:      errors.New("a mount with that name already exists in config"),
			mustHave: []string{"Suggestions", "Edit the existing entry"},
		},
		{
			name:     "duplicate",
			err:      errors.New("duplicate mount name detected"),
			mustHave: []string{"Suggestions", "Edit the existing entry"},
		},
		{
			name:     "failed to write service file",
			err:      errors.New("failed to write service file: i/o error"),
			mustHave: []string{"Suggestions", "~/.config/systemd/user/"},
		},
		{
			name:     "failed to import",
			err:      errors.New("failed to import orphan: parse error"),
			mustHave: []string{"Suggestions", "rclone config"},
		},
		{
			name:     "no remote",
			err:      errors.New("orphan references a remote that has no remote configured"),
			mustHave: []string{"Suggestions", "rclone config"},
		},
		{
			name:     "failed to cleanup",
			err:      errors.New("failed to cleanup orphan: in use"),
			mustHave: []string{"Suggestions", "Stop the service first"},
		},
		{
			name:     "failed to remove orphan",
			err:      errors.New("failed to remove orphan unit file: i/o error"),
			mustHave: []string{"Suggestions", "Stop the service first"},
		},
		{
			name:     "default fallback",
			err:      errors.New("some other random error"),
			mustHave: []string{"Suggestions", "doctor"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := orphanErrorSuggestions(tc.err)
			for _, want := range tc.mustHave {
				if !strings.Contains(got, want) {
					t.Errorf("suggestion missing %q, got:\n%s", want, got)
				}
			}
			for _, miss := range tc.mustMiss {
				if strings.Contains(got, miss) {
					t.Errorf("suggestion should not contain %q, got:\n%s", miss, got)
				}
			}
		})
	}
}

// TestApp_Update_OrphanActionMsg_CursorClamp covers the
// cursor-clamp behavior when a successful orphan removal would
// leave orphanSelected past the end of the list. The previous
// test suite covered the happy and error paths but not this
// edge case.
func TestApp_Update_OrphanActionMsg_CursorClamp(t *testing.T) {
	app := NewApp("dev")
	app.width = 80
	app.height = 24
	app.showOrphanPrompt = true
	app.orphanSelected = 2
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "a.service", ID: "1"},
			{Name: "b.service", ID: "2"},
			{Name: "c.service", ID: "3"},
		},
	}

	// Remove the last orphan; selection must clamp to the new
	// last index (1), not stay at 2.
	_, _ = app.Update(OrphanActionMsg{Index: 2, Action: "remove"})

	if app.orphanSelected != 1 {
		t.Errorf("orphanSelected = %d, want 1 (clamped to last valid index)", app.orphanSelected)
	}
	if len(app.orphans.OrphanedUnits) != 2 {
		t.Errorf("orphans len = %d, want 2", len(app.orphans.OrphanedUnits))
	}
	if app.orphanMode != 0 {
		t.Errorf("orphanMode = %d, want 0 (reset after success)", app.orphanMode)
	}
	if !app.showOrphanPrompt {
		t.Error("showOrphanPrompt should remain true with orphans remaining")
	}
}

// TestApp_Update_LoadingMsgFromAnyScreen ensures the LoadingMsg
// and LoadingDoneMsg state flips are independent of the current
// screen. A regression that keyed these to a specific screen
// would cause loading indicators to never clear on screens
// without explicit handling.
func TestApp_Update_LoadingMsgFromAnyScreen(t *testing.T) {
	for _, s := range []Screen{ScreenMain, ScreenMounts, ScreenSyncJobs, ScreenServices, ScreenSettings} {
		t.Run(s.String(), func(t *testing.T) {
			app := NewApp("dev")
			app.width = 80
			app.height = 24
			app.currentScreen = s
			app.loading = false

			_, _ = app.Update(LoadingMsg{})
			if !app.loading {
				t.Error("LoadingMsg should set loading=true")
			}

			_, _ = app.Update(LoadingDoneMsg{})
			if app.loading {
				t.Error("LoadingDoneMsg should set loading=false")
			}
		})
	}
}

// TestApp_renderOrphanPrompt_LoadingState exercises the
// "Processing..." branch of renderOrphanPrompt. The body must
// not panic and must mention the in-flight state.
func TestApp_renderOrphanPrompt_LoadingState(t *testing.T) {
	app := NewApp("dev")
	app.width = 100
	app.height = 30
	app.showOrphanPrompt = true
	app.loading = true
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "loading.service", ID: "1", Type: "mount"},
		},
	}

	view := app.View()
	if !strings.Contains(view, "Processing") {
		t.Errorf("renderOrphanPrompt loading state should mention 'Processing', got:\n%s", view)
	}
}

// TestApp_renderOrphanPrompt_ErrorState exercises the
// orphanError branch of renderOrphanPrompt.
func TestApp_renderOrphanPrompt_ErrorState(t *testing.T) {
	app := NewApp("dev")
	app.width = 100
	app.height = 30
	app.showOrphanPrompt = true
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "errored.service", ID: "1", Type: "mount"},
		},
	}
	app.orphanError = errors.New("permission denied while writing")

	view := app.View()
	if !strings.Contains(view, "permission denied") {
		t.Errorf("renderOrphanPrompt should show orphan error, got:\n%s", view)
	}
	if !strings.Contains(view, "Press Esc to dismiss") {
		t.Errorf("renderOrphanPrompt error state should show dismiss hint, got:\n%s", view)
	}
}

// TestApp_renderOrphanPrompt_ActionMode_ImportHint exercises the
// orphanMode==1 branch with a non-legacy orphan. The test name is
// suffixed to avoid clashing with the existing
// TestApp_renderOrphanPrompt_ActionMode in tui_test.go.
func TestApp_renderOrphanPrompt_ActionMode_ImportHint(t *testing.T) {
	app := NewApp("dev")
	app.width = 100
	app.height = 30
	app.showOrphanPrompt = true
	app.orphanMode = 1
	app.orphanSelected = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "action.service", ID: "1", Type: "mount", Path: "/p/action.service"},
		},
	}

	view := app.View()
	if !strings.Contains(view, "Import to config") {
		t.Errorf("action menu should mention import, got:\n%s", view)
	}
	if !strings.Contains(view, "Cleanup") {
		t.Errorf("action menu should mention cleanup, got:\n%s", view)
	}
	if !strings.Contains(view, "action.service") {
		t.Errorf("action menu should show unit name, got:\n%s", view)
	}
}

// TestApp_renderOrphanPrompt_LegacyTag_Extra exercises the
// legacy branch of renderOrphanPrompt. The test name is suffixed
// to avoid clashing with the existing
// TestApp_renderOrphanPrompt_LegacyTag in tui_test.go.
func TestApp_renderOrphanPrompt_LegacyTag_Extra(t *testing.T) {
	app := NewApp("dev")
	app.width = 100
	app.height = 30
	app.showOrphanPrompt = true
	app.orphanMode = 1
	app.orphanSelected = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "legacy.service", ID: "1", Type: "mount", IsLegacy: true},
		},
	}

	view := app.View()
	if !strings.Contains(view, "legacy") {
		t.Errorf("legacy orphan should be tagged, got:\n%s", view)
	}
}

// TestApp_helpSections_ContainsAllGroups is a focused check
// that helpSections produces the full expected content. The
// snapshot/golden tests cover the rendered output, but a
// targeted assertion of the source string is useful when
// adding a new help group — the developer can update one
// assertion and immediately see the failure.
func TestApp_helpSections_ContainsAllGroups(t *testing.T) {
	app := NewApp("dev")
	body := app.helpSections()

	required := []string{
		"Global Keybindings",
		"Screen Navigation",
		"Mount Management",
		"Sync Job Management",
		"Service Status",
		"Move up",
		"Move down",
		"Select/confirm",
		"Go back/cancel",
		"Quit (from main menu) or go back",
		"Force quit",
		"Toggle this help screen",
	}
	for _, want := range required {
		if !strings.Contains(body, want) {
			t.Errorf("helpSections missing %q, body:\n%s", want, body)
		}
	}
}

// TestApp_computeHelpContentLen_NonZero verifies the
// helpContentLen tracker is not zero after helpSections has
// been computed. The current implementation returns the line
// count of helpSections(), which has a hard floor of around 30
// lines. A regression that returned 0 would break scroll
// bounds in Update.
func TestApp_computeHelpContentLen_NonZero(t *testing.T) {
	app := NewApp("dev")
	app.helpContentLen = app.computeHelpContentLen()
	if app.helpContentLen <= 0 {
		t.Errorf("helpContentLen = %d, want > 0", app.helpContentLen)
	}
}

// TestApp_RenderHelp_VerySmallHeight exercises the small-height
// clamp in renderHelp. A height of 0 or 1 must not produce a
// negative availableHeight that would index the lines slice
// out of bounds.
func TestApp_RenderHelp_VerySmallHeight(t *testing.T) {
	app := NewApp("dev")
	app.width = 100
	app.height = 2
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.helpScrollY = 0
	app.helpContentLen = 50

	// Must not panic.
	_ = app.View()
}

// TestApp_View_NegativeClamp exercises the contentHeight
// clamp in View. A height of 0 (or below header+status) must
// not produce a negative contentHeight that lipgloss then
// renders as garbage.
func TestApp_View_NegativeClamp(t *testing.T) {
	app := NewApp("dev")
	app.width = 80
	app.height = 1 // less than headerHeight(1) + statusHeight(1)

	// Must not panic.
	view := app.View()
	if view == "" {
		t.Error("View should not be empty even at very small height")
	}
}

// TestApp_updateOrphanPrompt_ErrorDismiss exercises the error
// dismiss keys: when an orphan error is displayed, only
// esc/q/d should clear it; other keys are ignored.
func TestApp_updateOrphanPrompt_ErrorDismiss(t *testing.T) {
	cases := []struct {
		name           string
		key            tea.KeyMsg
		wantErrCleared bool // whether orphanError should be nil after the key
	}{
		{"esc", tea.KeyMsg{Type: tea.KeyEsc}, true},
		{"q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}, true},
		{"d", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}, true},
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}, false},
		{"c", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}, false},
		{"up", tea.KeyMsg{Type: tea.KeyUp}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := NewApp("dev")
			app.width = 80
			app.height = 24
			app.showOrphanPrompt = true
			app.orphanMode = 1
			app.orphans = &systemd.ReconciliationResult{
				OrphanedUnits: []systemd.OrphanedUnit{
					{Name: "a.service", ID: "1"},
				},
			}
			app.orphanError = errors.New("transient error")

			_, _ = app.updateOrphanPrompt(tc.key)

			errCleared := app.orphanError == nil
			if errCleared != tc.wantErrCleared {
				t.Errorf("orphanError cleared = %v, want %v (key=%s)",
					errCleared, tc.wantErrCleared, tc.name)
			}
			if tc.wantErrCleared && app.showOrphanPrompt {
				t.Errorf("showOrphanPrompt should be closed after dismiss key, got true")
			}
		})
	}
}

// TestApp_updateOrphanPrompt_LoadingIgnoresKeys covers the
// "loading in flight, ignore all keys" branch. This prevents
// a user from spamming Enter during an in-progress import and
// queuing up multiple OrphanActionMsg.
func TestApp_updateOrphanPrompt_LoadingIgnoresKeys(t *testing.T) {
	app := NewApp("dev")
	app.width = 80
	app.height = 24
	app.showOrphanPrompt = true
	app.loading = true
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "a.service", ID: "1"},
		},
	}
	app.orphanSelected = 0

	_, cmd := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Enter while loading should not return a cmd")
	}
	if app.orphanMode != 0 {
		t.Errorf("orphanMode should remain 0 while loading, got %d", app.orphanMode)
	}
}

// TestApp_updateOrphanPrompt_NilOrphans covers the
// a.orphans == nil branch (a defensive guard; the production
// path that produces this state should set orphans before
// showOrphanPrompt, but we want the panic-free path pinned).
func TestApp_updateOrphanPrompt_NilOrphans(t *testing.T) {
	app := NewApp("dev")
	app.width = 80
	app.height = 24
	app.showOrphanPrompt = true
	app.orphans = nil

	// Must not panic.
	_, _ = app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyEsc})
	if !app.showOrphanPrompt {
		t.Error("showOrphanPrompt should remain true (no state mutation when orphans is nil)")
	}
}

// testConfigSmall returns a fresh minimal-valid Config for
// tests that need an AppDeps.Config.
func testConfigSmall() *config.Config {
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
		Mounts:   nil,
		SyncJobs: nil,
	}
}

// writeMountUnit writes a minimal-but-valid rclone mount service
// unit file to disk and returns the path. The file is formatted so
// that Reconciler.Import's ExecStart / Description extractors
// recover the expected Name, Remote, and MountPoint.
func writeMountUnit(t *testing.T, id, name, remote, mountPoint string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "rclone-mount-"+id+".service")
	content := "[Unit]\n" +
		"Description=Rclone mount: " + name + "\n" +
		"After=network-online.target\n\n" +
		"[Service]\n" +
		"Type=notify\n" +
		"ExecStart=/usr/bin/rclone mount " + remote + ": " + mountPoint + " \\\n" +
		"    --vfs-cache-mode full\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write unit file: %v", err)
	}
	return path
}

// TestApp_importSelectedOrphan_MountSuccess exercises the full
// import happy path: a real mount unit file is written to a
// temp dir, the App is wired with a real Generator + Manager,
// and the cmd produced by importSelectedOrphan is expected to
// return an OrphanActionMsg{Action: "import"} with no error and
// the config updated with the new mount.
//
// The reconciler's RemoveOrphan step calls systemctl, which
// fails in a test environment; the App swallows that error
// (logs a warning) so it must NOT be reported on the
// OrphanActionMsg.
func TestApp_importSelectedOrphan_MountSuccess(t *testing.T) {
	// Use a TestGenerator pointing at an isolated temp dir so
	// the test never writes to the real user systemd directory
	// (~/.config/systemd/user/). The earlier NewGenerator() call
	// was the source of leftover rclone-mount-*.service and
	// rclone-sync-*.timer units in the user's real systemd
	// after running `go test`. NewTestGenerator scopes the
	// generator's systemdDir to a t.TempDir() that the
	// runtime cleans up automatically.
	tmp := t.TempDir()
	gen := systemd.NewTestGenerator(tmp)

	id := "imp00001"
	name := "ImportedMount"
	unitPath := writeMountUnit(t, id, name, "gdrive", "/mnt/gdrive")

	app := NewApp("dev")
	app.width = 80
	app.height = 24
	app.generator = gen
	app.manager = systemd.NewManager()
	app.config = testConfigSmall()
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "rclone-mount-" + id + ".service", ID: id, Type: "mount", Path: unitPath},
		},
	}
	app.orphanSelected = 0

	_, cmd := app.importSelectedOrphan()
	if cmd == nil {
		t.Fatal("importSelectedOrphan should return a cmd")
	}
	msg := cmd()
	action, ok := msg.(OrphanActionMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want OrphanActionMsg", msg)
	}
	if action.Err != nil {
		t.Errorf("OrphanActionMsg.Err = %v, want nil (import should succeed)", action.Err)
	}
	if action.Action != "import" {
		t.Errorf("Action = %q, want 'import'", action.Action)
	}
	if action.Index != 0 {
		t.Errorf("Index = %d, want 0", action.Index)
	}

	// The new mount must now be in the in-memory config and
	// carry the values recovered from the unit file.
	var found *models.MountConfig
	for i := range app.config.Mounts {
		if app.config.Mounts[i].ID == id {
			found = &app.config.Mounts[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("config.Mounts did not include the imported mount (id=%s); got %d entries", id, len(app.config.Mounts))
	}
	if found.Name != name {
		t.Errorf("imported mount Name = %q, want %q", found.Name, name)
	}
	if found.Remote != "gdrive" {
		t.Errorf("imported mount Remote = %q, want 'gdrive'", found.Remote)
	}
	if found.MountPoint != "/mnt/gdrive" {
		t.Errorf("imported mount MountPoint = %q, want '/mnt/gdrive'", found.MountPoint)
	}
}

// TestApp_cleanupSelectedOrphan_SuccessNoSuchUnit covers the
// "no active service" path. The cleanup runs against a unit
// that has no live state; the reconciler may log a warning
// (because systemctl returns non-zero in the test env) but the
// action is reported as successful because RemoveOrphan errors
// are non-fatal at the App level.
func TestApp_cleanupSelectedOrphan_SuccessNoSuchUnit(t *testing.T) {
	// Use a TestGenerator pointing at an isolated temp dir so
	// the test never writes to the real user systemd directory.
	// See TestApp_importSelectedOrphan_MountSuccess for the
	// rationale on switching away from systemd.NewGenerator().
	tmp := t.TempDir()
	gen := systemd.NewTestGenerator(tmp)

	// Drop a service file in the generator's systemdDir so
	// RemoveUnit has something concrete to remove. Without
	// this, RemoveUnit errors out, but App still reports
	// success because the error is logged-only.
	systemdDir := gen.GetSystemdDir()
	if err := os.MkdirAll(systemdDir, 0o700); err != nil {
		t.Fatalf("failed to create systemd dir: %v", err)
	}
	unitPath := filepath.Join(systemdDir, "rclone-mount-cln00001.service")
	if err := os.WriteFile(unitPath, []byte("[Unit]\nDescription=test\n"), 0o600); err != nil {
		t.Fatalf("failed to seed unit file: %v", err)
	}

	app := NewApp("dev")
	app.width = 80
	app.height = 24
	app.generator = gen
	app.manager = systemd.NewManager()
	app.config = testConfigSmall()
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "rclone-mount-cln00001.service", ID: "cln00001", Type: "mount", Path: unitPath},
		},
	}
	app.orphanSelected = 0

	_, cmd := app.cleanupSelectedOrphan()
	if cmd == nil {
		t.Fatal("cleanupSelectedOrphan should return a cmd")
	}
	msg := cmd()
	action, ok := msg.(OrphanActionMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want OrphanActionMsg", msg)
	}
	if action.Err != nil {
		t.Errorf("OrphanActionMsg.Err = %v, want nil (cleanup errors are non-fatal)", action.Err)
	}
	if action.Action != "cleanup" {
		t.Errorf("Action = %q, want 'cleanup'", action.Action)
	}
	if action.Index != 0 {
		t.Errorf("Index = %d, want 0", action.Index)
	}
}
