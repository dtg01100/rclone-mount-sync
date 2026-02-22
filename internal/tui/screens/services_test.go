package screens

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
)

// Test errors for services
var errTestServiceNotFound = errors.New("service not found")
var errTestServiceFailed = errors.New("service failed")

// Helper function to create a test services screen
func createTestServicesScreen() *ServicesScreen {
	return NewServicesScreen()
}

// Helper function to create test service info
func createTestServices() []ServiceInfo {
	return []ServiceInfo{
		{
			Name:       "rclone-mount-gdrive",
			Type:       "mount",
			Status:     "active",
			SubState:   "running",
			Enabled:    true,
			MountPoint: "/mnt/gdrive",
			Remote:     "gdrive:",
		},
		{
			Name:       "rclone-mount-dropbox",
			Type:       "mount",
			Status:     "inactive",
			SubState:   "dead",
			Enabled:    false,
			MountPoint: "/mnt/dropbox",
			Remote:     "dropbox:",
		},
		{
			Name:        "rclone-sync-backup",
			Type:        "sync",
			Status:      "active",
			SubState:    "running",
			Enabled:     true,
			Source:      "gdrive:/Documents",
			Destination: "/home/user/backup",
			TimerActive: true,
			NextRun:     time.Now().Add(1 * time.Hour),
		},
		{
			Name:        "rclone-sync-photos",
			Type:        "sync",
			Status:      "failed",
			SubState:    "failed",
			Enabled:     true,
			Source:      "dropbox:/Photos",
			Destination: "/home/user/photos",
			TimerActive: false,
		},
	}
}

// Helper function to create a test config with mounts and sync jobs
func createTestConfigForServices() *config.Config {
	return &config.Config{
		Version: "1.0",
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
		},
		Mounts: []models.MountConfig{
			{
				ID:          "mount1",
				Name:        "gdrive",
				Remote:      "gdrive:",
				RemotePath:  "/",
				MountPoint:  "/mnt/gdrive",
				Description: "Google Drive mount",
				Enabled:     true,
			},
			{
				ID:          "mount2",
				Name:        "dropbox",
				Remote:      "dropbox:",
				RemotePath:  "/",
				MountPoint:  "/mnt/dropbox",
				Description: "Dropbox mount",
				Enabled:     false,
			},
		},
		SyncJobs: []models.SyncJobConfig{
			{
				ID:          "sync1",
				Name:        "backup",
				Source:      "gdrive:/Documents",
				Destination: "/home/user/backup",
				Description: "Backup sync job",
				Enabled:     true,
			},
			{
				ID:          "sync2",
				Name:        "photos",
				Source:      "dropbox:/Photos",
				Destination: "/home/user/photos",
				Description: "Photos sync job",
				Enabled:     true,
			},
		},
	}
}

func TestNewServicesScreen(t *testing.T) {
	screen := NewServicesScreen()

	if screen == nil {
		t.Fatal("NewServicesScreen() returned nil")
	}

	// Verify initial mode
	if screen.mode != ServicesModeList {
		t.Errorf("mode = %q, want %q (ServicesModeList)", screen.mode, ServicesModeList)
	}

	// Verify initial filter
	if screen.filter != FilterAll {
		t.Errorf("filter = %q, want %q (FilterAll)", screen.filter, FilterAll)
	}

	// Verify initial log filter
	if screen.logFilter != "all" {
		t.Errorf("logFilter = %q, want 'all'", screen.logFilter)
	}

	// Verify initial cursor
	if screen.cursor != 0 {
		t.Errorf("cursor = %d, want 0", screen.cursor)
	}

	// Verify goBack is false
	if screen.goBack {
		t.Error("goBack should be false initially")
	}

	// Verify services slices are initialized
	if screen.services == nil {
		t.Error("services should be initialized")
	}

	if screen.filteredServices == nil {
		t.Error("filteredServices should be initialized")
	}

	// Verify status message type
	if screen.statusMessageType != "info" {
		t.Errorf("statusMessageType = %q, want 'info'", screen.statusMessageType)
	}
}

func TestServicesScreen_SetSize(t *testing.T) {
	screen := NewServicesScreen()

	// Set size
	screen.SetSize(100, 30)

	if screen.width != 100 {
		t.Errorf("width = %d, want 100", screen.width)
	}

	if screen.height != 30 {
		t.Errorf("height = %d, want 30", screen.height)
	}
}

func TestServicesScreen_CursorNavigation(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.filteredServices = createTestServices()

	// Start at first item (index 0)
	if screen.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", screen.cursor)
	}

	// Press up - should stay at 0 (can't go above first item)
	screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	if screen.cursor != 0 {
		t.Errorf("cursor after up at top = %d, want 0", screen.cursor)
	}

	// Move down through all items
	for i := 0; i < len(screen.filteredServices)-1; i++ {
		screen.Update(tea.KeyMsg{Type: tea.KeyDown})
		expected := i + 1
		if screen.cursor != expected {
			t.Errorf("cursor after down %d times = %d, want %d", i+1, screen.cursor, expected)
		}
	}

	// Try to move down past last item - should stay at last
	lastIndex := len(screen.filteredServices) - 1
	screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	if screen.cursor != lastIndex {
		t.Errorf("cursor after down at bottom = %d, want %d", screen.cursor, lastIndex)
	}

	// Move back up
	screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	if screen.cursor != lastIndex-1 {
		t.Errorf("cursor after up = %d, want %d", screen.cursor, lastIndex-1)
	}
}

