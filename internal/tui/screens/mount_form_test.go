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
)

// Helper function to create a test config
func createTestConfig() *config.Config {
	return &config.Config{
		Version: "1.0",
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
			Sync: config.SyncDefaults{
				LogLevel:  "INFO",
				Transfers: 4,
				Checkers:  8,
			},
		},
		Settings: config.Settings{
			DefaultMountDir: "~/mnt",
		},
		Mounts:   []models.MountConfig{},
		SyncJobs: []models.SyncJobConfig{},
	}
}

// Helper function to create test remotes
func createTestRemotes() []rclone.Remote {
	return []rclone.Remote{
		{Name: "gdrive", Type: "drive", RootPath: "gdrive:"},
		{Name: "dropbox", Type: "dropbox", RootPath: "dropbox:"},
		{Name: "s3", Type: "s3", RootPath: "s3:"},
	}
}

func TestNewMountForm_Create(t *testing.T) {
	cfg := createTestConfig()
	remotes := createTestRemotes()

	form := NewMountForm(nil, remotes, cfg, nil, nil, nil, false)

	if form == nil {
		t.Fatal("NewMountForm() returned nil")
	}

	// Verify initial state
	if form.isEdit {
		t.Error("isEdit should be false for create mode")
	}

	if form.mount != nil {
		t.Error("mount should be nil for create mode")
	}

	// Verify defaults from config are applied
	if form.vfsCacheMode != cfg.Defaults.Mount.VFSCacheMode {
		t.Errorf("vfsCacheMode = %q, want %q", form.vfsCacheMode, cfg.Defaults.Mount.VFSCacheMode)
	}

	if form.bufferSize != cfg.Defaults.Mount.BufferSize {
		t.Errorf("bufferSize = %q, want %q", form.bufferSize, cfg.Defaults.Mount.BufferSize)
	}

	if form.logLevel != cfg.Defaults.Mount.LogLevel {
		t.Errorf("logLevel = %q, want %q", form.logLevel, cfg.Defaults.Mount.LogLevel)
	}
}

func TestNewMountForm_Edit(t *testing.T) {
	cfg := createTestConfig()
	remotes := createTestRemotes()

	// Create an existing mount to edit
	existingMount := &models.MountConfig{
		ID:          "t1e2s3t4",
		Name:        "Test Mount",
		Remote:      "gdrive",
		RemotePath:  "/Photos",
		MountPoint:  "/mnt/gdrive",
		Description: "Test mount for editing",
		MountOptions: models.MountOptions{
			VFSCacheMode: "writes",
			BufferSize:   "32M",
			LogLevel:     "DEBUG",
			AllowOther:   true,
			ReadOnly:     true,
		},
		AutoStart: true,
		Enabled:   true,
	}

	form := NewMountForm(existingMount, remotes, cfg, nil, nil, nil, true)

	if form == nil {
		t.Fatal("NewMountForm() returned nil")
	}

	// Verify edit mode
	if !form.isEdit {
		t.Error("isEdit should be true for edit mode")
	}

	// Verify existing values are populated
	if form.name != existingMount.Name {
		t.Errorf("name = %q, want %q", form.name, existingMount.Name)
	}

	if form.remote != existingMount.Remote+":" {
		t.Errorf("remote = %q, want %q", form.remote, existingMount.Remote+":")
	}

	if form.remotePath != existingMount.RemotePath {
		t.Errorf("remotePath = %q, want %q", form.remotePath, existingMount.RemotePath)
	}

	if form.mountPoint != existingMount.MountPoint {
		t.Errorf("mountPoint = %q, want %q", form.mountPoint, existingMount.MountPoint)
	}

	if form.vfsCacheMode != existingMount.MountOptions.VFSCacheMode {
		t.Errorf("vfsCacheMode = %q, want %q", form.vfsCacheMode, existingMount.MountOptions.VFSCacheMode)
	}

	if form.allowOther != existingMount.MountOptions.AllowOther {
		t.Errorf("allowOther = %v, want %v", form.allowOther, existingMount.MountOptions.AllowOther)
	}

	if form.readOnly != existingMount.MountOptions.ReadOnly {
		t.Errorf("readOnly = %v, want %v", form.readOnly, existingMount.MountOptions.ReadOnly)
	}

	if form.autoStart != existingMount.AutoStart {
		t.Errorf("autoStart = %v, want %v", form.autoStart, existingMount.AutoStart)
	}
}

