// Package systemd provides functionality for managing systemd user services.
package systemd

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/dlafreniere/rclone-mount-sync/internal/models"
)

// Manager handles systemd user service operations.
type Manager struct {
	systemctlPath string
}

// NewManager creates a new systemd manager.
func NewManager() *Manager {
	systemctlPath, err := exec.LookPath("systemctl")
	if err != nil {
		// Return a manager with default path - operations will fail gracefully
		return &Manager{systemctlPath: "/usr/bin/systemctl"}
	}
	return &Manager{systemctlPath: systemctlPath}
}

// ServiceStatus represents the status of a systemd service.
type ServiceStatus struct {
	Name     string
	Active   bool
	State    string // active, inactive, failed, activating
	SubState string // running, dead, exited
	Enabled  bool
}

// IsSystemdAvailable checks if systemd is available on the system.
func (m *Manager) IsSystemdAvailable() bool {
	cmd := exec.Command(m.systemctlPath, "--user", "status")
	err := cmd.Run()
	return err == nil
}

// DaemonReload reloads the systemd daemon to pick up unit file changes.
func (m *Manager) DaemonReload() error {
	cmd := exec.Command(m.systemctlPath, "--user", "daemon-reload")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("daemon-reload failed: %w, output: %s", err, string(output))
	}
	return nil
}

// Enable enables a systemd user unit.
func (m *Manager) Enable(name string) error {
	cmd := exec.Command(m.systemctlPath, "--user", "enable", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("enable %s failed: %w, output: %s", name, err, string(output))
	}
	return nil
}

// Disable disables a systemd user unit.
func (m *Manager) Disable(name string) error {
	cmd := exec.Command(m.systemctlPath, "--user", "disable", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("disable %s failed: %w, output: %s", name, err, string(output))
	}
	return nil
}

// Start starts a systemd user unit.
func (m *Manager) Start(name string) error {
	cmd := exec.Command(m.systemctlPath, "--user", "start", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("start %s failed: %w, output: %s", name, err, string(output))
	}
	return nil
}

// Stop stops a systemd user unit.
func (m *Manager) Stop(name string) error {
	cmd := exec.Command(m.systemctlPath, "--user", "stop", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop %s failed: %w, output: %s", name, err, string(output))
	}
	return nil
}

// Restart restarts a systemd user unit.
func (m *Manager) Restart(name string) error {
	cmd := exec.Command(m.systemctlPath, "--user", "restart", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart %s failed: %w, output: %s", name, err, string(output))
	}
	return nil
}

// Status returns the status of a systemd user unit.
func (m *Manager) Status(name string) (*ServiceStatus, error) {
	status := &ServiceStatus{
		Name: name,
	}

	// Get active state
	cmd := exec.Command(m.systemctlPath, "--user", "show", name,
		"--property=ActiveState,SubState,LoadState")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get status for %s: %w", name, err)
	}

	// Parse output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		switch key {
		case "ActiveState":
			status.State = value
			status.Active = value == "active"
		case "SubState":
			status.SubState = value
		}
	}

	// Get enabled status
	enabled, err := m.IsEnabled(name)
	if err == nil {
		status.Enabled = enabled
	}

	return status, nil
}

// IsEnabled checks if a unit is enabled.
func (m *Manager) IsEnabled(name string) (bool, error) {
	cmd := exec.Command(m.systemctlPath, "--user", "is-enabled", name)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(output)) == "enabled", nil
}

// IsActive checks if a unit is currently active.
func (m *Manager) IsActive(name string) (bool, error) {
	cmd := exec.Command(m.systemctlPath, "--user", "is-active", name)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(output)) == "active", nil
}

// ListServices lists all rclone services (mounts and sync jobs).
func (m *Manager) ListServices() ([]ServiceStatus, error) {
	// List all rclone services
	cmd := exec.Command(m.systemctlPath, "--user", "list-unit-files",
		"--type=service", "--no-legend", "rclone-*.service")
	output, err := cmd.Output()
	if err != nil {
		// If no units found, return empty list
		return []ServiceStatus{}, nil
	}

	var services []ServiceStatus
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse line: "rclone-mount-name.service enabled"
		parts := strings.Fields(line)
		if len(parts) < 1 {
			continue
		}

		unitName := parts[0]
		name := strings.TrimSuffix(unitName, ".service")

		status := &ServiceStatus{
			Name:    name,
			Enabled: len(parts) > 1 && parts[1] == "enabled",
		}

		// Get current state
		if isActive, _ := m.IsActive(name); isActive {
			status.Active = true
			status.State = "active"
		} else {
			status.Active = false
			status.State = "inactive"
		}

		services = append(services, *status)
	}

	return services, nil
}

// GetLogs returns the last N lines of logs for a service.
func (m *Manager) GetLogs(name string, lines int) (string, error) {
	cmd := exec.Command(m.systemctlPath, "--user", "journalctl",
		"-u", name, "-n", strconv.Itoa(lines), "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get logs for %s: %w", name, err)
	}
	return string(output), nil
}