func TestServicesScreen_VimNavigation(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.filteredServices = createTestServices()

	// Test 'k' key (up) - should stay at 0
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if screen.cursor != 0 {
		t.Errorf("cursor after 'k' at top = %d, want 0", screen.cursor)
	}

	// Test 'j' key (down)
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if screen.cursor != 1 {
		t.Errorf("cursor after 'j' = %d, want 1", screen.cursor)
	}

	// Test 'k' key (up) again
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if screen.cursor != 0 {
		t.Errorf("cursor after 'k' = %d, want 0", screen.cursor)
	}
}

func TestServicesScreen_ModeTransitions(t *testing.T) {
	tests := []struct {
		name         string
		key          tea.KeyMsg
		setupScreen  func(*ServicesScreen)
		expectedMode string
	}{
		{
			name:         "Enter details mode",
			key:          tea.KeyMsg{Type: tea.KeyEnter},
			setupScreen:  func(s *ServicesScreen) { s.filteredServices = createTestServices() },
			expectedMode: ServicesModeDetails,
		},
		{
			name:         "Enter logs mode from list",
			key:          tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")},
			setupScreen:  func(s *ServicesScreen) { s.filteredServices = createTestServices(); s.manager = &systemd.Manager{} },
			expectedMode: ServicesModeLogs,
		},
		{
			name:         "Enter actions mode",
			key:          tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")},
			setupScreen:  func(s *ServicesScreen) { s.filteredServices = createTestServices() },
			expectedMode: ServicesModeActions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewServicesScreen()
			screen.SetSize(80, 24)
			tt.setupScreen(screen)

			// Ensure cursor is valid
			if screen.cursor >= len(screen.filteredServices) {
				screen.cursor = 0
			}

			screen.Update(tt.key)

			if screen.mode != tt.expectedMode {
				t.Errorf("mode = %q, want %q", screen.mode, tt.expectedMode)
			}
		})
	}
}

func TestServicesScreen_LoadServices(t *testing.T) {
	screen := NewServicesScreen()
	cfg := createTestConfigForServices()
	screen.cfg = cfg
	screen.manager = &systemd.Manager{}

	// Call loadServices
	msg := screen.loadServices()

	// Check message type
	loadedMsg, ok := msg.(ServicesLoadedMsg)
	if !ok {
		t.Fatalf("expected ServicesLoadedMsg, got %T", msg)
	}

	// Services should be loaded (may be empty if systemd not available)
	if loadedMsg.Services == nil {
		t.Error("Services should not be nil")
	}
}

func TestServicesScreen_LoadServicesNilManager(t *testing.T) {
	screen := NewServicesScreen()
	// Don't set manager - it should be nil

	// Call loadServices
	msg := screen.loadServices()

	// Check message type
	loadedMsg, ok := msg.(ServicesLoadedMsg)
	if !ok {
		t.Fatalf("expected ServicesLoadedMsg, got %T", msg)
	}

	// Should return empty services
	if len(loadedMsg.Services) != 0 {
		t.Errorf("loaded services = %d, want 0", len(loadedMsg.Services))
	}
}

func TestServicesScreen_ServicesLoadedMsg(t *testing.T) {
	screen := NewServicesScreen()
	screen.loading = true

	services := createTestServices()
	msg := ServicesLoadedMsg{Services: services}

	screen.Update(msg)

	// Verify services were set
	if len(screen.services) != len(services) {
		t.Errorf("services = %d, want %d", len(screen.services), len(services))
	}

	// Verify loading is false
	if screen.loading {
		t.Error("loading should be false after loading")
	}
}

func TestServicesScreen_FilterTypes(t *testing.T) {
	tests := []struct {
		name           string
		filter         string
		expectedCount  int
		expectedInList []string
	}{
		{
			name:           "Filter all",
			filter:         FilterAll,
			expectedCount:  4,
			expectedInList: []string{"rclone-mount-gdrive", "rclone-mount-dropbox", "rclone-sync-backup", "rclone-sync-photos"},
		},
		{
			name:           "Filter running",
			filter:         FilterRunning,
			expectedCount:  2,
			expectedInList: []string{"rclone-mount-gdrive", "rclone-sync-backup"},
		},
		{
			name:           "Filter stopped",
			filter:         FilterStopped,
			expectedCount:  1,
			expectedInList: []string{"rclone-mount-dropbox"},
		},
		{
			name:           "Filter failed",
			filter:         FilterFailed,
			expectedCount:  1,
			expectedInList: []string{"rclone-sync-photos"},
		},
		{
			name:           "Filter mounts",
			filter:         FilterMounts,
			expectedCount:  2,
			expectedInList: []string{"rclone-mount-gdrive", "rclone-mount-dropbox"},
		},
		{
			name:           "Filter sync jobs",
			filter:         FilterSyncJobs,
			expectedCount:  2,
			expectedInList: []string{"rclone-sync-backup", "rclone-sync-photos"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewServicesScreen()
			screen.services = createTestServices()
			screen.filter = tt.filter

			screen.applyFilter()

			if len(screen.filteredServices) != tt.expectedCount {
				t.Errorf("filtered services count = %d, want %d", len(screen.filteredServices), tt.expectedCount)
			}

			// Check expected services are in the filtered list
			for _, expectedName := range tt.expectedInList {
				found := false
				for _, svc := range screen.filteredServices {
					if svc.Name == expectedName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected service %q not found in filtered list", expectedName)
				}
			}
		})
	}
}

func TestServicesScreen_CycleFilter(t *testing.T) {
	screen := NewServicesScreen()
	screen.services = createTestServices()

	// Test filter cycling order
	expectedFilters := []string{
		FilterAll,
		FilterRunning,
		FilterStopped,
		FilterFailed,
		FilterMounts,
		FilterSyncJobs,
		FilterAll, // Cycles back to all
	}

	for i, expected := range expectedFilters {
		if screen.filter != expected {
			t.Errorf("step %d: filter = %q, want %q", i, screen.filter, expected)
		}
		screen.cycleFilter()
	}
}

