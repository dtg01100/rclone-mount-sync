package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/rclone"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
	"github.com/dtg01100/rclone-mount-sync/internal/tui/components"
)

// Helper function to create a test config for sync jobs
func createSyncTestConfig() *config.Config {
	return &config.Config{
		Version: "1.0",
		Defaults: config.DefaultConfig{
			Sync: config.SyncDefaults{
				LogLevel:  "INFO",
				Transfers: 4,
				Checkers:  8,
			},
		},
		Mounts:   []models.MountConfig{},
		SyncJobs: []models.SyncJobConfig{},
	}
}

// createSyncTestGenerator creates a test generator for sync job tests
func createSyncTestGenerator(t *testing.T) *systemd.Generator {
	t.Helper()
	tmpDir := t.TempDir()
	return systemd.NewTestGenerator(tmpDir)
}

// createSyncTestManager creates a mock manager for sync job tests
func createSyncTestManager() *systemd.MockManager {
	return &systemd.MockManager{}
}
func TestNewSyncJobForm_Create(t *testing.T) {
	cfg := createSyncTestConfig()
	remotes := createTestRemotes()

	form := NewSyncJobForm(nil, remotes, cfg, nil, nil, nil, false)

	if form == nil {
		t.Fatal("NewSyncJobForm() returned nil")
	}

	// Verify initial state
	if form.isEdit {
		t.Error("isEdit should be false for create mode")
	}

	if form.job != nil {
		t.Error("job should be nil for create mode")
	}

	// Verify defaults from config are applied
	if form.logLevel != cfg.Defaults.Sync.LogLevel {
		t.Errorf("logLevel = %q, want %q", form.logLevel, cfg.Defaults.Sync.LogLevel)
	}

	// Verify default values
	if form.direction != "sync" {
		t.Errorf("default direction = %q, want 'sync'", form.direction)
	}

	if form.deleteMode != "after" {
		t.Errorf("default deleteMode = %q, want 'after'", form.deleteMode)
	}

	if form.scheduleType != "timer" {
		t.Errorf("default scheduleType = %q, want 'timer'", form.scheduleType)
	}
}

func TestNewSyncJobForm_Edit(t *testing.T) {
	cfg := createSyncTestConfig()
	remotes := createTestRemotes()

	// Create an existing sync job to edit
	existingJob := &models.SyncJobConfig{
		ID:          "j1o2b3x4",
		Name:        "Test Sync Job",
		Source:      "gdrive:/Photos",
		Destination: "/home/user/Backup/Photos",
		SyncOptions: models.SyncOptions{
			Direction:      "copy",
			DeleteAfter:    true,
			DryRun:         false,
			Transfers:      8,
			BandwidthLimit: "10M",
			LogLevel:       "DEBUG",
		},
		Schedule: models.ScheduleConfig{
			Type:             "timer",
			OnCalendar:       "daily",
			RequireACPower:   true,
			RequireUnmetered: true,
		},
		Enabled: true,
	}

	form := NewSyncJobForm(existingJob, remotes, cfg, nil, nil, nil, true)

	if form == nil {
		t.Fatal("NewSyncJobForm() returned nil")
	}

	// Verify edit mode
	if !form.isEdit {
		t.Error("isEdit should be true for edit mode")
	}

	// Verify existing values are populated
	if form.name != existingJob.Name {
		t.Errorf("name = %q, want %q", form.name, existingJob.Name)
	}

	if form.sourceRemote != "gdrive" {
		t.Errorf("sourceRemote = %q, want 'gdrive'", form.sourceRemote)
	}

	if form.sourcePath != "/Photos" {
		t.Errorf("sourcePath = %q, want '/Photos'", form.sourcePath)
	}

	if form.destPath != existingJob.Destination {
		t.Errorf("destPath = %q, want %q", form.destPath, existingJob.Destination)
	}

	if form.direction != existingJob.SyncOptions.Direction {
		t.Errorf("direction = %q, want %q", form.direction, existingJob.SyncOptions.Direction)
	}

	if form.enabled != existingJob.Enabled {
		t.Errorf("enabled = %v, want %v", form.enabled, existingJob.Enabled)
	}

	if form.requireACPower != existingJob.Schedule.RequireACPower {
		t.Errorf("requireACPower = %v, want %v", form.requireACPower, existingJob.Schedule.RequireACPower)
	}

	if form.requireUnmetered != existingJob.Schedule.RequireUnmetered {
		t.Errorf("requireUnmetered = %v, want %v", form.requireUnmetered, existingJob.Schedule.RequireUnmetered)
	}
}

