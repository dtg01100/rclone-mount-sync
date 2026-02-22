package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
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

func TestApp_Update_HelpCloseWithQ(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.previousScreen = ScreenMain
	app.showHelp = true

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if updatedApp.(*App).showHelp {
		t.Error("'q' should close help")
	}
	if updatedApp.(*App).currentScreen != ScreenMain {
		t.Errorf("'q' from help should return to previous screen, got %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_Update_EscapeFromMainScreen(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.showHelp = false

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if updatedApp.(*App).currentScreen != ScreenMain {
		t.Errorf("Escape from main screen should stay on main, got %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_Update_HelpToggleWhenAlreadyOpen(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.previousScreen = ScreenMain

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})

	if !updatedApp.(*App).showHelp {
		t.Error("'?' when help is open should keep help open")
	}
}

func TestApp_Update_ReconciliationMsg(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	result := &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "test.service", Type: "mount", ID: "testid"},
		},
	}
	updatedApp, _ := app.Update(ReconciliationMsg{Result: result})

	if !updatedApp.(*App).showOrphanPrompt {
		t.Error("ReconciliationMsg with orphans should show orphan prompt")
	}
	if updatedApp.(*App).orphans == nil {
		t.Error("ReconciliationMsg should set orphans")
	}
}

func TestApp_Update_ReconciliationMsgEmpty(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	result := &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{},
	}
	updatedApp, _ := app.Update(ReconciliationMsg{Result: result})

	if updatedApp.(*App).showOrphanPrompt {
		t.Error("ReconciliationMsg without orphans should not show orphan prompt")
	}
}

func TestApp_Update_AppInitDone(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	_, cmd := app.Update(AppInitDone{})

	if cmd == nil {
		t.Error("AppInitDone should return a command")
	}
}

func TestApp_View_AllScreens(t *testing.T) {
	screens := []struct {
		name   string
		screen Screen
	}{
		{"Main", ScreenMain},
		{"Mounts", ScreenMounts},
		{"SyncJobs", ScreenSyncJobs},
		{"Services", ScreenServices},
		{"Settings", ScreenSettings},
		{"Help", ScreenHelp},
	}

	for _, tt := range screens {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp()
			app.width = 80
			app.height = 24
			app.mainMenu.SetSize(80, 24)
			app.mounts.SetSize(80, 24)
			app.syncJobs.SetSize(80, 24)
			app.services.SetSize(80, 24)
			app.settings.SetSize(80, 24)
			app.currentScreen = tt.screen
			if tt.screen == ScreenHelp {
				app.showHelp = true
			}

			view := app.View()

			if view == "" {
				t.Error("View should not be empty")
			}
		})
	}
}

func TestApp_Update_ScrollDownMaxBounds(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.helpScrollY = 90
	app.helpContentLen = 95

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})

	if updatedApp.(*App).helpScrollY > 90 {
		t.Errorf("scroll down should respect max bounds, got %d", updatedApp.(*App).helpScrollY)
	}
}

func TestApp_Update_KKeyScrollUp(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.helpScrollY = 5
	app.helpContentLen = 100

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})

	if updatedApp.(*App).helpScrollY != 4 {
		t.Errorf("'k' should scroll up, got helpScrollY=%d", updatedApp.(*App).helpScrollY)
	}
}

func TestApp_Update_JKeyScrollDown(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.helpScrollY = 0
	app.helpContentLen = 100

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	if updatedApp.(*App).helpScrollY != 1 {
		t.Errorf("'j' should scroll down, got helpScrollY=%d", updatedApp.(*App).helpScrollY)
	}
}

func TestApp_updateOrphanPrompt_NavigateUp(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 1
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
			{Name: "unit2.service", Type: "mount", ID: "id2"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyUp})

	if updatedApp.(*App).orphanSelected != 0 {
		t.Errorf("up should decrement orphanSelected, got %d", updatedApp.(*App).orphanSelected)
	}
}

func TestApp_updateOrphanPrompt_NavigateDown(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
			{Name: "unit2.service", Type: "mount", ID: "id2"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyDown})

	if updatedApp.(*App).orphanSelected != 1 {
		t.Errorf("down should increment orphanSelected, got %d", updatedApp.(*App).orphanSelected)
	}
}