func TestServicesScreen_CycleFilterKey(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.services = createTestServices()
	screen.applyFilter()

	initialFilter := screen.filter

	// Press 'f' to cycle filter
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})

	if screen.filter == initialFilter {
		t.Errorf("filter should have changed from %q", initialFilter)
	}
}

func TestServicesScreen_ServiceActions(t *testing.T) {
	tests := []struct {
		name        string
		key         tea.KeyMsg
		action      string
		expectCmd   bool
	}{
		{
			name:      "Start service with 's'",
			key:       tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")},
			action:    "start",
			expectCmd: true,
		},
		{
			name:      "Stop service with 'x'",
			key:       tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")},
			action:    "stop",
			expectCmd: true,
		},
		{
			name:      "Restart service with 'r'",
			key:       tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")},
			action:    "restart",
			expectCmd: true,
		},
		{
			name:      "Enable service with 'e'",
			key:       tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")},
			action:    "enable",
			expectCmd: true,
		},
		{
			name:      "Disable service with 'd'",
			key:       tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")},
			action:    "disable",
			expectCmd: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewServicesScreen()
			screen.SetSize(80, 24)
			screen.filteredServices = createTestServices()
			screen.manager = &systemd.Manager{}
			screen.cursor = 0

			_, cmd := screen.Update(tt.key)

			if tt.expectCmd && cmd == nil {
				t.Error("expected command to be returned, got nil")
			}
		})
	}
}

func TestServicesScreen_ServiceActionsNoServices(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	// No services
	screen.manager = &systemd.Manager{}

	// Try various action keys - should not panic
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
}

func TestServicesScreen_ActionResultMsg(t *testing.T) {
	tests := []struct {
		name            string
		msg             ServiceActionResultMsg
		expectedType    string
		expectedInMsg   string
	}{
		{
			name: "Success result",
			msg: ServiceActionResultMsg{
				Name:    "test-service",
				Action:  "start",
				Success: true,
			},
			expectedType:  "success",
			expectedInMsg: "completed successfully",
		},
		{
			name: "Failure result",
			msg: ServiceActionResultMsg{
				Name:    "test-service",
				Action:  "start",
				Success: false,
				Error:   "permission denied",
			},
			expectedType:  "error",
			expectedInMsg: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewServicesScreen()
			screen.manager = &systemd.Manager{}

			screen.Update(tt.msg)

			if screen.statusMessageType != tt.expectedType {
				t.Errorf("statusMessageType = %q, want %q", screen.statusMessageType, tt.expectedType)
			}

			if !strings.Contains(screen.statusMessage, tt.expectedInMsg) {
				t.Errorf("statusMessage = %q, should contain %q", screen.statusMessage, tt.expectedInMsg)
			}
		})
	}
}

func TestServicesScreen_LogFiltering(t *testing.T) {
	screen := NewServicesScreen()
	screen.logs = `2024-01-01 10:00:00 ERROR Something went wrong
2024-01-01 10:00:01 INFO Starting service
2024-01-01 10:00:02 WARNING Low disk space
2024-01-01 10:00:03 DEBUG Debug message here
2024-01-01 10:00:04 INFO Another info message`

	tests := []struct {
		name          string
		logFilter     string
		expectedLines int
	}{
		{
			name:          "Filter all logs",
			logFilter:     "all",
			expectedLines: 5,
		},
		{
			name:          "Filter error logs",
			logFilter:     "error",
			expectedLines: 1, // Only ERROR line
		},
		{
			name:          "Filter warning logs",
			logFilter:     "warning",
			expectedLines: 1, // Only WARNING line
		},
		{
			name:          "Filter info logs",
			logFilter:     "info",
			expectedLines: 2, // INFO lines
		},
		{
			name:          "Filter debug logs",
			logFilter:     "debug",
			expectedLines: 1, // DEBUG line
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen.logFilter = tt.logFilter
			filtered := screen.filterLogs()

			// Count non-empty lines
			lines := strings.Split(filtered, "\n")
			nonEmptyLines := 0
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					nonEmptyLines++
				}
			}

			if nonEmptyLines != tt.expectedLines {
				t.Errorf("filtered lines = %d, want %d", nonEmptyLines, tt.expectedLines)
			}
		})
	}
}

func TestServicesScreen_CycleLogFilter(t *testing.T) {
	screen := NewServicesScreen()

	// Test log filter cycling order
	expectedFilters := []string{
		"all",
		"error",
		"warning",
		"info",
		"debug",
		"all", // Cycles back to all
	}

	for i, expected := range expectedFilters {
		if screen.logFilter != expected {
			t.Errorf("step %d: logFilter = %q, want %q", i, screen.logFilter, expected)
		}
		screen.cycleLogFilter()
	}
}

func TestServicesScreen_EscapeKey(t *testing.T) {
	tests := []struct {
		name         string
		initialMode  string
		expectedMode string
		shouldGoBack bool
	}{
		{
			name:         "Escape from list mode",
			initialMode:  ServicesModeList,
			expectedMode: ServicesModeList,
			shouldGoBack: true,
		},
		{
			name:         "Escape from details mode",
			initialMode:  ServicesModeDetails,
			expectedMode: ServicesModeList,
			shouldGoBack: false,
		},
		{
			name:         "Escape from logs mode",
			initialMode:  ServicesModeLogs,
			expectedMode: ServicesModeDetails,
			shouldGoBack: false,
		},
		{
			name:         "Escape from actions mode",
			initialMode:  ServicesModeActions,
			expectedMode: ServicesModeList,
			shouldGoBack: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewServicesScreen()
			screen.SetSize(80, 24)
			screen.mode = tt.initialMode
			screen.filteredServices = createTestServices()
			screen.selectedService = &screen.filteredServices[0]

			// Reset goBack
			screen.goBack = false

			screen.Update(tea.KeyMsg{Type: tea.KeyEsc})

			if screen.mode != tt.expectedMode {
				t.Errorf("mode = %q, want %q", screen.mode, tt.expectedMode)
			}

			if screen.ShouldGoBack() != tt.shouldGoBack {
				t.Errorf("ShouldGoBack() = %v, want %v", screen.ShouldGoBack(), tt.shouldGoBack)
			}
		})
	}
}