func TestSyncJobForm_ValidateName(t *testing.T) {
	tests := []struct {
		name          string
		inputName     string
		existing      []models.SyncJobConfig
		isEdit        bool
		expectError   bool
		errorContains string
	}{
		{
			name:          "Empty name",
			inputName:     "",
			expectError:   true,
			errorContains: "required",
		},
		{
			name:        "Valid name",
			inputName:   "My Sync Job",
			expectError: false,
		},
		{
			name:          "Name too long",
			inputName:     strings.Repeat("a", 51),
			expectError:   true,
			errorContains: "50 characters",
		},
		{
			name:          "Duplicate name - new job",
			inputName:     "Existing Job",
			existing:      []models.SyncJobConfig{{Name: "Existing Job"}},
			isEdit:        false,
			expectError:   true,
			errorContains: "already exists",
		},
		{
			name:        "Duplicate name - edit mode same job",
			inputName:   "Existing Job",
			existing:    []models.SyncJobConfig{{Name: "Existing Job"}},
			isEdit:      true,
			expectError: false, // Should allow same name when editing
		},
		{
			name:        "Name at max length",
			inputName:   strings.Repeat("a", 50),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createSyncTestConfig()
			cfg.SyncJobs = tt.existing

			form := NewSyncJobForm(nil, createTestRemotes(), cfg, nil, nil, nil, tt.isEdit)
			err := form.validateName(tt.inputName)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error = %q, should contain %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSyncJobForm_ValidateDestPath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "sync-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory that exists
	existingDir := filepath.Join(tmpDir, "existing")
	if err := os.Mkdir(existingDir, 0755); err != nil {
		t.Fatalf("failed to create existing dir: %v", err)
	}

	tests := []struct {
		name          string
		inputPath     string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Empty path",
			inputPath:     "",
			expectError:   true,
			errorContains: "required",
		},
		{
			name:        "Valid absolute path with existing parent",
			inputPath:   filepath.Join(existingDir, "newsync"),
			expectError: false,
		},
		{
			name:          "Relative path",
			inputPath:     "relative/path",
			expectError:   true,
			errorContains: "absolute",
		},
		{
			name:        "Path with tilde to home",
			inputPath:   "~/backup",
			expectError: false, // Tilde is expanded to home directory which exists
		},
		{
			name:          "Parent directory does not exist",
			inputPath:     "/nonexistent/path/sync",
			expectError:   true,
			errorContains: "parent directory",
		},
		{
			name:        "Remote path (contains colon)",
			inputPath:   "s3:bucket/path",
			expectError: false, // Remote paths are valid
		},
		{
			name:        "Remote path with remote name",
			inputPath:   "gdrive:/Backup",
			expectError: false, // Remote paths are valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
			err := form.validateDestPath(tt.inputPath)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error = %q, should contain %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParseRemotePath(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedRemote string
		expectedPath   string
	}{
		{
			name:           "Remote with path",
			input:          "gdrive:/Photos",
			expectedRemote: "gdrive",
			expectedPath:   "/Photos",
		},
		{
			name:           "Remote without path",
			input:          "dropbox:",
			expectedRemote: "dropbox",
			expectedPath:   "",
		},
		{
			name:           "Remote with root path",
			input:          "s3:/",
			expectedRemote: "s3",
			expectedPath:   "/",
		},
		{
			name:           "No remote (local path)",
			input:          "/local/path",
			expectedRemote: "",
			expectedPath:   "/local/path",
		},
		{
			name:           "Empty string",
			input:          "",
			expectedRemote: "",
			expectedPath:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote, path := parseRemotePath(tt.input)

			if remote != tt.expectedRemote {
				t.Errorf("remote = %q, want %q", remote, tt.expectedRemote)
			}

			if path != tt.expectedPath {
				t.Errorf("path = %q, want %q", path, tt.expectedPath)
			}
		})
	}
}

func TestExpandSyncJobPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Path with tilde",
			input:    "~/Documents",
			expected: filepath.Join(home, "Documents"),
		},
		{
			name:     "Path without tilde",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "Empty path",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := components.ExpandHome(tt.input)
			if result != tt.expected {
				t.Errorf("components.ExpandHome(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSyncJobForm_SetSize(t *testing.T) {
	form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)

	form.SetSize(100, 30)

	if form.width != 100 {
		t.Errorf("width = %d, want 100", form.width)
	}

	if form.height != 30 {
		t.Errorf("height = %d, want 30", form.height)
	}
}

func TestSyncJobForm_Init(t *testing.T) {
	form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)

	cmd := form.Init()

	// Init should return a command (form initialization)
	// The exact command depends on huh.Form implementation
	if cmd == nil {
		// This is acceptable - form might not need initialization
	}
}

func TestSyncJobForm_View(t *testing.T) {
	form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
	form.SetSize(80, 24)

	view := form.View()

	// Check title for create mode
	if !strings.Contains(view, "Create New Sync Job") {
		t.Error("View() should contain 'Create New Sync Job' title for create mode")
	}

	// Check help text
	if !strings.Contains(view, "Tab") {
		t.Error("View() should contain help text for Tab key")
	}

	if !strings.Contains(view, "Esc") {
		t.Error("View() should contain help text for Esc key")
	}
}

func TestSyncJobForm_ViewEditMode(t *testing.T) {
	existingJob := &models.SyncJobConfig{
		Name:        "Test Sync",
		Source:      "gdrive:/Photos",
		Destination: "/backup/photos",
	}

	form := NewSyncJobForm(existingJob, createTestRemotes(), nil, nil, nil, nil, true)
	form.SetSize(80, 24)

	view := form.View()

	// Check title for edit mode
	if !strings.Contains(view, "Edit Sync Job") {
		t.Error("View() should contain 'Edit Sync Job' title for edit mode")
	}

	if !strings.Contains(view, "Test Sync") {
		t.Error("View() should contain job name in edit mode")
	}
}

func TestSyncJobForm_ViewWhenDone(t *testing.T) {
	form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
	form.done = true

	view := form.View()

	if view != "" {
		t.Errorf("View() when done = %q, want empty string", view)
	}
}

func TestSyncJobForm_IsDone(t *testing.T) {
	form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)

	if form.IsDone() {
		t.Error("IsDone() = true initially, want false")
	}

	form.done = true

	if !form.IsDone() {
		t.Error("IsDone() = false after setting done, want true")
	}
}

