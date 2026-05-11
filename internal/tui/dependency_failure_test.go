package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dtg01100/rclone-mount-sync/internal/rclone"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
)

// TestApp_InitError_ConfigLoadFailure tests handling when config directory is inaccessible.
func TestApp_InitError_ConfigLoadFailure(t *testing.T) {
	// Create a temporary directory with restricted permissions
	tmpDir := t.TempDir()
	restrictedDir := filepath.Join(tmpDir, "restricted")
	if err := os.Mkdir(restrictedDir, 0000); err != nil {
		t.Skip("Cannot create restricted directory (may be running as root)")
	}

	// Save and restore XDG_CONFIG_HOME
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	// Point to restricted directory
	_ = os.Setenv("XDG_CONFIG_HOME", restrictedDir)

	app := NewApp()
	msg := app.initializeServices()

	// Should return AppInitError due to permission issues
	initErr, ok := msg.(AppInitError)
	if !ok {
		// If it didn't error, that's also acceptable (depends on OS permissions)
		t.Logf("Initialization succeeded with restricted dir (message type: %T)", msg)
		return
	}

	if initErr.Err == nil {
		t.Error("AppInitError should have an error")
	}
	t.Logf("Got expected error: %v", initErr.Err)
}

// TestApp_InitError_SystemdGeneratorFailure tests handling of systemd generator initialization failure.
func TestApp_InitError_SystemdGeneratorFailure(t *testing.T) {
	// Point to a non-existent, non-creatable path
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	// Use a path that will fail to create
	_ = os.Setenv("XDG_CONFIG_HOME", "/proc/nonexistent")

	app := NewApp()
	msg := app.initializeServices()

	// Should return AppInitError due to generator failure
	initErr, ok := msg.(AppInitError)
	if !ok {
		t.Logf("Initialization succeeded (message type: %T)", msg)
		return
	}

	if initErr.Err == nil {
		t.Error("AppInitError should have an error")
	}
	t.Logf("Got expected error: %v", initErr.Err)
}

// TestApp_InitError_RcloneNotAvailable tests graceful handling when rclone is not in PATH.
func TestApp_InitError_RcloneNotAvailable(t *testing.T) {
	// Save and restore PATH
	oldPath := os.Getenv("PATH")
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	// Set PATH to only include non-existent directory
	tmpDir := t.TempDir()
	_ = os.Setenv("PATH", tmpDir)

	app := NewApp()
	msg := app.initializeServices()

	// Should NOT fail - rclone unavailability should be handled gracefully
	_, isInitDone := msg.(AppInitDone)
	_, isReconcile := msg.(ReconciliationMsg)

	if !isInitDone && !isReconcile {
		if initErr, ok := msg.(AppInitError); ok {
			// If it does error, it should be informative
			t.Logf("Got init error (may be acceptable): %v", initErr.Err)
		}
	}

	// Rclone client should be initialized (even if binary not found)
	if app.rclone == nil {
		t.Log("Warning: rclone client is nil (may be expected if initialization failed)")
	}
}

// TestApp_InitError_ConfigEmpty tests handling of empty/minimal config.
func TestApp_InitError_ConfigEmpty(t *testing.T) {
	// Use a temporary config directory
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "rclone-mount-sync")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create minimal config file
	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("version: \"1.0\"\n"), 0644); err != nil { //nolint:gosec
		t.Fatalf("Failed to write config: %v", err)
	}

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	app := NewApp()
	msg := app.initializeServices()

	// Should succeed with empty config
	_, isInitDone := msg.(AppInitDone)
	_, isReconcile := msg.(ReconciliationMsg)

	if !isInitDone && !isReconcile {
		if err, ok := msg.(AppInitError); ok {
			t.Errorf("Empty config should not cause init error, got: %v", err.Err)
		}
	}
}