func TestServicesScreen_GoBack(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)

	// Initially should not go back
	if screen.ShouldGoBack() {
		t.Error("ShouldGoBack() = true initially, want false")
	}

	// Trigger go back
	screen.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if !screen.ShouldGoBack() {
		t.Error("ShouldGoBack() = false after escape, want true")
	}

	// Reset go back
	screen.ResetGoBack()

	if screen.ShouldGoBack() {
		t.Error("ShouldGoBack() = true after reset, want false")
	}
}

func TestServicesScreen_ResetGoBack(t *testing.T) {
	screen := NewServicesScreen()
	screen.goBack = true

	screen.ResetGoBack()

	if screen.goBack {
		t.Error("goBack should be false after ResetGoBack")
	}
}

func TestServicesScreen_View(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.services = createTestServices()
	screen.filteredServices = screen.services

	view := screen.View()

	// Check title is rendered
	if !strings.Contains(view, "Service Status") {
		t.Error("View() should contain 'Service Status' title")
	}

	// Check service names are rendered
	for _, svc := range screen.filteredServices {
		if !strings.Contains(view, svc.Name) {
			t.Errorf("View() should contain service name '%s'", svc.Name)
		}
	}

	// Check help text is present
	if !strings.Contains(view, "navigate") {
		t.Error("View() should contain help text for navigation")
	}

	// Check selection marker is present
	if !strings.Contains(view, "▸") {
		t.Error("View() should contain selection marker '▸'")
	}
}

func TestServicesScreen_ViewEmpty(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	// No services

	view := screen.View()

	// Check empty state message
	if !strings.Contains(view, "No services match the current filter") {
		t.Error("View() should contain 'No services match the current filter' message")
	}

	// Check hint
	if !strings.Contains(view, "Add mounts or sync jobs") {
		t.Error("View() should contain hint to add mounts or sync jobs")
	}
}

func TestServicesScreen_ViewLoading(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.loading = true

	view := screen.View()

	// Check loading message
	if !strings.Contains(view, "Loading services") {
		t.Error("View() should contain 'Loading services' message")
	}
}

func TestServicesScreen_ViewWithError(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.statusMessage = "Error: service not found"
	screen.statusMessageType = "error"

	view := screen.View()

	// Check error is rendered
	if !strings.Contains(view, "service not found") {
		t.Error("View() should contain error message")
	}
}

func TestServicesScreen_ViewWithSuccess(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.statusMessage = "Service started successfully"
	screen.statusMessageType = "success"

	view := screen.View()

	// Check success message is rendered
	if !strings.Contains(view, "Service started successfully") {
		t.Error("View() should contain success message")
	}
}

func TestServicesScreen_ViewDetailsMode(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeDetails
	services := createTestServices()
	screen.selectedService = &services[0]

	view := screen.View()

	// Check details view is rendered
	if !strings.Contains(view, "Service Details") {
		t.Error("View() should contain 'Service Details' title in details mode")
	}

	if !strings.Contains(view, services[0].Name) {
		t.Error("View() should contain selected service name")
	}
}

func TestServicesScreen_ViewLogsMode(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeLogs
	services := createTestServices()
	screen.selectedService = &services[0]
	screen.logs = "Sample log line 1\nSample log line 2"

	view := screen.View()

	// Check logs view is rendered - title changes to "Logs - <service>" when service is selected
	if !strings.Contains(view, "Logs") {
		t.Error("View() should contain 'Logs' title in logs mode")
	}

	if !strings.Contains(view, "Filter:") {
		t.Error("View() should contain filter indicator in logs mode")
	}
}

func TestServicesScreen_ViewLogsModeLoading(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeLogs
	services := createTestServices()
	screen.selectedService = &services[0]
	screen.logsLoading = true

	view := screen.View()

	// Check loading message
	if !strings.Contains(view, "Loading logs") {
		t.Error("View() should contain 'Loading logs' message when logs are loading")
	}
}

func TestServicesScreen_ViewActionsMode(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeActions
	services := createTestServices()
	screen.selectedService = &services[0]

	view := screen.View()

	// Check actions view is rendered - title changes to "Actions - <service>" when service is selected
	if !strings.Contains(view, "Actions") {
		t.Error("View() should contain 'Actions' title in actions mode")
	}

	// Check action options
	expectedActions := []string{"Start", "Stop", "Restart", "Enable", "Disable", "View Logs", "Back"}
	for _, action := range expectedActions {
		if !strings.Contains(view, action) {
			t.Errorf("View() should contain action '%s'", action)
		}
	}
}

func TestServicesScreen_ActionsModeNavigation(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeActions
	services := createTestServices()
	screen.selectedService = &services[0]

	// Test navigation in actions menu
	// Initial cursor should be 0
	if screen.actionCursor != 0 {
		t.Errorf("initial actionCursor = %d, want 0", screen.actionCursor)
	}

	// Move down
	screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	if screen.actionCursor != 1 {
		t.Errorf("actionCursor after down = %d, want 1", screen.actionCursor)
	}

	// Move up
	screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	if screen.actionCursor != 0 {
		t.Errorf("actionCursor after up = %d, want 0", screen.actionCursor)
	}

	// Test vim navigation
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if screen.actionCursor != 1 {
		t.Errorf("actionCursor after 'j' = %d, want 1", screen.actionCursor)
	}

	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if screen.actionCursor != 0 {
		t.Errorf("actionCursor after 'k' = %d, want 0", screen.actionCursor)
	}
}

