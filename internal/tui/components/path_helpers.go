package components

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func GetCommonDirectories() []string {
	var dirs []string

	if _, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, "~/")
		dirs = append(dirs, "~/mnt/")
		dirs = append(dirs, "~/mounts/")
	}

	dirs = append(dirs, "/mnt/")
	dirs = append(dirs, "/media/")

	return dirs
}

func GetPathSuggestions(recentPaths []string, existingPaths []string) []string {
	seen := make(map[string]bool)
	var suggestions []string

	maxRecent := 5
	for i, path := range recentPaths {
		if i >= maxRecent {
			break
		}
		expanded := ExpandHome(path)
		if !seen[expanded] {
			seen[expanded] = true
			suggestions = append(suggestions, path)
		}
	}

	for _, path := range existingPaths {
		expanded := ExpandHome(path)
		if !seen[expanded] {
			seen[expanded] = true
			suggestions = append(suggestions, path)
		}
	}

	for _, dir := range GetCommonDirectories() {
		expanded := ExpandHome(dir)
		if !seen[expanded] {
			seen[expanded] = true
			suggestions = append(suggestions, dir)
		}
	}

	return suggestions
}

func ExpandHome(path string) string {
	if path == "" {
		return path
	}

	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}

	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return homeDir
	}

	if strings.HasPrefix(path, "~") {
		end := strings.Index(path, "/")
		if end == -1 {
			end = len(path)
		}
		username := path[1:end]
		u, err := user.Lookup(username)
		if err != nil {
			return path
		}
		rest := ""
		if end < len(path) {
			rest = path[end:]
		}
		return filepath.Join(u.HomeDir, rest)
	}

	return path
}

func ContractHome(path string) string {
	if path == "" {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == homeDir {
		return "~"
	}

	if strings.HasPrefix(path, homeDir+string(filepath.Separator)) {
		return "~" + path[len(homeDir):]
	}

	return path
}

// GetParentDirectory returns the parent directory of the given path.
// If the path is already at root, it returns the root path.
func GetParentDirectory(path string) string {
	if path == "" {
		return "/"
	}

	// Expand the path first
	expandedPath := ExpandHome(path)

	// Get the parent directory
	parent := filepath.Dir(expandedPath)

	// filepath.Dir returns "." for empty paths, handle that
	if parent == "." {
		return "/"
	}

	return parent
}

// GetBreadcrumbSegments splits a path into segments for breadcrumb navigation.
// It returns a slice of path segment names suitable for display.
func GetBreadcrumbSegments(path string) []string {
	if path == "" {
		return []string{}
	}

	// Expand the path first
	expandedPath := ExpandHome(path)

	// Handle home directory specially
	homeDir, _ := os.UserHomeDir()
	var segments []string
	
	if homeDir != "" && strings.HasPrefix(expandedPath, homeDir) {
		segments = append(segments, "~")
		remaining := strings.TrimPrefix(expandedPath, homeDir)
		remaining = strings.Trim(remaining, string(filepath.Separator))
		if remaining != "" {
			parts := strings.Split(remaining, string(filepath.Separator))
			for _, part := range parts {
				if part != "" {
					segments = append(segments, part)
				}
			}
		}
		return segments
	}

	// Handle absolute paths
	if filepath.IsAbs(expandedPath) {
		// Start with root indicator
		parts := strings.Split(expandedPath, string(filepath.Separator))
		for _, part := range parts {
			if part != "" {
				segments = append(segments, part)
			}
		}
		// If path is just root, return empty (breadcrumb will show just home icon)
		if len(segments) == 0 && expandedPath == "/" {
			return []string{}
		}
		return segments
	}

	// Relative path - just split by separator
	parts := strings.Split(expandedPath, string(filepath.Separator))
	for _, part := range parts {
		if part != "" {
			segments = append(segments, part)
		}
	}

	return segments
}

// PathExists checks if a path exists on the filesystem.
func PathExists(path string) bool {
	_, err := os.Stat(ExpandHome(path))
	return err == nil
}

// IsDirectory checks if a path is a directory.
func IsDirectory(path string) bool {
	info, err := os.Stat(ExpandHome(path))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetDisplayPath returns a shortened display-friendly version of a path.
// It contracts the home directory and truncates if necessary.
func GetDisplayPath(path string, maxLen int) string {
	if path == "" {
		return ""
	}

	// Contract home directory first
	displayPath := ContractHome(ExpandHome(path))

	// Truncate if necessary
	if maxLen > 0 && len(displayPath) > maxLen {
		// Try to keep the end of the path (more relevant)
		if maxLen <= 3 {
			return displayPath[:maxLen]
		}
		return "..." + displayPath[len(displayPath)-maxLen+3:]
	}

	return displayPath
}
