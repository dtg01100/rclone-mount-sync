package components

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewMenu(t *testing.T) {
	items := []MenuItem{
		{Label: "Item 1", Description: "First item", Key: "1"},
		{Label: "Item 2", Description: "Second item", Key: "2"},
		{Label: "Item 3", Description: "", Key: "3"},
	}

	menu := NewMenu(items)

	if menu == nil {
		t.Fatal("NewMenu returned nil")
	}
	if len(menu.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(menu.Items))
	}
	if menu.Cursor != 0 {
		t.Errorf("Expected cursor at 0, got %d", menu.Cursor)
	}
	if !menu.ShowKeys {
		t.Error("Expected ShowKeys to be true by default")
	}
}

func TestMenu_SetWidth(t *testing.T) {
	menu := NewMenu([]MenuItem{{Label: "Test"}})
	menu.SetWidth(100)
	if menu.Width != 100 {
		t.Errorf("Expected width 100, got %d", menu.Width)
	}
}

func TestMenu_Up(t *testing.T) {
	tests := []struct {
		name     string
		items    []MenuItem
		initial  int
		expected int
	}{
		{
			name:     "move up from middle",
			items:    []MenuItem{{Label: "1"}, {Label: "2"}, {Label: "3"}},
			initial:  2,
			expected: 1,
		},
		{
			name:     "move up from first item",
			items:    []MenuItem{{Label: "1"}, {Label: "2"}},
			initial:  0,
			expected: 0,
		},
		{
			name:     "move up in single item menu",
			items:    []MenuItem{{Label: "1"}},
			initial:  0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			menu := NewMenu(tt.items)
			menu.Cursor = tt.initial
			menu.Up()
			if menu.Cursor != tt.expected {
				t.Errorf("Expected cursor at %d, got %d", tt.expected, menu.Cursor)
			}
		})
	}
}

func TestMenu_Down(t *testing.T) {
	tests := []struct {
		name     string
		items    []MenuItem
		initial  int
		expected int
	}{
		{
			name:     "move down from first item",
			items:    []MenuItem{{Label: "1"}, {Label: "2"}, {Label: "3"}},
			initial:  0,
			expected: 1,
		},
		{
			name:     "move down from last item",
			items:    []MenuItem{{Label: "1"}, {Label: "2"}},
			initial:  1,
			expected: 1,
		},
		{
			name:     "move down in single item menu",
			items:    []MenuItem{{Label: "1"}},
			initial:  0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			menu := NewMenu(tt.items)
			menu.Cursor = tt.initial
			menu.Down()
			if menu.Cursor != tt.expected {
				t.Errorf("Expected cursor at %d, got %d", tt.expected, menu.Cursor)
			}
		})
	}
}

func TestMenu_Selected(t *testing.T) {
	tests := []struct {
		name      string
		items     []MenuItem
		cursor    int
		wantLabel string
		wantDesc  string
		wantKey   string
		wantEmpty bool
	}{
		{
			name:      "select first item",
			items:     []MenuItem{{Label: "A", Description: "Desc A", Key: "a"}, {Label: "B", Description: "Desc B", Key: "b"}},
			cursor:    0,
			wantLabel: "A",
			wantDesc:  "Desc A",
			wantKey:   "a",
		},
		{
			name:      "select second item",
			items:     []MenuItem{{Label: "A"}, {Label: "B"}},
			cursor:    1,
			wantLabel: "B",
		},
		{
			name:      "empty menu",
			items:     []MenuItem{},
			cursor:    0,
			wantEmpty: true,
		},
		{
			name:      "invalid cursor negative",
			items:     []MenuItem{{Label: "A"}},
			cursor:    -1,
			wantEmpty: true,
		},
		{
			name:      "invalid cursor too high",
			items:     []MenuItem{{Label: "A"}},
			cursor:    10,
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			menu := NewMenu(tt.items)
			menu.Cursor = tt.cursor
			selected := menu.Selected()

			if tt.wantEmpty {
				if selected.Label != "" || selected.Description != "" || selected.Key != "" {
					t.Errorf("Expected empty MenuItem, got %+v", selected)
				}
				return
			}

			if selected.Label != tt.wantLabel {
				t.Errorf("Expected label %q, got %q", tt.wantLabel, selected.Label)
			}
			if selected.Description != tt.wantDesc {
				t.Errorf("Expected description %q, got %q", tt.wantDesc, selected.Description)
			}
			if selected.Key != tt.wantKey {
				t.Errorf("Expected key %q, got %q", tt.wantKey, selected.Key)
			}
		})
	}
}