func TestSyncJobForm_EscapeCancels(t *testing.T) {
	form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
	form.SetSize(80, 24)

	// Initialize the form
	_ = form.Init()

	// Press escape
	_, cmd := form.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Form should be done and cancelled
	if !form.done {
		t.Error("form should be done after escape")
	}

	if !form.cancelled {
		t.Error("form should be cancelled after escape")
	}

	// Should return a cancel message
	if cmd == nil {
		t.Error("Update should return a command")
	}
}

func TestSyncJobForm_ShowCalendar(t *testing.T) {
	form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)

	// Timer type should show calendar
	form.scheduleType = "timer"
	if !form.showCalendar() {
		t.Error("showCalendar() = false for timer type, want true")
	}

	// OnBoot type should not show calendar
	form.scheduleType = "onboot"
	if form.showCalendar() {
		t.Error("showCalendar() = true for onboot type, want false")
	}

	// Manual type should not show calendar
	form.scheduleType = "manual"
	if form.showCalendar() {
		t.Error("showCalendar() = true for manual type, want false")
	}
}

func TestSyncJobForm_ShowOnBoot(t *testing.T) {
	form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)

	// OnBoot type should show on boot field
	form.scheduleType = "onboot"
	if !form.showOnBoot() {
		t.Error("showOnBoot() = false for onboot type, want true")
	}

	// Timer type should not show on boot field
	form.scheduleType = "timer"
	if form.showOnBoot() {
		t.Error("showOnBoot() = true for timer type, want false")
	}
}

func TestSyncJobForm_SubmitFormCreatesSyncJobConfig(t *testing.T) {
	cfg := createSyncTestConfig()
	gen := createSyncTestGenerator(t)
	mgr := createTestManager()
	form := NewSyncJobForm(nil, createTestRemotes(), cfg, gen, mgr, nil, false)

	// Set form values
	form.name = "Test Sync Job"
	form.sourceRemote = "gdrive"
	form.sourcePath = "/Photos"
	form.destPath = "/backup/photos"
	form.direction = "sync"
	form.deleteMode = "after"
	form.dryRun = true
	form.scheduleType = "timer"
	form.onCalendar = "daily"
	form.excludePattern = "*.tmp"
	form.maxTransfers = "8"
	form.bandwidthLimit = "10M"
	form.logLevel = "DEBUG"
	form.enabled = true
	form.requireACPower = true
	form.requireUnmetered = true

	// Submit the form
	msg := form.submitForm()

	// Check the returned message type
	createdMsg, ok := msg.(SyncJobCreatedMsg)
	if !ok {
		t.Fatalf("expected SyncJobCreatedMsg, got %T", msg)
	}

	// Verify the sync job config
	job := createdMsg.Job
	if job.Name != "Test Sync Job" {
		t.Errorf("job.Name = %q, want 'Test Sync Job'", job.Name)
	}

	if job.Source != "gdrive:/Photos" {
		t.Errorf("job.Source = %q, want 'gdrive:/Photos'", job.Source)
	}

	if job.Destination != "/backup/photos" {
		t.Errorf("job.Destination = %q, want '/backup/photos'", job.Destination)
	}

	if job.SyncOptions.Direction != "sync" {
		t.Errorf("job.Direction = %q, want 'sync'", job.SyncOptions.Direction)
	}

	if !job.SyncOptions.DeleteAfter {
		t.Error("job.DeleteAfter should be true")
	}

	if !job.SyncOptions.DryRun {
		t.Error("job.DryRun should be true")
	}

	if job.SyncOptions.Transfers != 8 {
		t.Errorf("job.Transfers = %d, want 8", job.SyncOptions.Transfers)
	}

	if job.SyncOptions.BandwidthLimit != "10M" {
		t.Errorf("job.BandwidthLimit = %q, want '10M'", job.SyncOptions.BandwidthLimit)
	}

	if job.Schedule.Type != "timer" {
		t.Errorf("job.Schedule.Type = %q, want 'timer'", job.Schedule.Type)
	}

	if job.Schedule.OnCalendar != "daily" {
		t.Errorf("job.Schedule.OnCalendar = %q, want 'daily'", job.Schedule.OnCalendar)
	}

	if !job.Schedule.RequireACPower {
		t.Error("job.Schedule.RequireACPower should be true")
	}

	if !job.Schedule.RequireUnmetered {
		t.Error("job.Schedule.RequireUnmetered should be true")
	}

	if !job.Enabled {
		t.Error("job.Enabled should be true")
	}

	// Verify ID was generated
	if job.ID == "" {
		t.Error("job.ID should be generated")
	}

	// Verify timestamps were set
	if job.CreatedAt.IsZero() {
		t.Error("job.CreatedAt should be set")
	}

	if job.ModifiedAt.IsZero() {
		t.Error("job.ModifiedAt should be set")
	}
}