func TestMountForm_ValidateName(t *testing.T) {
	tests := []struct {
		name          string
		inputName     string
		existing      []models.MountConfig
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
			inputName:   "My Mount",
			expectError: false,
		},
		{
			name:          "Name too long",
			inputName:     strings.Repeat("a", 51),
			expectError:   true,
			errorContains: "50 characters",
		},
		{
			name:      "Duplicate name - new mount",
			inputName: "Existing Mount",
			existing: []models.MountConfig{
				{Name: "Existing Mount"},
			},
			isEdit:        false,
			expectError:   true,
			errorContains: "already exists",
		},
		{
			name:      "Duplicate name - edit mode same mount",
			inputName: "Existing Mount",
			existing: []models.MountConfig{
				{Name: "Existing Mount"},
			},
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
			cfg := createTestConfig()
			cfg.Mounts = tt.existing

			form := NewMountForm(nil, createTestRemotes(), cfg, nil, nil, nil, tt.isEdit)
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

func TestMountForm_ValidateMountPoint(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "mount-test-*")
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
			inputPath:   filepath.Join(existingDir, "newmount"),
			expectError: false,
		},
		{
			name:          "Relative path",
			inputPath:     "relative/path",
			expectError:   true,
			errorContains: "absolute path",
		},
		{
			name:        "Path with tilde to home",
			inputPath:   "~/test_mount",
			expectError: false, // Tilde is expanded to home directory which exists
		},
		{
			name:          "Parent directory does not exist",
			inputPath:     "/nonexistent/path/mount",
			expectError:   true,
			errorContains: "parent directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
			err := form.validateMountPoint(tt.inputPath)

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

func TestExpandPath(t *testing.T) {
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
			name:     "Just tilde",
			input:    "~",
			expected: "~", // Not expanded since it doesn't have /
		},
		{
			name:     "Empty path",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			if result != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMountForm_SetSize(t *testing.T) {
	form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)

	form.SetSize(100, 30)

	if form.width != 100 {
		t.Errorf("width = %d, want 100", form.width)
	}

	if form.height != 30 {
		t.Errorf("height = %d, want 30", form.height)
	}
}

func TestMountForm_Init(t *testing.T) {
	form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)

	cmd := form.Init()

	// Init should return a command (form initialization)
	// The exact command depends on huh.Form implementation
	if cmd == nil {
		// This is acceptable - form might not need initialization
	}
}

func TestMountForm_View(t *testing.T) {
	form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
	form.SetSize(80, 24)

	view := form.View()

	// Check title for create mode
	if !strings.Contains(view, "Create New Mount") {
		t.Error("View() should contain 'Create New Mount' title for create mode")
	}

	// Check help text
	if !strings.Contains(view, "Tab") {
		t.Error("View() should contain help text for Tab key")
	}

	if !strings.Contains(view, "Esc") {
		t.Error("View() should contain help text for Esc key")
	}
}

func TestMountForm_ViewEditMode(t *testing.T) {
	existingMount := &models.MountConfig{
		Name:       "Test Mount",
		Remote:     "gdrive",
		RemotePath: "/",
		MountPoint: "/mnt/test",
	}

	form := NewMountForm(existingMount, createTestRemotes(), nil, nil, nil, nil, true)
	form.SetSize(80, 24)

	view := form.View()

	// Check title for edit mode
	if !strings.Contains(view, "Edit Mount") {
		t.Error("View() should contain 'Edit Mount' title for edit mode")
	}

	if !strings.Contains(view, "Test Mount") {
		t.Error("View() should contain mount name in edit mode")
	}
}

func TestMountForm_ViewWhenDone(t *testing.T) {
	form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
	form.done = true

	view := form.View()

	if view != "" {
		t.Errorf("View() when done = %q, want empty string", view)
	}
}

func TestMountForm_IsDone(t *testing.T) {
	form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)

	if form.IsDone() {
		t.Error("IsDone() = true initially, want false")
	}

	form.done = true

	if !form.IsDone() {
		t.Error("IsDone() = false after setting done, want true")
	}
}

