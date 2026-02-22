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
