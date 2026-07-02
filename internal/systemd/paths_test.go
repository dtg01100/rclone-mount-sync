package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetUserSystemdPath_NoConfigDir(t *testing.T) {
	// t.Setenv to "" clears the var for the duration of the test.
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	_, err := GetUserSystemdPath()
	if err == nil {
		t.Error("GetUserSystemdPath() should return error when config dir cannot be determined")
	}
}

func TestGetUserSystemdPath_WithEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

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
	t.Setenv("HOME", "")
	t.Setenv("USER", "")

	input := "~/Documents"
	got := expandPath(input)

	if got != input {
		t.Errorf("expandPath(%q) = %q, want %q (original path when home unavailable)", input, got, input)
	}
}

func TestExpandPath_NoHomeDirWithAbsolutePath(t *testing.T) {
	t.Setenv("HOME", "")

	input := "/absolute/path"
	got := expandPath(input)

	if got != input {
		t.Errorf("expandPath(%q) = %q, want %q", input, got, input)
	}
}

func TestExpandPath_NoHomeDirWithRelativePath(t *testing.T) {
	t.Setenv("HOME", "")

	input := "relative/path"
	got := expandPath(input)

	if got != input {
		t.Errorf("expandPath(%q) = %q, want %q", input, got, input)
	}
}

func TestGetLogDir_NoHomeDir(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_STATE_HOME", "")

	_, err := getLogDir()
	if err == nil {
		t.Error("getLogDir() should return error when home dir cannot be determined")
	}
}

func TestGetLogDir_MkdirAllPermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	readonlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readonlyDir, 0555); err != nil { //nolint:gosec
		t.Fatalf("Failed to create readonly dir: %v", err)
	}

	t.Setenv("XDG_STATE_HOME", readonlyDir)

	_, err := getLogDir()
	if err == nil {
		t.Error("getLogDir() should return error when mkdir fails due to permission")
	}
}

func TestGetLogDir_XdgStateHomeMkdirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil { //nolint:gosec
		t.Fatalf("Failed to create file: %v", err)
	}

	t.Setenv("XDG_STATE_HOME", filePath)

	_, err := getLogDir()
	if err == nil {
		t.Error("getLogDir() should return error when mkdir fails on file path")
	}
}

func TestGetRcloneConfigPath_NoHomeDir(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("RCLONE_CONFIG", "")
	t.Setenv("USER", "testuser")

	path := getRcloneConfigPath()

	if path != "" {
		t.Errorf("getRcloneConfigPath() = %q, want empty string when HOME is unset (no USER fallback for safety)", path)
	}
}

func TestGetRcloneConfigPath_NoHomeDirNoUser(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USER", "")
	t.Setenv("RCLONE_CONFIG", "")

	path := getRcloneConfigPath()

	if path != "" {
		t.Errorf("getRcloneConfigPath() = %q, want empty string when HOME and USER are unset", path)
	}
}

func TestGetRcloneConfigPath_EnvOverride(t *testing.T) {
	customPath := "/custom/path/rclone.conf"
	t.Setenv("RCLONE_CONFIG", customPath)

	path := getRcloneConfigPath()

	if path != customPath {
		t.Errorf("getRcloneConfigPath() = %q, want %q", path, customPath)
	}
}

func TestGetLogDir_Success(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

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
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_STATE_HOME", "")

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
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	got := expandPath("~")

	if got != home {
		t.Errorf("expandPath(\"~\") = %q, want %q", got, home)
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