func TestMountForm_EscapeCancels(t *testing.T) {
	form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
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

func TestMountForm_DefaultValues(t *testing.T) {
	// Test with nil config - should use hardcoded defaults
	form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)

	if form.vfsCacheMode != "full" {
		t.Errorf("default vfsCacheMode = %q, want 'full'", form.vfsCacheMode)
	}

	if form.bufferSize != "16M" {
		t.Errorf("default bufferSize = %q, want '16M'", form.bufferSize)
	}

	if form.logLevel != "INFO" {
		t.Errorf("default logLevel = %q, want 'INFO'", form.logLevel)
	}

	if form.remotePath != "/" {
		t.Errorf("default remotePath = %q, want '/'", form.remotePath)
	}
}

func TestMountForm_RemoteOptions(t *testing.T) {
	remotes := createTestRemotes()
	form := NewMountForm(nil, remotes, nil, nil, nil, nil, false)

	// Verify remotes are stored
	if len(form.remotes) != len(remotes) {
		t.Errorf("remotes count = %d, want %d", len(form.remotes), len(remotes))
	}

	for i, r := range remotes {
		if i >= len(form.remotes) {
			t.Errorf("missing remote at index %d", i)
			continue
		}
		if form.remotes[i].Name != r.Name {
			t.Errorf("remote %d name = %q, want %q", i, form.remotes[i].Name, r.Name)
		}
	}
}

func TestMountForm_SubmitFormCreatesMountConfig(t *testing.T) {
	cfg := createTestConfig()
	form := NewMountForm(nil, createTestRemotes(), cfg, nil, nil, nil, false)

	// Set form values
	form.name = "Test Mount"
	form.remote = "gdrive:"
	form.remotePath = "/Photos"
	form.mountPoint = "/mnt/test"
	form.vfsCacheMode = "full"
	form.bufferSize = "16M"
	form.logLevel = "INFO"
	form.allowOther = true
	form.readOnly = false
	form.autoStart = true
	form.enabled = true

	// Submit the form
	msg := form.submitForm()

	// Check the returned message type
	createdMsg, ok := msg.(MountCreatedMsg)
	if !ok {
		t.Fatalf("expected MountCreatedMsg, got %T", msg)
	}

	// Verify the mount config
	mount := createdMsg.Mount
	if mount.Name != "Test Mount" {
		t.Errorf("mount.Name = %q, want 'Test Mount'", mount.Name)
	}

	if mount.Remote != "gdrive" {
		t.Errorf("mount.Remote = %q, want 'gdrive'", mount.Remote)
	}

	if mount.RemotePath != "/Photos" {
		t.Errorf("mount.RemotePath = %q, want '/Photos'", mount.RemotePath)
	}

	if mount.MountPoint != "/mnt/test" {
		t.Errorf("mount.MountPoint = %q, want '/mnt/test'", mount.MountPoint)
	}

	if mount.MountOptions.VFSCacheMode != "full" {
		t.Errorf("mount.VFSCacheMode = %q, want 'full'", mount.MountOptions.VFSCacheMode)
	}

	if mount.MountOptions.AllowOther != true {
		t.Errorf("mount.AllowOther = %v, want true", mount.MountOptions.AllowOther)
	}

	if mount.AutoStart != true {
		t.Errorf("mount.AutoStart = %v, want true", mount.AutoStart)
	}

	// Verify ID was generated
	if mount.ID == "" {
		t.Error("mount.ID should be generated")
	}

	// Verify timestamps were set
	if mount.CreatedAt.IsZero() {
		t.Error("mount.CreatedAt should be set")
	}

	if mount.ModifiedAt.IsZero() {
		t.Error("mount.ModifiedAt should be set")
	}
}

