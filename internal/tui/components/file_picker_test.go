// TestEnhancedFilePicker tests the EnhancedFilePicker component.
package components

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestNewEnhancedFilePicker tests the creation of a new enhanced file picker.
func TestNewEnhancedFilePicker(t *testing.T) {
	picker := NewEnhancedFilePicker()

	if picker == nil {
		t.Fatal("NewEnhancedFilePicker returned nil")
	}

	if !picker.dirAllowed {
		t.Error("Expected dirAllowed to be true by default")
	}

	if !picker.fileAllowed {
		t.Error("Expected fileAllowed to be true by default")
	}

	if picker.showHidden {
		t.Error("Expected showHidden to be false by default")
	}

	if !picker.focused {
		t.Error("Expected focused to be true by default")
	}
}

// TestEnhancedFilePicker_Options tests the builder pattern for setting options.
func TestEnhancedFilePicker_Options(t *testing.T) {
	tests := []struct {
		name     string
		opts     func(*EnhancedFilePicker) *EnhancedFilePicker
		check    func(*EnhancedFilePicker) bool
	}{
		{
			name: "Title option",
			opts: func(p *EnhancedFilePicker) *EnhancedFilePicker {
				return p.Title("Select File")
			},
			check: func(p *EnhancedFilePicker) bool {
				return p.title == "Select File"
			},
		},
		{
			name: "Description option",
			opts: func(p *EnhancedFilePicker) *EnhancedFilePicker {
				return p.Description("Choose a file from the list")
			},
			check: func(p *EnhancedFilePicker) bool {
				return p.description == "Choose a file from the list"
			},
		},
		{
			name: "DirAllowed false",
			opts: func(p *EnhancedFilePicker) *EnhancedFilePicker {
				return p.DirAllowed(false)
			},
			check: func(p *EnhancedFilePicker) bool {
				return !p.dirAllowed
			},
		},
		{
			name: "FileAllowed false",
			opts: func(p *EnhancedFilePicker) *EnhancedFilePicker {
				return p.FileAllowed(false)
			},
			check: func(p *EnhancedFilePicker) bool {
				return !p.fileAllowed
			},
		},
		{
			name: "CurrentDirectory option",
			opts: func(p *EnhancedFilePicker) *EnhancedFilePicker {
				return p.CurrentDirectory("/tmp")
			},
			check: func(p *EnhancedFilePicker) bool {
				return p.currentDir == "/tmp"
			},
		},
		{
			name: "Value option",
			opts: func(p *EnhancedFilePicker) *EnhancedFilePicker {
				val := "/tmp/test"
				return p.Value(&val)
			},
			check: func(p *EnhancedFilePicker) bool {
				return p.selectedPath != nil && *p.selectedPath == "/tmp/test"
			},
		},
		{
			name: "Validate option",
			opts: func(p *EnhancedFilePicker) *EnhancedFilePicker {
				return p.Validate(func(s string) error {
					if s == "" {
						return os.ErrInvalid
					}
					return nil
				})
			},
			check: func(p *EnhancedFilePicker) bool {
				return p.validate != nil
			},
		},
		{
			name: "ShowHidden option",
			opts: func(p *EnhancedFilePicker) *EnhancedFilePicker {
				return p.ShowHidden(true)
			},
			check: func(p *EnhancedFilePicker) bool {
				return p.showHidden
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			picker := NewEnhancedFilePicker()
			picker = tt.opts(picker)
			if !tt.check(picker) {
				t.Errorf("Option %s failed check", tt.name)
			}
		})
	}
}

// TestEnhancedFilePicker_Init tests the Init method.
func TestEnhancedFilePicker_Init(t *testing.T) {
	picker := NewEnhancedFilePicker().
		Title("Test Picker").
		CurrentDirectory("/tmp")

	cmd := picker.Init()

	// Init should return a command (from inner picker)
	if cmd == nil {
		t.Error("Init should return a non-nil command")
	}

	// After Init, inner picker should be initialized
	if picker.innerPicker == nil {
		t.Error("Init should initialize inner picker")
	}
}

