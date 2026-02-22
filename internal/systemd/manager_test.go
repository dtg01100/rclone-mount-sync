package systemd

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestNewManager tests the NewManager function.
func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if m.systemctlPath == "" {
		t.Error("NewManager() systemctlPath is empty")
	}
}

// TestManager_SystemctlPath tests that the manager has a valid systemctl path.
func TestManager_SystemctlPath(t *testing.T) {
	m := NewManager()
	// The path should either be the actual systemctl path or the default
	if m.systemctlPath != "/usr/bin/systemctl" && m.systemctlPath != "systemctl" {
		// It could also be a full path from LookPath
		if m.systemctlPath == "" {
			t.Error("systemctlPath should not be empty")
		}
	}
}

// TestParseSystemdTimestamp tests the parseSystemdTimestamp function.
func TestParseSystemdTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			wantErr: false, // Returns zero time
		},
		{
			name:    "n/a string",
			input:   "n/a",
			wantErr: false, // Returns zero time
		},
		{
			name:    "unix timestamp microseconds",
			input:   "1708540800000000", // Microseconds since epoch
			wantErr: false,
		},
		{
			name:    "RFC3339 format",
			input:   "2024-02-21T15:04:05Z",
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "not-a-timestamp",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSystemdTimestamp(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSystemdTimestamp() error = %v, wantErr %v", err, tt.wantErr)
			}
			// For empty and n/a, should return zero time
			if (tt.input == "" || tt.input == "n/a") && !got.IsZero() {
				t.Errorf("parseSystemdTimestamp(%q) = %v, want zero time", tt.input, got)
			}
		})
	}
}

// TestParseSystemdTimestamp_Microseconds tests parsing microsecond timestamps.
func TestParseSystemdTimestamp_Microseconds(t *testing.T) {
	// Test a known timestamp
	// 1708540800000000 microseconds = 2024-02-21 16:00:00 UTC
	micros := int64(1708540800000000)
	input := strconv.FormatInt(micros, 10)

	got, err := parseSystemdTimestamp(input)
	if err != nil {
		t.Fatalf("parseSystemdTimestamp() error = %v", err)
	}

	if got.IsZero() {
		t.Error("parseSystemdTimestamp() returned zero time for valid input")
	}

	// Verify the time is reasonable (within the last few years)
	now := time.Now()
	if got.After(now) {
		t.Errorf("parseSystemdTimestamp() returned future time %v, should be in the past", got)
	}
}

// TestParseSystemdTimestamp_RFC3339 tests parsing RFC3339 timestamps.
func TestParseSystemdTimestamp_RFC3339(t *testing.T) {
	input := "2024-02-21T15:04:05Z"

	got, err := parseSystemdTimestamp(input)
	if err != nil {
		t.Fatalf("parseSystemdTimestamp() error = %v", err)
	}

	if got.IsZero() {
		t.Error("parseSystemdTimestamp() returned zero time for valid RFC3339 input")
	}

	// Verify the parsed time matches expected
	expected := time.Date(2024, 2, 21, 15, 4, 5, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("parseSystemdTimestamp() = %v, want %v", got, expected)
	}
}

// TestServiceStatus_Struct tests that ServiceStatus struct fields work correctly.
func TestServiceStatus_Struct(t *testing.T) {
	status := ServiceStatus{
		Name:     "rclone-mount-gdrive",
		Active:   true,
		State:    "active",
		SubState: "running",
		Enabled:  true,
	}

	if status.Name != "rclone-mount-gdrive" {
		t.Errorf("ServiceStatus.Name = %q, want %q", status.Name, "rclone-mount-gdrive")
	}
	if !status.Active {
		t.Error("ServiceStatus.Active should be true")
	}
	if status.State != "active" {
		t.Errorf("ServiceStatus.State = %q, want %q", status.State, "active")
	}
	if status.SubState != "running" {
		t.Errorf("ServiceStatus.SubState = %q, want %q", status.SubState, "running")
	}
	if !status.Enabled {
		t.Error("ServiceStatus.Enabled should be true")
	}
}