func TestServicesScreen_ActionsModeSelection(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeActions
	screen.manager = &systemd.Manager{}
	services := createTestServices()
	screen.selectedService = &services[0]

	// Test selecting "Back" (last option, index 6)
	screen.actionCursor = 6
	screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if screen.mode != ServicesModeList {
		t.Errorf("mode after selecting Back = %q, want %q", screen.mode, ServicesModeList)
	}

	if screen.showActions {
		t.Error("showActions should be false after selecting Back")
	}
}

func TestServicesScreen_DetailsModeActions(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeDetails
	screen.manager = &systemd.Manager{}
	services := createTestServices()
	screen.selectedService = &services[0]

	// Test action keys in details mode
	actionKeys := []string{"s", "x", "r", "e", "d", "l"}
	for _, key := range actionKeys {
		_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		// Most actions should return a command (except 'l' which needs logs)
		if key != "l" && cmd == nil {
			t.Errorf("key %q should return a command", key)
		}
	}
}

func TestServicesScreen_DetailsModeRefresh(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeDetails
	screen.manager = &systemd.Manager{}
	services := createTestServices()
	screen.selectedService = &services[0]

	// Test refresh key
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})

	if cmd == nil {
		t.Error("Refresh key 'R' should return a command")
	}

	if !screen.loading {
		t.Error("loading should be true after refresh")
	}
}

func TestServicesScreen_RefreshKey(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.filteredServices = createTestServices()
	screen.manager = &systemd.Manager{}

	// Press 'R' (uppercase) to refresh
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})

	// Should return a command
	if cmd == nil {
		t.Error("Update should return a command for refresh")
	}

	if !screen.loading {
		t.Error("loading should be true after refresh")
	}
}

func TestServicesScreen_RefreshKeyCtrlR(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.filteredServices = createTestServices()
	screen.manager = &systemd.Manager{}

	// Press Ctrl+R to refresh
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyCtrlR})

	// Should return a command
	if cmd == nil {
		t.Error("Update should return a command for Ctrl+R refresh")
	}

	if !screen.loading {
		t.Error("loading should be true after refresh")
	}
}

func TestServicesScreen_Init(t *testing.T) {
	screen := NewServicesScreen()

	cmd := screen.Init()

	// Init should return a command (loadServices)
	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestServicesScreen_SetServices(t *testing.T) {
	screen := NewServicesScreen()
	cfg := &config.Config{}
	mgr := &systemd.Manager{}

	screen.SetServices(cfg, mgr)

	if screen.cfg != cfg {
		t.Error("cfg should be set")
	}
	if screen.manager != mgr {
		t.Error("manager should be set")
	}
}

func TestServicesScreen_ErrorMsg(t *testing.T) {
	screen := NewServicesScreen()
	screen.loading = true

	msg := ServicesErrorMsg{Err: errTestServiceNotFound}
	screen.Update(msg)

	// Verify error message was set
	if !strings.Contains(screen.statusMessage, "Error:") {
		t.Error("statusMessage should contain 'Error:'")
	}

	// Verify status message type
	if screen.statusMessageType != "error" {
		t.Errorf("statusMessageType = %q, want 'error'", screen.statusMessageType)
	}

	// Verify loading is false
	if screen.loading {
		t.Error("loading should be false after error")
	}
}

func TestServicesScreen_RefreshServicesMsg(t *testing.T) {
	screen := NewServicesScreen()
	screen.manager = &systemd.Manager{}

	msg := RefreshServicesMsg{}
	_, cmd := screen.Update(msg)

	// Should return a command
	if cmd == nil {
		t.Error("RefreshServicesMsg should return a command")
	}

	// Should set loading to true
	if !screen.loading {
		t.Error("loading should be true after RefreshServicesMsg")
	}
}

func TestServicesScreen_ServiceLogsLoadedMsg(t *testing.T) {
	screen := NewServicesScreen()
	screen.logsLoading = true

	msg := ServiceLogsLoadedMsg{
		Name: "test-service",
		Logs: "Log line 1\nLog line 2",
	}
	screen.Update(msg)

	// Verify logs were set
	if screen.logs != msg.Logs {
		t.Errorf("logs = %q, want %q", screen.logs, msg.Logs)
	}

	// Verify loading is false
	if screen.logsLoading {
		t.Error("logsLoading should be false after logs loaded")
	}
}

func TestServicesScreen_EnterNoServices(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	// No services

	// Press enter
	screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should stay in list mode
	if screen.mode != ServicesModeList {
		t.Errorf("mode = %q, want %q (ServicesModeList)", screen.mode, ServicesModeList)
	}
}

func TestServicesScreen_LogsNoService(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeLogs
	// No selected service

	// Press 'f' to cycle filter
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})

	// Should not panic, mode should stay in logs
	if screen.mode != ServicesModeLogs {
		t.Errorf("mode = %q, want %q", screen.mode, ServicesModeLogs)
	}
}

func TestServicesScreen_DetailsModeNoService(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeDetails
	// No selected service

	view := screen.View()

	// Should show error message
	if !strings.Contains(view, "No service selected") {
		t.Error("View() should contain 'No service selected' error")
	}
}

func TestServicesScreen_ViewSystemdStatus(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.systemdStatus = SystemdStatus{
		Available:      true,
		FailedUnits:     2,
		ActiveServices: 3,
		ActiveTimers:   1,
	}

	view := screen.renderSystemdStatus()

	// Check systemd status is rendered
	if !strings.Contains(view, "Systemd:") {
		t.Error("renderSystemdStatus() should contain 'Systemd:'")
	}

	if !strings.Contains(view, "Available") {
		t.Error("renderSystemdStatus() should contain 'Available'")
	}

	if !strings.Contains(view, "Failed:") {
		t.Error("renderSystemdStatus() should contain 'Failed:'")
	}
}

