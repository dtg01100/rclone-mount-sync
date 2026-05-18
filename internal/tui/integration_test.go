package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
)

// minimalApp returns an App with all services pre-set so Init() is fast and deterministic.
func minimalApp(t *testing.T) *App {
	t.Helper()
	app := NewApp("dev")
	app.config = &config.Config{
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
	app.width = 80
	app.height = 24
	return app
}

// sendMsg is a helper that calls app.Update and returns the updated *App.
func sendMsg(t *testing.T, app *App, msg tea.Msg) *App {
	t.Helper()
	updated, _ := app.Update(msg)
	return updated.(*App)
}

// sendKey is a helper for sending key messages.
func sendKey(t *testing.T, app *App, key tea.KeyMsg) *App {
	return sendMsg(t, app, key)
}

// keyRune returns a KeyMsg for a rune character.
func keyRune(r string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r)}
}

// TestIntegration_AppInitAndRender verifies the app starts, renders, and processes keys.
func TestIntegration_AppInitAndRender(t *testing.T) {
	app := minimalApp(t)

	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil command")
	}

	view := app.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
	if !strings.Contains(view, "Rclone Mount Sync") {
		t.Error("View() should contain app title")
	}
}

// TestIntegration_MainMenuQuickJumpKeys tests that quick-jump keys from the main
// menu navigate to the correct screen.
func TestIntegration_MainMenuQuickJumpKeys(t *testing.T) {
	tests := []struct {
		key          string
		expectedScreen Screen
	}{
		{"m", ScreenMounts},
		{"s", ScreenSyncJobs},
		{"v", ScreenServices},
		{"t", ScreenSettings},
	}

	for _, tt := range tests {
		t.Run("key_"+tt.key, func(t *testing.T) {
			app := minimalApp(t)
			m := sendKey(t, app, keyRune(tt.key))
			if m.currentScreen != tt.expectedScreen {
				t.Errorf("currentScreen = %d (%s), want %d (%s)",
					m.currentScreen, m.currentScreen,
					tt.expectedScreen, tt.expectedScreen)
			}
		})
	}
}

// TestIntegration_EscapeReturnsToMain tests that pressing Escape from any
// sub-screen returns to the main menu.
func TestIntegration_EscapeReturnsToMain(t *testing.T) {
	// Navigate to each sub-screen, then press Escape, then verify we're back at main.
	screens := []struct {
		name     string
		screen   Screen
		jumpKey  string
	}{
		{"Mounts", ScreenMounts, "m"},
		{"SyncJobs", ScreenSyncJobs, "s"},
		{"Services", ScreenServices, "v"},
		{"Settings", ScreenSettings, "t"},
	}

	for _, tt := range screens {
		t.Run(tt.name, func(t *testing.T) {
			app := minimalApp(t)

			// Jump to sub-screen
			m := sendKey(t, app, keyRune(tt.jumpKey))
			if m.currentScreen != tt.screen {
				t.Fatalf("setup: currentScreen = %d, want %d", m.currentScreen, tt.screen)
			}

			// Press Escape
			m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
			if m.currentScreen != ScreenMain {
				t.Errorf("currentScreen = %d after Escape, want ScreenMain (%d)",
					m.currentScreen, ScreenMain)
			}
		})
	}
}

// TestIntegration_QKeyFromSubScreen tests that 'q' from a sub-screen goes to main.
func TestIntegration_QKeyFromSubScreen(t *testing.T) {
	app := minimalApp(t)

	// Navigate to mounts
	m := sendKey(t, app, keyRune("m"))
	if m.currentScreen != ScreenMounts {
		t.Fatalf("setup: currentScreen = %d, want ScreenMounts", m.currentScreen)
	}

	// Press 'q' to go back
	m = sendKey(t, m, keyRune("q"))
	if m.currentScreen != ScreenMain {
		t.Errorf("currentScreen = %d after 'q', want ScreenMain", m.currentScreen)
	}
}

// TestIntegration_HelpToggle tests that '?' opens help and Escape closes it.
func TestIntegration_HelpToggle(t *testing.T) {
	app := minimalApp(t)

	// Open help with '?'
	m := sendKey(t, app, keyRune("?"))
	if !m.showHelp {
		t.Error("showHelp = false, want true after '?'")
	}
	if m.currentScreen != ScreenHelp {
		t.Errorf("currentScreen = %d, want ScreenHelp (%d)", m.currentScreen, ScreenHelp)
	}

	// Close help with Escape
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.showHelp {
		t.Error("showHelp = true, want false after Escape")
	}
	if m.currentScreen != ScreenMain {
		t.Errorf("currentScreen = %d, want ScreenMain after closing help", m.currentScreen)
	}
}

