package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetUserSystemdPath_NoConfigDir(t *testing.T) {
	originalHome := os.Getenv("HOME")
	originalXdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_CONFIG_HOME", originalXdgConfigHome)
	}()

	os.Unsetenv("HOME")
	os.Unsetenv("XDG_CONFIG_HOME")

	_, err := GetUserSystemdPath()
	if err == nil {
		t.Error("GetUserSystemdPath() should return error when config dir cannot be determined")
	}
}

func TestGetUserSystemdPath_WithEnv(t *testing.T) {
	originalXdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXdgConfigHome)

	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := GetUserSystemdPath()
	if err != nil {
		t.Fatalf("GetUserSystemdPath() error = %v", err)
	}

	expected := filepath.Join(tmpDir, "systemd", "user")
	if path != expected {
		t.Errorf("GetUserSystemdPath() = %q, want %q", path, expected)
	}
}

func TestExpandPath_NoHomeDir(t *testing.T) {
	originalHome := os.Getenv("HOME")
	originalUserEnv := os.Getenv("USER")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("USER", originalUserEnv)
	}()

	os.Unsetenv("HOME")
	os.Unsetenv("USER")

	input := "~/Documents"
	got := expandPath(input)

	if got != input {
		t.Errorf("expandPath(%q) = %q, want %q (original path when home unavailable)", input, got, input)
	}
}

func TestExpandPath_NoHomeDirWithAbsolutePath(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Unsetenv("HOME")

	input := "/absolute/path"
	got := expandPath(input)

	if got != input {
		t.Errorf("expandPath(%q) = %q, want %q", input, got, input)
	}
}

func TestExpandPath_NoHomeDirWithRelativePath(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Unsetenv("HOME")

	input := "relative/path"
	got := expandPath(input)

	if got != input {
		t.Errorf("expandPath(%q) = %q, want %q", input, got, input)
	}
}

func TestGetLogDir_NoHomeDir(t *testing.T) {
	originalHome := os.Getenv("HOME")
	originalXdgStateHome := os.Getenv("XDG_STATE_HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_STATE_HOME", originalXdgStateHome)
	}()

	os.Unsetenv("HOME")
	os.Unsetenv("XDG_STATE_HOME")

	_, err := getLogDir()
	if err == nil {
		t.Error("getLogDir() should return error when home dir cannot be determined")
	}
}

func TestGetLogDir_MkdirAllPermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	originalXdgStateHome := os.Getenv("XDG_STATE_HOME")
	defer os.Setenv("XDG_STATE_HOME", originalXdgStateHome)

	tmpDir := t.TempDir()
	readonlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readonlyDir, 0555); err != nil {
		t.Fatalf("Failed to create readonly dir: %v", err)
	}

	os.Setenv("XDG_STATE_HOME", readonlyDir)

	_, err := getLogDir()
	if err == nil {
		t.Error("getLogDir() should return error when mkdir fails due to permission")
	}
}

func TestGetLogDir_XdgStateHomeMkdirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	originalXdgStateHome := os.Getenv("XDG_STATE_HOME")
	defer os.Setenv("XDG_STATE_HOME", originalXdgStateHome)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	os.Setenv("XDG_STATE_HOME", filePath)

	_, err := getLogDir()
	if err == nil {
		t.Error("getLogDir() should return error when mkdir fails on file path")
	}
}

func TestGetRcloneConfigPath_NoHomeDir(t *testing.T) {
	originalHome := os.Getenv("HOME")
	originalUser := os.Getenv("USER")
	originalRcloneConfig := os.Getenv("RCLONE_CONFIG")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("USER", originalUser)
		os.Setenv("RCLONE_CONFIG", originalRcloneConfig)
	}()

	os.Unsetenv("HOME")
	os.Unsetenv("RCLONE_CONFIG")
	os.Setenv("USER", "testuser")

	path := getRcloneConfigPath()

	expected := filepath.Join("/home", "testuser", ".config", "rclone", "rclone.conf")
	if path != expected {
		t.Errorf("getRcloneConfigPath() = %q, want %q", path, expected)
	}
}