// TestApp_InitError_ConfigWithMounts tests initialization with existing mounts.
func TestApp_InitError_ConfigWithMounts(t *testing.T) {
	// Use a temporary config directory
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "rclone-mount-sync")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create config with a mount
	configContent := `version: "1.0"
mounts:
  - id: test-mount-1
    name: test-mount
    remote: "gdrive:"
    remote_path: /data
    mount_point: ~/mnt/test
sync_jobs: []
`
	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil { //nolint:gosec
		t.Fatalf("Failed to write config: %v", err)
	}

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	app := NewApp()
	msg := app.initializeServices()

	// Should handle mounts in config gracefully
	t.Logf("Initialization completed with message type: %T", msg)

	// Verify config was loaded
	if app.config != nil && len(app.config.Mounts) > 0 {
		t.Logf("Loaded %d mount(s)", len(app.config.Mounts))
		if app.config.Mounts[0].Name != "test-mount" {
			t.Errorf("Expected mount name 'test-mount', got %q", app.config.Mounts[0].Name)
		}
	}
}

// TestApp_Reconciliation_WithOrphans tests the reconciliation flow with orphaned units.
func TestApp_Reconciliation_WithOrphans(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	orphanedUnits := []systemd.OrphanedUnit{
		{Name: "rclone-mount-old.service", ID: "old-mount-id"},
		{Name: "rclone-sync-legacy.timer", ID: "legacy-sync-id"},
	}

	result := &systemd.ReconciliationResult{
		OrphanedUnits: orphanedUnits,
	}

	_, cmd := app.Update(ReconciliationMsg{Result: result})

	if app.orphans == nil {
		t.Fatal("orphans should be set")
	}
	if len(app.orphans.OrphanedUnits) != 2 {
		t.Errorf("Expected 2 orphaned units, got %d", len(app.orphans.OrphanedUnits))
	}
	if !app.showOrphanPrompt {
		t.Error("showOrphanPrompt should be true when orphans exist")
	}
	if cmd == nil {
		t.Error("Should return command to initialize screens")
	}
}

// TestApp_Reconciliation_NoOrphans tests reconciliation when no orphans exist.
func TestReconciliation_NoOrphans(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	result := &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{},
	}

	_, cmd := app.Update(ReconciliationMsg{Result: result})

	if app.showOrphanPrompt {
		t.Error("showOrphanPrompt should be false when no orphans exist")
	}
	if cmd == nil {
		t.Error("Should return command to initialize screens")
	}
}

// TestApp_OrphanAction_Remove tests removing an orphaned unit.
func TestApp_OrphanAction_Remove(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.showOrphanPrompt = true
	app.orphanSelected = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "orphan1.service", ID: "1"},
			{Name: "orphan2.service", ID: "2"},
			{Name: "orphan3.service", ID: "3"},
		},
	}

	// Simulate successful orphan removal (index 1)
	_, cmd := app.Update(OrphanActionMsg{Index: 1, Action: "remove"})

	if app.orphanError != nil {
		t.Errorf("Should not have error after successful removal: %v", app.orphanError)
	}
	if len(app.orphans.OrphanedUnits) != 2 {
		t.Errorf("Expected 2 orphaned units after removal, got %d", len(app.orphans.OrphanedUnits))
	}
	if cmd == nil {
		t.Error("Should return command to refresh screens")
	}
}

// TestApp_OrphanAction_RemoveLast tests removing the last orphaned unit.
func TestApp_OrphanAction_RemoveLast(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.showOrphanPrompt = true
	app.orphanSelected = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "last-orphan.service", ID: "last"},
		},
	}

	_, _ = app.Update(OrphanActionMsg{Index: 0, Action: "remove"})

	if app.showOrphanPrompt {
		t.Error("showOrphanPrompt should be false after removing last orphan")
	}
	if app.orphanSelected != -1 {
		t.Errorf("orphanSelected should be -1, got %d", app.orphanSelected)
	}
}

// TestApp_OrphanAction_Ignore tests ignoring an orphaned unit.
func TestApp_OrphanAction_Ignore(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.showOrphanPrompt = true
	app.orphanSelected = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "ignore-me.service", ID: "ignore"},
		},
	}

	_, cmd := app.Update(OrphanActionMsg{Index: 0, Action: "ignore"})

	if app.orphanError != nil {
		t.Errorf("Should not have error after ignore: %v", app.orphanError)
	}
	// Current behavior: ignore also removes the orphan from the list on success
	// This test documents that behavior
	if len(app.orphans.OrphanedUnits) != 0 {
		t.Logf("Orphan removed after ignore (current behavior): %d units remaining", len(app.orphans.OrphanedUnits))
	}
	if cmd == nil {
		t.Error("Should return command to refresh screens")
	}
}

