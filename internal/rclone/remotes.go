package rclone

import (
	"context"
	"errors"
	"fmt"
	"log"
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
func (c *Client) ListRemotes(ctx context.Context) ([]Remote, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	output, err := c.runCommandWithRetry(ctx, "listremotes")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("failed to list remotes: %w", err)
		}
		return nil, fmt.Errorf("failed to list remotes: %w", err)
	}

	// Output format: one remote per line with trailing colon
	// gdrive:
	// dropbox:
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	remotes := make([]Remote, 0, len(lines))

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
		remoteType, err := c.GetRemoteType(ctx, name)
		if err != nil {
			// Log warning but continue - remote might still be usable
			log.Printf("Warning: failed to get remote type for %s: %v", name, err)
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
func (c *Client) GetRemoteType(ctx context.Context, remote string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	output, err := c.runCommandWithRetry(ctx, "config", "show", remote)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("failed to get remote type: %w", err)
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
		if strings.HasPrefix(line, "type") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[0]) == "type" {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "", fmt.Errorf("could not find type for remote %s", remote)
}

// ListRemotePath lists the contents of a path on an rclone remote.
// Returns a slice of entry names (directories and files).
func (c *Client) ListRemotePath(ctx context.Context, remote, path string) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	remotePath := remote + ":" + path

	output, err := c.runCommandWithRetry(ctx, "lsf", remotePath)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("failed to list remote path: %w", err)
		}
		return nil, fmt.Errorf("failed to list remote path: %w", err)
	}

	// Output format: one entry per line, directories end with "/"
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	entries := make([]string, 0, len(lines))

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
func (c *Client) ListRemoteDirectories(ctx context.Context, remote, path string) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	remotePath := remote + ":" + path

	output, err := c.runCommandWithRetry(ctx, "lsf", remotePath, "--dirs-only")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("failed to list remote directories: %w", err)
		}
		return nil, fmt.Errorf("failed to list remote directories: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	directories := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			directories = append(directories, strings.TrimSuffix(line, "/"))
		}
	}

	return directories, nil
}

// ListRootDirectories lists directories at the root of a remote.
func (c *Client) ListRootDirectories(ctx context.Context, remote string) ([]string, error) {
	return c.ListRemoteDirectories(ctx, remote, "")
}

// ValidateRemote checks if a remote exists in the rclone configuration.
func (c *Client) ValidateRemote(ctx context.Context, remote string) error {
	remotes, err := c.ListRemotes(ctx)
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
func (c *Client) TestRemoteAccess(ctx context.Context, remote, path string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	remotePath := remote + ":" + path

	_, err := c.runCommandWithRetry(ctx, "lsf", remotePath, "--max-depth", "1")
	if err != nil {
		return fmt.Errorf("failed to access remote path %q: %w", remotePath, err)
	}

	return nil
}
