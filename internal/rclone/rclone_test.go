package rclone

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNewClientUsesEnv(t *testing.T) {
	os.Setenv("RCLONE_BINARY_PATH", "/custom/bin/rclone")
	os.Setenv("RCLONE_CONFIG", "/tmp/rclone.conf")
	defer os.Unsetenv("RCLONE_BINARY_PATH")
	defer os.Unsetenv("RCLONE_CONFIG")

	c := NewClient()
	if c.binaryPath != "/custom/bin/rclone" {
		t.Errorf("binaryPath = %q, want %q", c.binaryPath, "/custom/bin/rclone")
	}
	if c.configPath != "/tmp/rclone.conf" {
		t.Errorf("configPath = %q, want %q", c.configPath, "/tmp/rclone.conf")
	}
}

func TestNewClientWithPath(t *testing.T) {
	os.Setenv("RCLONE_CONFIG", "/tmp/rc.conf")
	defer os.Unsetenv("RCLONE_CONFIG")

	c := NewClientWithPath("/usr/bin/foobar")
	if c.binaryPath != "/usr/bin/foobar" {
		t.Errorf("binaryPath = %q, want %q", c.binaryPath, "/usr/bin/foobar")
	}
	if c.configPath != "/tmp/rc.conf" {
		t.Errorf("configPath = %q, want %q", c.configPath, "/tmp/rc.conf")
	}
}

func TestNewClientDefaultValues(t *testing.T) {
	os.Unsetenv("RCLONE_BINARY_PATH")
	os.Unsetenv("RCLONE_CONFIG")

	c := NewClient()
	if c.binaryPath != "rclone" {
		t.Errorf("binaryPath = %q, want %q", c.binaryPath, "rclone")
	}
	if c.configPath != "" {
		t.Errorf("configPath = %q, want empty", c.configPath)
	}
}

func TestClientIsInstalledCustomBinary(t *testing.T) {
	path, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found in PATH")
	}
	c := NewClientWithPath(path)
	if !c.IsInstalled() {
		t.Errorf("expected IsInstalled() true for %s", path)
	}
}

func TestIsInstalledPackageUsesPathEnv(t *testing.T) {
	oldPath := os.Getenv("PATH")
	defer os.Setenv("PATH", oldPath)
	os.Setenv("PATH", "")

	if IsInstalled() {
		t.Error("IsInstalled() = true with empty PATH, want false")
	}
}

func TestClientIsInstalledNotFound(t *testing.T) {
	c := NewClientWithPath("/nonexistent/path/to/rclone")
	if c.IsInstalled() {
		t.Error("IsInstalled() = true for nonexistent path, want false")
	}
}

func createMockRclone(t *testing.T, script string) string {
	t.Helper()
	tmpDir := t.TempDir()
	mockPath := filepath.Join(tmpDir, "rclone")
	if runtime.GOOS == "windows" {
		mockPath += ".bat"
	}
	if err := os.WriteFile(mockPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create mock rclone: %v", err)
	}
	return mockPath
}

func TestListRemotes(t *testing.T) {
	mockScript := `#!/bin/sh
case "$1" in
	listremotes)
		echo "gdrive:"
		echo "dropbox:"
		echo "s3:"
		;;
	config)
		if [ "$2" = "show" ]; then
			case "$3" in
				gdrive) echo "[gdrive]"; echo "type = drive" ;;
				dropbox) echo "[dropbox]"; echo "type = dropbox" ;;
				s3) echo "[s3]"; echo "type = s3" ;;
			esac
		fi
		;;
esac
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	remotes, err := c.ListRemotes()
	if err != nil {
		t.Fatalf("ListRemotes() error = %v", err)
	}

	if len(remotes) != 3 {
		t.Fatalf("ListRemotes() returned %d remotes, want 3", len(remotes))
	}

	expected := []struct {
		name, remoteType string
	}{
		{"gdrive", "drive"},
		{"dropbox", "dropbox"},
		{"s3", "s3"},
	}

	for i, exp := range expected {
		if remotes[i].Name != exp.name {
			t.Errorf("remotes[%d].Name = %q, want %q", i, remotes[i].Name, exp.name)
		}
		if remotes[i].Type != exp.remoteType {
			t.Errorf("remotes[%d].Type = %q, want %q", i, remotes[i].Type, exp.remoteType)
		}
	}
}

func TestListRemotesEmpty(t *testing.T) {
	mockScript := `#!/bin/sh
echo ""
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	remotes, err := c.ListRemotes()
	if err != nil {
		t.Fatalf("ListRemotes() error = %v", err)
	}

	if len(remotes) != 0 {
		t.Errorf("ListRemotes() returned %d remotes, want 0", len(remotes))
	}
}

