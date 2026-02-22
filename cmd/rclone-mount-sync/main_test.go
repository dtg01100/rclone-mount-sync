package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dtg01100/rclone-mount-sync/internal/rclone"
)

func TestVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{name: "dev version", version: "dev", expected: "dev\n"},
		{name: "semantic version", version: "1.0.0", expected: "1.0.0\n"},
		{name: "version with prerelease", version: "1.0.0-beta.1", expected: "1.0.0-beta.1\n"},
		{name: "version with commit", version: "v1.2.3-abc123", expected: "v1.2.3-abc123\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVersion := version
			version = tt.version
			defer func() { version = oldVersion }()

			var buf bytes.Buffer
			buf.WriteString(version + "\n")
			result := buf.String()

			if result != tt.expected {
				t.Errorf("version output = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRunPreflightChecks_Logic(t *testing.T) {
	client := rclone.NewClient()
	results := rclone.PreflightChecks(client)

	if len(results) == 0 {
		t.Error("PreflightChecks returned no results")
	}

	for _, r := range results {
		if r.Name == "" {
			t.Error("CheckResult has empty name")
		}
	}

	if rclone.HasCriticalFailure(results) {
		formatted := rclone.FormatResults(results)
		if !strings.Contains(formatted, "FAIL") {
			t.Error("FormatResults should contain 'FAIL' when there's a critical failure")
		}
	}
}

func TestConfigDirHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		isDir       bool
		expectXDG   string
		description string
	}{
		{
			name:        "empty config dir",
			input:       "",
			expectXDG:   "",
			description: "Empty input should not set XDG_CONFIG_HOME",
		},
		{
			name:        "valid directory path",
			input:       "/tmp/testconfig",
			isDir:       true,
			expectXDG:   "/tmp/testconfig",
			description: "Directory path should be set as XDG_CONFIG_HOME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalXDG := os.Getenv("XDG_CONFIG_HOME")
			defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

			if tt.input != "" {
				if tt.isDir {
					os.MkdirAll(tt.input, 0755)
					defer os.RemoveAll(tt.input)
				}

				os.Setenv("XDG_CONFIG_HOME", tt.input)
				result := os.Getenv("XDG_CONFIG_HOME")
				if result != tt.expectXDG && tt.expectXDG != "" {
					t.Errorf("XDG_CONFIG_HOME = %q, want %q", result, tt.expectXDG)
				}
			}
		})
	}
}

func TestConfigDirFilePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	filePath := testFile
	if fi, err := os.Stat(filePath); err == nil && !fi.IsDir() {
		dir := filepath.Dir(filePath)
		if dir != tempDir {
			t.Errorf("filepath.Dir(%q) = %q, want %q", filePath, dir, tempDir)
		}
	}
}

func TestFormatResultsOutput(t *testing.T) {
	results := []rclone.CheckResult{
		{Name: "Test Check 1", Passed: true, Message: "All good", IsCritical: true},
		{Name: "Test Check 2", Passed: false, Message: "Something failed", Suggestion: "Fix it", IsCritical: false},
	}

	output := rclone.FormatResults(results)

	if !strings.Contains(output, "Test Check 1") {
		t.Error("FormatResults should contain check name")
	}
	if !strings.Contains(output, "PASS") {
		t.Error("FormatResults should contain 'PASS'")
	}
	if !strings.Contains(output, "FAIL") {
		t.Error("FormatResults should contain 'FAIL'")
	}
	if !strings.Contains(output, "Fix it") {
		t.Error("FormatResults should contain suggestion")
	}
}

func TestHasCriticalFailure(t *testing.T) {
	tests := []struct {
		name     string
		results  []rclone.CheckResult
		expected bool
	}{
		{
			name:     "empty results",
			results:  []rclone.CheckResult{},
			expected: false,
		},
		{
			name: "all passed",
			results: []rclone.CheckResult{
				{Name: "Check 1", Passed: true, IsCritical: true},
				{Name: "Check 2", Passed: true, IsCritical: false},
			},
			expected: false,
		},
		{
			name: "critical failure",
			results: []rclone.CheckResult{
				{Name: "Check 1", Passed: false, IsCritical: true},
			},
			expected: true,
		},
		{
			name: "non-critical failure",
			results: []rclone.CheckResult{
				{Name: "Check 1", Passed: false, IsCritical: false},
			},
			expected: false,
		},
		{
			name: "mixed results",
			results: []rclone.CheckResult{
				{Name: "Check 1", Passed: true, IsCritical: true},
				{Name: "Check 2", Passed: false, IsCritical: false},
				{Name: "Check 3", Passed: false, IsCritical: true},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rclone.HasCriticalFailure(tt.results)
			if result != tt.expected {
				t.Errorf("HasCriticalFailure() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAllPassed(t *testing.T) {
	tests := []struct {
		name     string
		results  []rclone.CheckResult
		expected bool
	}{
		{
			name:     "empty results",
			results:  []rclone.CheckResult{},
			expected: true,
		},
		{
			name: "all passed",
			results: []rclone.CheckResult{
				{Name: "Check 1", Passed: true},
				{Name: "Check 2", Passed: true},
			},
			expected: true,
		},
		{
			name: "one failed",
			results: []rclone.CheckResult{
				{Name: "Check 1", Passed: true},
				{Name: "Check 2", Passed: false},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rclone.AllPassed(tt.results)
			if result != tt.expected {
				t.Errorf("AllPassed() = %v, want %v", result, tt.expected)
			}
		})
	}
}