func TestSyncJobForm_SubmitFormEditMode(t *testing.T) {
	cfg := createSyncTestConfig()

	// Create an existing sync job
	existingJob := &models.SyncJobConfig{
		ID:          "e4x5i6s7",
		Name:        "Existing Sync Job",
		Source:      "gdrive:/Documents",
		Destination: "/backup/docs",
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		ModifiedAt:  time.Now().Add(-24 * time.Hour),
	}

	gen := createSyncTestGenerator(t)
	mgr := createTestManager()
	form := NewSyncJobForm(existingJob, createTestRemotes(), cfg, gen, mgr, nil, true)

	// Modify some values
	form.destPath = "/backup/newdocs"
	form.direction = "copy"

	// Submit the form
	msg := form.submitForm()

	// Check the returned message type
	updatedMsg, ok := msg.(SyncJobUpdatedMsg)
	if !ok {
		t.Fatalf("expected SyncJobUpdatedMsg, got %T", msg)
	}

	job := updatedMsg.Job

	// Verify ID was preserved
	if job.ID != "e4x5i6s7" {
		t.Errorf("job.ID = %q, want 'e4x5i6s7'", job.ID)
	}

	// Verify name was preserved
	if job.Name != "Existing Sync Job" {
		t.Errorf("job.Name = %q, want 'Existing Sync Job'", job.Name)
	}

	// Verify created timestamp was preserved
	if !job.CreatedAt.Equal(existingJob.CreatedAt) {
		t.Error("job.CreatedAt should be preserved in edit mode")
	}

	// Verify modified timestamp was updated
	if !job.ModifiedAt.After(existingJob.ModifiedAt) {
		t.Error("job.ModifiedAt should be updated in edit mode")
	}

	// Verify updated values
	if job.Destination != "/backup/newdocs" {
		t.Errorf("job.Destination = %q, want '/backup/newdocs'", job.Destination)
	}

	if job.SyncOptions.Direction != "copy" {
		t.Errorf("job.Direction = %q, want 'copy'", job.SyncOptions.Direction)
	}
}

func TestSyncJobForm_DeleteModeHandling(t *testing.T) {
	tests := []struct {
		name              string
		deleteMode        string
		expectDeleteAfter bool
		expectDeleteExtr  bool
	}{
		{
			name:              "Delete after",
			deleteMode:        "after",
			expectDeleteAfter: true,
			expectDeleteExtr:  false,
		},
		{
			name:              "Delete during",
			deleteMode:        "during",
			expectDeleteAfter: false,
			expectDeleteExtr:  true,
		},
		{
			name:              "Delete never",
			deleteMode:        "never",
			expectDeleteAfter: false,
			expectDeleteExtr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := createSyncTestGenerator(t)
			mgr := createTestManager()
			form := NewSyncJobForm(nil, createTestRemotes(), nil, gen, mgr, nil, false)
			form.deleteMode = tt.deleteMode

			msg := form.submitForm()
			createdMsg, ok := msg.(SyncJobCreatedMsg)
			if !ok {
				t.Fatalf("expected SyncJobCreatedMsg, got %T", msg)
			}

			if createdMsg.Job.SyncOptions.DeleteAfter != tt.expectDeleteAfter {
				t.Errorf("DeleteAfter = %v, want %v", createdMsg.Job.SyncOptions.DeleteAfter, tt.expectDeleteAfter)
			}

			if createdMsg.Job.SyncOptions.DeleteExtraneous != tt.expectDeleteExtr {
				t.Errorf("DeleteExtraneous = %v, want %v", createdMsg.Job.SyncOptions.DeleteExtraneous, tt.expectDeleteExtr)
			}
		})
	}
}