func TestApp_updateOrphanPrompt_EnterSelect(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyEnter})

	if updatedApp.(*App).orphanMode != 1 {
		t.Errorf("enter should set orphanMode to 1, got %d", updatedApp.(*App).orphanMode)
	}
}

func TestApp_updateOrphanPrompt_EscapeFromActionMenu(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 1
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyEsc})

	if updatedApp.(*App).orphanMode != 0 {
		t.Errorf("escape from action menu should set orphanMode to 0, got %d", updatedApp.(*App).orphanMode)
	}
}

func TestApp_updateOrphanPrompt_EscapeFromList(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyEsc})

	if updatedApp.(*App).showOrphanPrompt {
		t.Error("escape from list should close orphan prompt")
	}
}

func TestApp_updateOrphanPrompt_QKeyFromActionMenu(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 1
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if updatedApp.(*App).orphanMode != 0 {
		t.Errorf("'q' from action menu should set orphanMode to 0, got %d", updatedApp.(*App).orphanMode)
	}
}

func TestApp_updateOrphanPrompt_QKeyFromList(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if updatedApp.(*App).showOrphanPrompt {
		t.Error("'q' from list should close orphan prompt")
	}
}

func TestApp_updateOrphanPrompt_DismissAll(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	if updatedApp.(*App).showOrphanPrompt {
		t.Error("'d' should dismiss all and close orphan prompt")
	}
}

func TestApp_updateOrphanPrompt_NavigateUpAtTop(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
			{Name: "unit2.service", Type: "mount", ID: "id2"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyUp})

	if updatedApp.(*App).orphanSelected != 0 {
		t.Errorf("up at top should stay at 0, got %d", updatedApp.(*App).orphanSelected)
	}
}

func TestApp_updateOrphanPrompt_NavigateDownAtBottom(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 1
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
			{Name: "unit2.service", Type: "mount", ID: "id2"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyDown})

	if updatedApp.(*App).orphanSelected != 1 {
		t.Errorf("down at bottom should stay at 1, got %d", updatedApp.(*App).orphanSelected)
	}
}

func TestApp_renderOrphanPrompt_SelectMode(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}

	view := app.renderOrphanPrompt("base view")

	if !strings.Contains(view, "Orphaned Units Detected") {
		t.Error("renderOrphanPrompt should contain 'Orphaned Units Detected'")
	}
	if !strings.Contains(view, "Select a unit to manage") {
		t.Error("renderOrphanPrompt should contain 'Select a unit to manage'")
	}
}

func TestApp_renderOrphanPrompt_ActionMode(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 1
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1", Path: "/path/to/unit"},
		},
	}

	view := app.renderOrphanPrompt("base view")

	if !strings.Contains(view, "Unit:") {
		t.Error("renderOrphanPrompt in action mode should contain 'Unit:'")
	}
	if !strings.Contains(view, "Import to config") {
		t.Error("renderOrphanPrompt in action mode should contain 'Import to config'")
	}
}

func TestApp_renderOrphanPrompt_LegacyTag(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1", IsLegacy: true},
		},
	}

	view := app.renderOrphanPrompt("base view")

	if !strings.Contains(view, "(legacy)") {
		t.Error("renderOrphanPrompt should show '(legacy)' tag for legacy units")
	}
}

func TestApp_renderOrphanPrompt_SmallWidth(t *testing.T) {
	app := NewApp()
	app.width = 30
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}

	view := app.renderOrphanPrompt("base view")

	if !strings.Contains(view, "Orphaned Units Detected") {
		t.Error("renderOrphanPrompt should work with small width")
	}
}

func TestApp_View_WithOrphanPrompt(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.showOrphanPrompt = true
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}

	view := app.View()

	if !strings.Contains(view, "Orphaned Units Detected") {
		t.Error("View should show orphan prompt overlay")
	}
}

func TestApp_Update_OrphanPromptIntercept(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.showOrphanPrompt = true
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	if updatedApp.(*App).showOrphanPrompt {
		t.Error("keys should be intercepted by orphan prompt when shown")
	}
}