func TestMenu_Render(t *testing.T) {
	tests := []struct {
		name     string
		items    []MenuItem
		cursor   int
		showKeys bool
		wantLen  int
	}{
		{
			name:     "render with keys",
			items:    []MenuItem{{Label: "Item 1", Key: "1"}},
			cursor:   0,
			showKeys: true,
			wantLen:  1,
		},
		{
			name:     "render without keys",
			items:    []MenuItem{{Label: "Item 1"}},
			cursor:   0,
			showKeys: false,
			wantLen:  1,
		},
		{
			name: "render multiple items with descriptions",
			items: []MenuItem{
				{Label: "Item 1", Description: "Desc 1", Key: "1"},
				{Label: "Item 2", Description: "Desc 2", Key: "2"},
			},
			cursor:   1,
			showKeys: true,
			wantLen:  4,
		},
		{
			name:     "render empty menu",
			items:    []MenuItem{},
			cursor:   0,
			showKeys: true,
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			menu := NewMenu(tt.items)
			menu.Cursor = tt.cursor
			menu.ShowKeys = tt.showKeys
			rendered := menu.Render()

			lines := strings.Count(rendered, "\n")
			if lines != tt.wantLen {
				t.Errorf("Expected %d lines, got %d", tt.wantLen, lines)
			}

			if len(tt.items) > 0 {
				for _, item := range tt.items {
					if !strings.Contains(rendered, item.Label) {
						t.Errorf("Expected rendered output to contain %q", item.Label)
					}
				}
			}
		})
	}
}

func TestNewButton(t *testing.T) {
	button := NewButton("Click Me")
	if button == nil {
		t.Fatal("NewButton returned nil")
	}
	if button.Label != "Click Me" {
		t.Errorf("Expected label %q, got %q", "Click Me", button.Label)
	}
	if button.Focus {
		t.Error("Expected Focus to be false by default")
	}
}

func TestButton_Render(t *testing.T) {
	tests := []struct {
		name  string
		label string
		focus bool
	}{
		{name: "unfocused button", label: "Click Me", focus: false},
		{name: "focused button", label: "Submit", focus: true},
		{name: "empty label", label: "", focus: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			button := &Button{Label: tt.label, Focus: tt.focus}
			rendered := button.Render()

			if rendered == "" && tt.label != "" {
				t.Error("Expected non-empty render output")
			}
		})
	}
}

func TestHelpBar(t *testing.T) {
	tests := []struct {
		name  string
		width int
		items []HelpItem
	}{
		{
			name:  "normal width",
			width: 80,
			items: []HelpItem{
				{Key: "↑↓", Desc: "Navigate"},
				{Key: "Enter", Desc: "Select"},
				{Key: "q", Desc: "Quit"},
			},
		},
		{
			name:  "narrow width",
			width: 20,
			items: []HelpItem{
				{Key: "↑↓", Desc: "Navigate"},
				{Key: "Enter", Desc: "Select"},
				{Key: "q", Desc: "Quit"},
			},
		},
		{
			name:  "very narrow width",
			width: 10,
			items: []HelpItem{
				{Key: "↑↓", Desc: "Navigate"},
				{Key: "Enter", Desc: "Select"},
			},
		},
		{
			name:  "empty items",
			width: 80,
			items: []HelpItem{},
		},
		{
			name:  "single item",
			width: 80,
			items: []HelpItem{{Key: "q", Desc: "Quit"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := HelpBar(tt.width, tt.items)
			if rendered == "" {
				t.Error("HelpBar returned empty string")
			}
		})
	}
}

func TestTitleBar(t *testing.T) {
	tests := []struct {
		name    string
		width   int
		title   string
		version string
	}{
		{name: "normal width", width: 80, title: "App", version: "1.0.0"},
		{name: "narrow width", width: 30, title: "MyApp", version: "1.0.0"},
		{name: "very narrow width", width: 10, title: "App", version: "1.0.0"},
		{name: "empty title", width: 80, title: "", version: "1.0.0"},
		{name: "long title", width: 80, title: "Very Long Application Title Here", version: "2.0.0-beta"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := TitleBar(tt.width, tt.title, tt.version)
			if rendered == "" {
				t.Error("TitleBar returned empty string")
			}
		})
	}
}

