package rclone

import (
	"os"
	"os/exec"
	"testing"
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
