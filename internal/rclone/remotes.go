package rclone

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// Remote represents an rclone remote configuration.
type Remote struct {
	Name     string // Remote name (e.g., "gdrive")
	Type     string // Remote type (e.g., "drive", "s3", "dropbox")
	RootPath string // Root path for the remote (e.g., "gdrive:")
}

// ValidateRemoteName returns nil if name is a syntactically valid rclone
// remote name. rclone's own rules are intentionally permissive (any non-empty
// name is accepted), but this app uses `remote + ":" + path` to build the
// final path string, so a name containing `:` would let a caller smuggle
// a different remote. We restrict the set to a safe character class to
// prevent that and to avoid surprises in unit-file generation.
func ValidateRemoteName(name string) error {
	if name == "" {
		return fmt.Errorf("remote name must not be empty")
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("remote name %q must not start with -", name)
	}
	if strings.Contains(name, ":") {
		return fmt.Errorf("remote name %q must not contain a colon (the app adds the separator)", name)
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-' || r == '.' || r == '@' || r == '+' || r == '!':
		default:
			return fmt.Errorf("remote name %q contains illegal character %q", name, r)
		}
	}
	return nil
}

// RemotePath represents a path on an rclone remote.
type RemotePath struct {
	Remote string // Remote name (e.g., "gdrive")
	Path   string // Path on the remote (e.g., "/Photos")
}

// ListRemotes returns a list of configured rclone remotes.
//
// Performance: types are fetched in a single \`rclone config show\` call
// rather than one \`config show <name>\` per remote. With 50 remotes the
// old N+1 pattern blocked for 30-60 seconds on cold cache; the single
// call returns in <100ms.
func (c *Client) ListRemotes(ctx context.Context) ([]Remote, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	output, err := c.runCommandWithRetry(ctx, "listremotes")
	if err != nil {
		return nil, fmt.Errorf("failed to list remotes: %w", err)
	}

	// Get all types in a single \`config show\` call. If this fails
	// (e.g. rclone version too old to support \`config show\` without a
	// name, or corrupted config), fall back to "unknown" for every
	// remote so the list still loads.
	types, typeErr := c.GetAllRemoteTypes(ctx)
	if typeErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get remote types in bulk: %v\n", typeErr)
		types = nil
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

		remoteType := "unknown"
		if t, ok := types[name]; ok {
			remoteType = t
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

// GetAllRemoteTypes returns a name->type map for every configured remote in
// a single \`rclone config show\` subprocess call. This replaces the
// previous N+1 pattern in ListRemotes that called \`config show <name>\`
// for each remote (with 50 remotes the pre-flight check blocked for
// 30-60 seconds on cold cache; the single call returns in <100ms).
//
// Output format from rclone config show (no remote name):
//
//	; --------------------
//	; gdrive
//	; --------------------
//	[gdrive]
//	type = drive
//	client_id =
//	...
//
//	; --------------------
//	; dropbox
//	; --------------------
//	[dropbox]
//	type = dropbox
//	...
//
// We track the current section name ([xxx]) and capture the first
// "type = xxx" line within it. Remotes without a type (corrupt config)
// are omitted from the result; callers must treat absence as "unknown".
func (c *Client) GetAllRemoteTypes(ctx context.Context) (map[string]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	output, err := c.runCommandWithRetry(ctx, "config", "show")
	if err != nil {
		return nil, fmt.Errorf("failed to get remote types: %w", err)
	}

	result := make(map[string]string)
	var current string
	for _, raw := range strings.Split(string(output), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "["):
			// New section. Strip brackets, then drop anything from
			// the first inline comment marker ('#' or ';') onwards so
			// hand-edited lines like "[gdrive] # primary" still
			// resolve to section "gdrive". Sections without a name
			// (e.g. "[]") leave current empty and the type line is
			// skipped.
			stripped := strings.TrimPrefix(line, "[")
			if idx := strings.IndexAny(stripped, "#;"); idx >= 0 {
				stripped = stripped[:idx]
			}
			name := strings.TrimSuffix(strings.TrimSpace(stripped), "]")
			current = name
		case current != "" && strings.HasPrefix(line, "type"):
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[0]) == "type" {
				typeVal := strings.TrimSpace(parts[1])
				if typeVal != "" {
					result[current] = typeVal
				}
			}
		}
	}

	return result, nil
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