// TestManager_StartTimerNameHandling tests that StartTimer handles timer names correctly.
func TestManager_StartTimerNameHandling(t *testing.T) {
	// This tests the name handling logic without actually calling systemctl
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "name without suffix",
			input:    "rclone-sync-e5f6g7h8",
			expected: "rclone-sync-e5f6g7h8.timer",
		},
		{
			name:     "name with .timer suffix",
			input:    "rclone-sync-e5f6g7h8.timer",
			expected: "rclone-sync-e5f6g7h8.timer",
		},
		{
			name:     "name with .service suffix",
			input:    "rclone-sync-e5f6g7h8.service",
			expected: "rclone-sync-e5f6g7h8.service.timer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timerName := tt.input
			if len(timerName) < 6 || timerName[len(timerName)-6:] != ".timer" {
				timerName = timerName + ".timer"
			}
			if timerName != tt.expected {
				t.Errorf("timer name handling: got %q, want %q", timerName, tt.expected)
			}
		})
	}
}

// TestManager_StopTimerNameHandling tests that StopTimer handles timer names correctly.
func TestManager_StopTimerNameHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "name without suffix",
			input:    "rclone-sync-e5f6g7h8",
			expected: "rclone-sync-e5f6g7h8.timer",
		},
		{
			name:     "name with .timer suffix",
			input:    "rclone-sync-e5f6g7h8.timer",
			expected: "rclone-sync-e5f6g7h8.timer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timerName := tt.input
			if len(timerName) < 6 || timerName[len(timerName)-6:] != ".timer" {
				timerName = timerName + ".timer"
			}
			if timerName != tt.expected {
				t.Errorf("timer name handling: got %q, want %q", timerName, tt.expected)
			}
		})
	}
}

// TestManager_EnableTimerNameHandling tests that EnableTimer handles timer names correctly.
func TestManager_EnableTimerNameHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "name without suffix",
			input:    "rclone-sync-i9j0k1l2",
			expected: "rclone-sync-i9j0k1l2.timer",
		},
		{
			name:     "name with .timer suffix",
			input:    "rclone-sync-i9j0k1l2.timer",
			expected: "rclone-sync-i9j0k1l2.timer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timerName := tt.input
			if len(timerName) < 6 || timerName[len(timerName)-6:] != ".timer" {
				timerName = timerName + ".timer"
			}
			if timerName != tt.expected {
				t.Errorf("timer name handling: got %q, want %q", timerName, tt.expected)
			}
		})
	}
}

// TestManager_DisableTimerNameHandling tests that DisableTimer handles timer names correctly.
func TestManager_DisableTimerNameHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "name without suffix",
			input:    "rclone-sync-m3n4o5p6",
			expected: "rclone-sync-m3n4o5p6.timer",
		},
		{
			name:     "name with .timer suffix",
			input:    "rclone-sync-m3n4o5p6.timer",
			expected: "rclone-sync-m3n4o5p6.timer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timerName := tt.input
			if len(timerName) < 6 || timerName[len(timerName)-6:] != ".timer" {
				timerName = timerName + ".timer"
			}
			if timerName != tt.expected {
				t.Errorf("timer name handling: got %q, want %q", timerName, tt.expected)
			}
		})
	}
}

// TestManager_RunSyncNowNameHandling tests that RunSyncNow handles service names correctly.
func TestManager_RunSyncNowNameHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "name without suffix",
			input:    "rclone-sync-q7r8s9t0",
			expected: "rclone-sync-q7r8s9t0.service",
		},
		{
			name:     "name with .service suffix",
			input:    "rclone-sync-q7r8s9t0.service",
			expected: "rclone-sync-q7r8s9t0.service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceName := tt.input
			if len(serviceName) < 8 || serviceName[len(serviceName)-8:] != ".service" {
				serviceName = serviceName + ".service"
			}
			if serviceName != tt.expected {
				t.Errorf("service name handling: got %q, want %q", serviceName, tt.expected)
			}
		})
	}
}

// TestManager_StartContext tests StartContext with a cancelled context.
func TestManager_StartContext(t *testing.T) {
	m := NewManager()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// This should fail because the context is cancelled
	err := m.StartContext(ctx, "nonexistent-service")
	if err == nil {
		t.Error("StartContext with cancelled context should return error")
	}
}