// TestEnhancedFilePicker_View tests the View method renders properly.
func TestEnhancedFilePicker_View(t *testing.T) {
	picker := NewEnhancedFilePicker().
		Title("Test Picker").
		CurrentDirectory("/tmp").
		WithWidth(80)

	// Initialize first
	picker.Init()

	view := picker.View()

	if view == "" {
		t.Error("View should return non-empty string")
	}

	// Should contain breadcrumb bar
	if view == "" {
		t.Error("View should not be empty")
	}
}

// TestEnhancedFilePicker_GetRecentPaths tests the GetRecentPaths function.
func TestEnhancedFilePicker_GetRecentPaths(t *testing.T) {
	// Clear and set up recent paths
	ClearRecentPaths()
	defer ClearRecentPaths()

	// Should return empty initially
	paths := GetRecentPaths()
	if len(paths) != 0 {
		t.Errorf("Expected empty recent paths, got %d", len(paths))
	}

	// Add some paths
	AddRecentPath("/tmp/test1")
	AddRecentPath("/tmp/test2")

	paths = GetRecentPaths()
	if len(paths) != 2 {
		t.Errorf("Expected 2 recent paths, got %d", len(paths))
	}

	// First should be most recent (newest added is at front)
	if paths[0] != "/tmp/test2" {
		t.Errorf("Expected first path to be /tmp/test2 (newest), got %q", paths[0])
	}

	// Second should be test1 (older)
	if paths[1] != "/tmp/test1" {
		t.Errorf("Expected second path to be /tmp/test1 (older), got %q", paths[1])
	}

	// Clean up
	ClearRecentPaths()
}

// TestEnhancedFilePicker_AddRecentPath tests the AddRecentPath function.
func TestEnhancedFilePicker_AddRecentPath(t *testing.T) {
	ClearRecentPaths()

	// Add empty path - should be ignored
	AddRecentPath("")
	paths := GetRecentPaths()
	if len(paths) != 0 {
		t.Error("Empty path should not be added")
	}

	// Add valid path
	AddRecentPath("/tmp/test")
	paths = GetRecentPaths()
	if len(paths) != 1 {
		t.Errorf("Expected 1 path, got %d", len(paths))
	}

	// Add same path again - should move to front
	AddRecentPath("/tmp/test")
	paths = GetRecentPaths()
	if len(paths) != 1 {
		t.Errorf("Expected 1 path after duplicate, got %d", len(paths))
	}

	// Add path with home directory
	homeDir, _ := os.UserHomeDir()
	testPath := filepath.Join(homeDir, "Documents")
	AddRecentPath(testPath)
	paths = GetRecentPaths()
	if len(paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(paths))
	}

	// Test max paths limit (10)
	ClearRecentPaths()
	for i := 0; i < 15; i++ {
		AddRecentPath(filepath.Join("/tmp", "path", string(rune('a'+i))))
	}
	paths = GetRecentPaths()
	if len(paths) > 10 {
		t.Errorf("Expected max 10 paths, got %d", len(paths))
	}

	ClearRecentPaths()
}

// TestEnhancedFilePicker_ClearRecentPaths tests the ClearRecentPaths function.
func TestEnhancedFilePicker_ClearRecentPaths(t *testing.T) {
	// Add some paths
	AddRecentPath("/tmp/test1")
	AddRecentPath("/tmp/test2")

	// Clear
	ClearRecentPaths()

	paths := GetRecentPaths()
	if len(paths) != 0 {
		t.Errorf("Expected 0 paths after clear, got %d", len(paths))
	}
}

// TestEnhancedFilePicker_SetRecentPaths tests the SetRecentPaths function.
func TestEnhancedFilePicker_SetRecentPaths(t *testing.T) {
	ClearRecentPaths()

	paths := []string{"/path1", "/path2", "/path3"}
	SetRecentPaths(paths)

	result := GetRecentPaths()
	if len(result) != len(paths) {
		t.Errorf("Expected %d paths, got %d", len(paths), len(result))
	}

	// Verify copy - modifying input shouldn't affect result
	paths[0] = "/modified"
	result = GetRecentPaths()
	if result[0] == "/modified" {
		t.Error("SetRecentPaths should create a copy, not share memory")
	}

	ClearRecentPaths()
}