// TestIntegration_HelpCloseWithQ tests that 'q' also closes the help screen.
func TestIntegration_HelpCloseWithQ(t *testing.T) {
	app := minimalApp(t)
	m := sendKey(t, app, keyRune("?"))
	if m.currentScreen != ScreenHelp {
		t.Fatalf("setup: currentScreen = %d, want ScreenHelp", m.currentScreen)
	}

	m = sendKey(t, m, keyRune("q"))
	if m.showHelp {
		t.Error("showHelp = true, want false after 'q'")
	}
	if m.currentScreen != ScreenMain {
		t.Errorf("currentScreen = %d, want ScreenMain after 'q' from help", m.currentScreen)
	}
}

// TestIntegration_NavigateAllScreensAndBack tests a full round-trip navigation
// flow through all screens using Enter key.
func TestIntegration_NavigateAllScreensAndBack(t *testing.T) {
	app := minimalApp(t)

	// Build a sequence: navigate to each screen via quick-jump, then Escape back.
	steps := []struct {
		name       string
		key        tea.KeyMsg
		wantScreen Screen
	}{
		{"to mounts", keyRune("m"), ScreenMounts},
		{"back from mounts", tea.KeyMsg{Type: tea.KeyEsc}, ScreenMain},
		{"to sync jobs", keyRune("s"), ScreenSyncJobs},
		{"back from sync jobs", tea.KeyMsg{Type: tea.KeyEsc}, ScreenMain},
		{"to services", keyRune("v"), ScreenServices},
		{"back from services", tea.KeyMsg{Type: tea.KeyEsc}, ScreenMain},
		{"to settings", keyRune("t"), ScreenSettings},
		{"back from settings", tea.KeyMsg{Type: tea.KeyEsc}, ScreenMain},
	}

	m := app
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			m = sendKey(t, m, step.key)
			if m.currentScreen != step.wantScreen {
				t.Errorf("currentScreen = %d (%s), want %d (%s)",
					m.currentScreen, m.currentScreen,
					step.wantScreen, step.wantScreen)
			}
		})
	}
}

// TestIntegration_WindowSizeUpdate tests that WindowSizeMsg updates dimensions.
func TestIntegration_WindowSizeUpdate(t *testing.T) {
	app := minimalApp(t)
	app.width = 80
	app.height = 24

	m := sendMsg(t, app, tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}
	if m.height != 40 {
		t.Errorf("height = %d, want 40", m.height)
	}
}

// TestIntegration_ReconciliationMsgWithOrphans tests that orphaned units trigger
// the orphan prompt.
func TestIntegration_ReconciliationMsgWithOrphans(t *testing.T) {
	app := minimalApp(t)

	orphaned := &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "rclone-mount-orphan.service", Type: "mount", Path: "/home/user/.config/systemd/user/rclone-mount-orphan.service", ID: "orphan1", IsLegacy: false},
		},
	}

	m := sendMsg(t, app, ReconciliationMsg{Result: orphaned})

	if !m.showOrphanPrompt {
		t.Error("showOrphanPrompt = false, want true after ReconciliationMsg with orphans")
	}
	if m.orphans == nil {
		t.Error("orphans should be set")
	}
	if len(m.orphans.OrphanedUnits) != 1 {
		t.Errorf("len(orphans.OrphanedUnits) = %d, want 1", len(m.orphans.OrphanedUnits))
	}
}

// TestIntegration_ReconciliationMsgEmpty does not show the orphan prompt when
// there are no orphans.
func TestIntegration_ReconciliationMsgEmpty(t *testing.T) {
	app := minimalApp(t)

	m := sendMsg(t, app, ReconciliationMsg{Result: &systemd.ReconciliationResult{}})

	if m.showOrphanPrompt {
		t.Error("showOrphanPrompt = true, want false for empty reconciliation")
	}
}

// TestIntegration_QuitFromMainMenu tests that both 'q' and Ctrl+C from the main
// menu produce quit commands.
func TestIntegration_QuitFromMainMenu(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"q key", keyRune("q")},
		{"ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := minimalApp(t)
			_, cmd := app.Update(tt.key)
			if cmd == nil {
				t.Error("quit key should return a non-nil command")
			}
		})
	}
}