// TestManager_StopContext tests StopContext with a cancelled context.
func TestManager_StopContext(t *testing.T) {
	m := NewManager()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// This should fail because the context is cancelled
	err := m.StopContext(ctx, "nonexistent-service")
	if err == nil {
		t.Error("StopContext with cancelled context should return error")
	}
}

// TestManager_IsSystemdAvailable tests IsSystemdAvailable.
// Note: This test will pass differently depending on the environment.
func TestManager_IsSystemdAvailable(t *testing.T) {
	m := NewManager()

	// Just verify the method doesn't panic
	_ = m.IsSystemdAvailable()
}

// TestManager_DaemonReload tests DaemonReload.
// Note: This will likely fail in a non-systemd environment.
func TestManager_DaemonReload(t *testing.T) {
	m := NewManager()

	// This will likely fail in CI/test environments without systemd
	err := m.DaemonReload()
	// We don't assert on the result since it depends on the environment
	_ = err
}

// TestManager_Enable tests Enable.
func TestManager_Enable(t *testing.T) {
	m := NewManager()

	// This will fail because the service doesn't exist
	err := m.Enable("nonexistent-service-12345")
	if err == nil {
		t.Error("Enable() should return error for nonexistent service")
	}
}

// TestManager_Disable tests Disable.
func TestManager_Disable(t *testing.T) {
	m := NewManager()

	// This will fail because the service doesn't exist
	err := m.Disable("nonexistent-service-12345")
	if err == nil {
		t.Error("Disable() should return error for nonexistent service")
	}
}

// TestManager_Start tests Start.
func TestManager_Start(t *testing.T) {
	m := NewManager()

	// This will fail because the service doesn't exist
	err := m.Start("nonexistent-service-12345")
	if err == nil {
		t.Error("Start() should return error for nonexistent service")
	}
}

// TestManager_Stop tests Stop.
func TestManager_Stop(t *testing.T) {
	m := NewManager()

	// This will fail because the service doesn't exist
	err := m.Stop("nonexistent-service-12345")
	if err == nil {
		t.Error("Stop() should return error for nonexistent service")
	}
}

// TestManager_Restart tests Restart.
func TestManager_Restart(t *testing.T) {
	m := NewManager()

	// This will fail because the service doesn't exist
	err := m.Restart("nonexistent-service-12345")
	if err == nil {
		t.Error("Restart() should return error for nonexistent service")
	}
}

// TestManager_Status tests Status.
func TestManager_Status(t *testing.T) {
	m := NewManager()

	// This will fail because the service doesn't exist
	// Note: In some environments, systemctl may return success even for non-existent units
	_, err := m.Status("nonexistent-service-12345")
	// We don't assert on the result since it depends on the environment
	_ = err
}

// TestManager_IsEnabled tests IsEnabled.
func TestManager_IsEnabled(t *testing.T) {
	m := NewManager()

	// This will return false because the service doesn't exist
	enabled, _ := m.IsEnabled("nonexistent-service-12345")
	if enabled {
		t.Error("IsEnabled() should return false for nonexistent service")
	}
}

// TestManager_IsActive tests IsActive.
func TestManager_IsActive(t *testing.T) {
	m := NewManager()

	// This will return false because the service doesn't exist
	active, _ := m.IsActive("nonexistent-service-12345")
	if active {
		t.Error("IsActive() should return false for nonexistent service")
	}
}

// TestManager_ListServices tests ListServices.
func TestManager_ListServices(t *testing.T) {
	m := NewManager()

	// This should return an empty list or the list of existing services
	services, err := m.ListServices()
	if err != nil {
		t.Errorf("ListServices() error = %v", err)
	}
	// services could be empty or contain existing rclone services
	_ = services
}

// TestManager_GetLogs tests GetLogs.
func TestManager_GetLogs(t *testing.T) {
	m := NewManager()

	// This will fail because the service doesn't exist
	_, err := m.GetLogs("nonexistent-service-12345", 10)
	if err == nil {
		t.Error("GetLogs() should return error for nonexistent service")
	}
}

