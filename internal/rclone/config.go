package rclone

import (
	"context"
	"fmt"
	"strings"
	"time"
)

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

	output, err := doRetryBytes(ctx, c.retryConfig, func() ([]byte, error) {
		return c.runCommand(ctx, args...)
	})
	if err != nil {
		return "", fmt.Errorf("failed to get rclone config path: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && i == len(lines)-1 {
			return line, nil
		}
	}

	return "", fmt.Errorf("could not parse rclone config path from output")
}