func TestStatusBar(t *testing.T) {
	tests := []struct {
		name  string
		width int
		text  string
	}{
		{name: "normal", width: 80, text: "Ready"},
		{name: "narrow", width: 10, text: "Processing..."},
		{name: "empty text", width: 80, text: ""},
		{name: "long text", width: 40, text: "This is a very long status message that might be truncated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := StatusBar(tt.width, tt.text)
			if rendered == "" {
				t.Error("StatusBar returned empty string")
			}
		})
	}
}

func TestBox(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		content string
		width   int
	}{
		{name: "with title", title: "Settings", content: "Content here", width: 40},
		{name: "without title", title: "", content: "Just content", width: 40},
		{name: "empty content", title: "Empty", content: "", width: 40},
		{name: "narrow width", title: "Box", content: "X", width: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := Box(tt.title, tt.content, tt.width)
			if rendered == "" {
				t.Error("Box returned empty string")
			}
		})
	}
}

func TestCenter(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
	}{
		{name: "short text", text: "Hi", width: 20},
		{name: "exact fit", text: "Hello", width: 5},
		{name: "text wider than width", text: "This is very long", width: 5},
		{name: "empty text", text: "", width: 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := Center(tt.text, tt.width)
			if rendered == "" {
				t.Error("Center returned empty string")
			}
		})
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		padding int
	}{
		{name: "normal padding", text: "Test", padding: 4},
		{name: "zero padding", text: "Test", padding: 0},
		{name: "large padding", text: "Test", padding: 20},
		{name: "empty text", text: "", padding: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := PadLeft(tt.text, tt.padding)
			if rendered == "" {
				t.Error("PadLeft returned empty string")
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		padding int
	}{
		{name: "normal padding", text: "Test", padding: 4},
		{name: "zero padding", text: "Test", padding: 0},
		{name: "large padding", text: "Test", padding: 20},
		{name: "empty text", text: "", padding: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := PadRight(tt.text, tt.padding)
			if rendered == "" {
				t.Error("PadRight returned empty string")
			}
		})
	}
}

func TestStatusIndicator(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		contains string
	}{
		{name: "active", status: "active", contains: "●"},
		{name: "running", status: "running", contains: "●"},
		{name: "mounted", status: "mounted", contains: "●"},
		{name: "inactive", status: "inactive", contains: "○"},
		{name: "stopped", status: "stopped", contains: "○"},
		{name: "unmounted", status: "unmounted", contains: "○"},
		{name: "failed", status: "failed", contains: "✗"},
		{name: "error", status: "error", contains: "✗"},
		{name: "unknown status", status: "unknown", contains: "○"},
		{name: "empty status", status: "", contains: "○"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StatusIndicator(tt.status)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("StatusIndicator(%q) = %q, expected to contain %q", tt.status, result, tt.contains)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxLen   int
		expected string
	}{
		{name: "shorter text", text: "Hi", maxLen: 10, expected: "Hi"},
		{name: "exact length", text: "Hello", maxLen: 5, expected: "Hello"},
		{name: "longer text", text: "Hello World", maxLen: 8, expected: "Hello..."},
		{name: "maxLen 3", text: "Hello", maxLen: 3, expected: "Hel"},
		{name: "maxLen 2", text: "Hello", maxLen: 2, expected: "He"},
		{name: "maxLen 1", text: "Hello", maxLen: 1, expected: "H"},
		{name: "maxLen 0", text: "Hello", maxLen: 0, expected: ""},
		{name: "empty text", text: "", maxLen: 5, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.text, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Truncate(%q, %d) = %q, expected %q", tt.text, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestRenderTitle(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "normal text", text: "Settings"},
		{name: "empty text", text: ""},
		{name: "special chars", text: "Hello! @#$%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := RenderTitle(tt.text)
			if rendered == "" && tt.text != "" {
				t.Error("RenderTitle returned empty string for non-empty input")
			}
		})
	}
}