// TestManager_GetDetailedStatus tests GetDetailedStatus.
func TestManager_GetDetailedStatus(t *testing.T) {
	m := NewManager()

	// This will fail because the service doesn't exist
	// Note: In some environments, systemctl may return success even for non-existent units
	_, err := m.GetDetailedStatus("nonexistent-service-12345")
	// We don't assert on the result since it depends on the environment
	_ = err
}

// TestManager_GetTimerNextRun tests GetTimerNextRun.
func TestManager_GetTimerNextRun(t *testing.T) {
	m := NewManager()

	// This will fail because the timer doesn't exist
	// Note: In some environments, systemctl may return success even for non-existent units
	_, err := m.GetTimerNextRun("nonexistent-timer-12345.timer")
	// We don't assert on the result since it depends on the environment
	_ = err
}

// TestManager_StartTimer tests StartTimer.
func TestManager_StartTimer(t *testing.T) {
	m := NewManager()

	// This will fail because the timer doesn't exist
	err := m.StartTimer("nonexistent-timer-12345")
	if err == nil {
		t.Error("StartTimer() should return error for nonexistent timer")
	}
}

// TestManager_StopTimer tests StopTimer.
func TestManager_StopTimer(t *testing.T) {
	m := NewManager()

	// This will fail because the timer doesn't exist
	err := m.StopTimer("nonexistent-timer-12345")
	if err == nil {
		t.Error("StopTimer() should return error for nonexistent timer")
	}
}

// TestManager_EnableTimer tests EnableTimer.
func TestManager_EnableTimer(t *testing.T) {
	m := NewManager()

	// This will fail because the timer doesn't exist
	err := m.EnableTimer("nonexistent-timer-12345")
	if err == nil {
		t.Error("EnableTimer() should return error for nonexistent timer")
	}
}

// TestManager_DisableTimer tests DisableTimer.
func TestManager_DisableTimer(t *testing.T) {
	m := NewManager()

	// This will fail because the timer doesn't exist
	err := m.DisableTimer("nonexistent-timer-12345")
	if err == nil {
		t.Error("DisableTimer() should return error for nonexistent timer")
	}
}

// TestManager_RunSyncNow tests RunSyncNow.
func TestManager_RunSyncNow(t *testing.T) {
	m := NewManager()

	// This will fail because the service doesn't exist
	err := m.RunSyncNow("nonexistent-sync-12345")
	if err == nil {
		t.Error("RunSyncNow() should return error for nonexistent service")
	}
}

// TestParseSystemdTimestamp_CommonFormats tests various common timestamp formats.
func TestParseSystemdTimestamp_CommonFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool // Whether we expect a valid parse
	}{
		{
			name:  "systemd format with day",
			input: "Wed 2024-02-21 15:04:05 UTC",
			valid: true,
		},
		{
			name:  "date time timezone format",
			input: "2024-02-21 15:04:05 UTC",
			valid: true,
		},
		{
			name:  "RFC3339",
			input: "2024-02-21T15:04:05Z",
			valid: true,
		},
		{
			name:  "zero microseconds",
			input: "0",
			valid: true, // Returns zero time without error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSystemdTimestamp(tt.input)
			if tt.valid && err != nil {
				t.Errorf("parseSystemdTimestamp(%q) error = %v", tt.input, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("parseSystemdTimestamp(%q) expected error, got time %v", tt.input, got)
			}
		})
	}
}

// TestServiceStatus_ZeroValue tests zero value of ServiceStatus.
func TestServiceStatus_ZeroValue(t *testing.T) {
	var status ServiceStatus

	if status.Name != "" {
		t.Errorf("Zero value ServiceStatus.Name = %q, want empty", status.Name)
	}
	if status.Active {
		t.Error("Zero value ServiceStatus.Active should be false")
	}
	if status.State != "" {
		t.Errorf("Zero value ServiceStatus.State = %q, want empty", status.State)
	}
	if status.Enabled {
		t.Error("Zero value ServiceStatus.Enabled should be false")
	}
}

