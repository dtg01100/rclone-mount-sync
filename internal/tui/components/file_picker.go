package components

import (
	"os"
	"path/filepath"
	"strings"
)

// EnhancedFilePicker is a component for picking files and directories
type EnhancedFilePicker struct {
	currentPath string
	suggestions []string
}

// NewFilePicker creates a new EnhancedFilePicker instance
func NewFilePicker() *EnhancedFilePicker {
	return &EnhancedFilePicker{
		currentPath: "",
		suggestions: []string{},
	}
}

// ValidateFilePath validates if a file path is valid and exists
func (fp *EnhancedFilePicker) ValidateFilePath(path string) bool {
	if path == "" {
		return false
	}

	// Expand home directory if the path starts with ~
	expandedPath := ExpandHome(path)

	// Check if the file or directory exists
	_, err := os.Stat(expandedPath)
	return err == nil
}

// ValidateFilePathStatic is a static function version of ValidateFilePath
func ValidateFilePathStatic(path string) bool {
	if path == "" {
		return false
	}

	// Expand home directory if the path starts with ~
	expandedPath := ExpandHome(path)

	// Check if the file or directory exists
	_, err := os.Stat(expandedPath)
	return err == nil
}

// PathExists checks if a path exists (file or directory)
func PathExists(path string) bool {
	if path == "" {
		return false
	}

	// Expand home directory if the path starts with ~
	expandedPath := ExpandHome(path)

	// Check if the path exists
	_, err := os.Stat(expandedPath)
	return err == nil
}

// ValidateFileExtension validates if a file has one of the allowed extensions
func (fp *EnhancedFilePicker) ValidateFileExtension(path string, allowedExtensions []string) bool {
	if len(allowedExtensions) == 0 {
		return true // If no extensions specified, allow any
	}

	ext := strings.ToLower(filepath.Ext(path))
	for _, allowedExt := range allowedExtensions {
		if strings.ToLower(allowedExt) == ext {
			return true
		}
	}
	return false
}

// GetFilesInDirectory returns a list of files in the specified directory
func (fp *EnhancedFilePicker) GetFilesInDirectory(dirPath string) ([]string, error) {
	expandedPath := ExpandHome(dirPath)

	entries, err := os.ReadDir(expandedPath)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// GetDirectoriesInDirectory returns a list of directories in the specified directory
func (fp *EnhancedFilePicker) GetDirectoriesInDirectory(dirPath string) ([]string, error) {
	expandedPath := ExpandHome(dirPath)

	entries, err := os.ReadDir(expandedPath)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}

	return dirs, nil
}

// SetCurrentPath sets the current path for the file picker
func (fp *EnhancedFilePicker) SetCurrentPath(path string) {
	fp.currentPath = path
}

// GetCurrentPath returns the current path for the file picker
func (fp *EnhancedFilePicker) GetCurrentPath() string {
	return fp.currentPath
}

// GetSuggestions returns path suggestions based on recent paths and common directories
func (fp *EnhancedFilePicker) GetSuggestions(recentPaths []string) []string {
	return GetPathSuggestions(recentPaths, []string{})
}