func TestListRemotesError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "error: config not found" >&2
exit 1
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	_, err := c.ListRemotes()
	if err == nil {
		t.Error("ListRemotes() expected error, got nil")
	}
}

func TestListRemotesWithConfig(t *testing.T) {
	mockScript := `#!/bin/sh
if [ "$1" = "--config" ]; then
	echo "config: $2" >&2
fi
case "$3" in
	listremotes)
		echo "remote1:"
		;;
	config)
		echo "[remote1]"; echo "type = test"
		;;
esac
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)
	c.SetConfigPath("/custom/config.conf")

	remotes, err := c.ListRemotes()
	if err != nil {
		t.Fatalf("ListRemotes() error = %v", err)
	}

	if len(remotes) != 1 {
		t.Errorf("ListRemotes() returned %d remotes, want 1", len(remotes))
	}
}

func TestGetRemoteType(t *testing.T) {
	mockScript := `#!/bin/sh
echo "[gdrive]"
echo "type = drive"
echo "client_id = xxx"
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	remoteType, err := c.GetRemoteType("gdrive")
	if err != nil {
		t.Fatalf("GetRemoteType() error = %v", err)
	}

	if remoteType != "drive" {
		t.Errorf("GetRemoteType() = %q, want %q", remoteType, "drive")
	}
}

func TestGetRemoteTypeNoSpace(t *testing.T) {
	mockScript := `#!/bin/sh
echo "[s3]"
echo "type=s3"
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	remoteType, err := c.GetRemoteType("s3")
	if err != nil {
		t.Fatalf("GetRemoteType() error = %v", err)
	}

	if remoteType != "s3" {
		t.Errorf("GetRemoteType() = %q, want %q", remoteType, "s3")
	}
}

func TestGetRemoteTypeNotFound(t *testing.T) {
	mockScript := `#!/bin/sh
echo "[other]"
echo "name = value"
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	_, err := c.GetRemoteType("gdrive")
	if err == nil {
		t.Error("GetRemoteType() expected error when type not found")
	}
}

func TestGetRemoteTypeError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "remote not found" >&2
exit 1
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	_, err := c.GetRemoteType("nonexistent")
	if err == nil {
		t.Error("GetRemoteType() expected error")
	}
}

func TestListRemotePath(t *testing.T) {
	mockScript := `#!/bin/sh
echo "Photos/"
echo "Documents/"
echo "file.txt"
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	entries, err := c.ListRemotePath("gdrive", "/")
	if err != nil {
		t.Fatalf("ListRemotePath() error = %v", err)
	}

	expected := []string{"Photos/", "Documents/", "file.txt"}
	if len(entries) != len(expected) {
		t.Fatalf("ListRemotePath() returned %d entries, want %d", len(entries), len(expected))
	}

	for i, exp := range expected {
		if entries[i] != exp {
			t.Errorf("entries[%d] = %q, want %q", i, entries[i], exp)
		}
	}
}

func TestListRemotePathEmpty(t *testing.T) {
	mockScript := `#!/bin/sh
echo ""
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	entries, err := c.ListRemotePath("gdrive", "/empty")
	if err != nil {
		t.Fatalf("ListRemotePath() error = %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("ListRemotePath() returned %d entries, want 0", len(entries))
	}
}

func TestListRemotePathError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "directory not found" >&2
exit 1
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	_, err := c.ListRemotePath("gdrive", "/nonexistent")
	if err == nil {
		t.Error("ListRemotePath() expected error")
	}
}

func TestListRemoteDirectories(t *testing.T) {
	mockScript := `#!/bin/sh
echo "Photos/"
echo "Documents/"
echo "Music/"
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	dirs, err := c.ListRemoteDirectories("gdrive", "/")
	if err != nil {
		t.Fatalf("ListRemoteDirectories() error = %v", err)
	}

	expected := []string{"Photos", "Documents", "Music"}
	if len(dirs) != len(expected) {
		t.Fatalf("ListRemoteDirectories() returned %d dirs, want %d", len(dirs), len(expected))
	}

	for i, exp := range expected {
		if dirs[i] != exp {
			t.Errorf("dirs[%d] = %q, want %q", i, dirs[i], exp)
		}
	}
}

func TestListRemoteDirectoriesEmpty(t *testing.T) {
	mockScript := `#!/bin/sh
echo ""
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	dirs, err := c.ListRemoteDirectories("gdrive", "/empty")
	if err != nil {
		t.Fatalf("ListRemoteDirectories() error = %v", err)
	}

	if len(dirs) != 0 {
		t.Errorf("ListRemoteDirectories() returned %d dirs, want 0", len(dirs))
	}
}

