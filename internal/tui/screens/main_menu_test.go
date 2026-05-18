package screens

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewMainMenuScreen(t *testing.T) {
	screen := NewMainMenuScreen()

	if screen == nil {
		t.Fatal("NewMainMenuScreen() returned nil")
	}

	// Verify menu is initialized
	if screen.menu == nil {
		t.Fatal("menu should be initialized")
	}

	// Verify menu items count
	if len(screen.menu.Items) != 5 {
		t.Errorf("menu items count = %d, want 5", len(screen.menu.Items))
	}
}

func TestMainMenuScreen_MenuItems(t *testing.T) {
	screen := NewMainMenuScreen()

	// Define expected menu items
	expectedItems := []struct {
		label string
		key   string
	}{
		{"Mount Management", "M"},
		{"Sync Job Management", "S"},
		{"Service Status", "V"},
		{"Settings", "T"},
		{"Quit", "Q"},
	}

	for i, expected := range expectedItems {
		if i >= len(screen.menu.Items) {
			t.Errorf("missing menu item at index %d", i)
			continue
		}

		if screen.menu.Items[i].Label != expected.label {
			t.Errorf("menu item %d label = %q, want %q", i, screen.menu.Items[i].Label, expected.label)
		}

		if screen.menu.Items[i].Key != expected.key {
			t.Errorf("menu item %d key = %q, want %q", i, screen.menu.Items[i].Key, expected.key)
		}
	}
}

func TestMainMenuScreen_NavigationUp(t *testing.T) {
	screen := NewMainMenuScreen()
	screen.SetSize(80, 24)

	// Start at first item (index 0)
	if screen.menu.Cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", screen.menu.Cursor)
	}

	// Press up - should stay at 0 (can't go above first item)
	screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	if screen.menu.Cursor != 0 {
		t.Errorf("cursor after up at top = %d, want 0", screen.menu.Cursor)
	}

	// Move down first
	screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	if screen.menu.Cursor != 1 {
		t.Fatalf("cursor after down = %d, want 1", screen.menu.Cursor)
	}

	// Now move up
	screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	if screen.menu.Cursor != 0 {
		t.Errorf("cursor after up = %d, want 0", screen.menu.Cursor)
	}
}

func TestMainMenuScreen_NavigationDown(t *testing.T) {
	screen := NewMainMenuScreen()
	screen.SetSize(80, 24)

	// Start at first item (index 0)
	if screen.menu.Cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", screen.menu.Cursor)
	}

	// Move down through all items
	for i := range len(screen.menu.Items) - 1 {
		screen.Update(tea.KeyMsg{Type: tea.KeyDown})
		expected := i + 1
		if screen.menu.Cursor != expected {
			t.Errorf("cursor after down %d times = %d, want %d", i+1, screen.menu.Cursor, expected)
		}
	}

	// Try to move down past last item - should stay at last
	lastIndex := len(screen.menu.Items) - 1
	screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	if screen.menu.Cursor != lastIndex {
		t.Errorf("cursor after down at bottom = %d, want %d", screen.menu.Cursor, lastIndex)
	}
}

func TestMainMenuScreen_NavigationKeys(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		expectedIndex int
	}{
		{"k key (vim up)", "k", 0},
		{"j key (vim down)", "j", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewMainMenuScreen()
			screen.SetSize(80, 24)

			// Reset cursor to 0
			screen.menu.Cursor = 0

			// Send key
			screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})

			if screen.menu.Cursor != tt.expectedIndex {
				t.Errorf("cursor after %s = %d, want %d", tt.key, screen.menu.Cursor, tt.expectedIndex)
			}
		})
	}
}

func TestMainMenuScreen_EnterKeyNavigation(t *testing.T) {
	tests := []struct {
		name           string
		startIndex     int
		expectedTarget string
		isQuit         bool
	}{
		{"Mount Management", 0, "mounts", false},
		{"Sync Job Management", 1, "sync_jobs", false},
		{"Service Status", 2, "services", false},
		{"Settings", 3, "settings", false},
		{"Quit", 4, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewMainMenuScreen()
			screen.SetSize(80, 24)
			screen.menu.Cursor = tt.startIndex

			// Press enter and capture the command
			_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

			if cmd == nil {
				t.Fatal("Update() returned nil command, want navigation command")
			}

			// Execute the command to get the message
			msg := cmd()
			if msg == nil {
				t.Fatal("command returned nil message")
			}

			// Quit uses tea.Quit which returns QuitMsg
			if tt.isQuit {
				if _, ok := msg.(tea.QuitMsg); !ok {
					t.Fatalf("message type = %T, want tea.QuitMsg", msg)
				}
				return
			}

			// Check the message type and target
			navigateMsg, ok := msg.(NavigateToMsg)
			if !ok {
				t.Fatalf("message type = %T, want NavigateToMsg", msg)
			}

			if navigateMsg.Target != tt.expectedTarget {
				t.Errorf("NavigateToMsg.Target = %q, want %q", navigateMsg.Target, tt.expectedTarget)
			}
		})
	}
}

func TestMainMenuScreen_QuickJumpKeys(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		expectedTarget string
	}{
		{"m key -> mounts", "m", "mounts"},
		{"s key -> sync_jobs", "s", "sync_jobs"},
		{"v key -> services", "v", "services"},
		{"t key -> settings", "t", "settings"},
		{"q key -> quit", "q", ""}, // Quit returns tea.Quit, not NavigateToMsg
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewMainMenuScreen()
			screen.SetSize(80, 24)

			// Send quick-jump key
			_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})

			if cmd == nil {
				t.Fatal("Update() returned nil command, want navigation command")
			}

			msg := cmd()
			if msg == nil {
				t.Fatal("command returned nil message")
			}

			if tt.key == "q" {
				// Quit should return tea.Quit which produces QuitMsg
				if _, ok := msg.(tea.QuitMsg); !ok {
					t.Errorf("message type = %T, want tea.QuitMsg", msg)
				}
				return
			}

			navigateMsg, ok := msg.(NavigateToMsg)
			if !ok {
				t.Fatalf("message type = %T, want NavigateToMsg", msg)
			}

			if navigateMsg.Target != tt.expectedTarget {
				t.Errorf("NavigateToMsg.Target = %q, want %q", navigateMsg.Target, tt.expectedTarget)
			}
		})
	}
}