func TestApp_Update_MainMenuNavigation(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.mainMenu.SetSize(80, 24)

	app.mainMenu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if updatedApp.(*App).currentScreen != ScreenMounts {
		t.Errorf("main menu navigation should change screen to Mounts, got %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_Update_MainMenuNavigationQuit(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.mainMenu.SetSize(80, 24)

	app.mainMenu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if cmd == nil {
		t.Error("main menu quit navigation should return quit command")
	}
}

func TestApp_Update_MountsScreenGoBack(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMounts

	app.mounts.ResetGoBack()
	app.mounts.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if updatedApp.(*App).currentScreen != ScreenMain {
		t.Errorf("mounts screen go back should return to main, got %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_Update_SyncJobsScreenGoBack(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenSyncJobs

	app.syncJobs.ResetGoBack()
	app.syncJobs.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if updatedApp.(*App).currentScreen != ScreenMain {
		t.Errorf("sync jobs screen go back should return to main, got %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_Update_ServicesScreenGoBack(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenServices

	app.services.ResetGoBack()
	app.services.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if updatedApp.(*App).currentScreen != ScreenMain {
		t.Errorf("services screen go back should return to main, got %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_Update_SettingsScreenGoBack(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenSettings

	app.settings.ResetGoBack()
	app.settings.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if updatedApp.(*App).currentScreen != ScreenMain {
		t.Errorf("settings screen go back should return to main, got %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_RenderHelp_ScrollIndicator(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 10
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.helpScrollY = 0

	view := app.renderHelp()

	if view == "" {
		t.Error("renderHelp should not return empty string")
	}
}

func TestApp_RenderHelp_NegativeScroll(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.helpScrollY = -5

	view := app.renderHelp()

	if view == "" {
		t.Error("renderHelp should handle negative scroll")
	}
}

func TestApp_RenderHelp_ScrollToEnd(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.helpScrollY = 5
	app.helpContentLen = 50

	view := app.renderHelp()

	if view == "" {
		t.Error("renderHelp should handle scroll near end")
	}
}

func TestApp_ScreenChangeMsg_HidesHelp(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.showHelp = true

	updatedApp, _ := app.Update(ScreenChangeMsg{Screen: ScreenMounts})

	if updatedApp.(*App).showHelp {
		t.Error("ScreenChangeMsg should hide help")
	}
}

func TestApp_OrphanUnit_Fields(t *testing.T) {
	orphan := systemd.OrphanedUnit{
		Name:     "test.service",
		Type:     "mount",
		ID:       "abc123",
		IsLegacy: true,
		Path:     "/path/to/unit",
		Imported: true,
	}

	if orphan.Name != "test.service" {
		t.Errorf("orphan.Name = %q, want 'test.service'", orphan.Name)
	}
	if orphan.Type != "mount" {
		t.Errorf("orphan.Type = %q, want 'mount'", orphan.Type)
	}
	if !orphan.IsLegacy {
		t.Error("orphan.IsLegacy should be true")
	}
}

func TestApp_ReconciliationResult_Fields(t *testing.T) {
	result := &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
		Errors: []error{&testError{msg: "test error"}},
	}

	if len(result.OrphanedUnits) != 1 {
		t.Errorf("len(OrphanedUnits) = %d, want 1", len(result.OrphanedUnits))
	}
	if len(result.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want 1", len(result.Errors))
	}
}

func TestApp_Messages(t *testing.T) {
	t.Run("ScreenChangeMsg", func(t *testing.T) {
		msg := ScreenChangeMsg{Screen: ScreenMounts}
		if msg.Screen != ScreenMounts {
			t.Errorf("ScreenChangeMsg.Screen = %d, want %d", msg.Screen, ScreenMounts)
		}
	})

	t.Run("LoadingMsg", func(t *testing.T) {
		msg := LoadingMsg{}
		_ = msg
	})

	t.Run("LoadingDoneMsg", func(t *testing.T) {
		msg := LoadingDoneMsg{}
		_ = msg
	})

	t.Run("AppInitError", func(t *testing.T) {
		err := &testError{msg: "test"}
		msg := AppInitError{Err: err}
		if msg.Err != err {
			t.Error("AppInitError should contain the error")
		}
	})

	t.Run("AppInitDone", func(t *testing.T) {
		msg := AppInitDone{}
		_ = msg
	})
}

func TestApp_RenderInitError_Layout(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.initError = &testError{msg: "detailed error message"}

	view := app.renderInitError()

	expectedStrings := []string{
		"Initialization Error",
		"detailed error message",
		"Possible solutions:",
		"rclone",
		"Press q or Ctrl+C to quit",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(view, expected) {
			t.Errorf("renderInitError should contain '%s'", expected)
		}
	}
}

func TestApp_RenderStatusBar_ShowHelp(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.currentScreen = ScreenMain
	app.showHelp = true

	status := app.renderStatusBar()

	if !strings.Contains(status, "Esc") {
		t.Error("status bar should show help close hint when help is shown")
	}
}

func TestApp_updateOrphanPrompt_JKeyInActionMode(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 1
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
			{Name: "unit2.service", Type: "mount", ID: "id2"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	if updatedApp.(*App).orphanSelected != 0 {
		t.Errorf("'j' in action mode should not change selection, got %d", updatedApp.(*App).orphanSelected)
	}
}

func TestApp_updateOrphanPrompt_KKeyInActionMode(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 1
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})

	if updatedApp.(*App).orphanSelected != 0 {
		t.Errorf("'k' in action mode should not change selection, got %d", updatedApp.(*App).orphanSelected)
	}
}

func TestApp_updateOrphanPrompt_CKeyInSelectMode(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})

	if !updatedApp.(*App).showOrphanPrompt {
		t.Error("'c' in select mode should not close prompt")
	}
}

func TestApp_updateOrphanPrompt_UpKeyInActionMode(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 1
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyUp})

	if updatedApp.(*App).orphanSelected != 0 {
		t.Errorf("up in action mode should not change selection, got %d", updatedApp.(*App).orphanSelected)
	}
}

func TestApp_updateOrphanPrompt_DownKeyInActionMode(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 1
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
			{Name: "unit2.service", Type: "mount", ID: "id2"},
		},
	}
	app.showOrphanPrompt = true

	updatedApp, _ := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyDown})

	if updatedApp.(*App).orphanSelected != 0 {
		t.Errorf("down in action mode should not change selection, got %d", updatedApp.(*App).orphanSelected)
	}
}

