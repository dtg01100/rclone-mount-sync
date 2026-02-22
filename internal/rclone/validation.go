// Package rclone provides validation and pre-flight checks for rclone environment.
package rclone

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// CheckResult represents the result of a single pre-flight check.
type CheckResult struct {
	Name        string // Name of the check
	Passed      bool   // Whether the check passed
	Message     string // Error or success message
	Suggestion  string // User-friendly suggestion for fixing the issue
	IsCritical  bool   // If true, the application cannot continue without this check passing
}

// PreflightChecks runs all pre-flight validation checks and returns the results.
// It uses the provided RcloneClient for rclone-specific checks.
func PreflightChecks(client *Client) []CheckResult {
	var results []CheckResult

	// 1. Check rclone binary exists
	results = append(results, checkRcloneBinary(client))

	// If rclone binary doesn't exist, we can't run other rclone checks
	if !results[0].Passed {
		// Add placeholder failures for rclone-dependent checks
		results = append(results, CheckResult{
			Name:       "Rclone Version",
			Passed:     false,
			Message:    "Skipped: rclone binary not found",
			Suggestion: "Install rclone first to check version",
			IsCritical: true,
		})
		results = append(results, CheckResult{
			Name:       "Configured Remotes",
			Passed:     false,
			Message:    "Skipped: rclone binary not found",
			Suggestion: "Install rclone first to check configured remotes",
			IsCritical: true,
		})
	} else {
		// 2. Check rclone version
		results = append(results, checkRcloneVersion(client))

		// 3. Check configured remotes
		results = append(results, checkConfiguredRemotes(client))
	}

	// 4. Check systemd user session
	results = append(results, checkSystemdUserSession())

	// 5. Check fusermount availability
	results = append(results, checkFusermount())

	return results
}

// checkRcloneBinary verifies that the rclone binary exists at the configured path or in PATH.
func checkRcloneBinary(client *Client) CheckResult {
	result := CheckResult{
		Name:       "Rclone Binary",
		IsCritical: true,
	}

	if client == nil {
		result.Passed = false
		result.Message = "Rclone client is not initialized"
		result.Suggestion = "Ensure the rclone client is properly created before running pre-flight checks"
		return result
	}

	if client.IsInstalled() {
		binaryPath := client.binaryPath
		if binaryPath == "rclone" {
			// Find the actual path
			if path, err := exec.LookPath("rclone"); err == nil {
				binaryPath = path
			}
		}
		result.Passed = true
		result.Message = fmt.Sprintf("Found rclone binary at: %s", binaryPath)
		return result
	}

	// Binary not found
	binaryPath := client.binaryPath
	if binaryPath == "rclone" {
		result.Message = "rclone binary not found in PATH"
		result.Suggestion = "Install rclone using your package manager (e.g., 'sudo apt install rclone') or download from https://rclone.org/install/"
	} else {
		result.Message = fmt.Sprintf("rclone binary not found at configured path: %s", binaryPath)
		result.Suggestion = "Verify the rclone_binary_path in your settings or install rclone"
	}

	return result
}

// checkRcloneVersion verifies that rclone version is at least 1.60.0.
func checkRcloneVersion(client *Client) CheckResult {
	result := CheckResult{
		Name:       "Rclone Version",
		IsCritical: true,
	}

	versionStr, err := client.GetVersion()
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Failed to get rclone version: %v", err)
		result.Suggestion = "Ensure rclone is properly installed and accessible"
		return result
	}

	// Parse version from output like "rclone v1.62.0" or just "v1.62.0"
	version, err := parseVersion(versionStr)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Failed to parse rclone version from '%s': %v", versionStr, err)
		result.Suggestion = "Ensure you have a valid rclone installation"
		return result
	}

	// Check minimum version (1.60.0)
	minVersion := versionTuple{1, 60, 0}
	if compareVersions(version, minVersion) >= 0 {
		result.Passed = true
		result.Message = fmt.Sprintf("Rclone version %d.%d.%d meets minimum requirement (1.60.0)", version.major, version.minor, version.patch)
	} else {
		result.Passed = false
		result.Message = fmt.Sprintf("Rclone version %d.%d.%d is below minimum required version 1.60.0", version.major, version.minor, version.patch)
		result.Suggestion = "Upgrade rclone to version 1.60.0 or later from https://rclone.org/install/"
	}

	return result
}

// checkConfiguredRemotes verifies that at least one rclone remote is configured.
func checkConfiguredRemotes(client *Client) CheckResult {
	result := CheckResult{
		Name:       "Configured Remotes",
		IsCritical: true,
	}

	// Use a context with timeout for the remote listing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a channel to receive the result
	type remoteResult struct {
		remotes []Remote
		err     error
	}
	resultChan := make(chan remoteResult, 1)

	go func() {
		remotes, err := client.ListRemotes()
		resultChan <- remoteResult{remotes: remotes, err: err}
	}()

	select {
	case <-ctx.Done():
		result.Passed = false
		result.Message = "Timeout while listing rclone remotes"
		result.Suggestion = "Check your rclone configuration and network connectivity"
		return result
	case res := <-resultChan:
		if res.err != nil {
			result.Passed = false
			result.Message = fmt.Sprintf("Failed to list rclone remotes: %v", res.err)
			result.Suggestion = "Ensure rclone configuration is accessible. Try running 'rclone listremotes' manually"
			return result
		}

		if len(res.remotes) == 0 {
			result.Passed = false
			result.Message = "No rclone remotes are configured"
			result.Suggestion = "Run 'rclone config' to set up a remote storage provider first"
			return result
		}

		result.Passed = true
		result.Message = fmt.Sprintf("Found %d configured remote(s): %s", len(res.remotes), formatRemoteNames(res.remotes))
		return result
	}
}

