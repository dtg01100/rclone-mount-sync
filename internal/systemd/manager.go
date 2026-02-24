// Package systemd provides functionality for managing systemd user services.
package systemd

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/dtg01100/rclone-mount-sync/internal/models"
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

// IsSystemdAvailable checks if systemd user manager is available on the system.
// It uses is-system-running which returns success if the manager is running,
// regardless of individual service states.
func (m *Manager) IsSystemdAvailable() bool {
	cmd := exec.Command(m.systemctlPath, "--user", "is-system-running")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	state := strings.TrimSpace(string(output))
	return state == "running" || state == "degraded"
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

// ResetFailed resets the failed state of a unit.
func (m *Manager) ResetFailed(name string) error {
	cmd := exec.Command(m.systemctlPath, "--user", "reset-failed", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("reset-failed failed: %w, output: %s", err, string(output))
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

// parseServiceListLine parses a single line from systemctl list-unit-files output.
// Expected format: "rclone-mount-name.service enabled" or "rclone-sync-name.service disabled"
// Returns the parsed name (without .service suffix) and enabled status.
func parseServiceListLine(line string) (name string, enabled bool, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false, false
	}

	parts := strings.Fields(line)
	if len(parts) < 1 {
		return "", false, false
	}

	unitName := parts[0]
	name = strings.TrimSuffix(unitName, ".service")
	enabled = len(parts) > 1 && parts[1] == "enabled"

	return name, enabled, true
}

// ListServices lists all rclone services (mounts and sync jobs).
func (m *Manager) ListServices() ([]ServiceStatus, error) {
	cmd := exec.Command(m.systemctlPath, "--user", "list-unit-files",
		"--type=service", "--no-legend", "rclone-*.service")
	output, err := cmd.Output()
	if err != nil {
		return []ServiceStatus{}, nil
	}

	var services []ServiceStatus
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		name, enabled, ok := parseServiceListLine(line)
		if !ok {
			continue
		}

		status := &ServiceStatus{
			Name:    name,
			Enabled: enabled,
		}

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

// ParseUnitID extracts the ID from a unit name like "rclone-mount-a1b2c3d4.service".
// Returns the ID and unit type ("mount" or "sync"). Returns empty strings if parsing fails.
func ParseUnitID(unitName string) (id string, unitType string) {
	// Remove .service or .timer suffix
	name := strings.TrimSuffix(unitName, ".service")
	name = strings.TrimSuffix(name, ".timer")

	// Parse rclone-{type}-{id}
	if strings.HasPrefix(name, "rclone-mount-") {
		return strings.TrimPrefix(name, "rclone-mount-"), "mount"
	}
	if strings.HasPrefix(name, "rclone-sync-") {
		return strings.TrimPrefix(name, "rclone-sync-"), "sync"
	}
	return "", ""
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

// ServiceManager defines the interface for systemd service operations.
// It allows for mocking in tests.
type ServiceManager interface {
	IsSystemdAvailable() bool
	DaemonReload() error
	Enable(name string) error
	Disable(name string) error
	Start(name string) error
	Stop(name string) error
	Restart(name string) error
	Status(name string) (*ServiceStatus, error)
	IsEnabled(name string) (bool, error)
	IsActive(name string) (bool, error)
	ListServices() ([]ServiceStatus, error)
	GetLogs(name string, lines int) (string, error)
	GetDetailedStatus(name string) (*models.ServiceStatus, error)
	GetTimerNextRun(timerName string) (time.Time, error)
	StartTimer(name string) error
	StopTimer(name string) error
	EnableTimer(name string) error
	DisableTimer(name string) error
	RunSyncNow(name string) error
	ResetFailed(name string) error
}

// MockManager is a mock implementation of ServiceManager for testing.
type MockManager struct {
	IsSystemdAvailableResult bool
	DaemonReloadErr          error
	EnableErr                error
	DisableErr               error
	StartErr                 error
	StopErr                  error
	RestartErr               error
	StatusResult             *ServiceStatus
	StatusErr                error
	IsEnabledResult          bool
	IsEnabledErr             error
	IsActiveResult           bool
	IsActiveErr              error
	ListServicesResult       []ServiceStatus
	ListServicesErr          error
	GetLogsResult            string
	GetLogsErr               error
	GetDetailedStatusResult  *models.ServiceStatus
	GetDetailedStatusErr     error
	GetTimerNextRunResult    time.Time
	GetTimerNextRunErr       error
	StartTimerErr            error
	StopTimerErr             error
	EnableTimerErr           error
	DisableTimerErr          error
	RunSyncNowErr            error
	ResetFailedErr           error
}

// IsSystemdAvailable mocks the IsSystemdAvailable method.
func (m *MockManager) IsSystemdAvailable() bool {
	return m.IsSystemdAvailableResult
}

// DaemonReload mocks the DaemonReload method.
func (m *MockManager) DaemonReload() error {
	return m.DaemonReloadErr
}

// Enable mocks the Enable method.
func (m *MockManager) Enable(name string) error {
	return m.EnableErr
}

// Disable mocks the Disable method.
func (m *MockManager) Disable(name string) error {
	return m.DisableErr
}

// Start mocks the Start method.
func (m *MockManager) Start(name string) error {
	return m.StartErr
}

// Stop mocks the Stop method.
func (m *MockManager) Stop(name string) error {
	return m.StopErr
}

// Restart mocks the Restart method.
func (m *MockManager) Restart(name string) error {
	return m.RestartErr
}

// Status mocks the Status method.
func (m *MockManager) Status(name string) (*ServiceStatus, error) {
	return m.StatusResult, m.StatusErr
}

// IsEnabled mocks the IsEnabled method.
func (m *MockManager) IsEnabled(name string) (bool, error) {
	return m.IsEnabledResult, m.IsEnabledErr
}

// IsActive mocks the IsActive method.
func (m *MockManager) IsActive(name string) (bool, error) {
	return m.IsActiveResult, m.IsActiveErr
}

// ListServices mocks the ListServices method.
func (m *MockManager) ListServices() ([]ServiceStatus, error) {
	return m.ListServicesResult, m.ListServicesErr
}

// GetLogs mocks the GetLogs method.
func (m *MockManager) GetLogs(name string, lines int) (string, error) {
	return m.GetLogsResult, m.GetLogsErr
}

// GetDetailedStatus mocks the GetDetailedStatus method.
func (m *MockManager) GetDetailedStatus(name string) (*models.ServiceStatus, error) {
	return m.GetDetailedStatusResult, m.GetDetailedStatusErr
}

// GetTimerNextRun mocks the GetTimerNextRun method.
func (m *MockManager) GetTimerNextRun(timerName string) (time.Time, error) {
	return m.GetTimerNextRunResult, m.GetTimerNextRunErr
}

// StartTimer mocks the StartTimer method.
func (m *MockManager) StartTimer(name string) error {
	return m.StartTimerErr
}

// StopTimer mocks the StopTimer method.
func (m *MockManager) StopTimer(name string) error {
	return m.StopTimerErr
}

// EnableTimer mocks the EnableTimer method.
func (m *MockManager) EnableTimer(name string) error {
	return m.EnableTimerErr
}

// DisableTimer mocks the DisableTimer method.
func (m *MockManager) DisableTimer(name string) error {
	return m.DisableTimerErr
}

// RunSyncNow mocks the RunSyncNow method.
func (m *MockManager) RunSyncNow(name string) error {
	return m.RunSyncNowErr
}

// ResetFailed mocks the ResetFailed method.
func (m *MockManager) ResetFailed(name string) error {
	return m.ResetFailedErr
}
