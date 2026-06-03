package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
)

// TestApp_StateMachine_Navigation tests all valid screen transitions
// using a state machine approach to ensure navigation integrity.
func TestApp_StateMachine_Navigation(t *testing.T) {
	// Define all valid transitions from key presses
	transitions := []struct {
		name         string
		startScreen  Screen
		key          string
		expectScreen Screen
		expectQuit   bool
	}{
		// Main menu navigation
		{"main->mounts via m", ScreenMain, "m", ScreenMounts, false},
		{"main->sync via s", ScreenMain, "s", ScreenSyncJobs, false},
		{"main->services via v", ScreenMain, "v", ScreenServices, false},
		{"main->settings via t", ScreenMain, "t", ScreenSettings, false},
		{"main->quit via q", ScreenMain, "q", ScreenMain, true},

		// Go back from screens to main
		{"mounts->main via q", ScreenMounts, "q", ScreenMain, false},
		{"sync->main via q", ScreenSyncJobs, "q", ScreenMain, false},
		{"services->main via q", ScreenServices, "q", ScreenMain, false},
		{"settings->main via q", ScreenSettings, "q", ScreenMain, false},

		// Escape navigation
		{"mounts->main via esc", ScreenMounts, "esc", ScreenMain, false},
		{"sync->main via esc", ScreenSyncJobs, "esc", ScreenMain, false},
		{"services->main via esc", ScreenServices, "esc", ScreenMain, false},
		{"settings->main via esc", ScreenSettings, "esc", ScreenMain, false},
		{"main stays via esc", ScreenMain, "esc", ScreenMain, false},

		// Ctrl+C always quits
		{"main->quit via ctrl+c", ScreenMain, "ctrl+c", ScreenMain, true},
		{"mounts->quit via ctrl+c", ScreenMounts, "ctrl+c", ScreenMounts, true},
	}

	for _, tt := range transitions {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp("dev")
			app.width = 80
			app.height = 24
			app.currentScreen = tt.startScreen

			var cmd tea.Cmd
			switch tt.key {
			case "ctrl+c":
				_, cmd = app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
			case "esc":
				_, cmd = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
			default:
				_, cmd = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			}

			if tt.expectQuit {
				if cmd == nil {
					t.Error("Expected quit command, got nil")
				} else {
					// A bare `cmd != nil` is necessary but not
					// sufficient: a NavigateToMsg-wrapping cmd is
					// also non-nil. Invoke the cmd and assert it
					// actually produces a tea.QuitMsg — otherwise a
					// regression that returned a navigation cmd from
					// "q" would silently pass.
					got := cmd()
					if _, ok := got.(tea.QuitMsg); !ok {
						t.Errorf("quit cmd produced %T, want tea.QuitMsg", got)
					}
				}
			} else {
				if app.currentScreen != tt.expectScreen {
					t.Errorf("currentScreen = %d, want %d", app.currentScreen, tt.expectScreen)
				}
			}
		})
	}
}

// TestApp_StateMachine_HelpScreen tests help screen transitions.
func TestApp_StateMachine_HelpScreen(t *testing.T) {
	transitions := []struct {
		name         string
		setup        func(*App)
		key          string
		expectScreen Screen
		expectHelp   bool
	}{
		{
			name: "main->help via ?",
			setup: func(a *App) {
				a.currentScreen = ScreenMain
				a.showHelp = false
			},
			key:          "?",
			expectScreen: ScreenHelp,
			expectHelp:   true,
		},
		{
			name: "help->previous via esc",
			setup: func(a *App) {
				a.currentScreen = ScreenHelp
				a.previousScreen = ScreenMounts
				a.showHelp = true
			},
			key:          "esc",
			expectScreen: ScreenMounts,
			expectHelp:   false,
		},
		{
			name: "help->previous via q",
			setup: func(a *App) {
				a.currentScreen = ScreenHelp
				a.previousScreen = ScreenMounts
				a.showHelp = true
			},
			key:          "q",
			expectScreen: ScreenMounts, // 'q' from help returns to previousScreen
			expectHelp:   false,
		},
		{
			name: "mounts->help via ?",
			setup: func(a *App) {
				a.currentScreen = ScreenMounts
				a.showHelp = false
			},
			key:          "?",
			expectScreen: ScreenHelp,
			expectHelp:   true,
		},
	}

	for _, tt := range transitions {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp("dev")
			app.width = 80
			app.height = 24
			tt.setup(app)

			var key tea.KeyMsg
			switch tt.key {
			case "?":
				key = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
			case "esc":
				key = tea.KeyMsg{Type: tea.KeyEsc}
			case "q":
				key = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
			}

			_, _ = app.Update(key)

			if app.currentScreen != tt.expectScreen {
				t.Errorf("currentScreen = %d, want %d", app.currentScreen, tt.expectScreen)
			}
			if app.showHelp != tt.expectHelp {
				t.Errorf("showHelp = %v, want %v", app.showHelp, tt.expectHelp)
			}
		})
	}
}