func TestRenderError(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "normal error", text: "Something went wrong"},
		{name: "empty error", text: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := RenderError(tt.text)
			if !strings.Contains(rendered, "✗") {
				t.Error("RenderError should contain ✗ symbol")
			}
		})
	}
}

func TestRenderSuccess(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "normal success", text: "Operation completed"},
		{name: "empty success", text: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := RenderSuccess(tt.text)
			if !strings.Contains(rendered, "✓") {
				t.Error("RenderSuccess should contain ✓ symbol")
			}
		})
	}
}

func TestRenderWarning(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "normal warning", text: "Low disk space"},
		{name: "empty warning", text: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := RenderWarning(tt.text)
			if !strings.Contains(rendered, "⚠") {
				t.Error("RenderWarning should contain ⚠ symbol")
			}
		})
	}
}

func TestRenderInfo(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "normal info", text: "Press any key"},
		{name: "empty info", text: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := RenderInfo(tt.text)
			if !strings.Contains(rendered, "ℹ") {
				t.Error("RenderInfo should contain ℹ symbol")
			}
		})
	}
}

func TestGetCommonDirectories(t *testing.T) {
	dirs := GetCommonDirectories()

	if len(dirs) == 0 {
		t.Error("GetCommonDirectories returned empty slice")
	}

	for _, dir := range dirs {
		if dir == "" {
			t.Error("GetCommonDirectories returned empty string")
		}
	}

	hasMnt := false
	hasMedia := false
	for _, dir := range dirs {
		if dir == "/mnt/" {
			hasMnt = true
		}
		if dir == "/media/" {
			hasMedia = true
		}
	}
	if !hasMnt {
		t.Error("GetCommonDirectories should include /mnt/")
	}
	if !hasMedia {
		t.Error("GetCommonDirectories should include /media/")
	}
}