func TestListRemoteDirectoriesError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "access denied" >&2
exit 1
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	_, err := c.ListRemoteDirectories("gdrive", "/private")
	if err == nil {
		t.Error("ListRemoteDirectories() expected error")
	}
}

func TestListRootDirectories(t *testing.T) {
	mockScript := `#!/bin/sh
echo "Photos/"
echo "Documents/"
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	dirs, err := c.ListRootDirectories("gdrive")
	if err != nil {
		t.Fatalf("ListRootDirectories() error = %v", err)
	}

	if len(dirs) != 2 {
		t.Errorf("ListRootDirectories() returned %d dirs, want 2", len(dirs))
	}
}

func TestValidateRemoteFound(t *testing.T) {
	mockScript := `#!/bin/sh
case "$1" in
	listremotes)
		echo "gdrive:"
		echo "dropbox:"
		;;
	config)
		if [ "$2" = "show" ]; then
			case "$3" in
				gdrive) echo "[gdrive]"; echo "type = drive" ;;
				dropbox) echo "[dropbox]"; echo "type = dropbox" ;;
			esac
		fi
		;;
esac
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	err := c.ValidateRemote("gdrive")
	if err != nil {
		t.Errorf("ValidateRemote() error = %v", err)
	}
}

func TestValidateRemoteNotFound(t *testing.T) {
	mockScript := `#!/bin/sh
case "$1" in
	listremotes)
		echo "gdrive:"
		;;
	config)
		echo "[gdrive]"; echo "type = drive"
		;;
esac
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	err := c.ValidateRemote("dropbox")
	if err == nil {
		t.Error("ValidateRemote() expected error for nonexistent remote")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("ValidateRemote() error should contain 'not found', got: %v", err)
	}
}

func TestValidateRemoteListError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "error" >&2
exit 1
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	err := c.ValidateRemote("gdrive")
	if err == nil {
		t.Error("ValidateRemote() expected error")
	}
}

func TestTestRemoteAccess(t *testing.T) {
	mockScript := `#!/bin/sh
echo ""
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	err := c.TestRemoteAccess("gdrive", "/")
	if err != nil {
		t.Errorf("TestRemoteAccess() error = %v", err)
	}
}

func TestTestRemoteAccessError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "access denied" >&2
exit 1
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	err := c.TestRemoteAccess("gdrive", "/private")
	if err == nil {
		t.Error("TestRemoteAccess() expected error")
	}
}

func TestGetVersion(t *testing.T) {
	mockScript := `#!/bin/sh
echo "rclone v1.62.0"
echo "- os/version: linux"
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	version, err := c.GetVersion()
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}

	if version != "rclone v1.62.0" {
		t.Errorf("GetVersion() = %q, want %q", version, "rclone v1.62.0")
	}
}

func TestGetVersionError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "not found" >&2
exit 1
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	_, err := c.GetVersion()
	if err == nil {
		t.Error("GetVersion() expected error")
	}
}

func TestGetConfigPathCustom(t *testing.T) {
	c := NewClientWithPath("rclone")
	c.SetConfigPath("/custom/rclone.conf")

	path, err := c.GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() error = %v", err)
	}

	if path != "/custom/rclone.conf" {
		t.Errorf("GetConfigPath() = %q, want %q", path, "/custom/rclone.conf")
	}
}

func TestGetConfigPathFromRclone(t *testing.T) {
	mockScript := `#!/bin/sh
echo "Configuration file is stored at:"
echo "/home/user/.config/rclone/rclone.conf"
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	path, err := c.GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() error = %v", err)
	}

	if path != "/home/user/.config/rclone/rclone.conf" {
		t.Errorf("GetConfigPath() = %q, want %q", path, "/home/user/.config/rclone/rclone.conf")
	}
}

func TestGetConfigPathError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "error" >&2
exit 1
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	_, err := c.GetConfigPath()
	if err == nil {
		t.Error("GetConfigPath() expected error")
	}
}

