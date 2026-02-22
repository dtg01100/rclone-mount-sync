// Package rclone provides a client wrapper for interacting with rclone commands.
package rclone

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Client wraps rclone command execution.
type Client struct {
	binaryPath string
	configPath string
}

// NewClient creates a new rclone client.
// It first checks for a custom binary path via the RCLONE_BINARY_PATH environment variable,
// then falls back to searching for "rclone" in PATH.
func NewClient() *Client {
	binaryPath := os.Getenv("RCLONE_BINARY_PATH")
	if binaryPath == "" {
		binaryPath = "rclone"
	}

	configPath := os.Getenv("RCLONE_CONFIG")

	return &Client{
		binaryPath: binaryPath,
		configPath: configPath,
	}
}

// NewClientWithPath creates a new rclone client with a specific binary path.
func NewClientWithPath(binaryPath string) *Client {
	return &Client{
		binaryPath: binaryPath,
		configPath: os.Getenv("RCLONE_CONFIG"),
	}
}

// IsInstalled checks if rclone is available in the system PATH.
func (c *Client) IsInstalled() bool {
	// If binaryPath is just "rclone", check PATH
	if c.binaryPath == "rclone" {
		_, err := exec.LookPath("rclone")
		return err == nil
	}
	// Otherwise check if the specific path exists and is executable
	_, err := exec.LookPath(c.binaryPath)
	return err == nil
}

// IsInstalled checks if rclone is available in the system PATH.
// This is a package-level convenience function.
func IsInstalled() bool {
	_, err := exec.LookPath("rclone")
	return err == nil
}

// GetVersion returns the installed rclone version.
func (c *Client) GetVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.binaryPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get rclone version: %w", err)
	}

	// Output format: "rclone v1.62.0\n..."
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		// Extract version from "rclone v1.62.0" or just return the line
		return firstLine, nil
	}

	return "", fmt.Errorf("could not parse rclone version from output")
}

// runCommand is a helper to run rclone commands with context and config.
func (c *Client) runCommand(ctx context.Context, args ...string) ([]byte, error) {
	if c.configPath != "" {
		args = append([]string{"--config", c.configPath}, args...)
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	return cmd.Output()
}