func TestServicesScreen_ViewSystemdStatusUnavailable(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.systemdStatus = SystemdStatus{
		Available: false,
	}

	view := screen.renderSystemdStatus()

	// Check unavailable status is rendered
	if !strings.Contains(view, "Unavailable") {
		t.Error("renderSystemdStatus() should contain 'Unavailable' when systemd is not available")
	}
}

func TestGetFilterDescription(t *testing.T) {
	tests := []struct {
		filter    string
		expected  string
	}{
		{FilterAll, "All"},
		{FilterRunning, "Running"},
		{FilterStopped, "Stopped"},
		{FilterFailed, "Failed"},
		{FilterMounts, "Mounts"},
		{FilterSyncJobs, "Sync Jobs"},
		{"unknown", "All"}, // Default case
	}

	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			result := getFilterDescription(tt.filter)
			if result != tt.expected {
				t.Errorf("getFilterDescription(%q) = %q, want %q", tt.filter, result, tt.expected)
			}
		})
	}
}

func TestServicesScreen_ApplyFilterResetsCursor(t *testing.T) {
	screen := NewServicesScreen()
	screen.services = createTestServices()
	screen.cursor = 10 // Out of bounds

	// Apply filter should reset cursor
	screen.applyFilter()

	if screen.cursor >= len(screen.filteredServices) {
		t.Errorf("cursor = %d, should be less than %d", screen.cursor, len(screen.filteredServices))
	}
}

func TestServicesScreen_ApplyFilterEmptyResult(t *testing.T) {
	screen := NewServicesScreen()
	screen.services = createTestServices()
	screen.filter = FilterFailed
	screen.cursor = 0

	// First, let's test with failed filter which has 1 result
	screen.applyFilter()

	if len(screen.filteredServices) != 1 {
		t.Errorf("filtered services = %d, want 1", len(screen.filteredServices))
	}

	// Now test with a filter that has no results
	// Create services with no running ones
	screen.services = []ServiceInfo{
		{Name: "svc1", Status: "inactive", Type: "mount"},
		{Name: "svc2", Status: "failed", Type: "sync"},
	}
	screen.filter = FilterRunning
	screen.applyFilter()

	if len(screen.filteredServices) != 0 {
		t.Errorf("filtered services = %d, want 0", len(screen.filteredServices))
	}

	// Cursor should be reset to 0
	if screen.cursor != 0 {
		t.Errorf("cursor = %d, want 0", screen.cursor)
	}
}

func TestServicesScreen_RenderLogLine(t *testing.T) {
	screen := NewServicesScreen()

	tests := []struct {
		name     string
		line     string
	}{
		{"Error line", "2024-01-01 ERROR Something failed"},
		{"Warning line", "2024-01-01 WARNING Check this"},
		{"Info line", "2024-01-01 INFO Normal message"},
		{"Debug line", "2024-01-01 DEBUG Debug info"},
		{"Normal line", "2024-01-01 Normal output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := screen.renderLogLine(tt.line)
			if result == "" {
				t.Error("renderLogLine should return non-empty string")
			}
		})
	}
}

func TestServicesScreen_LoadDetailedStatus(t *testing.T) {
	screen := NewServicesScreen()
	screen.manager = &systemd.Manager{}
	services := createTestServices()
	screen.selectedService = &services[0]

	// Should not panic even with real manager (will fail gracefully)
	screen.loadDetailedStatus()

	// detailedStatus may be nil if manager fails, which is fine
}

func TestServicesScreen_LoadDetailedStatusNilManager(t *testing.T) {
	screen := NewServicesScreen()
	// manager is nil
	services := createTestServices()
	screen.selectedService = &services[0]

	// Should not panic
	screen.loadDetailedStatus()

	// detailedStatus should be nil
	if screen.detailedStatus != nil {
		t.Error("detailedStatus should be nil with nil manager")
	}
}

func TestServicesScreen_LoadDetailedStatusNilService(t *testing.T) {
	screen := NewServicesScreen()
	screen.manager = &systemd.Manager{}
	// selectedService is nil

	// Should not panic
	screen.loadDetailedStatus()

	// detailedStatus should be nil
	if screen.detailedStatus != nil {
		t.Error("detailedStatus should be nil with nil selectedService")
	}
}

func TestServicesScreen_DetailsViewForSyncJob(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeDetails

	// Use a sync job
	services := createTestServices()
	screen.selectedService = &services[2] // rclone-sync-backup

	view := screen.View()

	// Check sync-specific fields are rendered
	if !strings.Contains(view, "Source:") {
		t.Error("View() should contain 'Source:' for sync job")
	}

	if !strings.Contains(view, "Destination:") {
		t.Error("View() should contain 'Destination:' for sync job")
	}

	if !strings.Contains(view, "Timer:") {
		t.Error("View() should contain 'Timer:' for sync job")
	}

	if !strings.Contains(view, "Next Run:") {
		t.Error("View() should contain 'Next Run:' for sync job")
	}
}

func TestServicesScreen_DetailsViewForMount(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeDetails

	// Use a mount
	services := createTestServices()
	screen.selectedService = &services[0] // rclone-mount-gdrive

	view := screen.View()

	// Check mount-specific fields are rendered
	if !strings.Contains(view, "Mount Point:") {
		t.Error("View() should contain 'Mount Point:' for mount")
	}

	if !strings.Contains(view, "Remote:") {
		t.Error("View() should contain 'Remote:' for mount")
	}
}