// TestIntegration_UnknownKeyDoesNotCrash tests that unknown keys leave state
// unchanged without panicking.
func TestIntegration_UnknownKeyDoesNotCrash(t *testing.T) {
	app := minimalApp(t)
	initialScreen := app.currentScreen

	randomKeys := []tea.KeyMsg{
		keyRune("z"),
		keyRune("1"),
		keyRune("@"),
		{Type: tea.KeyF1},
		{Type: tea.KeyF2},
	}

	m := app
	for _, k := range randomKeys {
		m = sendKey(t, m, k)
	}

	if m.currentScreen != initialScreen {
		t.Errorf("currentScreen drifted to %d after random keys, want %d",
			m.currentScreen, initialScreen)
	}
}

// TestIntegration_EscapeFromMainScreenIsNoOp tests that Escape from the main
// screen does nothing (no crash, stays on main).
func TestIntegration_EscapeFromMainScreenIsNoOp(t *testing.T) {
	app := minimalApp(t)

	m := sendKey(t, app, tea.KeyMsg{Type: tea.KeyEsc})
	if m.currentScreen != ScreenMain {
		t.Errorf("currentScreen = %d after Escape from main, want ScreenMain",
			m.currentScreen)
	}
}

// TestIntegration_MultipleEscapesInSubScreen tests that repeated Escape key
// presses in a sub-screen always return to main (not toggling between two
// screens).
func TestIntegration_MultipleEscapesInSubScreen(t *testing.T) {
	app := minimalApp(t)

	// Navigate to mounts
	m := sendKey(t, app, keyRune("m"))
	if m.currentScreen != ScreenMounts {
		t.Fatalf("setup: currentScreen = %d, want ScreenMounts", m.currentScreen)
	}

	// First Escape
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.currentScreen != ScreenMain {
		t.Errorf("currentScreen = %d after 1st Escape, want ScreenMain", m.currentScreen)
	}

	// Second Escape from main — should still be main (no-op)
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.currentScreen != ScreenMain {
		t.Errorf("currentScreen = %d after 2nd Escape, want ScreenMain", m.currentScreen)
	}
}

// TestIntegration_ScreenChangeMsgDirectly tests that ScreenChangeMsg changes
// screen directly regardless of current screen.
func TestIntegration_ScreenChangeMsgDirectly(t *testing.T) {
	app := minimalApp(t)

	targetScreens := []Screen{ScreenMounts, ScreenSyncJobs, ScreenServices, ScreenSettings}
	for _, target := range targetScreens {
		m := sendMsg(t, app, ScreenChangeMsg{Screen: target})
		if m.currentScreen != target {
			t.Errorf("currentScreen = %d after ScreenChangeMsg{%s}, want %d",
				m.currentScreen, target, target)
		}
	}
}

// TestIntegration_AppInitError renders an error screen when initError is set.
func TestIntegration_AppInitError(t *testing.T) {
	app := minimalApp(t)
	app.initError = &testConfigError{msg: "config file not found"}

	view := app.View()
	if !strings.Contains(view, "Initialization Error") {
		t.Error("View() should contain 'Initialization Error' when initError is set")
	}
	if !strings.Contains(view, "config file not found") {
		t.Error("View() should contain the error message")
	}
}

// TestIntegration_LoadingState tests that View() returns "Loading..." when
// dimensions are zero (the app's defined loading/empty state trigger).
// Note: the app.loading field is informational; it does not suppress rendering.
func TestIntegration_LoadingState(t *testing.T) {
	app := minimalApp(t)
	app.width = 0
	app.height = 0

	view := app.View()
	if view != "Loading..." {
		t.Errorf("View() = %q with zero size, want 'Loading...'", view)
	}
}

func TestIntegration_ZeroSizeView(t *testing.T) {
	app := minimalApp(t)
	app.width = 0
	app.height = 0

	view := app.View()
	if view != "Loading..." {
		t.Errorf("View() = %q with zero size, want 'Loading...'", view)
	}
}