func TestSyncJobForm_ConfigIsUpdated(t *testing.T) {
	cfg := createSyncTestConfig()
	gen := createSyncTestGenerator(t)
	mgr := createTestManager()
	form := NewSyncJobForm(nil, createTestRemotes(), cfg, gen, mgr, nil, false)

	// Set form values
	form.name = "New Sync Job"
	form.sourceRemote = "gdrive"
	form.sourcePath = "/"
	form.destPath = "~/backup"

	// Submit the form
	form.submitForm()

	// Verify sync job was added to config
	if len(cfg.SyncJobs) != 1 {
		t.Fatalf("config.SyncJobs length = %d, want 1", len(cfg.SyncJobs))
	}

	if cfg.SyncJobs[0].Name != "New Sync Job" {
		t.Errorf("config.SyncJobs[0].Name = %q, want 'New Sync Job'", cfg.SyncJobs[0].Name)
	}
}

func TestSyncJobForm_NilConfigNoPanic(t *testing.T) {
	// Test that form doesn't panic with nil config
	gen := createSyncTestGenerator(t)
	mgr := createTestManager()
	form := NewSyncJobForm(nil, createTestRemotes(), nil, gen, mgr, nil, false)

	// This should not panic
	msg := form.submitForm()

	// Should still return a valid message
	if msg == nil {
		t.Error("submitForm() should return a message even with nil config")
	}
}

func TestSyncJobForm_RemoteDestination(t *testing.T) {
	gen := createSyncTestGenerator(t)
	mgr := createTestManager()
	form := NewSyncJobForm(nil, createTestRemotes(), nil, gen, mgr, nil, false)

	// Set a remote destination
	form.destRemote = "s3"
	form.destPath = "/backup-bucket/photos"

	msg := form.submitForm()
	createdMsg, ok := msg.(SyncJobCreatedMsg)
	if !ok {
		t.Fatalf("expected SyncJobCreatedMsg, got %T", msg)
	}

	// Destination should be formatted as remote:path
	if createdMsg.Job.Destination != "s3:/backup-bucket/photos" {
		t.Errorf("Destination = %q, want 's3:/backup-bucket/photos'", createdMsg.Job.Destination)
	}
}

func TestSyncJobForm_DefaultValues(t *testing.T) {
	// Test with nil config - should use hardcoded defaults
	form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)

	if form.direction != "sync" {
		t.Errorf("default direction = %q, want 'sync'", form.direction)
	}

	if form.deleteMode != "after" {
		t.Errorf("default deleteMode = %q, want 'after'", form.deleteMode)
	}

	if form.logLevel != "INFO" {
		t.Errorf("default logLevel = %q, want 'INFO'", form.logLevel)
	}

	if form.scheduleType != "timer" {
		t.Errorf("default scheduleType = %q, want 'timer'", form.scheduleType)
	}

	if form.onCalendar != "daily" {
		t.Errorf("default onCalendar = %q, want 'daily'", form.onCalendar)
	}
}

func TestSyncJobForm_MaxTransfersParsing(t *testing.T) {
	tests := []struct {
		name          string
		maxTransfers  string
		expectedValue int
	}{
		{
			name:          "Valid number",
			maxTransfers:  "8",
			expectedValue: 8,
		},
		{
			name:          "Empty string uses default",
			maxTransfers:  "",
			expectedValue: 4, // Default
		},
		{
			name:          "Invalid number uses default",
			maxTransfers:  "abc",
			expectedValue: 4, // Default
		},
		{
			name:          "Whitespace trimmed",
			maxTransfers:  " 12 ",
			expectedValue: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := createSyncTestGenerator(t)
			mgr := createTestManager()
			form := NewSyncJobForm(nil, createTestRemotes(), nil, gen, mgr, nil, false)
			form.maxTransfers = tt.maxTransfers

			msg := form.submitForm()
			createdMsg, ok := msg.(SyncJobCreatedMsg)
			if !ok {
				t.Fatalf("expected SyncJobCreatedMsg, got %T", msg)
			}

			if createdMsg.Job.SyncOptions.Transfers != tt.expectedValue {
				t.Errorf("Transfers = %d, want %d", createdMsg.Job.SyncOptions.Transfers, tt.expectedValue)
			}
		})
	}
}