func TestMainMenuScreen_SetSize(t *testing.T) {
	screen := NewMainMenuScreen()

	// Set size
	screen.SetSize(100, 30)

	if screen.width != 100 {
		t.Errorf("width = %d, want 100", screen.width)
	}

	if screen.height != 30 {
		t.Errorf("height = %d, want 30", screen.height)
	}

	// Menu width should be set (width - 8 for padding)
	if screen.menu.Width != 92 {
		t.Errorf("menu width = %d, want 92", screen.menu.Width)
	}
}

func TestMainMenuScreen_Init(t *testing.T) {
	screen := NewMainMenuScreen()

	cmd := screen.Init()

	if cmd != nil {
		t.Error("Init() should return nil command")
	}
}

func TestMainMenuScreen_View(t *testing.T) {
	screen := NewMainMenuScreen()
	screen.SetSize(80, 24)

	view := screen.View()

	// Check title is rendered
	if !strings.Contains(view, "Main Menu") {
		t.Error("View() should contain 'Main Menu' title")
	}

	// Check all menu items are rendered
	expectedLabels := []string{
		"Mount Management",
		"Sync Job Management",
		"Service Status",
		"Settings",
		"Quit",
	}

	for _, label := range expectedLabels {
		if !strings.Contains(view, label) {
			t.Errorf("View() should contain menu item '%s'", label)
		}
	}

	// Check help text is present
	if !strings.Contains(view, "navigate") {
		t.Error("View() should contain help text for navigation")
	}

	if !strings.Contains(view, "select") {
		t.Error("View() should contain help text for select")
	}

	// Check selection marker is present
	if !strings.Contains(view, "▸") {
		t.Error("View() should contain selection marker '▸'")
	}
}

func TestMainMenuScreen_ViewContainsDescriptions(t *testing.T) {
	screen := NewMainMenuScreen()
	screen.SetSize(80, 24)

	view := screen.View()

	// Check descriptions are rendered
	expectedDescriptions := []string{
		"Configure and manage rclone mount points",
		"Configure and schedule rclone sync operations",
		"View and control systemd services",
		"Application configuration",
		"Exit the application",
	}

	for _, desc := range expectedDescriptions {
		if !strings.Contains(view, desc) {
			t.Errorf("View() should contain description '%s'", desc)
		}
	}
}

func TestMainMenuScreen_SpaceKeySelects(t *testing.T) {
	screen := NewMainMenuScreen()
	screen.SetSize(80, 24)
	screen.menu.Cursor = 0

	// Press space to select
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeySpace})

	if cmd == nil {
		t.Fatal("Update() returned nil command")
	}

	msg := cmd()
	navigateMsg, ok := msg.(NavigateToMsg)
	if !ok {
		t.Fatalf("message type = %T, want NavigateToMsg", msg)
	}

	if navigateMsg.Target != "mounts" {
		t.Errorf("NavigateToMsg.Target = %q, want 'mounts'", navigateMsg.Target)
	}
}

func TestMainMenuScreen_ViewWithSmallWidth(t *testing.T) {
	screen := NewMainMenuScreen()
	screen.SetSize(40, 24) // Small width

	view := screen.View()

	// Should still render title
	if !strings.Contains(view, "Main Menu") {
		t.Error("View() should contain 'Main Menu' even with small width")
	}
}

func TestMainMenuScreen_ViewWithLargeWidth(t *testing.T) {
	screen := NewMainMenuScreen()
	screen.SetSize(200, 50) // Large width

	view := screen.View()

	// Should still render title
	if !strings.Contains(view, "Main Menu") {
		t.Error("View() should contain 'Main Menu' even with large width")
	}
}

func TestMainMenuScreen_UnknownKey(t *testing.T) {
	screen := NewMainMenuScreen()
	screen.SetSize(80, 24)

	// Press an unknown key - should not navigate
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	// No command should be returned for unknown keys
	if cmd != nil {
		t.Error("Update() should return nil command for unknown keys")
	}
}

func TestMainMenuScreen_MultipleNavigations(t *testing.T) {
	screen := NewMainMenuScreen()
	screen.SetSize(80, 24)

	// Navigate to mounts
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	msg := cmd()
	navigateMsg := msg.(NavigateToMsg)
	if navigateMsg.Target != "mounts" {
		t.Errorf("target = %q, want 'mounts'", navigateMsg.Target)
	}

	// Navigate to sync_jobs
	_, cmd = screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	msg = cmd()
	navigateMsg = msg.(NavigateToMsg)
	if navigateMsg.Target != "sync_jobs" {
		t.Errorf("target = %q, want 'sync_jobs'", navigateMsg.Target)
	}

	// Navigate to services
	_, cmd = screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	msg = cmd()
	navigateMsg = msg.(NavigateToMsg)
	if navigateMsg.Target != "services" {
		t.Errorf("target = %q, want 'services'", navigateMsg.Target)
	}

	// Navigate to settings
	_, cmd = screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	msg = cmd()
	navigateMsg = msg.(NavigateToMsg)
	if navigateMsg.Target != "settings" {
		t.Errorf("target = %q, want 'settings'", navigateMsg.Target)
	}
}