func TestMountForm_SubmitFormEditMode(t *testing.T) {
	cfg := createTestConfig()

	// Create an existing mount
	existingMount := &models.MountConfig{
		ID:         "e1x2i3s4",
		Name:       "Existing Mount",
		Remote:     "gdrive",
		RemotePath: "/",
		MountPoint: "/mnt/old",
		CreatedAt:  time.Now().Add(-24 * time.Hour),
		ModifiedAt: time.Now().Add(-24 * time.Hour),
	}

	form := NewMountForm(existingMount, createTestRemotes(), cfg, nil, nil, nil, true)

	// Modify some values
	form.mountPoint = "/mnt/new"
	form.vfsCacheMode = "writes"

	// Submit the form
	msg := form.submitForm()

	// Check the returned message type
	updatedMsg, ok := msg.(MountUpdatedMsg)
	if !ok {
		t.Fatalf("expected MountUpdatedMsg, got %T", msg)
	}

	mount := updatedMsg.Mount

	// Verify ID was preserved
	if mount.ID != "e1x2i3s4" {
		t.Errorf("mount.ID = %q, want 'e1x2i3s4'", mount.ID)
	}

	// Verify name was preserved
	if mount.Name != "Existing Mount" {
		t.Errorf("mount.Name = %q, want 'Existing Mount'", mount.Name)
	}

	// Verify created timestamp was preserved
	if !mount.CreatedAt.Equal(existingMount.CreatedAt) {
		t.Error("mount.CreatedAt should be preserved in edit mode")
	}

	// Verify modified timestamp was updated
	if !mount.ModifiedAt.After(existingMount.ModifiedAt) {
		t.Error("mount.ModifiedAt should be updated in edit mode")
	}

	// Verify updated values
	if mount.MountPoint != "/mnt/new" {
		t.Errorf("mount.MountPoint = %q, want '/mnt/new'", mount.MountPoint)
	}

	if mount.MountOptions.VFSCacheMode != "writes" {
		t.Errorf("mount.VFSCacheMode = %q, want 'writes'", mount.MountOptions.VFSCacheMode)
	}
}

func TestMountForm_ConfigIsUpdated(t *testing.T) {
	cfg := createTestConfig()
	form := NewMountForm(nil, createTestRemotes(), cfg, nil, nil, nil, false)

	// Set form values
	form.name = "New Mount"
	form.remote = "gdrive:"
	form.remotePath = "/"
	form.mountPoint = "/mnt/new"

	// Submit the form
	form.submitForm()

	// Verify mount was added to config
	if len(cfg.Mounts) != 1 {
		t.Fatalf("config.Mounts length = %d, want 1", len(cfg.Mounts))
	}

	if cfg.Mounts[0].Name != "New Mount" {
		t.Errorf("config.Mounts[0].Name = %q, want 'New Mount'", cfg.Mounts[0].Name)
	}
}

func TestMountForm_NilConfigNoPanic(t *testing.T) {
	// Test that form doesn't panic with nil config
	form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)

	// This should not panic
	msg := form.submitForm()

	// Should still return a valid message
	if msg == nil {
		t.Error("submitForm() should return a message even with nil config")
	}
}

func TestMountForm_GetRemotePathSuggestions_NilClient(t *testing.T) {
	form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
	form.rcloneClient = nil
	form.remote = "gdrive:"

	suggestions := form.getRemotePathSuggestions()

	if len(suggestions) == 0 {
		t.Error("getRemotePathSuggestions() should return static suggestions when client is nil")
	}

	if suggestions[0] != "/" {
		t.Errorf("first suggestion = %q, want '/'", suggestions[0])
	}
}

func TestMountForm_GetRemotePathSuggestions_EmptyRemote(t *testing.T) {
	form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
	form.remote = ""

	suggestions := form.getRemotePathSuggestions()

	if len(suggestions) == 0 {
		t.Error("getRemotePathSuggestions() should return static suggestions when remote is empty")
	}
}

func TestMountForm_GetRemotePathSuggestions_RemoteOnlyColon(t *testing.T) {
	form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
	form.remote = ":"

	suggestions := form.getRemotePathSuggestions()

	if len(suggestions) == 0 {
		t.Error("getRemotePathSuggestions() should return static suggestions")
	}
}

func TestMountForm_SubmitForm_NoRemoteSelected(t *testing.T) {
	form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
	form.remote = ""
	form.name = "Test"
	form.mountPoint = "/mnt/test"

	msg := form.submitForm()

	errMsg, ok := msg.(MountsErrorMsg)
	if !ok {
		t.Fatalf("expected MountsErrorMsg, got %T", msg)
	}

	if !strings.Contains(errMsg.Err.Error(), "no remote selected") {
		t.Errorf("error = %q, should contain 'no remote selected'", errMsg.Err.Error())
	}
	if !strings.Contains(errMsg.Err.Error(), "rclone config") {
		t.Errorf("error = %q, should contain 'rclone config'", errMsg.Err.Error())
	}
}