func TestRunCommand(t *testing.T) {
	mockScript := `#!/bin/sh
echo "output: $*"
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	ctx := context.Background()
	output, err := c.runCommand(ctx, "listremotes")
	if err != nil {
		t.Fatalf("runCommand() error = %v", err)
	}

	if !strings.Contains(string(output), "listremotes") {
		t.Errorf("runCommand() output should contain 'listremotes', got: %s", output)
	}
}

func TestRunCommandWithConfig(t *testing.T) {
	mockScript := `#!/bin/sh
echo "args: $*"
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)
	c.SetConfigPath("/custom.conf")

	ctx := context.Background()
	output, err := c.runCommand(ctx, "listremotes")
	if err != nil {
		t.Fatalf("runCommand() error = %v", err)
	}

	out := string(output)
	if !strings.Contains(out, "--config") {
		t.Errorf("runCommand() should include --config flag, got: %s", out)
	}
	if !strings.Contains(out, "/custom.conf") {
		t.Errorf("runCommand() should include config path, got: %s", out)
	}
}

func TestRunCommandTimeout(t *testing.T) {
	mockScript := `#!/bin/sh
sleep 5
echo "done"
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := c.runCommand(ctx, "slow-command")
	if err == nil {
		t.Error("runCommand() expected timeout error")
	}
}

func TestRemotePathStructure(t *testing.T) {
	rp := RemotePath{
		Remote: "gdrive",
		Path:   "/Photos/2024",
	}

	if rp.Remote != "gdrive" {
		t.Errorf("RemotePath.Remote = %q, want %q", rp.Remote, "gdrive")
	}
	if rp.Path != "/Photos/2024" {
		t.Errorf("RemotePath.Path = %q, want %q", rp.Path, "/Photos/2024")
	}
}

func TestListRemotesWithUnknownType(t *testing.T) {
	mockScript := `#!/bin/sh
case "$1" in
	listremotes)
		echo "gdrive:"
		;;
	config)
		echo "error" >&2
		exit 1
		;;
esac
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	remotes, err := c.ListRemotes()
	if err != nil {
		t.Fatalf("ListRemotes() error = %v", err)
	}

	if len(remotes) != 1 {
		t.Fatalf("ListRemotes() returned %d remotes, want 1", len(remotes))
	}

	if remotes[0].Name != "gdrive" {
		t.Errorf("remotes[0].Name = %q, want %q", remotes[0].Name, "gdrive")
	}
	if remotes[0].Type != "unknown" {
		t.Errorf("remotes[0].Type = %q, want 'unknown'", remotes[0].Type)
	}
}

func TestListRemotesWithEmptyLines(t *testing.T) {
	mockScript := `#!/bin/sh
case "$1" in
	listremotes)
		echo ""
		echo "gdrive:"
		echo ""
		echo ""
		;;
	config)
		echo "[gdrive]"; echo "type = drive"
		;;
esac
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	remotes, err := c.ListRemotes()
	if err != nil {
		t.Fatalf("ListRemotes() error = %v", err)
	}

	if len(remotes) != 1 {
		t.Errorf("ListRemotes() returned %d remotes, want 1", len(remotes))
	}
}

func TestListRemotesWithColonOnly(t *testing.T) {
	mockScript := `#!/bin/sh
case "$1" in
	listremotes)
		echo ":"
		echo "gdrive:"
		;;
	config)
		echo "[gdrive]"; echo "type = drive"
		;;
esac
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	remotes, err := c.ListRemotes()
	if err != nil {
		t.Fatalf("ListRemotes() error = %v", err)
	}

	if len(remotes) != 1 {
		t.Errorf("ListRemotes() returned %d remotes, want 1 (empty name should be skipped)", len(remotes))
	}
}

func TestListRemotePathWithEmptyLines(t *testing.T) {
	mockScript := `#!/bin/sh
echo ""
echo "file1.txt"
echo ""
echo "file2.txt"
echo ""
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	entries, err := c.ListRemotePath("gdrive", "/")
	if err != nil {
		t.Fatalf("ListRemotePath() error = %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("ListRemotePath() returned %d entries, want 2", len(entries))
	}
}

func TestListRemoteDirectoriesWithEmptyLines(t *testing.T) {
	mockScript := `#!/bin/sh
echo ""
echo "dir1/"
echo ""
echo "dir2/"
echo ""
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)

	dirs, err := c.ListRemoteDirectories("gdrive", "/")
	if err != nil {
		t.Fatalf("ListRemoteDirectories() error = %v", err)
	}

	if len(dirs) != 2 {
		t.Errorf("ListRemoteDirectories() returned %d dirs, want 2", len(dirs))
	}
}

