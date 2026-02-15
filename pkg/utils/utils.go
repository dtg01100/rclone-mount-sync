// Package utils provides utility functions for the rclone-mount-sync application.
package utils

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// ExpandHome expands ~ to the user's home directory in a path.
// This is a convenience function that returns the expanded path without error.
// If expansion fails, the original path is returned.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		usr, err := user.Current()
		if err != nil {
			return path
		}
		return filepath.Join(usr.HomeDir, path[2:])
	}
	return path
}

// ExpandPath expands ~ to the user's home directory in a path.
// Returns an error if the home directory cannot be determined.
func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		return filepath.Join(usr.HomeDir, path[2:]), nil
	}
	return path, nil
}

// FileExists checks if a file exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// EnsureDir creates a directory if it doesn't exist.
// It creates all necessary parent directories with mode 0755.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

// GetHomeDir returns the current user's home directory.
func GetHomeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.HomeDir, nil
}

// GetConfigDir returns the user's config directory.
func GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return configDir, nil
}

// SanitizeName sanitizes a name for use in filenames and systemd unit names.
func SanitizeName(name string) string {
	// Replace spaces and special characters with dashes
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "-")
	result = strings.ReplaceAll(result, "_", "-")

	// Remove any characters that aren't alphanumeric or dashes
	var cleaned strings.Builder
	for _, r := range result {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			cleaned.WriteRune(r)
		}
	}

	return cleaned.String()
}

// ValidateMountPath validates that a path is suitable for mounting.
func ValidateMountPath(path string) error {
	expanded, err := ExpandPath(path)
	if err != nil {
		return err
	}

	// Check if path is absolute
	if !filepath.IsAbs(expanded) {
		return os.ErrInvalid
	}

	return nil
}
