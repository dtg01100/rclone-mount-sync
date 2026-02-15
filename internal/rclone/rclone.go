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

// Remote represents an rclone remote configuration.
type Remote struct {
	Name     string // Remote name (e.g., "gdrive")
	Type     string // Remote type (e.g., "drive", "s3", "dropbox")
	RootPath string // Root path for the remote (e.g., "gdrive:")
}

// RemotePath represents a path on an rclone remote.
type RemotePath struct {
	Remote string // Remote name (e.g., "gdrive")
	Path   string // Path on the remote (e.g., "/Photos")
}

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

// SetConfigPath sets a custom rclone configuration file path.
func (c *Client) SetConfigPath(path string) {
	c.configPath = path
}

// GetConfigPath returns the path to the rclone configuration file.
// It returns the custom path if set, otherwise queries rclone for the config path.
func (c *Client) GetConfigPath() (string, error) {
	if c.configPath != "" {
		return c.configPath, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{"config", "file"}
	if c.configPath != "" {
		args = append([]string{"--config", c.configPath}, args...)
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get rclone config path: %w", err)
	}

	// Output format: "Configuration file is stored at:\n/path/to/config/rclone.conf\n"
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && i == len(lines)-1 {
			return line, nil
		}
	}

	return "", fmt.Errorf("could not parse rclone config path from output")
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

// ListRemotes returns a list of configured rclone remotes.
func (c *Client) ListRemotes() ([]Remote, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{"listremotes"}
	if c.configPath != "" {
		args = append([]string{"--config", c.configPath}, args...)
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to list remotes: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to list remotes: %w", err)
	}

	// Output format: one remote per line with trailing colon
	// gdrive:
	// dropbox:
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var remotes []Remote

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove trailing colon to get remote name
		name := strings.TrimSuffix(line, ":")
		if name == "" {
			continue
		}

		// Get the remote type
		remoteType, err := c.GetRemoteType(name)
		if err != nil {
			// Log warning but continue - remote might still be usable
			remoteType = "unknown"
		}

		remotes = append(remotes, Remote{
			Name:     name,
			Type:     remoteType,
			RootPath: line, // Keep the colon for root path
		})
	}

	return remotes, nil
}

// GetRemoteType returns the type of a specific remote (e.g., "drive", "s3", "dropbox").
func (c *Client) GetRemoteType(remote string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{"config", "show", remote}
	if c.configPath != "" {
		args = append([]string{"--config", c.configPath}, args...)
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("failed to get remote type: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to get remote type: %w", err)
	}

	// Output format:
	// [gdrive]
	// type = drive
	// ...
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "type = ") {
			return strings.TrimPrefix(line, "type = "), nil
		}
		if strings.HasPrefix(line, "type=") {
			return strings.TrimPrefix(line, "type="), nil
		}
	}

	return "", fmt.Errorf("could not find type for remote %s", remote)
}

// ListRemotePath lists the contents of a path on an rclone remote.
// Returns a slice of entry names (directories and files).
func (c *Client) ListRemotePath(remote, path string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Build the remote path (e.g., "gdrive:/Photos")
	remotePath := remote + ":" + path

	args := []string{"lsf", remotePath}
	if c.configPath != "" {
		args = append([]string{"--config", c.configPath}, args...)
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to list remote path: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to list remote path: %w", err)
	}

	// Output format: one entry per line, directories end with "/"
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var entries []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			entries = append(entries, line)
		}
	}

	return entries, nil
}

// ValidateRemote checks if a remote exists in the rclone configuration.
func (c *Client) ValidateRemote(remote string) error {
	remotes, err := c.ListRemotes()
	if err != nil {
		return fmt.Errorf("failed to validate remote: %w", err)
	}

	for _, r := range remotes {
		if r.Name == remote {
			return nil
		}
	}

	return fmt.Errorf("remote %q not found in rclone configuration", remote)
}

// TestRemoteAccess tests if a remote path is accessible.
// This performs a simple directory listing to verify connectivity.
func (c *Client) TestRemoteAccess(remote, path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build the remote path
	remotePath := remote + ":" + path

	args := []string{"lsf", remotePath, "--max-depth", "1"}
	if c.configPath != "" {
		args = append([]string{"--config", c.configPath}, args...)
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to access remote path %q: %s", remotePath, string(output))
	}

	return nil
}

// runCommand is a helper to run rclone commands with context and config.
func (c *Client) runCommand(ctx context.Context, args ...string) ([]byte, error) {
	if c.configPath != "" {
		args = append([]string{"--config", c.configPath}, args...)
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	return cmd.Output()
}
