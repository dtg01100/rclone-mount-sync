package systemd

import (
	"fmt"
	"os"
	"path/filepath"
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

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}

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
		return ""
	}
	return filepath.Join(home, ".config", "rclone", "rclone.conf")
}

// getLogDir returns the directory for log files.
func getLogDir() (string, error) {
	// Use XDG_STATE_HOME if available, otherwise ~/.local/state
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir != "" {
		logDir := filepath.Join(stateDir, "rclone-mount-sync")
		if err := os.MkdirAll(logDir, 0750); err != nil { //nolint:gosec
			return "", err
		}
		return logDir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	logDir := filepath.Join(home, ".local", "state", "rclone-mount-sync")
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return "", err
	}
	return logDir, nil
}