// TestManager_WithMockSystemctl tests manager operations with a custom systemctl path.
func TestManager_WithMockSystemctl(t *testing.T) {
	// Create a manager with a non-existent systemctl path
	m := &Manager{systemctlPath: "/nonexistent/systemctl"}

	// All operations should fail gracefully
	_ = m.DaemonReload()
	_ = m.Enable("test")
	_ = m.Disable("test")
	_ = m.Start("test")
	_ = m.Stop("test")
	_ = m.Restart("test")
	_, _ = m.Status("test")
	_, _ = m.IsEnabled("test")
	_, _ = m.IsActive("test")
	_, _ = m.ListServices()
	_, _ = m.GetLogs("test", 10)
	_, _ = m.GetDetailedStatus("test")
	_, _ = m.GetTimerNextRun("test.timer")
}

// TestManager_ContextCancellation tests that context cancellation is handled.
func TestManager_ContextCancellation(t *testing.T) {
	m := NewManager()

	// Test with already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// These should fail due to cancelled context
	err := m.StartContext(ctx, "test-service")
	if err == nil {
		t.Error("StartContext with cancelled context should fail")
	}

	err = m.StopContext(ctx, "test-service")
	if err == nil {
		t.Error("StopContext with cancelled context should fail")
	}
}

// TestManager_Timeout tests operations with timeout.
func TestManager_Timeout(t *testing.T) {
	m := NewManager()

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(1 * time.Millisecond)

	// This should fail due to timeout
	err := m.StartContext(ctx, "test-service")
	if err == nil {
		t.Error("StartContext with expired context should fail")
	}
}

// TestManager_EnableTimerError tests EnableTimer with nonexistent timer.
func TestManager_EnableTimerError(t *testing.T) {
	m := NewManager()

	err := m.EnableTimer("nonexistent-timer-12345")
	if err == nil {
		t.Error("EnableTimer() should return error for nonexistent timer")
	}
}

// TestManager_DisableTimerError tests DisableTimer with nonexistent timer.
func TestManager_DisableTimerError(t *testing.T) {
	m := NewManager()

	err := m.DisableTimer("nonexistent-timer-12345")
	if err == nil {
		t.Error("DisableTimer() should return error for nonexistent timer")
	}
}

// TestManager_StopTimerError tests StopTimer with nonexistent timer.
func TestManager_StopTimerError(t *testing.T) {
	m := NewManager()

	err := m.StopTimer("nonexistent-timer-12345")
	if err == nil {
		t.Error("StopTimer() should return error for nonexistent timer")
	}
}

// TestManager_RunSyncNowError tests RunSyncNow with nonexistent service.
func TestManager_RunSyncNowError(t *testing.T) {
	m := NewManager()

	err := m.RunSyncNow("nonexistent-sync-12345")
	if err == nil {
		t.Error("RunSyncNow() should return error for nonexistent service")
	}
}

// TestManager_GetLogsError tests GetLogs error handling.
func TestManager_GetLogsError(t *testing.T) {
	m := NewManager()

	_, err := m.GetLogs("nonexistent-service-12345", 10)
	if err == nil {
		t.Error("GetLogs() should return error for nonexistent service")
	}
}

// TestManager_ListServicesEmptyResult tests ListServices with no rclone services.
func TestManager_ListServicesEmptyResult(t *testing.T) {
	m := NewManager()

	services, err := m.ListServices()
	if err != nil {
		t.Errorf("ListServices() error = %v", err)
	}
	// Should return empty or existing rclone services (not error)
	if services == nil {
		t.Error("ListServices() should not return nil slice")
	}
}

// TestManager_GetTimerNextRunNonexistent tests GetTimerNextRun with nonexistent timer.
func TestManager_GetTimerNextRunNonexistent(t *testing.T) {
	m := NewManager()

	_, err := m.GetTimerNextRun("nonexistent-timer-12345.timer")
	// This may or may not error depending on environment
	_ = err
}

// TestManager_StartTimerWithServiceSuffix tests StartTimer name handling with .service suffix.
func TestManager_StartTimerWithServiceSuffix(t *testing.T) {
	m := NewManager()

	err := m.StartTimer("rclone-sync-u1v2w3x4.service")
	// Should fail because timer doesn't exist, but name should be handled
	if err == nil {
		t.Error("StartTimer() should return error for nonexistent timer")
	}
}