// TestIntegration_HelpScrolling tests up/down scrolling within the help screen.
func TestIntegration_HelpScrolling(t *testing.T) {
	app := minimalApp(t)
	app.showHelp = true
	app.currentScreen = ScreenHelp
	app.helpScrollY = 5
	app.helpContentLen = 100
	app.width = 80
	app.height = 24

	// Scroll up
	m := sendKey(t, app, tea.KeyMsg{Type: tea.KeyUp})
	if m.helpScrollY != 4 {
		t.Errorf("helpScrollY = %d after up, want 4", m.helpScrollY)
	}

	// Scroll down
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.helpScrollY != 5 {
		t.Errorf("helpScrollY = %d after down, want 5", m.helpScrollY)
	}

	// Scroll up again
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.helpScrollY != 4 {
		t.Errorf("helpScrollY = %d after second up, want 4", m.helpScrollY)
	}
}

// TestIntegration_HelpScrollAtTop tests that scrolling up at the top does not
// go negative.
func TestIntegration_HelpScrollAtTop(t *testing.T) {
	app := minimalApp(t)
	app.showHelp = true
	app.currentScreen = ScreenHelp
	app.helpScrollY = 0
	app.helpContentLen = 10
	app.width = 80
	app.height = 24

	m := sendKey(t, app, tea.KeyMsg{Type: tea.KeyUp})
	if m.helpScrollY < 0 {
		t.Errorf("helpScrollY = %d, want >= 0 at top boundary", m.helpScrollY)
	}
}

// TestIntegration_ConsecutiveEscapesOnlyAffectSubScreens tests that consecutive
// Escape presses don't "accumulate" and only affect the current sub-screen.
func TestIntegration_ConsecutiveEscapesOnlyAffectSubScreens(t *testing.T) {
	app := minimalApp(t)

	// Navigate to mounts then press Escape twice
	app = sendKey(t, app, keyRune("m"))
	app = sendKey(t, app, tea.KeyMsg{Type: tea.KeyEsc})
	app = sendKey(t, app, tea.KeyMsg{Type: tea.KeyEsc})

	// Second Escape from main should be no-op
	if app.currentScreen != ScreenMain {
		t.Errorf("currentScreen = %d after 2 escapes from mounts, want ScreenMain", app.currentScreen)
	}
}

// TestIntegration_AppInitErrorMsg tests that AppInitError sets the error and
// clears the loading flag.
func TestIntegration_AppInitErrorMsg(t *testing.T) {
	app := minimalApp(t)
	app.loading = true

	m := sendMsg(t, app, AppInitError{Err: &testConfigError{msg: "load failed"}})

	if m.initError == nil {
		t.Error("initError should be set")
	}
	if m.loading {
		t.Error("loading should be false after AppInitError")
	}
}

// TestIntegration_OrphanPromptDismissWithD tests that 'd' dismisses the orphan prompt.
func TestIntegration_OrphanPromptDismissWithD(t *testing.T) {
	app := minimalApp(t)

	// Show orphan prompt
	app.showOrphanPrompt = true
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "rclone-mount-orphan.service", Type: "mount", Path: "/home/user/.config/systemd/user/rclone-mount-orphan.service", ID: "orphan1", IsLegacy: false},
		},
	}
	app.currentScreen = ScreenMain
	app.width = 80
	app.height = 24

	m := sendKey(t, app, keyRune("d"))
	if m.showOrphanPrompt {
		t.Error("showOrphanPrompt = true after 'd', want false")
	}
}

// TestIntegration_OrphanPromptDismissWithEscape tests that 'esc' dismisses the orphan prompt.
func TestIntegration_OrphanPromptDismissWithEscape(t *testing.T) {
	app := minimalApp(t)

	app.showOrphanPrompt = true
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "rclone-mount-orphan.service", Type: "mount", Path: "/home/user/.config/systemd/user/rclone-mount-orphan.service", ID: "orphan1", IsLegacy: false},
		},
	}
	app.currentScreen = ScreenMain
	app.width = 80
	app.height = 24

	m := sendKey(t, app, tea.KeyMsg{Type: tea.KeyEsc})
	if m.showOrphanPrompt {
		t.Error("showOrphanPrompt = true after Escape, want false")
	}
}