// checkSystemdUserSession verifies that systemd user session is available.
func checkSystemdUserSession() CheckResult {
	result := CheckResult{
		Name:       "Systemd User Session",
		IsCritical: true,
	}

	// Check if systemctl exists
	systemctlPath, err := exec.LookPath("systemctl")
	if err != nil {
		result.Passed = false
		result.Message = "systemctl command not found"
		result.Suggestion = "This application requires systemd. Install systemd or use a systemd-based Linux distribution"
		return result
	}

	// Check if user session is available by running 'systemctl --user status'
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, systemctlPath, "--user", "is-active", "default.target")
	output, err := cmd.CombinedOutput()

	// The user session might report "inactive" or "active" - both mean systemd is working
	// An error with "Failed to connect to bus" indicates the user session is not available
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		// Check for specific error messages
		if strings.Contains(outputStr, "Failed to connect to bus") ||
			strings.Contains(outputStr, "No such file or directory") ||
			strings.Contains(outputStr, "Connection refused") {
			result.Passed = false
			result.Message = "Systemd user session is not available"
			result.Suggestion = "Ensure your system is running with a systemd user session. You may need to log in again or start the user session with 'systemctl --user start default.target'"
			return result
		}

		// Some other error - but systemctl exists, so we'll consider it a warning
		// The user session might still work for our purposes
		result.Passed = true
		result.Message = fmt.Sprintf("Systemd user session detected (status: %s)", outputStr)
		return result
	}

	// Success - systemd user session is active
	result.Passed = true
	if outputStr == "active" {
		result.Message = "Systemd user session is active and available"
	} else {
		result.Message = fmt.Sprintf("Systemd user session is available (status: %s)", outputStr)
	}

	return result
}

// checkFusermount verifies that fusermount or fusermount3 is available for mounting.
func checkFusermount() CheckResult {
	result := CheckResult{
		Name:       "Fusermount",
		IsCritical: false, // Not critical - sync jobs can work without it
	}

	// Try fusermount3 first (newer)
	if path, err := exec.LookPath("fusermount3"); err == nil {
		result.Passed = true
		result.Message = fmt.Sprintf("Found fusermount3 at: %s", path)
		return result
	}

	// Try fusermount (older version)
	if path, err := exec.LookPath("fusermount"); err == nil {
		result.Passed = true
		result.Message = fmt.Sprintf("Found fusermount at: %s", path)
		return result
	}

	// Neither found
	result.Passed = false
	result.Message = "Neither fusermount nor fusermount3 found"
	result.Suggestion = "Install FUSE to enable mounting: 'sudo apt install fuse3' or 'sudo apt install fuse'. Note: Sync jobs will still work without FUSE, but mount functionality will be unavailable."

	return result
}

// versionTuple represents a semantic version.
type versionTuple struct {
	major, minor, patch int
}

// parseVersion extracts version numbers from a version string.
// Handles formats like "rclone v1.62.0", "v1.62.0", "1.62.0", etc.
func parseVersion(versionStr string) (versionTuple, error) {
	// Regular expression to find version numbers
	re := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)

	if len(matches) != 4 {
		return versionTuple{}, fmt.Errorf("could not find version pattern in %q", versionStr)
	}

	var v versionTuple
	if _, err := fmt.Sscanf(matches[0], "%d.%d.%d", &v.major, &v.minor, &v.patch); err != nil {
		return versionTuple{}, fmt.Errorf("failed to parse version numbers: %w", err)
	}

	return v, nil
}

// compareVersions compares two version tuples.
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func compareVersions(a, b versionTuple) int {
	if a.major != b.major {
		if a.major < b.major {
			return -1
		}
		return 1
	}
	if a.minor != b.minor {
		if a.minor < b.minor {
			return -1
		}
		return 1
	}
	if a.patch != b.patch {
		if a.patch < b.patch {
			return -1
		}
		return 1
	}
	return 0
}

// formatRemoteNames creates a comma-separated list of remote names.
func formatRemoteNames(remotes []Remote) string {
	if len(remotes) == 0 {
		return ""
	}

	names := make([]string, len(remotes))
	for i, r := range remotes {
		names[i] = r.Name
	}

	if len(names) > 5 {
		return strings.Join(names[:5], ", ") + fmt.Sprintf(" (and %d more)", len(names)-5)
	}
	return strings.Join(names, ", ")
}

// HasCriticalFailure returns true if any check result has a critical failure.
func HasCriticalFailure(results []CheckResult) bool {
	for _, r := range results {
		if !r.Passed && r.IsCritical {
			return true
		}
	}
	return false
}

// AllPassed returns true if all checks passed.
func AllPassed(results []CheckResult) bool {
	for _, r := range results {
		if !r.Passed {
			return false
		}
	}
	return true
}

// FormatResults formats the check results for display.
func FormatResults(results []CheckResult) string {
	var sb strings.Builder

	sb.WriteString("Pre-flight Check Results:\n")
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	for _, r := range results {
		status := "✓ PASS"
		if !r.Passed {
			if r.IsCritical {
				status = "✗ FAIL (critical)"
			} else {
				status = "⚠ FAIL (optional)"
			}
		}

		sb.WriteString(fmt.Sprintf("\n[%s] %s\n", status, r.Name))
		sb.WriteString(fmt.Sprintf("  %s\n", r.Message))
		if r.Suggestion != "" {
			sb.WriteString(fmt.Sprintf("  Suggestion: %s\n", r.Suggestion))
		}
	}

	return sb.String()
}