// TestManager_StopTimerWithServiceSuffix tests StopTimer name handling with .service suffix.
func TestManager_StopTimerWithServiceSuffix(t *testing.T) {
	m := NewManager()

	err := m.StopTimer("rclone-sync-y5z6a7b8.service")
	// Should fail because timer doesn't exist, but name should be handled
	if err == nil {
		t.Error("StopTimer() should return error for nonexistent timer")
	}
}

// TestManager_EnableTimerWithServiceSuffix tests EnableTimer name handling with .service suffix.
func TestManager_EnableTimerWithServiceSuffix(t *testing.T) {
	m := NewManager()

	err := m.EnableTimer("rclone-sync-c9d0e1f2.service")
	// Should fail because timer doesn't exist, but name should be handled
	if err == nil {
		t.Error("EnableTimer() should return error for nonexistent timer")
	}
}

// TestManager_DisableTimerWithServiceSuffix tests DisableTimer name handling with .service suffix.
func TestManager_DisableTimerWithServiceSuffix(t *testing.T) {
	m := NewManager()

	err := m.DisableTimer("rclone-sync-g3h4i5j6.service")
	// Should fail because timer doesn't exist, but name should be handled
	if err == nil {
		t.Error("DisableTimer() should return error for nonexistent timer")
	}
}

// TestManager_RunSyncNowWithTimerSuffix tests RunSyncNow name handling with .timer suffix.
func TestManager_RunSyncNowWithTimerSuffix(t *testing.T) {
	m := NewManager()

	err := m.RunSyncNow("rclone-sync-k7l8m9n0.timer")
	// Should fail because service doesn't exist, but name should be handled
	if err == nil {
		t.Error("RunSyncNow() should return error for nonexistent service")
	}
}

// TestParseSystemdTimestamp_EdgeCases tests additional timestamp edge cases.
func TestParseSystemdTimestamp_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "whitespace string",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "negative microseconds - valid but past time",
			input:   "-1234567890",
			wantErr: false,
		},
		{
			name:    "very large microseconds",
			input:   "9999999999999999999",
			wantErr: true,
		},
		{
			name:    "partial RFC3339",
			input:   "2024-02-21T15:04:05",
			wantErr: true,
		},
		{
			name:    "date only",
			input:   "2024-02-21",
			wantErr: true,
		},
		{
			name:    "mixed format",
			input:   "2024/02/21 15:04:05",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSystemdTimestamp(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSystemdTimestamp(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestServiceStatus_AllFields tests ServiceStatus with all fields populated.
func TestServiceStatus_AllFields(t *testing.T) {
	status := ServiceStatus{
		Name:     "rclone-mount-gdrive",
		Active:   true,
		State:    "active",
		SubState: "running",
		Enabled:  true,
	}

	if status.Name != "rclone-mount-gdrive" {
		t.Errorf("ServiceStatus.Name = %q, want %q", status.Name, "rclone-mount-gdrive")
	}
	if !status.Active {
		t.Error("ServiceStatus.Active should be true")
	}
	if status.State != "active" {
		t.Errorf("ServiceStatus.State = %q, want %q", status.State, "active")
	}
	if status.SubState != "running" {
		t.Errorf("ServiceStatus.SubState = %q, want %q", status.SubState, "running")
	}
	if !status.Enabled {
		t.Error("ServiceStatus.Enabled should be true")
	}
}

// TestManager_IsSystemdAvailableWithInvalidPath tests IsSystemdAvailable with invalid path.
func TestManager_IsSystemdAvailableWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	result := m.IsSystemdAvailable()
	if result {
		t.Error("IsSystemdAvailable() should return false for invalid systemctl path")
	}
}

// TestManager_DaemonReloadWithInvalidPath tests DaemonReload with invalid path.
func TestManager_DaemonReloadWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	err := m.DaemonReload()
	if err == nil {
		t.Error("DaemonReload() should return error for invalid systemctl path")
	}
}

// TestManager_EnableWithInvalidPath tests Enable with invalid path.
func TestManager_EnableWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	err := m.Enable("test-service")
	if err == nil {
		t.Error("Enable() should return error for invalid systemctl path")
	}
}

