package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestScreen_String(t *testing.T) {
	tests := []struct {
		screen   Screen
		expected string
	}{
		{ScreenMain, "Main Menu"},
		{ScreenMounts, "Mount Management"},
		{ScreenSyncJobs, "Sync Job Management"},
		{ScreenServices, "Service Status"},
		{ScreenSettings, "Settings"},
		{ScreenHelp, "Help"},
		{Screen(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.screen.String()
			if result != tt.expected {
				t.Errorf("Screen(%d).String() = %q, want %q", tt.screen, result, tt.expected)
			}
		})
	}
}

func TestScreen_Constants(t *testing.T) {
	if ScreenMain != 0 {
		t.Errorf("ScreenMain = %d, want 0", ScreenMain)
	}
	if ScreenMounts != 1 {
		t.Errorf("ScreenMounts = %d, want 1", ScreenMounts)
	}
	if ScreenSyncJobs != 2 {
		t.Errorf("ScreenSyncJobs = %d, want 2", ScreenSyncJobs)
	}
	if ScreenServices != 3 {
		t.Errorf("ScreenServices = %d, want 3", ScreenServices)
	}
	if ScreenSettings != 4 {
		t.Errorf("ScreenSettings = %d, want 4", ScreenSettings)
	}
	if ScreenHelp != 5 {
		t.Errorf("ScreenHelp = %d, want 5", ScreenHelp)
	}
}

func TestNewApp(t *testing.T) {
	app := NewApp()

	if app == nil {
		t.Fatal("NewApp() returned nil")
	}

	if app.currentScreen != ScreenMain {
		t.Errorf("currentScreen = %d, want %d", app.currentScreen, ScreenMain)
	}

	if app.mainMenu == nil {
		t.Error("mainMenu should be initialized")
	}

	if app.mounts == nil {
		t.Error("mounts should be initialized")
	}

	if app.syncJobs == nil {
		t.Error("syncJobs should be initialized")
	}

	if app.services == nil {
		t.Error("services should be initialized")
	}

	if app.settings == nil {
		t.Error("settings should be initialized")
	}

	if app.loading {
		t.Error("loading should be false initially")
	}

	if app.showHelp {
		t.Error("showHelp should be false initially")
	}

	if app.initError != nil {
		t.Errorf("initError should be nil initially, got %v", app.initError)
	}
}