func TestServicesScreen_EnableDisableSyncJob(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeList
	screen.manager = &systemd.Manager{}

	// Use a sync job
	services := createTestServices()
	screen.filteredServices = services
	screen.cursor = 2 // rclone-sync-backup

	// Test enable - should use .timer for sync jobs
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if cmd == nil {
		t.Error("Enable should return a command for sync job")
	}

	// Test disable - should use .timer for sync jobs
	_, cmd = screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if cmd == nil {
		t.Error("Disable should return a command for sync job")
	}
}

func TestServicesScreen_EnableDisableMount(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeList
	screen.manager = &systemd.Manager{}

	// Use a mount
	services := createTestServices()
	screen.filteredServices = services
	screen.cursor = 0 // rclone-mount-gdrive

	// Test enable - should use .service for mounts
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if cmd == nil {
		t.Error("Enable should return a command for mount")
	}

	// Test disable - should use .service for mounts
	_, cmd = screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if cmd == nil {
		t.Error("Disable should return a command for mount")
	}
}

func TestServicesScreen_ActionsModeEnableDisableSync(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeActions
	screen.manager = &systemd.Manager{}

	// Use a sync job
	services := createTestServices()
	screen.selectedService = &services[2] // rclone-sync-backup

	// Test enable action (index 3)
	screen.actionCursor = 3
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("Enable action should return a command for sync job")
	}

	// Reset and test disable action (index 4)
	screen.mode = ServicesModeActions
	screen.actionCursor = 4
	_, cmd = screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("Disable action should return a command for sync job")
	}
}

func TestServicesScreen_ActionsModeViewLogs(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeActions
	screen.manager = &systemd.Manager{}

	services := createTestServices()
	screen.selectedService = &services[0]

	// Select "View Logs" (index 5)
	screen.actionCursor = 5
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Error("View Logs action should return a command")
	}

	if !screen.logsLoading {
		t.Error("logsLoading should be true after View Logs action")
	}
}

func TestServicesScreen_FilterLogsWithEmptyLogs(t *testing.T) {
	screen := NewServicesScreen()
	screen.logs = ""
	screen.logFilter = "error"

	// Should return empty string
	result := screen.filterLogs()
	if result != "" {
		t.Errorf("filterLogs() = %q, want empty string", result)
	}
}

func TestServicesScreen_FilterLogsWithUnknownFilter(t *testing.T) {
	screen := NewServicesScreen()
	screen.logs = "Some log content"
	screen.logFilter = "unknown"

	// Should return all logs
	result := screen.filterLogs()
	if result != screen.logs {
		t.Errorf("filterLogs() with unknown filter = %q, want %q", result, screen.logs)
	}
}

func TestServicesScreen_ViewServiceList(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.services = createTestServices()
	screen.filteredServices = screen.services

	list := screen.renderServiceList()

	// Check header is rendered
	if !strings.Contains(list, "Service") {
		t.Error("renderServiceList should contain 'Service' header")
	}

	if !strings.Contains(list, "Type") {
		t.Error("renderServiceList should contain 'Type' header")
	}

	if !strings.Contains(list, "Status") {
		t.Error("renderServiceList should contain 'Status' header")
	}

	if !strings.Contains(list, "Enabled") {
		t.Error("renderServiceList should contain 'Enabled' header")
	}
}

func TestServicesScreen_ViewServiceListWithTimer(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.services = []ServiceInfo{
		{
			Name:        "rclone-sync-backup",
			Type:        "sync",
			Status:      "active",
			TimerActive: true,
		},
	}
	screen.filteredServices = screen.services

	list := screen.renderServiceList()

	// Check timer indicator is shown
	if !strings.Contains(list, "timer") {
		t.Error("renderServiceList should show 'timer' for sync job with active timer")
	}
}

func TestServicesScreen_SystemdStatusStruct(t *testing.T) {
	status := SystemdStatus{
		Available:      true,
		FailedUnits:     2,
		SessionType:     "user@.service",
		ActiveServices: 5,
		ActiveTimers:   3,
	}

	if !status.Available {
		t.Error("Available should be true")
	}

	if status.FailedUnits != 2 {
		t.Errorf("FailedUnits = %d, want 2", status.FailedUnits)
	}

	if status.ActiveServices != 5 {
		t.Errorf("ActiveServices = %d, want 5", status.ActiveServices)
	}

	if status.ActiveTimers != 3 {
		t.Errorf("ActiveTimers = %d, want 3", status.ActiveTimers)
	}
}

func TestServicesScreen_ServiceInfoStruct(t *testing.T) {
	now := time.Now()
	svc := ServiceInfo{
		Name:        "test-service",
		Type:        "sync",
		Status:      "active",
		SubState:    "running",
		Enabled:     true,
		Source:      "remote:/source",
		Destination: "/local/dest",
		NextRun:     now,
		LastRun:     now.Add(-1 * time.Hour),
		TimerActive: true,
	}

	if svc.Name != "test-service" {
		t.Errorf("Name = %q, want 'test-service'", svc.Name)
	}

	if svc.Type != "sync" {
		t.Errorf("Type = %q, want 'sync'", svc.Type)
	}

	if !svc.TimerActive {
		t.Error("TimerActive should be true")
	}
}