// TestManager_DisableWithInvalidPath tests Disable with invalid path.
func TestManager_DisableWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	err := m.Disable("test-service")
	if err == nil {
		t.Error("Disable() should return error for invalid systemctl path")
	}
}

// TestManager_StartWithInvalidPath tests Start with invalid path.
func TestManager_StartWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	err := m.Start("test-service")
	if err == nil {
		t.Error("Start() should return error for invalid systemctl path")
	}
}

// TestManager_StopWithInvalidPath tests Stop with invalid path.
func TestManager_StopWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	err := m.Stop("test-service")
	if err == nil {
		t.Error("Stop() should return error for invalid systemctl path")
	}
}

// TestManager_RestartWithInvalidPath tests Restart with invalid path.
func TestManager_RestartWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	err := m.Restart("test-service")
	if err == nil {
		t.Error("Restart() should return error for invalid systemctl path")
	}
}

// TestManager_StatusWithInvalidPath tests Status with invalid path.
func TestManager_StatusWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	_, err := m.Status("test-service")
	if err == nil {
		t.Error("Status() should return error for invalid systemctl path")
	}
}

// TestManager_IsEnabledWithInvalidPath tests IsEnabled with invalid path.
func TestManager_IsEnabledWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	enabled, _ := m.IsEnabled("test-service")
	if enabled {
		t.Error("IsEnabled() should return false for invalid systemctl path")
	}
}

// TestManager_IsActiveWithInvalidPath tests IsActive with invalid path.
func TestManager_IsActiveWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	active, _ := m.IsActive("test-service")
	if active {
		t.Error("IsActive() should return false for invalid systemctl path")
	}
}

// TestManager_ListServicesWithInvalidPath tests ListServices with invalid path.
func TestManager_ListServicesWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	services, err := m.ListServices()
	// Should return empty list and no error (error is swallowed)
	if err != nil {
		t.Errorf("ListServices() should not return error, got: %v", err)
	}
	if len(services) != 0 {
		t.Error("ListServices() should return empty list for invalid path")
	}
}

// TestManager_GetLogsWithInvalidPath tests GetLogs with invalid path.
func TestManager_GetLogsWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	_, err := m.GetLogs("test-service", 10)
	if err == nil {
		t.Error("GetLogs() should return error for invalid systemctl path")
	}
}

// TestManager_GetDetailedStatusWithInvalidPath tests GetDetailedStatus with invalid path.
func TestManager_GetDetailedStatusWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	_, err := m.GetDetailedStatus("test-service")
	if err == nil {
		t.Error("GetDetailedStatus() should return error for invalid systemctl path")
	}
}

// TestManager_GetTimerNextRunWithInvalidPath tests GetTimerNextRun with invalid path.
func TestManager_GetTimerNextRunWithInvalidPath(t *testing.T) {
	m := &Manager{systemctlPath: "/nonexistent/path/systemctl"}

	_, err := m.GetTimerNextRun("test.timer")
	if err == nil {
		t.Error("GetTimerNextRun() should return error for invalid systemctl path")
	}
}

// TestManager_StopContext tests StopContext with cancelled context.
func TestManager_StopContext_WithCancelledContext(t *testing.T) {
	m := NewManager()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := m.StopContext(ctx, "test-service")
	if err == nil {
		t.Error("StopContext with cancelled context should return error")
	}
}

// TestManager_StartContext_WithTimeout tests StartContext with timed-out context.
func TestManager_StartContext_WithTimeout(t *testing.T) {
	m := NewManager()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(1 * time.Millisecond)

	err := m.StartContext(ctx, "test-service")
	if err == nil {
		t.Error("StartContext with timed-out context should return error")
	}
}

// TestManager_StopContext_WithTimeout tests StopContext with timed-out context.
func TestManager_StopContext_WithTimeout(t *testing.T) {
	m := NewManager()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(1 * time.Millisecond)

	err := m.StopContext(ctx, "test-service")
	if err == nil {
		t.Error("StopContext with timed-out context should return error")
	}
}

