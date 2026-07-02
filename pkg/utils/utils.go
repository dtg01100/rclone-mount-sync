// Package utils provides utility functions for the rclone-mount-sync application.
package utils

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Function variables for dependency injection (used in tests)
var (
	osUserHomeDir = os.UserHomeDir
	userLookup    = user.Lookup
	osMkdirAll    = os.MkdirAll
)

// ExpandHome expands ~ to the user's home directory in a path.
// Supports ~, ~/path, and ~username/path.
// If expansion fails, the original path is returned.
func ExpandHome(path string) string {
	if path == "" {
		return path
	}

	if path == "~" {
		homeDir, err := osUserHomeDir()
		if err != nil {
			return path
		}
		return homeDir
	}

	if strings.HasPrefix(path, "~/") {
		homeDir, err := osUserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}

	if strings.HasPrefix(path, "~") {
		end := strings.Index(path, "/")
		username := path[1:]
		rest := ""
		if end != -1 {
			username = path[1:end]
			rest = path[end:]
		}
		if username == "" {
			return path
		}
		u, err := userLookup(username)
		if err != nil {
			return path
		}
		if rest == "" {
			return u.HomeDir
		}
		return filepath.Join(u.HomeDir, rest)
	}

	return path
}

// ExpandPath expands ~ to the user's home directory in a path.
// Returns an error if the home directory cannot be determined.
func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, path[2:]), nil
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
// It creates all necessary parent directories with mode 0700 so the
// directory and its contents are not readable by other users; the
// application stores credentials (rclone remote config references) in
// directories created by this function.
func EnsureDir(path string) error {
	if err := osMkdirAll(path, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	return nil
}

// GetHomeDir returns the current user's home directory.
func GetHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homeDir, nil
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

// GenerateID generates a short unique identifier (8 characters, alphanumeric).
func GenerateID() string {
	return uuid.New().String()[:8]
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

// WarningSink is the destination for NoteWarning output. Defaults to
// os.Stderr; tests may swap it to capture or discard messages.
var WarningSink io.Writer = os.Stderr

// NoteWarning writes a single warning line to WarningSink in the
// canonical "Warning: <message>" format used throughout the codebase.
// Centralising the format here makes warnings easy to grep, easy to
// suppress in tests, and easy to redirect to a log file in the future.
func NoteWarning(format string, args ...any) {
	_, _ = fmt.Fprintf(WarningSink, "Warning: "+format+"\n", args...)
}

// ErrorSink is the destination for NoteError output. Defaults to
// os.Stderr; tests may swap it to capture or discard messages.
var ErrorSink io.Writer = os.Stderr

// NoteError writes a single error line to ErrorSink in the canonical
// "Error: <message>" format. It is the error-channel counterpart to
// NoteWarning: the two helpers exist in parallel so a future
// log-file redirection can route warnings to one file and errors to
// another without re-plumbing every call site.
func NoteError(format string, args ...any) {
	_, _ = fmt.Fprintf(ErrorSink, "Error: "+format+"\n", args...)
}