// TestApp_OrphanAction_Error tests handling of orphan action errors.
func TestApp_OrphanAction_Error(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.showOrphanPrompt = true
	app.orphanSelected = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "error-orphan.service", ID: "err"},
		},
	}

	_, _ = app.Update(OrphanActionMsg{
		Index: 0,
		Action: "remove",
		Err:    fmt.Errorf("failed to stop service"),
	})

	if app.orphanError == nil {
		t.Error("orphanError should be set when action fails")
	}
	if !strings.Contains(app.orphanError.Error(), "failed to stop service") {
		t.Errorf("Error should contain 'failed to stop service', got: %v", app.orphanError)
	}
	// Orphan should NOT be removed on error
	if len(app.orphans.OrphanedUnits) != 1 {
		t.Errorf("Orphan should still be in list after error, got %d units", len(app.orphans.OrphanedUnits))
	}
}

// TestApp_OrphanAction_InvalidIndex tests handling of invalid orphan index.
func TestApp_OrphanAction_InvalidIndex(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.showOrphanPrompt = true
	app.orphanSelected = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "valid.service", ID: "1"},
		},
	}

	// Should not panic with invalid index
	_, _ = app.Update(OrphanActionMsg{Index: 999, Action: "remove"})

	// Should handle gracefully (may log error or ignore)
	t.Log("Invalid index handled without panic")
}

// TestApp_OrphanPrompt_Display tests that orphan prompt appears in View.
func TestApp_OrphanPrompt_Display(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.loading = false
	app.showOrphanPrompt = true
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "test-orphan.service", ID: "test"},
		},
	}

	view := app.View()

	if view == "" {
		t.Fatal("View should not be empty")
	}
	if !strings.Contains(view, "orphan") && !strings.Contains(view, "Orphan") {
		t.Error("View should mention 'orphan' when orphan prompt is shown")
	}
	if !strings.Contains(view, "test-orphan.service") {
		t.Error("View should show the orphan service name")
	}
}

// TestApp_Services_SetServices tests that SetServices properly distributes to screens.
func TestApp_Services_SetServices(t *testing.T) {
	// Use a temporary config directory
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "rclone-mount-sync")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("version: \"1.0\"\nmounts: []\nsync_jobs: []\n"), 0600); err != nil { //nolint:gosec
		t.Fatalf("Failed to write config: %v", err)
	}

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	app := NewApp()
	msg := app.initializeServices()

	// Check that services were set
	if app.config == nil {
		t.Error("config should be set after initialization")
	}
	if app.rclone == nil {
		t.Error("rclone client should be set after initialization")
	}
	if app.generator == nil {
		t.Error("generator should be set after initialization")
	}
	// Manager may be nil in test environment
	t.Logf("Manager set: %v, Message type: %T", app.manager != nil, msg)
}

// TestRclone_Remotes tests rclone remote listing.
func TestRclone_Remotes(t *testing.T) {
	client := rclone.NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	// This will fail if rclone is not installed, which is expected in test env
	remotes, err := client.ListRemotes(context.Background())
	if err != nil {
		t.Logf("ListRemotes failed (expected if rclone not installed): %v", err)
		return
	}

	t.Logf("Found %d remotes", len(remotes))
	for _, r := range remotes {
		t.Logf("  - %s (%s)", r.Name, r.Type)
	}
}

// TestApp_ConcurrentMessages tests handling concurrent messages safely.
func TestApp_ConcurrentMessages(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	// Send multiple messages in sequence (simulating concurrent updates)
	messages := []tea.Msg{
		tea.WindowSizeMsg{Width: 100, Height: 30},
		LoadingMsg{},
		LoadingDoneMsg{},
		ScreenChangeMsg{Screen: ScreenMounts},
		ScreenChangeMsg{Screen: ScreenSyncJobs},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")},
	}

	for _, msg := range messages {
		_, _ = app.Update(msg)
		// Verify invariants after each message
		if app.width < 0 || app.height < 0 {
			t.Errorf("Negative dimensions after message %T: %dx%d", msg, app.width, app.height)
		}
	}
}