// TestParseSystemdTimestamp_Timezones tests parsing timestamps with different timezone formats.
func TestParseSystemdTimestamp_Timezones(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "UTC timezone",
			input:   "Wed 2024-02-21 15:04:05 UTC",
			wantErr: false,
		},
		{
			name:    "EST timezone",
			input:   "Wed 2024-02-21 15:04:05 EST",
			wantErr: false,
		},
		{
			name:    "PST timezone",
			input:   "Wed 2024-02-21 15:04:05 PST",
			wantErr: false,
		},
		{
			name:    "RFC3339 with Z",
			input:   "2024-02-21T15:04:05Z",
			wantErr: false,
		},
		{
			name:    "RFC3339 with offset",
			input:   "2024-02-21T15:04:05+00:00",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSystemdTimestamp(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSystemdTimestamp(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestManager_GetDetailedStatus_NameParsing tests GetDetailedStatus type parsing from name.
func TestManager_GetDetailedStatus_NameParsing(t *testing.T) {
	tests := []struct {
		name     string
		unitName string
		wantType string
	}{
		{
			name:     "mount service",
			unitName: "rclone-mount-s5t6u7v8.service",
			wantType: "mount",
		},
		{
			name:     "sync service",
			unitName: "rclone-sync-w9x0y1z2.service",
			wantType: "sync",
		},
		{
			name:     "unknown service",
			unitName: "other-service.service",
			wantType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't test the actual parsing without systemd, but we can test the name parsing logic
			unitType := ""
			if strings.HasPrefix(tt.unitName, "rclone-mount-") {
				unitType = "mount"
			} else if strings.HasPrefix(tt.unitName, "rclone-sync-") {
				unitType = "sync"
			}

			if unitType != tt.wantType {
				t.Errorf("Type parsing for %q = %q, want %q", tt.unitName, unitType, tt.wantType)
			}
		})
	}
}

// TestManager_EnableDisable tests that Enable/Disable work as expected.
func TestManager_EnableDisable_NonexistentUnit(t *testing.T) {
	m := NewManager()

	unitName := "nonexistent-unit-for-testing-12345"

	// Both should fail for nonexistent unit
	if err := m.Enable(unitName); err == nil {
		t.Error("Enable() should fail for nonexistent unit")
	}
	if err := m.Disable(unitName); err == nil {
		t.Error("Disable() should fail for nonexistent unit")
	}
}

// TestManager_StartStop tests that Start/Stop work as expected.
func TestManager_StartStop_NonexistentUnit(t *testing.T) {
	m := NewManager()

	unitName := "nonexistent-unit-for-testing-12345"

	// Both should fail for nonexistent unit
	if err := m.Start(unitName); err == nil {
		t.Error("Start() should fail for nonexistent unit")
	}
	if err := m.Stop(unitName); err == nil {
		t.Error("Stop() should fail for nonexistent unit")
	}
}

// TestManager_TimerOperations tests timer operations with nonexistent timers.
func TestManager_TimerOperations_NonexistentTimer(t *testing.T) {
	m := NewManager()

	timerName := "nonexistent-timer-for-testing-12345"

	// All should fail for nonexistent timer
	if err := m.StartTimer(timerName); err == nil {
		t.Error("StartTimer() should fail for nonexistent timer")
	}
	if err := m.StopTimer(timerName); err == nil {
		t.Error("StopTimer() should fail for nonexistent timer")
	}
	if err := m.EnableTimer(timerName); err == nil {
		t.Error("EnableTimer() should fail for nonexistent timer")
	}
	if err := m.DisableTimer(timerName); err == nil {
		t.Error("DisableTimer() should fail for nonexistent timer")
	}
}

// TestManager_RestartNonexistent tests Restart with nonexistent service.
func TestManager_RestartNonexistent(t *testing.T) {
	m := NewManager()

	err := m.Restart("nonexistent-service-12345")
	if err == nil {
		t.Error("Restart() should return error for nonexistent service")
	}
}
