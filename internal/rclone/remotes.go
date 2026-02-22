package rclone

import (
	"context"
	"fmt"
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

// ListRemoteDirectories lists only directories in a path on an rclone remote.
// Returns clean directory names without trailing slashes.
func (c *Client) ListRemoteDirectories(remote, path string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	remotePath := remote + ":" + path

	args := []string{"lsf", remotePath, "--dirs-only"}
	if c.configPath != "" {
		args = append([]string{"--config", c.configPath}, args...)
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to list remote directories: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to list remote directories: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var directories []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			directories = append(directories, strings.TrimSuffix(line, "/"))
		}
	}

	return directories, nil
}

// ListRootDirectories lists directories at the root of a remote.
func (c *Client) ListRootDirectories(remote string) ([]string, error) {
	return c.ListRemoteDirectories(remote, "")
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