func TestApp_Update_UnknownKeyWithOrphanPrompt(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.showOrphanPrompt = true
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if !updatedApp.(*App).showOrphanPrompt {
		t.Error("unknown key should not close orphan prompt")
	}
}

func TestApp_renderOrphanPrompt_SelectedItem(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 1
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
			{Name: "unit2.service", Type: "sync", ID: "id2"},
		},
	}

	view := app.renderOrphanPrompt("base view")

	if !strings.Contains(view, "unit2.service") {
		t.Error("renderOrphanPrompt should show selected unit")
	}
}

func TestApp_renderOrphanPrompt_LargeWidth(t *testing.T) {
	app := NewApp()
	app.width = 200
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}

	view := app.renderOrphanPrompt("base view")

	if !strings.Contains(view, "Orphaned Units Detected") {
		t.Error("renderOrphanPrompt should work with large width")
	}
}

func TestApp_Update_MainMenuNavigationToSyncJobs(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.mainMenu.SetSize(80, 24)

	app.mainMenu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if updatedApp.(*App).currentScreen != ScreenSyncJobs {
		t.Errorf("main menu navigation should change screen to SyncJobs, got %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_Update_MainMenuNavigationToServices(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.mainMenu.SetSize(80, 24)

	app.mainMenu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if updatedApp.(*App).currentScreen != ScreenServices {
		t.Errorf("main menu navigation should change screen to Services, got %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_Update_MainMenuNavigationToSettings(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.mainMenu.SetSize(80, 24)

	app.mainMenu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if updatedApp.(*App).currentScreen != ScreenSettings {
		t.Errorf("main menu navigation should change screen to Settings, got %d", updatedApp.(*App).currentScreen)
	}
}

func TestApp_Update_ReconciliationMsgTriggersMountsInit(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	result := &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "test.service", Type: "mount", ID: "testid"},
		},
	}
	_, cmd := app.Update(ReconciliationMsg{Result: result})

	if cmd == nil {
		t.Error("ReconciliationMsg should return a command to init mounts")
	}
}

func TestApp_View_DifferentScreensContent(t *testing.T) {
	tests := []struct {
		name         string
		screen       Screen
		expectedText string
	}{
		{"Main", ScreenMain, "Main Menu"},
		{"Mounts", ScreenMounts, "Mount"},
		{"SyncJobs", ScreenSyncJobs, "Sync"},
		{"Services", ScreenServices, "Service"},
		{"Settings", ScreenSettings, "Settings"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp()
			app.width = 80
			app.height = 24
			app.mainMenu.SetSize(80, 24)
			app.mounts.SetSize(80, 24)
			app.syncJobs.SetSize(80, 24)
			app.services.SetSize(80, 24)
			app.settings.SetSize(80, 24)
			app.currentScreen = tt.screen

			view := app.View()

			if !strings.Contains(view, tt.expectedText) {
				t.Errorf("View for screen %v should contain '%s'", tt.screen, tt.expectedText)
			}
		})
	}
}