func TestServicesScreen_Messages(t *testing.T) {
	// Test ServicesLoadedMsg
	loadedMsg := ServicesLoadedMsg{Services: createTestServices()}
	if len(loadedMsg.Services) != 4 {
		t.Errorf("ServicesLoadedMsg Services = %d, want 4", len(loadedMsg.Services))
	}

	// Test ServiceActionMsg
	actionMsg := ServiceActionMsg{Name: "test", Action: "start"}
	if actionMsg.Name != "test" {
		t.Errorf("ServiceActionMsg Name = %q, want 'test'", actionMsg.Name)
	}

	// Test ServiceActionResultMsg
	resultMsg := ServiceActionResultMsg{Name: "test", Action: "start", Success: true}
	if !resultMsg.Success {
		t.Error("ServiceActionResultMsg Success should be true")
	}

	// Test ServiceLogsMsg
	logsMsg := ServiceLogsMsg{Name: "test-service"}
	if logsMsg.Name != "test-service" {
		t.Errorf("ServiceLogsMsg Name = %q, want 'test-service'", logsMsg.Name)
	}

	// Test ServiceLogsLoadedMsg
	logsLoadedMsg := ServiceLogsLoadedMsg{Name: "test", Logs: "log content"}
	if logsLoadedMsg.Logs != "log content" {
		t.Errorf("ServiceLogsLoadedMsg Logs = %q, want 'log content'", logsLoadedMsg.Logs)
	}

	// Test ServicesErrorMsg
	errMsg := ServicesErrorMsg{Err: errTestServiceNotFound}
	if errMsg.Err != errTestServiceNotFound {
		t.Error("ServicesErrorMsg Err should be set")
	}

	// Test RefreshServicesMsg
	_ = RefreshServicesMsg{} // Should compile
}

func TestServicesScreen_DoServiceAction(t *testing.T) {
	screen := NewServicesScreen()
	screen.manager = &systemd.Manager{}

	// Get a command for start action
	cmd := screen.doServiceAction("test.service", "start")
	if cmd == nil {
		t.Error("doServiceAction should return a command")
	}

	// Execute the command - it will likely fail since systemd isn't available
	// but we're testing that the command structure is correct
	msg := cmd()
	result, ok := msg.(ServiceActionResultMsg)
	if !ok {
		t.Fatalf("expected ServiceActionResultMsg, got %T", msg)
	}

	// The action will fail because systemd isn't available, but that's expected
	// We're just testing the message structure
	if result.Name != "test.service" {
		t.Errorf("result Name = %q, want 'test.service'", result.Name)
	}

	if result.Action != "start" {
		t.Errorf("result Action = %q, want 'start'", result.Action)
	}
}

func TestServicesScreen_LoadServiceLogs(t *testing.T) {
	screen := NewServicesScreen()
	screen.manager = &systemd.Manager{}

	// Get a command for loading logs
	cmd := screen.loadServiceLogs("test.service")
	if cmd == nil {
		t.Error("loadServiceLogs should return a command")
	}

	// Execute the command
	msg := cmd()
	logsMsg, ok := msg.(ServiceLogsLoadedMsg)
	if !ok {
		t.Fatalf("expected ServiceLogsLoadedMsg, got %T", msg)
	}

	// The logs loading will fail because systemd isn't available
	// but we're testing that the message structure is correct
	if logsMsg.Name != "test.service" {
		t.Errorf("logsMsg Name = %q, want 'test.service'", logsMsg.Name)
	}
}

func TestServicesScreen_ViewDefaultMode(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.services = createTestServices()
	screen.filteredServices = screen.services

	// Set an unknown mode
	screen.mode = "unknown"

	view := screen.View()

	// Should fall back to list view
	if !strings.Contains(view, "Service Status") {
		t.Error("View() with unknown mode should fall back to list view")
	}
}

func TestServicesScreen_CursorBoundsCheck(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.services = createTestServices()
	screen.filteredServices = screen.services

	// Set cursor to last item
	screen.cursor = len(screen.filteredServices) - 1

	// Try to move down - should stay at last
	screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	if screen.cursor != len(screen.filteredServices)-1 {
		t.Errorf("cursor = %d, want %d", screen.cursor, len(screen.filteredServices)-1)
	}

	// Set cursor to first item
	screen.cursor = 0

	// Try to move up - should stay at 0
	screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	if screen.cursor != 0 {
		t.Errorf("cursor = %d, want 0", screen.cursor)
	}
}

func TestServicesScreen_ActionCursorBoundsCheck(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.mode = ServicesModeActions
	services := createTestServices()
	screen.selectedService = &services[0]

	// There are 7 actions (0-6)
	// Set cursor to last
	screen.actionCursor = 6

	// Try to move down - should stay at last
	screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	if screen.actionCursor != 6 {
		t.Errorf("actionCursor = %d, want 6", screen.actionCursor)
	}

	// Set cursor to first
	screen.actionCursor = 0

	// Try to move up - should stay at 0
	screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	if screen.actionCursor != 0 {
		t.Errorf("actionCursor = %d, want 0", screen.actionCursor)
	}
}

func TestServicesScreen_StatusIndicatorInList(t *testing.T) {
	screen := NewServicesScreen()
	screen.SetSize(80, 24)
	screen.services = []ServiceInfo{
		{Name: "active-svc", Status: "active", Type: "mount"},
		{Name: "inactive-svc", Status: "inactive", Type: "mount"},
		{Name: "failed-svc", Status: "failed", Type: "sync"},
	}
	screen.filteredServices = screen.services

	view := screen.View()

	// All services should be shown
	for _, svc := range screen.services {
		if !strings.Contains(view, svc.Name) {
			t.Errorf("View() should contain service '%s'", svc.Name)
		}
	}
}

func TestServicesScreen_SelectedServiceAfterFilter(t *testing.T) {
	screen := NewServicesScreen()
	screen.services = createTestServices()
	screen.filter = FilterMounts
	screen.applyFilter()

	// Select first mount
	screen.cursor = 0

	// Press enter to view details
	screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if screen.selectedService == nil {
		t.Fatal("selectedService should not be nil")
	}

	// Selected service should be a mount
	if screen.selectedService.Type != "mount" {
		t.Errorf("selectedService Type = %q, want 'mount'", screen.selectedService.Type)
	}
}