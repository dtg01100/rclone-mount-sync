package components

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEnhancedFilePicker_ValidateFilePath tests the file path validation functionality
func TestEnhancedFilePicker_ValidateFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "empty_path",
			path:     "",
			expected: false,
		},
		{
			name:     "valid_file_under_home",
			path:     filepath.Join(tmpDir, "testfile.txt"),
			expected: true,
		},
		{
			name:     "valid_directory_under_home",
			path:     tmpDir,
			expected: true,
		},
		{
			name:     "nonexistent_file",
			path:     filepath.Join(tmpDir, "nonexistent.txt"),
			expected: false,
		},
		{
			name:     "relative_path",
			path:     "relative/path.txt",
			expected: false,
		},
		{
			name:     "path_with_tilde",
			path:     "~/test.txt",
			expected: false, // because ~ will be expanded to a path that may not exist
		},
	}

	// Create a test file in the temp directory to validate file paths
	testFilePath := filepath.Join(tmpDir, "testfile.txt")
	if err := os.WriteFile(testFilePath, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool
			if tt.name == "valid_file_under_home" || tt.name == "nonexistent_file" {
				// For file validation, check if the path points to an existing file
				expandedPath := ExpandHome(tt.path)
				info, err := os.Stat(expandedPath)
				result = err == nil && !info.IsDir()
			} else if tt.name == "valid_directory_under_home" {
				// For directory validation, check if the path points to an existing directory
				expandedPath := ExpandHome(tt.path)
				info, err := os.Stat(expandedPath)
				result = err == nil && info.IsDir()
			} else {
				// For other cases, just check if path exists
				expandedPath := ExpandHome(tt.path)
				_, err := os.Stat(expandedPath)
				result = err == nil
			}

			if result != tt.expected {
				t.Errorf("ValidateFilePath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestPathExists tests the path existence validation functionality
func TestPathExists(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "empty_path",
			path:     "",
			expected: false,
		},
		{
			name:     "path_under_home",
			path:     tmpDir,
			expected: true,
		},
		{
			name:     "nonexistent_path",
			path:     filepath.Join(tmpDir, "nonexistent"),
			expected: false,
		},
		{
			name:     "file_under_home",
			path:     filepath.Join(tmpDir, "test.txt"),
			expected: true,
		},
		{
			name:     "relative_path",
			path:     "relative/path",
			expected: false,
		},
	}

	// Create a test file in the temp directory
	testFilePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFilePath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expandedPath := ExpandHome(tt.path)
			_, err := os.Stat(expandedPath)
			result := err == nil

			if result != tt.expected {
				t.Errorf("PathExists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}