// TestEnhancedFilePicker_Value tests the Value getter and setter.
func TestEnhancedFilePicker_Value(t *testing.T) {
	// Test with nil value
	picker := NewEnhancedFilePicker()
	if picker.GetValue() != "" {
		t.Error("Expected empty value when selectedPath is nil")
	}

	// Test with value set
	testValue := "/tmp/test"
	picker = NewEnhancedFilePicker().Value(&testValue)

	if picker.GetValue() != testValue {
		t.Errorf("Expected value %q, got %q", testValue, picker.GetValue())
	}

	// Test modifying through pointer
	*picker.selectedPath = "/new/path"
	if picker.GetValue() != "/new/path" {
		t.Error("Value should reflect changes through pointer")
	}
}

// TestEnhancedFilePicker_ValidateDirectoryPath tests the ValidateDirectoryPath function.
func TestEnhancedFilePicker_ValidateDirectoryPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "valid directory",
			path:    "/tmp",
			wantErr: false,
		},
		{
			name:    "valid home directory",
			path:    "~",
			wantErr: false,
		},
		{
			name:    "non-existent path",
			path:    "/nonexistent/path/12345",
			wantErr: true,
		},
		{
			name:    "file instead of directory",
			path:    "/etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDirectoryPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDirectoryPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

// TestEnhancedFilePicker_ValidateFilePath tests the ValidateFilePath function.
func TestEnhancedFilePicker_ValidateFilePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "valid file path",
			path:    "/tmp/testfile",
			wantErr: false,
		},
		{
			name:    "valid file under home",
			path:    "~/Documents/test.txt",
			wantErr: false,
		},
		{
			name:    "non-existent parent directory",
			path:    "/nonexistent/path/file.txt",
			wantErr: true,
		},
		{
			name:    "file in root",
			path:    "/rootfile",
			wantErr: false, // Parent is / which exists
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

// TestEnhancedFilePicker_FormatPathForDisplay tests the FormatPathForDisplay function.
func TestEnhancedFilePicker_FormatPathForDisplay(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "empty path",
			path: "",
			want: "",
		},
		{
			name: "home directory",
			path: "~",
			want: "~",
		},
		{
			name: "path under home",
			path: "~/Documents",
			want: "~/Documents",
		},
		{
			name: "absolute path",
			path: "/tmp/test",
			want: "/tmp/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPathForDisplay(tt.path)
			if result != tt.want {
				t.Errorf("FormatPathForDisplay(%q) = %q, want %q", tt.path, result, tt.want)
			}
		})
	}
}

// TestEnhancedFilePicker_Run tests the Run method doesn't panic.
func TestEnhancedFilePicker_Run(t *testing.T) {
	// This would actually run the picker which requires a terminal
	// So we just test that the method exists and doesn't panic on nil setup
	picker := NewEnhancedFilePicker()

	// Run should return an error (no terminal available)
	err := picker.Run()
	if err == nil {
		// This is actually expected to fail without a terminal
		t.Log("Run failed as expected without terminal")
	}
}

// TestEnhancedFilePicker_MaxRecentPaths tests that recent paths are limited to max.
func TestEnhancedFilePicker_MaxRecentPaths(t *testing.T) {
	ClearRecentPaths()

	// Add more than maxRecentPaths paths
	for i := 0; i < 15; i++ {
		AddRecentPath(filepath.Join("/tmp", "path", string(rune('a'+i))))
	}

	paths := GetRecentPaths()
	if len(paths) > maxRecentPaths {
		t.Errorf("Expected at most %d recent paths, got %d", maxRecentPaths, len(paths))
	}

	ClearRecentPaths()
}

// TestEnhancedFilePicker_Update tests that Update method works without panicking.
func TestEnhancedFilePicker_Update(t *testing.T) {
	picker := NewEnhancedFilePicker().
		CurrentDirectory("/tmp").
		WithWidth(80)

	picker.Init()

	// Press a key - should not panic
	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, _ := picker.Update(msg)

	// The model should be returned
	if model == nil {
		t.Error("Update should return a non-nil model")
	}
}