// TestApp_StateMachine_ScreenChangeMsg tests programmatic screen changes.
func TestApp_StateMachine_ScreenChangeMsg(t *testing.T) {
	screens := []Screen{ScreenMain, ScreenMounts, ScreenSyncJobs, ScreenServices, ScreenSettings}

	for _, target := range screens {
		t.Run("change to "+target.String(), func(t *testing.T) {
			app := NewApp("dev")
			app.width = 80
			app.height = 24
			app.currentScreen = ScreenMain
			app.showHelp = true // Screen change should close help

			_, _ = app.Update(ScreenChangeMsg{Screen: target})

			if app.currentScreen != target {
				t.Errorf("currentScreen = %d, want %d", app.currentScreen, target)
			}
			if app.showHelp {
				t.Error("ScreenChangeMsg should close help")
			}
		})
	}
}

// TestApp_MessageHandling_AllMessageTypes tests that all message types
// are handled without panicking and produce expected state changes.
func TestApp_MessageHandling_AllMessageTypes(t *testing.T) {
	t.Run("LoadingMsg", func(t *testing.T) {
		// LoadingMsg is not currently handled in Update(); the production
		// code initializes `app.loading = true` directly at startup. A
		// previous version of this test only logged and never asserted;
		// skipping instead keeps the gap visible without false coverage.
		t.Skip("LoadingMsg is not handled in Update(); loading state is set elsewhere")
	})

	t.Run("LoadingDoneMsg", func(t *testing.T) {
		// Same as LoadingMsg: not handled. Future improvement: add a
		// case in Update() to set loading=false and a View() overlay,
		// then assert here.
		t.Skip("LoadingDoneMsg is not handled in Update(); loading state is set elsewhere")
	})

	t.Run("AppInitError", func(t *testing.T) {
		app := NewApp("dev")
		app.width = 80
		app.height = 24
		app.loading = true

		testErr := &testError{msg: "init failed"}
		_, _ = app.Update(AppInitError{Err: testErr})

		if app.initError == nil {
			t.Error("AppInitError should set initError")
		}
		if app.loading {
			t.Error("AppInitError should set loading to false")
		}
	})

	t.Run("AppInitDone", func(t *testing.T) {
		app := NewApp("dev")
		app.width = 80
		app.height = 24
		app.loading = true

		_, cmd := app.Update(AppInitDone{})

		if cmd == nil {
			t.Error("AppInitDone should return a batch command")
		}
	})

	t.Run("ReconciliationMsg", func(t *testing.T) {
		app := NewApp("dev")
		app.width = 80
		app.height = 24

		result := &systemd.ReconciliationResult{
			OrphanedUnits: []systemd.OrphanedUnit{
				{Name: "rclone-mount-test.service", ID: "test-id"},
			},
		}

		_, cmd := app.Update(ReconciliationMsg{Result: result})

		if app.orphans == nil {
			t.Error("ReconciliationMsg should set orphans")
		}
		if !app.showOrphanPrompt {
			t.Error("ReconciliationMsg with orphans should show orphan prompt")
		}
		if cmd == nil {
			t.Error("ReconciliationMsg should return a batch command")
		}
	})

	t.Run("ReconciliationMsg_NoOrphans", func(t *testing.T) {
		app := NewApp("dev")
		app.width = 80
		app.height = 24

		result := &systemd.ReconciliationResult{
			OrphanedUnits: []systemd.OrphanedUnit{},
		}

		_, _ = app.Update(ReconciliationMsg{Result: result})

		if app.showOrphanPrompt {
			t.Error("ReconciliationMsg without orphans should not show orphan prompt")
		}
	})

	t.Run("WindowSizeMsg propagates to screens", func(t *testing.T) {
		app := NewApp("dev")

		_, _ = app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

		if app.width != 120 {
			t.Errorf("width = %d, want 120", app.width)
		}
		if app.height != 40 {
			t.Errorf("height = %d, want 40", app.height)
		}
		// Verify screens received size (mainMenu width is set via SetSize)
		if app.mainMenu == nil {
			t.Error("mainMenu should not be nil")
		}
	})
}