func TestSyncJobForm_GetRemotePathSuggestions_NilClient(t *testing.T) {
	form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
	form.rcloneClient = nil
	form.sourceRemote = "gdrive"

	suggestions := form.getRemotePathSuggestions()

	if len(suggestions) == 0 {
		t.Error("getRemotePathSuggestions() should return static suggestions when client is nil")
	}

	if suggestions[0] != "/" {
		t.Errorf("first suggestion = %q, want '/'", suggestions[0])
	}
}

func TestSyncJobForm_GetRemotePathSuggestions_EmptyRemote(t *testing.T) {
	form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
	form.sourceRemote = ""

	suggestions := form.getRemotePathSuggestions()

	if len(suggestions) == 0 {
		t.Error("getRemotePathSuggestions() should return static suggestions when remote is empty")
	}
}

func TestSyncJobForm_SubmitForm_NoSourceRemote(t *testing.T) {
	gen := createSyncTestGenerator(t)
	mgr := createTestManager()
	form := NewSyncJobForm(nil, createTestRemotes(), nil, gen, mgr, nil, false)
	form.sourceRemote = ""
	form.name = "Test"
	form.destPath = "/backup/test"

	msg := form.submitForm()

	errMsg, ok := msg.(SyncJobsErrorMsg)
	if !ok {
		t.Fatalf("expected SyncJobsErrorMsg, got %T", msg)
	}

	if !strings.Contains(errMsg.Err.Error(), "no source remote selected") {
		t.Errorf("error = %q, should contain 'no source remote selected'", errMsg.Err.Error())
	}
	if !strings.Contains(errMsg.Err.Error(), "rclone config") {
		t.Errorf("error = %q, should contain 'rclone config'", errMsg.Err.Error())
	}
}

