package systemd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// UserSystemdDir is the relative path to the user systemd directory.
const UserSystemdDir = ".config/systemd/user"

// GetUserSystemdPath returns the path to the user's systemd unit directory.
func GetUserSystemdPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, "systemd", "user"), nil
}

// sanitizeName sanitizes a name for use in a systemd unit filename.
func sanitizeName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace spaces and special characters with dashes
	reg := regexp.MustCompile(`[^a-z0-9_-]`)
	name = reg.ReplaceAllString(name, "-")

	// Remove consecutive dashes
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	// Trim leading and trailing dashes
	name = strings.Trim(name, "-")

	return name
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// getRcloneConfigPath returns the path to the rclone config file.
func getRcloneConfigPath() string {
	// Check RCLONE_CONFIG environment variable
	if configPath := os.Getenv("RCLONE_CONFIG"); configPath != "" {
		return configPath
	}

	// Default location
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/home", os.Getenv("USER"), ".config", "rclone", "rclone.conf")
	}
	return filepath.Join(home, ".config", "rclone", "rclone.conf")
}

// getLogDir returns the directory for log files.
func getLogDir() (string, error) {
	// Use XDG_STATE_HOME if available, otherwise ~/.local/state
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir != "" {
		logDir := filepath.Join(stateDir, "rclone-mount-sync")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return "", err
		}
		return logDir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	logDir := filepath.Join(home, ".local", "state", "rclone-mount-sync")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", err
	}
	return logDir, nil
}