func TestRunCommandWithRetrySuccessOnFirstTry(t *testing.T) {
	mockScript := `#!/bin/sh
echo "output: $*"
`
	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)
	c.SetRetryConfig(RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	})

	ctx := context.Background()
	output, err := c.runCommandWithRetry(ctx, "listremotes")
	if err != nil {
		t.Fatalf("runCommandWithRetry() error = %v", err)
	}

	if !strings.Contains(string(output), "listremotes") {
		t.Errorf("runCommandWithRetry() output should contain 'listremotes', got: %s", output)
	}
}

func TestRunCommandWithRetrySuccessAfterRetry(t *testing.T) {
	attemptFile := filepath.Join(t.TempDir(), "attempts")

	mockScript := fmt.Sprintf(`#!/bin/sh
attempt=$(cat %s 2>/dev/null || echo 0)
attempt=$((attempt + 1))
echo $attempt > %s

if [ "$attempt" -lt 2 ]; then
    echo "connection timeout" >&2
    exit 1
fi

echo "success: $*"
`, attemptFile, attemptFile)

	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)
	c.SetRetryConfig(RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	})

	ctx := context.Background()
	output, err := c.runCommandWithRetry(ctx, "listremotes")
	if err != nil {
		t.Fatalf("runCommandWithRetry() error = %v", err)
	}

	if !strings.Contains(string(output), "success") {
		t.Errorf("runCommandWithRetry() output should contain 'success', got: %s", output)
	}

	attemptCount, _ := os.ReadFile(attemptFile)
	if strings.TrimSpace(string(attemptCount)) != "2" {
		t.Errorf("expected 2 attempts, got %s", string(attemptCount))
	}
}

func TestRunCommandWithRetryFailureAfterMaxRetries(t *testing.T) {
	mockScript := `#!/bin/sh
echo "connection timeout" >&2
exit 1
`

	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)
	c.SetRetryConfig(RetryConfig{
		MaxRetries:      2,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	})

	ctx := context.Background()
	_, err := c.runCommandWithRetry(ctx, "listremotes")
	if err == nil {
		t.Fatal("runCommandWithRetry() should return error after max retries")
	}

	if !strings.Contains(err.Error(), "failed after") {
		t.Errorf("error should mention failed attempts, got: %v", err)
	}
}

func TestRunCommandWithRetryContextCancellation(t *testing.T) {
	mockScript := `#!/bin/sh
echo "connection timeout" >&2
exit 1
`

	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)
	c.SetRetryConfig(RetryConfig{
		MaxRetries:      5,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        1 * time.Second,
		RetryMultiplier: 2.0,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := c.runCommandWithRetry(ctx, "listremotes")
	if err != context.Canceled {
		t.Errorf("runCommandWithRetry() should return context.Canceled, got %v", err)
	}
}

func TestRunCommandWithRetryPermanentError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "config file not found" >&2
exit 1
`

	mockPath := createMockRclone(t, mockScript)
	c := NewClientWithPath(mockPath)
	c.SetRetryConfig(RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	})

	ctx := context.Background()
	_, err := c.runCommandWithRetry(ctx, "listremotes")
	if err == nil {
		t.Fatal("runCommandWithRetry() should return error for permanent error")
	}

	if !IsPermanentError(err) {
		t.Errorf("error should be permanent, got: %v", err)
	}
}

func TestRunCommandWithRetryUsesClientConfig(t *testing.T) {
	mockScript := `#!/bin/sh
echo "success"
`

	mockPath := createMockRclone(t, mockScript)

	customConfig := RetryConfig{
		MaxRetries:      7,
		InitialDelay:    123 * time.Millisecond,
		MaxDelay:        5 * time.Second,
		RetryMultiplier: 1.5,
	}

	c := NewClientWithPath(mockPath)
	c.SetRetryConfig(customConfig)

	retrievedConfig := c.GetRetryConfig()
	if retrievedConfig.MaxRetries != customConfig.MaxRetries {
		t.Errorf("GetRetryConfig().MaxRetries = %d, want %d", retrievedConfig.MaxRetries, customConfig.MaxRetries)
	}
	if retrievedConfig.InitialDelay != customConfig.InitialDelay {
		t.Errorf("GetRetryConfig().InitialDelay = %v, want %v", retrievedConfig.InitialDelay, customConfig.InitialDelay)
	}
	if retrievedConfig.MaxDelay != customConfig.MaxDelay {
		t.Errorf("GetRetryConfig().MaxDelay = %v, want %v", retrievedConfig.MaxDelay, customConfig.MaxDelay)
	}
	if retrievedConfig.RetryMultiplier != customConfig.RetryMultiplier {
		t.Errorf("GetRetryConfig().RetryMultiplier = %v, want %v", retrievedConfig.RetryMultiplier, customConfig.RetryMultiplier)
	}
}