func TestGetPathSuggestions(t *testing.T) {
	tests := []struct {
		name          string
		recentPaths   []string
		existingPaths []string
		wantMinLen    int
		checkOrder    bool
	}{
		{
			name:          "empty inputs",
			recentPaths:   []string{},
			existingPaths: []string{},
			wantMinLen:    1,
		},
		{
			name:          "with recent paths",
			recentPaths:   []string{"/home/user/path1", "/home/user/path2"},
			existingPaths: []string{},
			wantMinLen:    2,
		},
		{
			name:          "with existing paths",
			recentPaths:   []string{},
			existingPaths: []string{"/mnt/data", "/media/usb"},
			wantMinLen:    2,
		},
		{
			name:          "with both recent and existing",
			recentPaths:   []string{"/home/user/recent"},
			existingPaths: []string{"/mnt/data"},
			wantMinLen:    2,
		},
		{
			name:          "deduplication",
			recentPaths:   []string{"/same/path", "/same/path"},
			existingPaths: []string{"/same/path"},
			wantMinLen:    1,
		},
		{
			name:          "order: recent first",
			recentPaths:   []string{"/recent/path"},
			existingPaths: []string{"/existing/path"},
			checkOrder:    true,
			wantMinLen:    2,
		},
		{
			name:          "max 5 recent paths",
			recentPaths:   []string{"/p1", "/p2", "/p3", "/p4", "/p5", "/p6", "/p7"},
			existingPaths: []string{},
			wantMinLen:    5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := GetPathSuggestions(tt.recentPaths, tt.existingPaths)

			if len(suggestions) < tt.wantMinLen {
				t.Errorf("Expected at least %d suggestions, got %d", tt.wantMinLen, len(suggestions))
			}

			if tt.checkOrder && len(suggestions) >= 2 {
				if suggestions[0] != tt.recentPaths[0] {
					t.Errorf("Expected first suggestion to be %q, got %q", tt.recentPaths[0], suggestions[0])
				}
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get user home directory")
	}

	tests := []struct {
		name     string
		path     string
		want     string
		skipUser bool
	}{
		{
			name: "empty string",
			path: "",
			want: "",
		},
		{
			name: "tilde alone",
			path: "~",
			want: homeDir,
		},
		{
			name: "tilde with path",
			path: "~/Documents",
			want: filepath.Join(homeDir, "Documents"),
		},
		{
			name: "tilde with subpath",
			path: "~/a/b/c",
			want: filepath.Join(homeDir, "a", "b", "c"),
		},
		{
			name: "absolute path",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "relative path",
			path: "relative/path",
			want: "relative/path",
		},
		{
			name:     "user tilde (~root)",
			path:     "~root",
			skipUser: true,
		},
		{
			name:     "user tilde with path (~root/tmp)",
			path:     "~root/tmp",
			skipUser: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandHome(tt.path)

			if tt.skipUser {
				if strings.HasPrefix(tt.path, "~root") {
					u, err := user.Lookup("root")
					if err == nil {
						expected := filepath.Join(u.HomeDir, strings.TrimPrefix(tt.path, "~root"))
						if result != expected {
							t.Errorf("ExpandHome(%q) = %q, expected %q", tt.path, result, expected)
						}
					}
				}
				return
			}

			if result != tt.want {
				t.Errorf("ExpandHome(%q) = %q, expected %q", tt.path, result, tt.want)
			}
		})
	}
}

func TestExpandHome_WithNonexistentUser(t *testing.T) {
	path := "~nonexistentuser12345/some/path"
	result := ExpandHome(path)

	if result != path {
		t.Errorf("ExpandHome(%q) = %q, expected original path unchanged", path, result)
	}
}

func TestContractHome(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get user home directory")
	}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "empty string",
			path:     "",
			expected: "",
		},
		{
			name:     "home directory exactly",
			path:     homeDir,
			expected: "~",
		},
		{
			name:     "path under home",
			path:     filepath.Join(homeDir, "Documents"),
			expected: "~/Documents",
		},
		{
			name:     "nested path under home",
			path:     filepath.Join(homeDir, "a", "b", "c"),
			expected: "~/a/b/c",
		},
		{
			name:     "path outside home",
			path:     "/usr/local/bin",
			expected: "/usr/local/bin",
		},
		{
			name:     "path that starts with home but isn't under it",
			path:     homeDir + "extra",
			expected: homeDir + "extra",
		},
		{
			name:     "relative path",
			path:     "relative/path",
			expected: "relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContractHome(tt.path)
			if result != tt.expected {
				t.Errorf("ContractHome(%q) = %q, expected %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestExpandContractRoundTrip(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get user home directory")
	}

	tests := []struct {
		name string
		path string
	}{
		{name: "home directory", path: "~"},
		{name: "subdirectory", path: "~/Documents"},
		{name: "nested path", path: "~/a/b/c/d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expanded := ExpandHome(tt.path)
			contracted := ContractHome(expanded)

			if contracted != tt.path {
				t.Errorf("Round trip failed: %q -> %q -> %q", tt.path, expanded, contracted)
			}

			if expanded == tt.path {
				t.Errorf("ExpandHome should have expanded %q", tt.path)
			}

			if !filepath.IsAbs(expanded) {
				t.Errorf("ExpandHome(%q) = %q should be absolute", tt.path, expanded)
			}

			if !strings.HasPrefix(expanded, homeDir) {
				t.Errorf("ExpandHome(%q) = %q should start with home dir %q", tt.path, expanded, homeDir)
			}
		})
	}
}