// TestIntegration_OrphanPromptNavigation tests up/down navigation between orphans.
func TestIntegration_OrphanPromptNavigation(t *testing.T) {
	app := minimalApp(t)

	app.showOrphanPrompt = true
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "rclone-mount-orphan.service", Type: "mount", Path: "/home/user/.config/systemd/user/rclone-mount-orphan.service", ID: "orphan1", IsLegacy: false},
			{Name: "rclone-sync-orphan.service", Type: "sync", Path: "/home/user/.config/systemd/user/rclone-sync-orphan.service", ID: "orphan2", IsLegacy: false},
		},
	}
	app.currentScreen = ScreenMain
	app.orphanSelected = 0
	app.width = 80
	app.height = 24

	// Navigate down
	m := sendKey(t, app, tea.KeyMsg{Type: tea.KeyDown})
	if m.orphanSelected != 1 {
		t.Errorf("orphanSelected = %d after KeyDown, want 1", m.orphanSelected)
	}

	// Navigate up
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.orphanSelected != 0 {
		t.Errorf("orphanSelected = %d after KeyUp, want 0", m.orphanSelected)
	}
}

// TestIntegration_OrphanActionMsgError sets orphanError when OrphanActionMsg has an error.
func TestIntegration_OrphanActionMsgError(t *testing.T) {
	app := minimalApp(t)
	app.loading = true
	app.showOrphanPrompt = true
	app.currentScreen = ScreenMain
	app.width = 80
	app.height = 24

	m := sendMsg(t, app, OrphanActionMsg{Err: &testConfigError{msg: "import failed"}})

	if m.orphanError == nil {
		t.Error("orphanError should be set")
	}
	if m.loading {
		t.Error("loading should be false after OrphanActionMsg")
	}
}

// TestIntegration_OrphanActionMsgSuccess dismisses prompt on successful orphan action.
func TestIntegration_OrphanActionMsgSuccess(t *testing.T) {
	app := minimalApp(t)
	app.loading = true
	app.showOrphanPrompt = true
	app.currentScreen = ScreenMain
	app.width = 80
	app.height = 24
	// Set up orphans so that len(a.orphans.OrphanedUnits) == 0 triggers dismissal
	app.orphans = &systemd.ReconciliationResult{OrphanedUnits: []systemd.OrphanedUnit{}}
	// Index 0 with 0 orphans means list is now empty after removal
	m := sendMsg(t, app, OrphanActionMsg{Action: "import", Index: 0})

	if m.showOrphanPrompt {
		t.Error("showOrphanPrompt = true after successful import with no remaining orphans, want false")
	}
	if m.loading {
		t.Error("loading should be false after OrphanActionMsg")
	}
}

// TestIntegration_OrphanModeEnterSelectAction tests that Enter on orphanMode=0 transitions to orphanMode=1.
func TestIntegration_OrphanModeEnterSelectAction(t *testing.T) {
	app := minimalApp(t)

	app.showOrphanPrompt = true
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "rclone-mount-orphan.service", Type: "mount", Path: "/home/user/.config/systemd/user/rclone-mount-orphan.service", ID: "orphan1", IsLegacy: false},
		},
	}
	app.orphanMode = 0
	app.orphanSelected = 0
	app.currentScreen = ScreenMain
	app.width = 80
	app.height = 24

	m := sendKey(t, app, tea.KeyMsg{Type: tea.KeyEnter})
	if m.orphanMode != 1 {
		t.Errorf("orphanMode = %d after Enter, want 1 (action selection)", m.orphanMode)
	}
}

// TestIntegration_OrphanModeEscapeReturnsToList tests that Escape in action menu (orphanMode=1) goes back to list (orphanMode=0) and keeps prompt open.
func TestIntegration_OrphanModeEscapeReturnsToList(t *testing.T) {
	app := minimalApp(t)

	app.showOrphanPrompt = true
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "rclone-mount-orphan.service", Type: "mount", Path: "/home/user/.config/systemd/user/rclone-mount-orphan.service", ID: "orphan1", IsLegacy: false},
		},
	}
	app.orphanMode = 1
	app.orphanSelected = 0
	app.currentScreen = ScreenMain
	app.width = 80
	app.height = 24

	m := sendKey(t, app, tea.KeyMsg{Type: tea.KeyEsc})
	if m.orphanMode != 0 {
		t.Errorf("orphanMode = %d after Escape from action menu, want 0", m.orphanMode)
	}
	// Note: showOrphanPrompt stays true when escaping from action menu - only dismisses from list view
	if !m.showOrphanPrompt {
		t.Error("showOrphanPrompt = false after Escape from action menu, want true (stays in prompt)")
	}
}


// --- test helpers ---

type testConfigError struct {
	msg string
}

func (e *testConfigError) Error() string {
	return e.msg
}