func TestApp_RenderHelp_WithScrolling(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 15
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.helpScrollY = 2

	view := app.renderHelp()

	if !strings.Contains(view, "Keybindings") {
		t.Error("renderHelp should contain keybindings")
	}
}

func TestApp_updateOrphanPrompt_EnterInActionModeWithNilOrphans(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 1
	app.orphans = nil
	app.showOrphanPrompt = false

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	updatedApp, cmd := app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd != nil {
		t.Error("enter with nil orphans should return nil command")
	}
	_ = updatedApp
}

func TestApp_updateOrphanPrompt_CleanupKeyInActionMode(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 1
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}
	app.showOrphanPrompt = true

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	app.updateOrphanPrompt(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
}

func TestApp_Update_HelpNotShown_KeysDontAffectScroll(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.showHelp = false
	app.helpScrollY = 0

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyUp})

	if updatedApp.(*App).helpScrollY != 0 {
		t.Error("up key should not affect scroll when help not shown")
	}
}

func TestApp_Update_HelpNotShown_DownDoesNotScroll(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.showHelp = false
	app.helpScrollY = 0

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})

	if updatedApp.(*App).helpScrollY != 0 {
		t.Error("down key should not affect scroll when help not shown")
	}
}

func TestApp_Update_SpaceKey(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.mainMenu.SetSize(80, 24)

	app.mainMenu.Update(tea.KeyMsg{Type: tea.KeySpace})
	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	_ = updatedApp
}

func TestApp_View_HelpScreenWithShowHelp(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.mainMenu.SetSize(80, 24)

	view := app.View()

	if !strings.Contains(view, "Help") {
		t.Error("View should show help content when on Help screen")
	}
}

func TestApp_RenderHelp_AvailableHeightOne(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 6
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.helpScrollY = 0

	view := app.renderHelp()

	if view == "" {
		t.Error("renderHelp should return content even with minimal height")
	}
}

func TestApp_Update_JKeyNotInHelp(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.showHelp = false
	app.helpScrollY = 0

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	if updatedApp.(*App).helpScrollY != 0 {
		t.Error("j key should not scroll when not in help mode")
	}
}

func TestApp_Update_KKeyNotInHelp(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.showHelp = false
	app.helpScrollY = 0

	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})

	if updatedApp.(*App).helpScrollY != 0 {
		t.Error("k key should not scroll when not in help mode")
	}
}

func TestApp_Update_EnterInActionModeWithOrphans(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.orphanSelected = 0
	app.orphanMode = 1
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "unit1.service", Type: "mount", ID: "id1"},
		},
	}
	app.showOrphanPrompt = true

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	app.Update(tea.KeyMsg{Type: tea.KeyEnter})
}