func TestMountForm_ValidateMountPoint_EdgeCases(t *testing.T) {
	// Create a temp directory to test paths
	tmpDir, err := os.MkdirTemp("", "mount-test-*")
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
			name:        "Root path",
			inputPath:   "/",
			expectError: false,
		},
		{
			name:        "Path without trailing slash under existing parent",
			inputPath:   tmpDir + "/test",
			expectError: false,
		},
		{
			name:        "Path with spaces under existing parent",
			inputPath:   tmpDir + "/my mount",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := NewMountForm(nil, createTestRemotes(), nil, nil, nil, nil, false)
			err := form.validateMountPoint(tt.inputPath)

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

func TestMountForm_EditPreservesAdvancedOptions(t *testing.T) {
	cfg := createTestConfig()
	remotes := createTestRemotes()

	existingMount := &models.MountConfig{
		ID:          "t1e2s3t4",
		Name:        "Test Mount",
		Remote:      "gdrive",
		RemotePath:  "/Photos",
		MountPoint:  "/mnt/gdrive",
		Description: "Test mount",
		MountOptions: models.MountOptions{
			VFSCacheMode:    "writes",
			VFSCacheMaxAge:  "12h",
			VFSCacheMaxSize: "5G",
			VFSWriteBack:    "10s",
			BufferSize:      "32M",
			AllowOther:      true,
			AllowRoot:       true,
			Umask:           "077",
			ReadOnly:        true,
			NoModTime:       true,
			NoChecksum:      true,
			LogLevel:        "DEBUG",
			ExtraArgs:       "--test-arg",
		},
		AutoStart: true,
		Enabled:   true,
	}

	form := NewMountForm(existingMount, remotes, cfg, nil, nil, nil, true)

	if form.vfsCacheMaxAge != "12h" {
		t.Errorf("vfsCacheMaxAge = %q, want '12h'", form.vfsCacheMaxAge)
	}
	if form.vfsCacheMaxSize != "5G" {
		t.Errorf("vfsCacheMaxSize = %q, want '5G'", form.vfsCacheMaxSize)
	}
	if form.vfsWriteBack != "10s" {
		t.Errorf("vfsWriteBack = %q, want '10s'", form.vfsWriteBack)
	}
	if form.allowRoot != true {
		t.Error("allowRoot should be true")
	}
	if form.umask != "077" {
		t.Errorf("umask = %q, want '077'", form.umask)
	}
	if form.noModtime != true {
		t.Error("noModtime should be true")
	}
	if form.noChecksum != true {
		t.Error("noChecksum should be true")
	}
	if form.extraArgs != "--test-arg" {
		t.Errorf("extraArgs = %q, want '--test-arg'", form.extraArgs)
	}
}

func TestMountForm_NoRemotesAvailable(t *testing.T) {
	cfg := createTestConfig()
	form := NewMountForm(nil, []rclone.Remote{}, cfg, nil, nil, nil, false)

	if form == nil {
		t.Fatal("NewMountForm() returned nil")
	}

	// Form should still be created even with no remotes
	if len(form.remotes) != 0 {
		t.Errorf("remotes count = %d, want 0", len(form.remotes))
	}
}

func TestMountForm_NoRemotesShowsHelpfulMessage(t *testing.T) {
	form := NewMountForm(nil, []rclone.Remote{}, nil, nil, nil, nil, false)
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
	form.mountPoint = "/mnt/test"
	form.remote = ""
	msg := form.submitForm()
	errMsg, ok := msg.(MountsErrorMsg)
	if !ok {
		t.Fatalf("expected MountsErrorMsg, got %T", msg)
	}
	if !strings.Contains(errMsg.Err.Error(), "rclone config") {
		t.Errorf("error should mention 'rclone config', got: %s", errMsg.Err.Error())
	}
}

func TestMountForm_ExpandPathWithHomeError(t *testing.T) {
	// Test expandPath function directly
	result := expandPath("~/test")
	home, err := os.UserHomeDir()
	if err == nil {
		expected := filepath.Join(home, "test")
		if result != expected {
			t.Errorf("expandPath('~/test') = %q, want %q", result, expected)
		}
	}
}

func TestMountForm_RollbackPreparationCreatesCopy(t *testing.T) {
	cfg := createTestConfig()
	cfg.Mounts = []models.MountConfig{
		{ID: "abc12345", Name: "Mount1"},
	}

	mgr := NewRollbackManager(cfg, nil, nil)
	data := mgr.PrepareMountRollback("new12345", "NewMount", OperationCreate)

	cfg.Mounts[0].Name = "ModifiedMount"

	if data.OriginalMounts[0].Name != "Mount1" {
		t.Error("OriginalMounts should be independent copy")
	}
}