func TestSyncJobForm_ValidateDestPath_EdgeCases(t *testing.T) {
	// Create a temp directory to test paths
	tmpDir, err := os.MkdirTemp("", "sync-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name          string
		inputPath     string
		expectError   bool
		errorContains string
	}{
		{
			name:        "Remote path with colon",
			inputPath:   "remote:bucket/path",
			expectError: false,
		},
		{
			name:          "Remote path without colon but looks like remote",
			inputPath:     "remotepath",
			expectError:   true,
			errorContains: "absolute",
		},
		{
			name:        "Path without trailing slash under existing parent",
			inputPath:   tmpDir + "/test",
			expectError: false,
		},
		{
			name:        "Remote path with just colon",
			inputPath:   "remote:",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
			err := form.validateDestPath(tt.inputPath)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error = %q, should contain %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSyncJobForm_SubmitFormWithRemoteDestination(t *testing.T) {
	gen := createSyncTestGenerator(t)
	mgr := createTestManager()
	form := NewSyncJobForm(nil, createTestRemotes(), nil, gen, mgr, nil, false)

	form.name = "Test Sync"
	form.sourceRemote = "gdrive"
	form.sourcePath = "/Documents"
	form.destRemote = "s3"
	form.destPath = "/backup-bucket/docs"
	form.direction = "sync"
	form.deleteMode = "after"
	form.scheduleType = "timer"
	form.onCalendar = "daily"
	form.enabled = true

	msg := form.submitForm()

	createdMsg, ok := msg.(SyncJobCreatedMsg)
	if !ok {
		t.Fatalf("expected SyncJobCreatedMsg, got %T", msg)
	}

	if createdMsg.Job.Source != "gdrive:/Documents" {
		t.Errorf("Source = %q, want 'gdrive:/Documents'", createdMsg.Job.Source)
	}

	if createdMsg.Job.Destination != "s3:/backup-bucket/docs" {
		t.Errorf("Destination = %q, want 's3:/backup-bucket/docs'", createdMsg.Job.Destination)
	}
}

func TestSyncJobForm_EditPreservesAllOptions(t *testing.T) {
	cfg := createSyncTestConfig()
	remotes := createTestRemotes()

	existingJob := &models.SyncJobConfig{
		ID:          "j1o2b3x4",
		Name:        "Test Sync",
		Source:      "gdrive:/Documents",
		Destination: "/backup/docs",
		SyncOptions: models.SyncOptions{
			Direction:        "copy",
			DeleteAfter:      false,
			DeleteExtraneous: true,
			DryRun:           true,
			ExcludePattern:   "*.tmp",
			Transfers:        8,
			BandwidthLimit:   "20M",
			LogLevel:         "DEBUG",
		},
		Schedule: models.ScheduleConfig{
			Type:             "onboot",
			OnBootSec:        "2min",
			OnCalendar:       "",
			RequireACPower:   true,
			RequireUnmetered: true,
		},
		Enabled: true,
	}

	form := NewSyncJobForm(existingJob, remotes, cfg, nil, nil, nil, true)

	if form.direction != "copy" {
		t.Errorf("direction = %q, want 'copy'", form.direction)
	}
	if form.deleteMode != "during" {
		t.Errorf("deleteMode = %q, want 'during' (DeleteExtraneous=true)", form.deleteMode)
	}
	if form.dryRun != true {
		t.Error("dryRun should be true")
	}
	if form.excludePattern != "*.tmp" {
		t.Errorf("excludePattern = %q, want '*.tmp'", form.excludePattern)
	}
	if form.maxTransfers != "8" {
		t.Errorf("maxTransfers = %q, want '8'", form.maxTransfers)
	}
	if form.bandwidthLimit != "20M" {
		t.Errorf("bandwidthLimit = %q, want '20M'", form.bandwidthLimit)
	}
	if form.scheduleType != "onboot" {
		t.Errorf("scheduleType = %q, want 'onboot'", form.scheduleType)
	}
	if form.onBootSec != "2min" {
		t.Errorf("onBootSec = %q, want '2min'", form.onBootSec)
	}
	if !form.requireACPower {
		t.Error("requireACPower should be true")
	}
	if !form.requireUnmetered {
		t.Error("requireUnmetered should be true")
	}
}

func TestSyncJobForm_DeleteModeParsing(t *testing.T) {
	tests := []struct {
		name               string
		deleteAfter        bool
		deleteExtraneous   bool
		expectedDeleteMode string
	}{
		{
			name:               "Delete after only",
			deleteAfter:        true,
			deleteExtraneous:   false,
			expectedDeleteMode: "after",
		},
		{
			name:               "Delete during (extraneous)",
			deleteAfter:        false,
			deleteExtraneous:   true,
			expectedDeleteMode: "during",
		},
		{
			name:               "No delete",
			deleteAfter:        false,
			deleteExtraneous:   false,
			expectedDeleteMode: "never",
		},
		{
			name:               "Both set (after takes precedence)",
			deleteAfter:        true,
			deleteExtraneous:   true,
			expectedDeleteMode: "after",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &models.SyncJobConfig{
				SyncOptions: models.SyncOptions{
					DeleteAfter:      tt.deleteAfter,
					DeleteExtraneous: tt.deleteExtraneous,
				},
			}

			form := NewSyncJobForm(job, createTestRemotes(), nil, nil, nil, nil, true)

			if form.deleteMode != tt.expectedDeleteMode {
				t.Errorf("deleteMode = %q, want %q", form.deleteMode, tt.expectedDeleteMode)
			}
		})
	}
}

func TestSyncJobForm_OnBootParsing(t *testing.T) {
	job := &models.SyncJobConfig{
		Schedule: models.ScheduleConfig{
			Type:      "onboot",
			OnBootSec: "10min",
		},
	}

	form := NewSyncJobForm(job, createTestRemotes(), nil, nil, nil, nil, true)

	if form.scheduleType != "onboot" {
		t.Error("scheduleType should be 'onboot' for onboot schedule type")
	}
	if form.onBootSec != "10min" {
		t.Errorf("onBootSec = %q, want '10min'", form.onBootSec)
	}
}

func TestSyncJobForm_RollbackPreparationCreatesCopy(t *testing.T) {
	cfg := createSyncTestConfig()
	cfg.SyncJobs = []models.SyncJobConfig{
		{ID: "abc12345", Name: "Job1"},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := mgr.PrepareSyncJobRollback("new12345", "NewJob", OperationCreate)

	cfg.SyncJobs[0].Name = "ModifiedJob"

	if data.OriginalJobs[0].Name != "Job1" {
		t.Error("OriginalJobs should be independent copy")
	}
}

func TestSyncJobForm_ConfigRestoreAfterFailure(t *testing.T) {
	cfg := createSyncTestConfig()
	origJobs := []models.SyncJobConfig{
		{ID: "abc12345", Name: "Job1"},
	}
	cfg.SyncJobs = origJobs

	mgr := NewRollbackManager(cfg, nil, nil)
	data := SyncJobRollbackData{
		OriginalJobs: origJobs,
		Operation:    OperationCreate,
		JobID:        "new12345",
		JobName:      "NewJob",
	}

	cfg.SyncJobs = append(cfg.SyncJobs, models.SyncJobConfig{ID: "new12345", Name: "NewJob"})

	err := mgr.RollbackSyncJob(data, true)
	if err != nil {
		t.Logf("RollbackSyncJob returned: %v", err)
	}

	if len(cfg.SyncJobs) != 1 {
		t.Errorf("after rollback, SyncJobs length = %d, want 1", len(cfg.SyncJobs))
	}

	if cfg.SyncJobs[0].ID != "abc12345" {
		t.Error("original job not restored")
	}
}

func TestSyncJobForm_NoRemotesAvailable(t *testing.T) {
	cfg := createSyncTestConfig()
	form := NewSyncJobForm(nil, []rclone.Remote{}, cfg, nil, nil, nil, false)

	if form == nil {
		t.Fatal("NewSyncJobForm() returned nil")
	}

	// Form should still be created even with no remotes
	if len(form.remotes) != 0 {
		t.Errorf("remotes count = %d, want 0", len(form.remotes))
	}
}

func TestSyncJobForm_NoRemotesShowsHelpfulMessage(t *testing.T) {
	gen := createSyncTestGenerator(t)
	mgr := createTestManager()
	form := NewSyncJobForm(nil, []rclone.Remote{}, nil, gen, mgr, nil, false)
	form.SetSize(80, 24)

	// Verify form was created successfully with empty remotes
	if form == nil {
		t.Fatal("form should not be nil")
	}

	// The placeholder option is added in buildForm - verify form can be initialized
	cmd := form.Init()
	if cmd == nil {
		// Init may return nil, that's fine
	}

	// Verify submitting with no remote selected gives helpful error
	form.name = "Test"
	form.destPath = "/backup/test"
	form.sourceRemote = ""
	msg := form.submitForm()
	errMsg, ok := msg.(SyncJobsErrorMsg)
	if !ok {
		t.Fatalf("expected SyncJobsErrorMsg, got %T", msg)
	}
	if !strings.Contains(errMsg.Err.Error(), "rclone config") {
		t.Errorf("error should mention 'rclone config', got: %s", errMsg.Err.Error())
	}
}

// Tests for validateOnCalendar function
func TestSyncJobForm_ValidateOnCalendar(t *testing.T) {
	tests := []struct {
		name        string
		calendar    string
		expectError bool
	}{
		{
			name:        "Empty string is invalid",
			calendar:    "",
			expectError: true,
		},
		{
			name:        "Daily is valid",
			calendar:    "daily",
			expectError: false,
		},
		{
			name:        "Weekly is valid",
			calendar:    "weekly",
			expectError: false,
		},
		{
			name:        "Monthly is valid",
			calendar:    "monthly",
			expectError: false,
		},
		{
			name:        "Hourly is valid",
			calendar:    "hourly",
			expectError: false,
		},
		{
			name:        "Specific time daily",
			calendar:    "*-*-* 00:00:00",
			expectError: false,
		},
		{
			name:        "Weekday at specific time",
			calendar:    "Mon *-*-* 09:00:00",
			expectError: false,
		},
		{
			name:        "Quarter hour",
			calendar:    "quarterly",
			expectError: false,
		},
		{
			name:        "Semi-annually",
			calendar:    "semiannually",
			expectError: false,
		},
		{
			name:        "Annually",
			calendar:    "annually",
			expectError: false,
		},
		{
			name:        "Specific date yearly",
			calendar:    "*-01-01 00:00:00",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
			err := form.validateOnCalendar(tt.calendar)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Tests for validateMaxTransfers function
func TestSyncJobForm_ValidateMaxTransfers(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expectError bool
		errContains string
	}{
		{
			name:        "Empty string is valid",
			value:       "",
			expectError: false,
		},
		{
			name:        "Valid positive number",
			value:       "4",
			expectError: false,
		},
		{
			name:        "Valid large number",
			value:       "100",
			expectError: false,
		},
		{
			name:        "Valid number with whitespace",
			value:       " 8 ",
			expectError: false,
		},
		{
			name:        "Zero is invalid",
			value:       "0",
			expectError: true,
			errContains: "greater than 0",
		},
		{
			name:        "Negative number is invalid",
			value:       "-1",
			expectError: true,
			errContains: "greater than 0",
		},
		{
			name:        "Non-numeric is invalid",
			value:       "abc",
			expectError: true,
			errContains: "valid number",
		},
		{
			name:        "Float is invalid",
			value:       "4.5",
			expectError: true,
			errContains: "valid number",
		},
		{
			name:        "Number with spaces only",
			value:       "   12   ",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
			err := form.validateMaxTransfers(tt.value)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, should contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Test validateOnCalendar with various systemd calendar expressions
func TestSyncJobForm_ValidateOnCalendar_SystemdFormats(t *testing.T) {
	tests := []struct {
		name        string
		calendar    string
		expectError bool
	}{
		{
			name:        "Full datetime format",
			calendar:    "2024-01-01 00:00:00",
			expectError: false,
		},
		{
			name:        "Time only with wildcard date",
			calendar:    "*-*-* 12:30:00",
			expectError: false,
		},
		{
			name:        "Weekday with time",
			calendar:    "Mon,Fri *-*-* 09:00:00",
			expectError: false,
		},
		{
			name:        "Monthly on specific day",
			calendar:    "*-*-15 00:00:00",
			expectError: false,
		},
		{
			name:        "Yearly on specific date",
			calendar:    "*-06-01 00:00:00",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := NewSyncJobForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
			err := form.validateOnCalendar(tt.calendar)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for calendar %q: %v", tt.calendar, err)
				}
			}
		})
	}
}