func TestApp_Init(t *testing.T) {
	app := NewApp()
	cmd := app.Init()

	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestApp_Update_QuitKey(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	if cmd == nil {
		t.Error("Update with Ctrl+C should return a quit command")
	}
}

func TestApp_Update_QKeyFromMainScreen(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if cmd == nil {
		t.Error("Update with 'q' from main screen should return a quit command")
	}
}

func TestApp_Update_QKeyFromOtherScreen(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMounts

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if updatedApp.(*App).currentScreen != ScreenMain {
		t.Errorf("'q' from non-main screen should navigate to main, got screen %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_Update_EscapeKey(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMounts
	app.showHelp = false

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if updatedApp.(*App).currentScreen != ScreenMain {
		t.Errorf("Escape from non-main screen should navigate to main, got screen %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_Update_HelpToggle(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.showHelp = false

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})

	if !updatedApp.(*App).showHelp {
		t.Error("'?' should toggle help on")
	}
	if updatedApp.(*App).currentScreen != ScreenHelp {
		t.Errorf("'?' should change screen to Help, got %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_Update_HelpClose(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.previousScreen = ScreenMounts
	app.showHelp = true

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if updatedApp.(*App).showHelp {
		t.Error("Escape should close help")
	}
	if updatedApp.(*App).currentScreen != ScreenMounts {
		t.Errorf("Escape from help should return to previous screen, got %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_Update_WindowSize(t *testing.T) {
	app := NewApp()

	_, _ = app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	if app.width != 100 {
		t.Errorf("width = %d, want 100", app.width)
	}
	if app.height != 30 {
		t.Errorf("height = %d, want 30", app.height)
	}
}

func TestApp_View_ZeroSize(t *testing.T) {
	app := NewApp()
	app.width = 0
	app.height = 0

	view := app.View()

	if view != "Loading..." {
		t.Errorf("View with zero size = %q, want 'Loading...'", view)
	}
}

func TestApp_View_Normal(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	view := app.View()

	if view == "" {
		t.Error("View should not be empty")
	}
	if !strings.Contains(view, "Rclone Mount Sync") {
		t.Error("View should contain 'Rclone Mount Sync'")
	}
}

func TestApp_View_WithInitError(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.initError = &testError{msg: "test error"}

	view := app.View()

	if !strings.Contains(view, "Initialization Error") {
		t.Error("View should contain 'Initialization Error'")
	}
	if !strings.Contains(view, "test error") {
		t.Error("View should contain the error message")
	}
}

func TestApp_View_HelpScreen(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.showHelp = true

	view := app.View()

	if !strings.Contains(view, "Help") {
		t.Error("View should contain 'Help'")
	}
}

func TestApp_ScreenChangeMsg(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	updatedApp, _ := app.Update(ScreenChangeMsg{Screen: ScreenMounts})

	if updatedApp.(*App).currentScreen != ScreenMounts {
		t.Errorf("ScreenChangeMsg should change screen to Mounts, got %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_AppInitError(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	testErr := &testError{msg: "init failed"}
	updatedApp, _ := app.Update(AppInitError{Err: testErr})

	if updatedApp.(*App).initError == nil {
		t.Error("AppInitError should set initError")
	}
	if updatedApp.(*App).loading {
		t.Error("AppInitError should set loading to false")
	}
}

func TestApp_RenderHeader(t *testing.T) {
	app := NewApp()
	app.width = 80

	header := app.renderHeader()

	if header == "" {
		t.Error("renderHeader should not return empty string")
	}
	if !strings.Contains(header, "Rclone Mount Sync") {
		t.Error("Header should contain 'Rclone Mount Sync'")
	}
}

func TestApp_RenderStatusBar(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.currentScreen = ScreenMain

	status := app.renderStatusBar()

	if status == "" {
		t.Error("renderStatusBar should not return empty string")
	}
	if !strings.Contains(status, "Main Menu") {
		t.Error("Status bar should contain current screen name")
	}
}

func TestApp_RenderStatusBar_HelpMode(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.showHelp = true

	status := app.renderStatusBar()

	if !strings.Contains(status, "Esc") {
		t.Error("Status bar in help mode should contain 'Esc'")
	}
}

func TestVersion(t *testing.T) {
	if Version != "dev" {
		t.Logf("Version = %q (default is 'dev', can be set at build time)", Version)
	}
}

func TestApp_Update_ScreenNavigation(t *testing.T) {
	tests := []struct {
		name          string
		startScreen   Screen
		targetScreen  Screen
		expectedAfter Screen
	}{
		{"navigate to mounts", ScreenMain, ScreenMounts, ScreenMounts},
		{"navigate to sync jobs", ScreenMain, ScreenSyncJobs, ScreenSyncJobs},
		{"navigate to services", ScreenMain, ScreenServices, ScreenServices},
		{"navigate to settings", ScreenMain, ScreenSettings, ScreenSettings},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp()
			app.width = 80
			app.height = 24
			app.currentScreen = tt.startScreen

			updatedApp, _ := app.Update(ScreenChangeMsg{Screen: tt.targetScreen})

			if updatedApp.(*App).currentScreen != tt.expectedAfter {
				t.Errorf("currentScreen = %d, want %d", updatedApp.(*App).currentScreen, tt.expectedAfter)
			}
		})
	}
}

func TestApp_Update_ScrollInHelp(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.helpScrollY = 1
	app.helpContentLen = 100

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyUp})

	if updatedApp.(*App).helpScrollY != 0 {
		t.Errorf("scroll up should decrement helpScrollY, got %d", updatedApp.(*App).helpScrollY)
	}
}

func TestApp_Update_ScrollDownInHelp(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.helpScrollY = 0
	app.helpContentLen = 100

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})

	if updatedApp.(*App).helpScrollY != 1 {
		t.Errorf("scroll down should increment helpScrollY, got %d", updatedApp.(*App).helpScrollY)
	}
}

func TestApp_Update_ScrollBounds(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.helpScrollY = 0
	app.helpContentLen = 10

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyUp})

	if updatedApp.(*App).helpScrollY < 0 {
		t.Errorf("scroll up at top should not go negative, got %d", updatedApp.(*App).helpScrollY)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
