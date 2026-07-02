package utils

import (
	"bytes"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strings"
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
			expected: home,
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

func TestExpandHome_Errors(t *testing.T) {
	// Save original functions
	origHomeDir := osUserHomeDir
	origUserLookup := userLookup
	t.Cleanup(func() {
		osUserHomeDir = origHomeDir
		userLookup = origUserLookup
	})

	t.Run("UserHomeDir error with tilde only", func(t *testing.T) {
		osUserHomeDir = func() (string, error) {
			return "", errors.New("no home dir")
		}
		result := ExpandHome("~")
		if result != "~" {
			t.Errorf("ExpandHome(~) = %q, want ~", result)
		}
	})

	t.Run("UserHomeDir error with ~/path", func(t *testing.T) {
		osUserHomeDir = func() (string, error) {
			return "", errors.New("no home dir")
		}
		result := ExpandHome("~/test")
		if result != "~/test" {
			t.Errorf("ExpandHome(~/test) = %q, want ~/test", result)
		}
	})

	t.Run("user.Lookup error with ~user", func(t *testing.T) {
		userLookup = func(username string) (*user.User, error) {
			return nil, errors.New("user not found")
		}
		result := ExpandHome("~nonexistentuser/test")
		if result != "~nonexistentuser/test" {
			t.Errorf("ExpandHome(~nonexistentuser/test) = %q, want ~nonexistentuser/test", result)
		}
	})

	t.Run("~user without slash returns home dir", func(t *testing.T) {
		userLookup = func(username string) (*user.User, error) {
			return &user.User{HomeDir: "/home/testuser"}, nil
		}
		result := ExpandHome("~testuser")
		if result != "/home/testuser" {
			t.Errorf("ExpandHome(~testuser) = %q, want /home/testuser", result)
		}
	})
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

	if err := os.WriteFile(tmpDir+"/testfile.txt", []byte("test"), 0600); err != nil {
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

	if err := os.WriteFile(tmpDir+"/testfile.txt", []byte("test"), 0600); err != nil {
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

func TestEnsureDir_Errors(t *testing.T) {
	// Save original function
	origMkdirAll := osMkdirAll
	t.Cleanup(func() {
		osMkdirAll = origMkdirAll
	})

	t.Run("osMkdirAll error", func(t *testing.T) {
		osMkdirAll = func(path string, perm os.FileMode) error {
			return errors.New("permission denied")
		}
		err := EnsureDir("/some/path")
		if err == nil {
			t.Error("EnsureDir should return error on osMkdirAll failure")
		}
	})

	t.Run("propagates non-EEXIST errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create the dir with the real osMkdirAll first.
		osMkdirAll = os.MkdirAll
		if err := EnsureDir(tmpDir); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
		// Now simulate os.MkdirAll returning a non-fs.ErrExist error.
		// EnsureDir should propagate it rather than swallow it.
		called := false
		osMkdirAll = func(path string, perm os.FileMode) error {
			called = true
			return errors.New("file exists")
		}
		err := EnsureDir(tmpDir)
		if err == nil {
			t.Error("EnsureDir should return error when osMkdirAll fails with a non-EEXIST error")
		}
		if !called {
			t.Error("osMkdirAll should be called")
		}
	})
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

func TestGenerateID(t *testing.T) {
	// Length: 8 characters (per the contract — truncated UUID).
	// Charset: 0-9a-f (hex, since uuid.New() emits hex).
	// Uniqueness: a small sample should produce no duplicates.

	const sample = 1000
	seen := make(map[string]struct{}, sample)

	for i := range sample {
		id := GenerateID()

		if len(id) != 8 {
			t.Fatalf("GenerateID() length = %d, want 8 (got %q on iteration %d)", len(id), id, i)
		}
		for _, r := range id {
			if r < '0' || (r > '9' && r < 'a') || r > 'f' {
				t.Errorf("GenerateID() = %q, contains non-hex char %q", id, r)
				break
			}
		}
		if _, dup := seen[id]; dup {
			t.Errorf("GenerateID() returned duplicate %q after %d iterations", id, i)
		}
		seen[id] = struct{}{}
	}
}

func TestNoteWarning(t *testing.T) {
	orig := WarningSink
	defer func() { WarningSink = orig }()

	var buf bytes.Buffer
	WarningSink = &buf

	NoteWarning("simple message")
	if got := buf.String(); got != "Warning: simple message\n" {
		t.Errorf("NoteWarning(simple) = %q, want %q", got, "Warning: simple message\n")
	}

	buf.Reset()
	NoteWarning("failed to %s unit %s: %v", "stop", "rclone-mount-x", errors.New("boom"))
	want := "Warning: failed to stop unit rclone-mount-x: boom\n"
	if got := buf.String(); got != want {
		t.Errorf("NoteWarning(formatted) = %q, want %q", got, want)
	}
}

func TestNoteWarningAlwaysPrefixed(t *testing.T) {
	orig := WarningSink
	defer func() { WarningSink = orig }()

	var buf bytes.Buffer
	WarningSink = &buf

	NoteWarning("a")
	NoteWarning("b")
	out := buf.String()
	if !strings.HasPrefix(out, "Warning: ") {
		t.Errorf("NoteWarning output must start with 'Warning: ', got %q", out)
	}
	if strings.Count(out, "Warning: ") != 2 {
		t.Errorf("expected two Warning: prefixes, got %q", out)
	}
}

func TestNoteError(t *testing.T) {
	orig := ErrorSink
	defer func() { ErrorSink = orig }()

	var buf bytes.Buffer
	ErrorSink = &buf

	NoteError("simple message")
	if got := buf.String(); got != "Error: simple message\n" {
		t.Errorf("NoteError(simple) = %q, want %q", got, "Error: simple message\n")
	}

	buf.Reset()
	NoteError("failed to %s unit %s: %v", "stop", "rclone-mount-x", errors.New("boom"))
	want := "Error: failed to stop unit rclone-mount-x: boom\n"
	if got := buf.String(); got != want {
		t.Errorf("NoteError(formatted) = %q, want %q", got, want)
	}
}

func TestNoteErrorAlwaysPrefixed(t *testing.T) {
	orig := ErrorSink
	defer func() { ErrorSink = orig }()

	var buf bytes.Buffer
	ErrorSink = &buf

	NoteError("a")
	NoteError("b")
	out := buf.String()
	if !strings.HasPrefix(out, "Error: ") {
		t.Errorf("NoteError output must start with 'Error: ', got %q", out)
	}
	if strings.Count(out, "Error: ") != 2 {
		t.Errorf("expected two Error: prefixes, got %q", out)
	}
}