// TestApp_Invariants validates that certain invariants hold after any update.
func TestApp_Invariants(t *testing.T) {
	t.Run("width and height are never negative", func(t *testing.T) {
		app := NewApp("dev")

		// Try various window sizes
		sizes := []struct{ w, h int }{
			{0, 0},
			{80, 24},
			{120, 40},
			{200, 60},
		}

		for _, sz := range sizes {
			_, _ = app.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
			if app.width < 0 {
				t.Errorf("width = %d, should be >= 0", app.width)
			}
			if app.height < 0 {
				t.Errorf("height = %d, should be >= 0", app.height)
			}
		}
	})

	t.Run("screen enum is always valid", func(t *testing.T) {
		app := NewApp("dev")
		app.width = 80
		app.height = 24

		// Navigate through all screens
		screens := []Screen{ScreenMain, ScreenMounts, ScreenSyncJobs, ScreenServices, ScreenSettings}
		for _, s := range screens {
			_, _ = app.Update(ScreenChangeMsg{Screen: s})
			if app.currentScreen < ScreenMain || app.currentScreen > ScreenHelp {
				t.Errorf("invalid screen value: %d", app.currentScreen)
			}
		}
	})

	t.Run("help state is consistent with help screen", func(t *testing.T) {
		app := NewApp("dev")
		app.width = 80
		app.height = 24

		// Open help
		_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		if !app.showHelp {
			t.Error("showHelp should be true after pressing ?")
		}
		if app.currentScreen != ScreenHelp {
			t.Errorf("currentScreen = %d, want ScreenHelp", app.currentScreen)
		}

		// Close help
		_, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if app.showHelp {
			t.Error("showHelp should be false after pressing Esc")
		}
		if app.currentScreen == ScreenHelp {
			t.Error("currentScreen should not be ScreenHelp after closing help")
		}
	})

	t.Run("View never panics with any state", func(t *testing.T) {
		// Test with various states
		states := []func(*App){
			func(a *App) { a.width = 0; a.height = 0 },
			func(a *App) { a.width = 80; a.height = 24 },
			func(a *App) { a.width = 80; a.height = 24; a.initError = &testError{msg: "err"} },
			func(a *App) { a.width = 80; a.height = 24; a.loading = true },
			func(a *App) { a.width = 80; a.height = 24; a.currentScreen = ScreenHelp; a.showHelp = true },
		}

		for i, setup := range states {
			app := NewApp("dev")
			setup(app)

			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("View() panicked on state %d: %v", i, r)
				}
			}()

			view := app.View()
			if view == "" && app.width > 0 && app.height > 0 {
				t.Errorf("View() returned empty string on state %d", i)
			}
		}
	})
}

// TestApp_RapidKeyPresses tests that rapid key presses don't cause issues.
func TestApp_RapidKeyPresses(t *testing.T) {
	app := NewApp("dev")
	app.width = 80
	app.height = 24

	// Send a flurry of keys
	keys := []tea.KeyMsg{
		{Type: tea.KeyUp},
		{Type: tea.KeyDown},
		{Type: tea.KeyRunes, Runes: []rune("m")},
		{Type: tea.KeyRunes, Runes: []rune("s")},
		{Type: tea.KeyRunes, Runes: []rune("v")},
		{Type: tea.KeyRunes, Runes: []rune("t")},
		{Type: tea.KeyRunes, Runes: []rune("?")},
		{Type: tea.KeyEsc},
		{Type: tea.KeyRunes, Runes: []rune("q")},
	}

	for _, key := range keys {
		_, _ = app.Update(key)
		// After each key, invariants should hold
		if app.currentScreen < ScreenMain || app.currentScreen > ScreenHelp {
			t.Errorf("invalid screen after key %v: %d", key, app.currentScreen)
		}
	}
}

// TestApp_ScrollBoundaries tests help scroll boundary conditions.
func TestApp_ScrollBoundaries(t *testing.T) {
	t.Run("scroll up at boundary", func(t *testing.T) {
		app := NewApp("dev")
		app.width = 80
		app.height = 24
		app.currentScreen = ScreenHelp
		app.showHelp = true
		app.helpScrollY = 0
		app.helpContentLen = 100

		_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})

		if app.helpScrollY < 0 {
			t.Errorf("helpScrollY = %d, should not be negative", app.helpScrollY)
		}
	})

	t.Run("scroll down at boundary", func(t *testing.T) {
		app := NewApp("dev")
		app.width = 80
		app.height = 24
		app.currentScreen = ScreenHelp
		app.showHelp = true
		app.helpScrollY = 1000 // Way past max
		app.helpContentLen = 100

		_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

		// The scroll logic only increments if below max, but doesn't clamp
		// an already-too-large value. This documents current behavior.
		maxScroll := app.helpContentLen - (app.height - 6)
		t.Logf("helpScrollY=%d, maxScroll=%d (documents current behavior)", app.helpScrollY, maxScroll)
	})

	t.Run("scroll with zero content length", func(t *testing.T) {
		app := NewApp("dev")
		app.width = 80
		app.height = 24
		app.currentScreen = ScreenHelp
		app.showHelp = true
		app.helpScrollY = 0
		app.helpContentLen = 0

		_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

		if app.helpScrollY != 0 {
			t.Errorf("helpScrollY = %d, should stay 0 with no content", app.helpScrollY)
		}
	})
}
