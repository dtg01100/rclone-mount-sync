package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "expand tilde",
			input:    "~/test/path",
			expected: filepath.Join(home, "test/path"),
		},
		{
			name:     "no expansion needed",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "tilde only",
			input:    "~",
			expected: "~",
		},
		{
			name:     "tilde in middle",
			input:    "/home/user~/test",
			expected: "/home/user~/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandHome(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "expand tilde",
			input:    "~/test/path",
			expected: filepath.Join(home, "test/path"),
			wantErr:  false,
		},
		{
			name:     "no expansion needed",
			input:    "/absolute/path",
			expected: "/absolute/path",
			wantErr:  false,
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing file",
			path:     tmpDir + "/testfile.txt",
			expected: true,
		},
		{
			name:     "non-existent file",
			path:     tmpDir + "/nonexistent.txt",
			expected: false,
		},
		{
			name:     "directory",
			path:     tmpDir,
			expected: false,
		},
	}

	if err := os.WriteFile(tmpDir+"/testfile.txt", []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FileExists(tt.path)
			if result != tt.expected {
				t.Errorf("FileExists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing directory",
			path:     tmpDir,
			expected: true,
		},
		{
			name:     "non-existent directory",
			path:     tmpDir + "/nonexistent",
			expected: false,
		},
		{
			name:     "file",
			path:     tmpDir + "/testfile.txt",
			expected: false,
		},
	}

	if err := os.WriteFile(tmpDir+"/testfile.txt", []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DirExists(tt.path)
			if result != tt.expected {
				t.Errorf("DirExists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "level1", "level2", "level3")

	if err := EnsureDir(newDir); err != nil {
		t.Errorf("EnsureDir(%q) error = %v", newDir, err)
	}

	if !DirExists(newDir) {
		t.Errorf("EnsureDir(%q) did not create directory", newDir)
	}

	if err := EnsureDir(newDir); err != nil {
		t.Errorf("EnsureDir(%q) on existing dir error = %v", newDir, err)
	}
}

func TestGetHomeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	result, err := GetHomeDir()
	if err != nil {
		t.Errorf("GetHomeDir() error = %v", err)
	}
	if result != home {
		t.Errorf("GetHomeDir() = %q, want %q", result, home)
	}
}

func TestGetConfigDir(t *testing.T) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("failed to get config dir: %v", err)
	}

	result, err := GetConfigDir()
	if err != nil {
		t.Errorf("GetConfigDir() error = %v", err)
	}
	if result != configDir {
		t.Errorf("GetConfigDir() = %q, want %q", result, configDir)
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "MyMount",
			expected: "mymount",
		},
		{
			name:     "spaces to dashes",
			input:    "My Mount Name",
			expected: "my-mount-name",
		},
		{
			name:     "underscores to dashes",
			input:    "my_mount_name",
			expected: "my-mount-name",
		},
		{
			name:     "special characters removed",
			input:    "My@Mount#Name!",
			expected: "mymountname",
		},
		{
			name:     "numbers preserved",
			input:    "Mount123",
			expected: "mount123",
		},
		{
			name:     "multiple dashes collapsed",
			input:    "my---mount---name",
			expected: "my---mount---name",
		},
		{
			name:     "leading dashes kept",
			input:    "-leading",
			expected: "-leading",
		},
		{
			name:     "trailing dashes kept",
			input:    "trailing-",
			expected: "trailing-",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateMountPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "absolute path",
			path:    "/mnt/data",
			wantErr: false,
		},
		{
			name:    "expanded tilde path",
			path:    filepath.Join(home, "mount"),
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    "relative/path",
			wantErr: true,
		},
		{
			name:    "tilde without expansion",
			path:    "~/mount",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMountPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMountPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