func TestGetRcloneConfigPath_NoHomeDirNoUser(t *testing.T) {
	originalHome := os.Getenv("HOME")
	originalUser := os.Getenv("USER")
	originalRcloneConfig := os.Getenv("RCLONE_CONFIG")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("USER", originalUser)
		os.Setenv("RCLONE_CONFIG", originalRcloneConfig)
	}()

	os.Unsetenv("HOME")
	os.Unsetenv("USER")
	os.Unsetenv("RCLONE_CONFIG")

	path := getRcloneConfigPath()

	if !strings.Contains(path, "rclone.conf") {
		t.Errorf("getRcloneConfigPath() = %q, should contain 'rclone.conf'", path)
	}
	if !strings.Contains(path, ".config") {
		t.Errorf("getRcloneConfigPath() = %q, should contain '.config'", path)
	}
}

func TestGetRcloneConfigPath_EnvOverride(t *testing.T) {
	originalRcloneConfig := os.Getenv("RCLONE_CONFIG")
	defer os.Setenv("RCLONE_CONFIG", originalRcloneConfig)

	customPath := "/custom/path/rclone.conf"
	os.Setenv("RCLONE_CONFIG", customPath)

	path := getRcloneConfigPath()

	if path != customPath {
		t.Errorf("getRcloneConfigPath() = %q, want %q", path, customPath)
	}
}

func TestSanitizeName_AllSpecialChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "only special characters",
			input: "@#$%^&*()",
			want:  "",
		},
		{
			name:  "mixed special with valid",
			input: "a@b#c$d",
			want:  "a-b-c-d",
		},
		{
			name:  "unicode characters",
			input: "日本語",
			want:  "",
		},
		{
			name:  "emoji",
			input: "test📁file",
			want:  "test-file",
		},
		{
			name:  "multiple consecutive special",
			input: "a@@@b",
			want:  "a-b",
		},
		{
			name:  "tabs and newlines",
			input: "a\tb\nc",
			want:  "a-b-c",
		},
		{
			name:  "slash characters",
			input: "path/to/file",
			want:  "path-to-file",
		},
		{
			name:  "backslash characters",
			input: "path\\to\\file",
			want:  "path-to-file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetLogDir_Success(t *testing.T) {
	originalXdgStateHome := os.Getenv("XDG_STATE_HOME")
	defer os.Setenv("XDG_STATE_HOME", originalXdgStateHome)

	tmpDir := t.TempDir()
	os.Setenv("XDG_STATE_HOME", tmpDir)

	dir, err := getLogDir()
	if err != nil {
		t.Fatalf("getLogDir() error = %v", err)
	}

	if !strings.HasPrefix(dir, tmpDir) {
		t.Errorf("getLogDir() = %q, should start with %q", dir, tmpDir)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("getLogDir() did not create directory %q", dir)
	}
}

func TestGetLogDir_UsesHomeWhenNoXdgStateHome(t *testing.T) {
	originalHome := os.Getenv("HOME")
	originalXdgStateHome := os.Getenv("XDG_STATE_HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_STATE_HOME", originalXdgStateHome)
	}()

	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("XDG_STATE_HOME")

	dir, err := getLogDir()
	if err != nil {
		t.Fatalf("getLogDir() error = %v", err)
	}

	expectedPrefix := filepath.Join(tmpDir, ".local", "state")
	if !strings.HasPrefix(dir, expectedPrefix) {
		t.Errorf("getLogDir() = %q, should start with %q", dir, expectedPrefix)
	}
}

func TestExpandPath_TildeOnly(t *testing.T) {
	got := expandPath("~")

	if got != "~" {
		t.Errorf("expandPath(\"~\") = %q, want \"~\" (unchanged, function only handles ~/)", got)
	}
}

func TestExpandPath_EmptyString(t *testing.T) {
	got := expandPath("")
	if got != "" {
		t.Errorf("expandPath(\"\") = %q, want \"\"", got)
	}
}

func TestExpandPath_MultipleTildes(t *testing.T) {
	input := "~~/path"
	got := expandPath(input)

	if got != input {
		t.Errorf("expandPath(%q) = %q, want %q (unchanged)", input, got, input)
	}
}

func TestExpandPath_TildeInMiddle(t *testing.T) {
	input := "/path/~user/file"
	got := expandPath(input)

	if got != input {
		t.Errorf("expandPath(%q) = %q, want %q (unchanged)", input, got, input)
	}
}