// GetDetailedStatus returns detailed status information for a service.
func (m *Manager) GetDetailedStatus(name string) (*models.ServiceStatus, error) {
	status := &models.ServiceStatus{
		Name: name,
	}

	// Determine type from name
	if strings.HasPrefix(name, "rclone-mount-") {
		status.Type = "mount"
	} else if strings.HasPrefix(name, "rclone-sync-") {
		status.Type = "sync"
	}

	// Get properties
	cmd := exec.Command(m.systemctlPath, "--user", "show", name,
		"--property=LoadState,ActiveState,SubState,MainPID,ExecMainStatus,ActiveEnterTimestamp,InactiveEnterTimestamp")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get detailed status for %s: %w", name, err)
	}

	// Parse output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		switch key {
		case "LoadState":
			status.LoadState = value
		case "ActiveState":
			status.ActiveState = value
		case "SubState":
			status.SubState = value
		case "MainPID":
			if pid, err := strconv.Atoi(value); err == nil {
				status.MainPID = pid
			}
		case "ExecMainStatus":
			if code, err := strconv.Atoi(value); err == nil {
				status.ExitCode = code
			}
		case "ActiveEnterTimestamp":
			if t, err := parseSystemdTimestamp(value); err == nil {
				status.ActivatedAt = t
			}
		case "InactiveEnterTimestamp":
			if t, err := parseSystemdTimestamp(value); err == nil {
				status.InactiveAt = t
			}
		}
	}

	// Get enabled status
	enabled, err := m.IsEnabled(name)
	if err == nil {
		status.Enabled = enabled
	}

	// For sync jobs, check timer status
	if status.Type == "sync" {
		timerName := strings.Replace(name, ".service", ".timer", 1)
		if isActive, _ := m.IsActive(timerName); isActive {
			status.TimerActive = true
		}

		// Get next run time from timer
		timerStatus, err := m.GetTimerNextRun(timerName)
		if err == nil && !timerStatus.IsZero() {
			status.NextRun = timerStatus
		}
	}

	return status, nil
}

// GetTimerNextRun returns the next run time for a timer.
func (m *Manager) GetTimerNextRun(timerName string) (time.Time, error) {
	cmd := exec.Command(m.systemctlPath, "--user", "show", timerName,
		"--property=NextElapseUSecMonotonic")
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get timer info for %s: %w", timerName, err)
	}

	// Parse output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "NextElapseUSecMonotonic=") {
			value := strings.TrimPrefix(line, "NextElapseUSecMonotonic=")
			value = strings.TrimSpace(value)
			if value == "" || value == "0" {
				continue
			}
			// Parse microseconds
			if micros, err := strconv.ParseInt(value, 10, 64); err == nil {
				return time.Now().Add(time.Duration(micros) * time.Microsecond), nil
			}
		}
	}

	return time.Time{}, nil
}

// StartTimer starts a systemd timer.
func (m *Manager) StartTimer(name string) error {
	// Ensure we're using the timer unit
	timerName := name
	if !strings.HasSuffix(timerName, ".timer") {
		timerName = timerName + ".timer"
	}
	return m.Start(timerName)
}

// StopTimer stops a systemd timer.
func (m *Manager) StopTimer(name string) error {
	// Ensure we're using the timer unit
	timerName := name
	if !strings.HasSuffix(timerName, ".timer") {
		timerName = timerName + ".timer"
	}
	return m.Stop(timerName)
}

// EnableTimer enables a systemd timer.
func (m *Manager) EnableTimer(name string) error {
	// Ensure we're using the timer unit
	timerName := name
	if !strings.HasSuffix(timerName, ".timer") {
		timerName = timerName + ".timer"
	}
	return m.Enable(timerName)
}

// DisableTimer disables a systemd timer.
func (m *Manager) DisableTimer(name string) error {
	// Ensure we're using the timer unit
	timerName := name
	if !strings.HasSuffix(timerName, ".timer") {
		timerName = timerName + ".timer"
	}
	return m.Disable(timerName)
}

// RunSyncNow triggers an immediate sync by starting the service.
func (m *Manager) RunSyncNow(name string) error {
	// Ensure we're using the service unit
	serviceName := name
	if !strings.HasSuffix(serviceName, ".service") {
		serviceName = serviceName + ".service"
	}
	return m.Start(serviceName)
}

// StartContext starts a systemd user unit with context for cancellation.
func (m *Manager) StartContext(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, m.systemctlPath, "--user", "start", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("start %s failed: %w, output: %s", name, err, string(output))
	}
	return nil
}

// StopContext stops a systemd user unit with context for cancellation.
func (m *Manager) StopContext(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, m.systemctlPath, "--user", "stop", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop %s failed: %w, output: %s", name, err, string(output))
	}
	return nil
}

// parseSystemdTimestamp parses a systemd timestamp string.
func parseSystemdTimestamp(s string) (time.Time, error) {
	if s == "" || s == "n/a" {
		return time.Time{}, nil
	}

	// Try parsing as Unix timestamp (microseconds)
	if micros, err := strconv.ParseInt(s, 10, 64); err == nil {
		seconds := micros / 1000000
		nanos := (micros % 1000000) * 1000
		return time.Unix(seconds, nanos), nil
	}

	// Try common systemd timestamp formats
	formats := []string{
		"Mon 2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05 MST",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", s)
}
