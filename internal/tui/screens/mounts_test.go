package screens

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/rclone"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
)

// Test errors for mounts
var errTestMountNotFound = errors.New("mount not found")

// Helper function to create a test mounts screen with services
func createTestMountsScreen() *MountsScreen {
	return NewMountsScreen()
}

// Helper function to create test mount configurations
func createTestMounts() []models.MountConfig {
	return []models.MountConfig{
		{
			ID:          "a1b2c3d4",
			Name:        "Google Drive",
			Remote:      "gdrive",
			RemotePath:  "/",
			MountPoint:  "/mnt/gdrive",
			Description: "Google Drive mount",
			MountOptions: models.MountOptions{
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
			AutoStart: true,
			Enabled:   true,
		},
		{
			ID:          "b2c3d4e5",
			Name:        "Dropbox",
			Remote:      "dropbox",
			RemotePath:  "/Photos",
			MountPoint:  "/mnt/dropbox",
			Description: "Dropbox photos mount",
			MountOptions: models.MountOptions{
				VFSCacheMode: "writes",
				BufferSize:   "32M",
			},
			AutoStart: false,
			Enabled:   true,
		},
		{
			ID:          "c3d4e5f6",
			Name:        "S3 Bucket",
			Remote:      "s3",
			RemotePath:  "/backup",
			MountPoint:  "/mnt/s3",
			Description: "S3 backup bucket",
			MountOptions: models.MountOptions{
				VFSCacheMode: "off",
				BufferSize:   "64M",
			},
			AutoStart: true,
			Enabled:   false,
		},
	}
}

// Helper function to create a test config with mounts
func createTestConfigWithMounts() *config.Config {
	return &config.Config{
		Version: "1.0",
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
			},
		},
		Mounts:   createTestMounts(),
		SyncJobs: []models.SyncJobConfig{},
	}
}

func TestNewMountsScreen(t *testing.T) {
	screen := NewMountsScreen()

	if screen == nil {
		t.Fatal("NewMountsScreen() returned nil")
	}

	// Verify initial mode
	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}

	// Verify statuses map is initialized
	if screen.statuses == nil {
		t.Error("statuses should be initialized")
	}

	// Verify initial cursor
	if screen.cursor != 0 {
		t.Errorf("cursor = %d, want 0", screen.cursor)
	}

	// Verify goBack is false
	if screen.goBack {
		t.Error("goBack should be false initially")
	}

	// Verify mounts is nil/empty
	if len(screen.mounts) != 0 {
		t.Errorf("mounts should be empty initially, got %d items", len(screen.mounts))
	}
}

func TestMountsScreen_SetSize(t *testing.T) {
	screen := NewMountsScreen()

	// Set size
	screen.SetSize(100, 30)

	if screen.width != 100 {
		t.Errorf("width = %d, want 100", screen.width)
	}

	if screen.height != 30 {
		t.Errorf("height = %d, want 30", screen.height)
	}
}

func TestMountsScreen_SetSizeWithForm(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)

	// Create a form and set it
	cfg := createTestConfigWithMounts()
	remotes := []rclone.Remote{{Name: "gdrive", Type: "drive"}}
	screen.form = NewMountForm(nil, remotes, cfg, nil, nil, nil, false)

	// Set size should propagate to form
	screen.SetSize(120, 40)

	if screen.width != 120 {
		t.Errorf("width = %d, want 120", screen.width)
	}

	if screen.form.width != 120 {
		t.Errorf("form width = %d, want 120", screen.form.width)
	}
}

func TestMountsScreen_CursorNavigation(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()

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
	for i := 0; i < len(screen.mounts)-1; i++ {
		screen.Update(tea.KeyMsg{Type: tea.KeyDown})
		expected := i + 1
		if screen.cursor != expected {
			t.Errorf("cursor after down %d times = %d, want %d", i+1, screen.cursor, expected)
		}
	}

	// Try to move down past last item - should stay at last
	lastIndex := len(screen.mounts) - 1
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

func TestMountsScreen_VimNavigation(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()

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

func TestMountsScreen_ModeTransitions(t *testing.T) {
	tests := []struct {
		name         string
		key          tea.KeyMsg
		setupScreen  func(*MountsScreen)
		expectedMode MountsScreenMode
	}{
		{
			name:         "Delete mode transition",
			key:          tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")},
			setupScreen:  func(s *MountsScreen) { s.mounts = createTestMounts() },
			expectedMode: MountsModeDelete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewMountsScreen()
			screen.SetSize(80, 24)
			tt.setupScreen(screen)

			// Ensure cursor is valid
			if screen.cursor >= len(screen.mounts) {
				screen.cursor = 0
			}

			screen.Update(tt.key)

			if screen.mode != tt.expectedMode {
				t.Errorf("mode = %d, want %d", screen.mode, tt.expectedMode)
			}
		})
	}
}

func TestMountsScreen_DetailsModeTransition(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.generator = &systemd.Generator{}
	screen.manager = &systemd.Manager{}

	// Ensure cursor is valid
	screen.cursor = 0

	// Press Enter to go to details mode
	screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if screen.mode != MountsModeDetails {
		t.Errorf("mode = %d, want %d (MountsModeDetails)", screen.mode, MountsModeDetails)
	}
}

func TestMountsScreen_DeleteModeNoMounts(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	// No mounts

	// Try to delete
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	// Should stay in list mode
	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}

	// delete should be nil
	if screen.delete != nil {
		t.Error("delete should be nil when no mounts")
	}
}

func TestMountsScreen_LoadMounts(t *testing.T) {
	screen := NewMountsScreen()
	cfg := createTestConfigWithMounts()
	screen.config = cfg

	// Create mock generator and manager
	screen.generator = &systemd.Generator{}
	screen.manager = &systemd.Manager{}

	// Call loadMounts
	msg := screen.loadMounts()

	// Check message type
	loadedMsg, ok := msg.(MountsLoadedMsg)
	if !ok {
		t.Fatalf("expected MountsLoadedMsg, got %T", msg)
	}

	// Verify mounts were loaded
	if len(loadedMsg.Mounts) != len(cfg.Mounts) {
		t.Errorf("loaded mounts = %d, want %d", len(loadedMsg.Mounts), len(cfg.Mounts))
	}
}

func TestMountsScreen_LoadMountsNilConfig(t *testing.T) {
	screen := NewMountsScreen()
	// Don't set config - it should be nil

	// Call loadMounts
	msg := screen.loadMounts()

	// Should return an error message
	errMsg, ok := msg.(MountsErrorMsg)
	if !ok {
		t.Fatalf("expected MountsErrorMsg, got %T", msg)
	}

	if errMsg.Err == nil {
		t.Error("expected error, got nil")
	}

	if !strings.Contains(errMsg.Err.Error(), "config not initialized") {
		t.Errorf("error = %q, should contain 'config not initialized'", errMsg.Err.Error())
	}
}

func TestMountsScreen_MountsLoadedMsg(t *testing.T) {
	screen := NewMountsScreen()
	screen.loading = true

	mounts := createTestMounts()
	msg := MountsLoadedMsg{Mounts: mounts}

	screen.Update(msg)

	// Verify mounts were set
	if len(screen.mounts) != len(mounts) {
		t.Errorf("mounts = %d, want %d", len(screen.mounts), len(mounts))
	}

	// Verify loading is false
	if screen.loading {
		t.Error("loading should be false after loading")
	}
}

func TestMountsScreen_MountCreatedMsg(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()

	newMount := models.MountConfig{
		ID:          "d4e5f6g7",
		Name:        "New Mount",
		Remote:      "newremote",
		RemotePath:  "/",
		MountPoint:  "/mnt/new",
		Description: "New mount",
	}

	msg := MountCreatedMsg{Mount: newMount}
	screen.Update(msg)

	// Verify mount was added
	if len(screen.mounts) != 4 {
		t.Errorf("mounts = %d, want 4", len(screen.mounts))
	}

	// Verify success message
	if screen.success == "" {
		t.Error("success message should be set")
	}

	// Verify mode is back to list
	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}

	// Verify error is cleared
	if screen.err != nil {
		t.Errorf("error should be cleared, got %v", screen.err)
	}
}

func TestMountsScreen_MountUpdatedMsg(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()

	// Update first mount
	updatedMount := screen.mounts[0]
	updatedMount.MountPoint = "/mnt/updated"

	msg := MountUpdatedMsg{Mount: updatedMount}
	screen.Update(msg)

	// Verify mount was updated
	if screen.mounts[0].MountPoint != "/mnt/updated" {
		t.Errorf("mount point = %q, want '/mnt/updated'", screen.mounts[0].MountPoint)
	}

	// Verify success message
	if screen.success == "" {
		t.Error("success message should be set")
	}

	// Verify mode is back to list
	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
}

func TestMountsScreen_MountDeletedMsg(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 1

	msg := MountDeletedMsg{Name: "Dropbox"}
	screen.Update(msg)

	// Verify mount was removed
	if len(screen.mounts) != 2 {
		t.Errorf("mounts = %d, want 2", len(screen.mounts))
	}

	// Verify cursor was reset
	if screen.cursor != 0 {
		t.Errorf("cursor = %d, want 0", screen.cursor)
	}

	// Verify success message
	if screen.success == "" {
		t.Error("success message should be set")
	}
}

func TestMountsScreen_MountStatusMsg(t *testing.T) {
	screen := NewMountsScreen()
	screen.statuses = make(map[string]*systemd.ServiceStatus)

	status := &systemd.ServiceStatus{
		Active: true,
		State:  "running",
	}

	msg := MountStatusMsg{Name: "Google Drive", Status: status}
	screen.Update(msg)

	// Verify status was set
	if screen.statuses["Google Drive"] != status {
		t.Error("status should be set for 'Google Drive'")
	}
}

func TestMountsScreen_ErrorMsg(t *testing.T) {
	screen := NewMountsScreen()
	screen.loading = true

	msg := MountsErrorMsg{Err: errTestMountNotFound}
	screen.Update(msg)

	// Verify error was set
	if screen.err != errTestMountNotFound {
		t.Errorf("error = %v, want %v", screen.err, errTestMountNotFound)
	}

	// Verify loading is false
	if screen.loading {
		t.Error("loading should be false after error")
	}
}

func TestMountsScreen_FormCancelMsg(t *testing.T) {
	screen := NewMountsScreen()
	screen.mode = MountsModeCreate
	screen.form = &MountForm{}

	msg := MountFormCancelMsg{}
	screen.Update(msg)

	// Verify mode is back to list
	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}

	// Verify form is nil
	if screen.form != nil {
		t.Error("form should be nil after cancel")
	}

	// Verify error is cleared
	if screen.err != nil {
		t.Errorf("error should be cleared, got %v", screen.err)
	}
}

func TestMountsScreen_EscapeKey(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)

	// Press escape
	screen.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if !screen.ShouldGoBack() {
		t.Error("ShouldGoBack() = false, want true")
	}
}

func TestMountsScreen_GoBack(t *testing.T) {
	screen := NewMountsScreen()
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

func TestMountsScreen_ResetGoBack(t *testing.T) {
	screen := NewMountsScreen()
	screen.goBack = true

	screen.ResetGoBack()

	if screen.goBack {
		t.Error("goBack should be false after ResetGoBack")
	}
}

func TestMountsScreen_View(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.loading = false // Set to false to show mount list
	screen.mounts = createTestMounts()

	view := screen.View()

	// Check title is rendered
	if !strings.Contains(view, "Mount Management") {
		t.Error("View() should contain 'Mount Management' title")
	}

	// Check mount names are rendered
	for _, mount := range screen.mounts {
		if !strings.Contains(view, mount.Name) {
			t.Errorf("View() should contain mount name '%s'", mount.Name)
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

func TestMountsScreen_ViewEmpty(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.loading = false // Set to false to show empty state
	// No mounts

	view := screen.View()

	// Check empty state message
	if !strings.Contains(view, "No mounts configured") {
		t.Error("View() should contain 'No mounts configured' message")
	}

	// Check add hint
	if !strings.Contains(view, "'a' to add") {
		t.Error("View() should contain hint to add mount")
	}
}

func TestMountsScreen_ViewLoading(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.loading = true

	view := screen.View()

	// Check loading message
	if !strings.Contains(view, "Loading mounts") {
		t.Error("View() should contain 'Loading mounts' message")
	}
}

func TestMountsScreen_ViewWithError(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.err = errTestMountNotFound

	view := screen.View()

	// Check error is rendered
	if !strings.Contains(view, errTestMountNotFound.Error()) {
		t.Errorf("View() should contain error message '%s'", errTestMountNotFound.Error())
	}
}

func TestMountsScreen_ViewWithSuccess(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.success = "Mount created successfully"

	view := screen.View()

	// Check success message is rendered
	if !strings.Contains(view, "Mount created successfully") {
		t.Error("View() should contain success message")
	}
}

func TestMountsScreen_ViewDeleteMode(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.mode = MountsModeDelete
	screen.delete = NewDeleteConfirm(screen.mounts[0])

	view := screen.View()

	// Check delete dialog is rendered
	if !strings.Contains(view, "Delete Mount") {
		t.Error("View() should contain 'Delete Mount' title in delete mode")
	}

	if !strings.Contains(view, "Are you sure") {
		t.Error("View() should contain confirmation message")
	}
}

func TestMountsScreen_ViewDetailsMode(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.mode = MountsModeDetails
	screen.generator = &systemd.Generator{}
	screen.manager = &systemd.Manager{}
	screen.details = NewMountDetails(screen.mounts[0], screen.manager, screen.generator)
	screen.details.SetSize(80, 24) // Set size on details component

	view := screen.View()

	// Check details view is rendered
	if !strings.Contains(view, "Mount:") {
		t.Error("View() should contain 'Mount:' title in details mode")
	}
}

func TestMountsScreen_ViewFormMode(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mode = MountsModeCreate
	screen.form = NewMountForm(nil, []rclone.Remote{{Name: "gdrive", Type: "drive"}}, nil, nil, nil, nil, false)

	view := screen.View()

	// Check form is rendered
	if !strings.Contains(view, "Create New Mount") {
		t.Error("View() should contain 'Create New Mount' title in create mode")
	}
}

func TestMountsScreen_Init(t *testing.T) {
	screen := NewMountsScreen()

	cmd := screen.Init()

	// Init should return a command (loadMounts)
	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestMountsScreen_SetServices(t *testing.T) {
	screen := NewMountsScreen()
	cfg := &config.Config{}
	rcloneClient := &rclone.Client{}
	gen := &systemd.Generator{}
	mgr := &systemd.Manager{}

	screen.SetServices(cfg, rcloneClient, gen, mgr)

	if screen.config != cfg {
		t.Error("config should be set")
	}
	if screen.rclone != rcloneClient {
		t.Error("rclone should be set")
	}
	if screen.generator != gen {
		t.Error("generator should be set")
	}
	if screen.manager != mgr {
		t.Error("manager should be set")
	}
}

// Tests for DeleteConfirm component

func TestNewDeleteConfirm(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)

	if dialog == nil {
		t.Fatal("NewDeleteConfirm() returned nil")
	}

	// Verify initial state
	if dialog.cursor != 0 {
		t.Errorf("cursor = %d, want 0", dialog.cursor)
	}

	if dialog.done {
		t.Error("done should be false initially")
	}

	if dialog.mount.Name != mount.Name {
		t.Errorf("mount name = %q, want %q", dialog.mount.Name, mount.Name)
	}
}

func TestDeleteConfirm_Navigation(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)

	// Move right (cursor should increase)
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if dialog.cursor != 1 {
		t.Errorf("cursor after 'l' = %d, want 1", dialog.cursor)
	}

	// Move right again
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if dialog.cursor != 2 {
		t.Errorf("cursor after 'l' = %d, want 2", dialog.cursor)
	}

	// Try to move past max (should stay at 2)
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if dialog.cursor != 2 {
		t.Errorf("cursor after 'l' at max = %d, want 2", dialog.cursor)
	}

	// Move left
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if dialog.cursor != 1 {
		t.Errorf("cursor after 'h' = %d, want 1", dialog.cursor)
	}
}

func TestDeleteConfirm_ArrowNavigation(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)

	// Move right with arrow
	dialog.Update(tea.KeyMsg{Type: tea.KeyRight})
	if dialog.cursor != 1 {
		t.Errorf("cursor after right = %d, want 1", dialog.cursor)
	}

	// Move left with arrow
	dialog.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if dialog.cursor != 0 {
		t.Errorf("cursor after left = %d, want 0", dialog.cursor)
	}
}

func TestDeleteConfirm_Escape(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)

	// Press escape
	dialog.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if !dialog.done {
		t.Error("done should be true after escape")
	}
}

func TestDeleteConfirm_EnterCancel(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)
	dialog.cursor = 0 // Cancel option

	// Press enter
	dialog.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !dialog.done {
		t.Error("done should be true after cancel")
	}
}

func TestDeleteConfirm_IsDone(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)

	if dialog.IsDone() {
		t.Error("IsDone() = true initially, want false")
	}

	dialog.done = true

	if !dialog.IsDone() {
		t.Error("IsDone() = false after setting done, want true")
	}
}

func TestDeleteConfirm_View(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)
	dialog.width = 80

	view := dialog.View()

	// Check title
	if !strings.Contains(view, "Delete Mount") {
		t.Error("View() should contain 'Delete Mount' title")
	}

	// Check mount name
	if !strings.Contains(view, mount.Name) {
		t.Errorf("View() should contain mount name '%s'", mount.Name)
	}

	// Check options
	if !strings.Contains(view, "Cancel") {
		t.Error("View() should contain 'Cancel' option")
	}

	if !strings.Contains(view, "Delete Service Only") {
		t.Error("View() should contain 'Delete Service Only' option")
	}

	if !strings.Contains(view, "Delete Service and Config") {
		t.Error("View() should contain 'Delete Service and Config' option")
	}

	// Check help text
	if !strings.Contains(view, "Enter") {
		t.Error("View() should contain help for Enter key")
	}

	if !strings.Contains(view, "Esc") {
		t.Error("View() should contain help for Esc key")
	}
}

func TestDeleteConfirm_SetServices(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)

	mgr := &systemd.Manager{}
	gen := &systemd.Generator{}
	cfg := &config.Config{}

	dialog.SetServices(mgr, gen, cfg)

	if dialog.manager != mgr {
		t.Error("manager should be set")
	}
	if dialog.generator != gen {
		t.Error("generator should be set")
	}
	if dialog.config != cfg {
		t.Error("config should be set")
	}
}

func TestDeleteConfirm_SetSize(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)

	dialog.SetSize(100, 30)

	if dialog.width != 100 {
		t.Errorf("width = %d, want 100", dialog.width)
	}
}

// Tests for MountDetails component

func TestNewMountDetails(t *testing.T) {
	mount := createTestMounts()[0]
	gen := &systemd.Generator{}
	mgr := &systemd.Manager{}
	details := NewMountDetails(mount, mgr, gen)

	if details == nil {
		t.Fatal("NewMountDetails() returned nil")
	}

	// Verify initial state
	if details.done {
		t.Error("done should be false initially")
	}

	if details.tab != 0 {
		t.Errorf("tab = %d, want 0", details.tab)
	}

	if details.mount.Name != mount.Name {
		t.Errorf("mount name = %q, want %q", details.mount.Name, mount.Name)
	}
}

func TestMountDetails_TabSwitching(t *testing.T) {
	mount := createTestMounts()[0]
	gen := &systemd.Generator{}
	mgr := &systemd.Manager{}
	details := NewMountDetails(mount, mgr, gen)

	// Initial tab is 0 (Details)
	if details.tab != 0 {
		t.Errorf("initial tab = %d, want 0", details.tab)
	}

	// Press tab to switch to Logs
	details.Update(tea.KeyMsg{Type: tea.KeyTab})
	if details.tab != 1 {
		t.Errorf("tab after Tab = %d, want 1", details.tab)
	}

	// Press tab again to wrap around to Details
	details.Update(tea.KeyMsg{Type: tea.KeyTab})
	if details.tab != 0 {
		t.Errorf("tab after Tab wrap = %d, want 0", details.tab)
	}
}

func TestMountDetails_Escape(t *testing.T) {
	mount := createTestMounts()[0]
	gen := &systemd.Generator{}
	mgr := &systemd.Manager{}
	details := NewMountDetails(mount, mgr, gen)

	// Press escape
	details.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if !details.done {
		t.Error("done should be true after escape")
	}
}

func TestMountDetails_QKey(t *testing.T) {
	mount := createTestMounts()[0]
	gen := &systemd.Generator{}
	mgr := &systemd.Manager{}
	details := NewMountDetails(mount, mgr, gen)

	// Press 'q'
	details.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if !details.done {
		t.Error("done should be true after 'q'")
	}
}

func TestMountDetails_IsDone(t *testing.T) {
	mount := createTestMounts()[0]
	gen := &systemd.Generator{}
	mgr := &systemd.Manager{}
	details := NewMountDetails(mount, mgr, gen)

	if details.IsDone() {
		t.Error("IsDone() = true initially, want false")
	}

	details.done = true

	if !details.IsDone() {
		t.Error("IsDone() = false after setting done, want true")
	}
}

func TestMountDetails_View(t *testing.T) {
	mount := createTestMounts()[0]
	gen := &systemd.Generator{}
	mgr := &systemd.Manager{}
	details := NewMountDetails(mount, mgr, gen)
	details.width = 80

	view := details.View()

	// Check title
	if !strings.Contains(view, "Mount:") {
		t.Error("View() should contain 'Mount:' title")
	}

	// Check mount name
	if !strings.Contains(view, mount.Name) {
		t.Errorf("View() should contain mount name '%s'", mount.Name)
	}

	// Check tabs
	if !strings.Contains(view, "Details") {
		t.Error("View() should contain 'Details' tab")
	}

	if !strings.Contains(view, "Logs") {
		t.Error("View() should contain 'Logs' tab")
	}

	// Check mount info
	if !strings.Contains(view, mount.Remote) {
		t.Errorf("View() should contain remote '%s'", mount.Remote)
	}

	if !strings.Contains(view, mount.MountPoint) {
		t.Errorf("View() should contain mount point '%s'", mount.MountPoint)
	}
}

func TestMountDetails_ViewLogsTab(t *testing.T) {
	mount := createTestMounts()[0]
	gen := &systemd.Generator{}
	mgr := &systemd.Manager{}
	details := NewMountDetails(mount, mgr, gen)
	details.width = 80
	details.tab = 1 // Logs tab
	details.logs = "Sample log line 1\nSample log line 2"

	view := details.View()

	// Check logs are rendered
	if !strings.Contains(view, "Sample log line") {
		t.Error("View() should contain log content")
	}
}

func TestMountDetails_ViewLogsEmpty(t *testing.T) {
	mount := createTestMounts()[0]
	gen := &systemd.Generator{}
	mgr := &systemd.Manager{}
	details := NewMountDetails(mount, mgr, gen)
	details.width = 80
	details.tab = 1 // Logs tab
	details.logs = ""

	view := details.View()

	// Check empty logs message
	if !strings.Contains(view, "No logs available") {
		t.Error("View() should contain 'No logs available' message")
	}
}

func TestMountDetails_SetSize(t *testing.T) {
	mount := createTestMounts()[0]
	gen := &systemd.Generator{}
	mgr := &systemd.Manager{}
	details := NewMountDetails(mount, mgr, gen)

	details.SetSize(100, 30)

	if details.width != 100 {
		t.Errorf("width = %d, want 100", details.width)
	}

	if details.height != 30 {
		t.Errorf("height = %d, want 30", details.height)
	}
}

func TestMountDetails_Init(t *testing.T) {
	mount := createTestMounts()[0]
	gen := &systemd.Generator{}
	mgr := &systemd.Manager{}
	details := NewMountDetails(mount, mgr, gen)

	cmd := details.Init()

	if cmd != nil {
		t.Error("Init() should return nil command")
	}
}

// Tests for updateForm mode handling

func TestMountsScreen_UpdateFormNilForm(t *testing.T) {
	screen := NewMountsScreen()
	screen.mode = MountsModeCreate
	// form is nil

	// Send any key
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	// Should return to list mode
	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
}

// Tests for updateDelete mode handling

func TestMountsScreen_UpdateDeleteNilDelete(t *testing.T) {
	screen := NewMountsScreen()
	screen.mode = MountsModeDelete
	// delete is nil

	// Send any key
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	// Should return to list mode
	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
}

// Tests for updateDetails mode handling

func TestMountsScreen_UpdateDetailsNilDetails(t *testing.T) {
	screen := NewMountsScreen()
	screen.mode = MountsModeDetails
	// details is nil

	// Send any key
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	// Should return to list mode
	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
}

// Tests for refresh key

func TestMountsScreen_RefreshKey(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.config = createTestConfigWithMounts()
	screen.generator = &systemd.Generator{}
	screen.manager = &systemd.Manager{}

	// Press 'r' to refresh
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	// Should return a command
	if cmd == nil {
		t.Error("Update should return a command for refresh")
	}
}

// Tests for edit key with no mounts

func TestMountsScreen_EditNoMounts(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	// No mounts

	// Try to edit
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})

	// Should stay in list mode
	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
}

// Tests for enter key with no mounts

func TestMountsScreen_EnterNoMounts(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	// No mounts

	// Press enter
	screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should stay in list mode
	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
}

// Tests for getMountStatus

func TestMountsScreen_GetMountStatus(t *testing.T) {
	screen := NewMountsScreen()
	screen.statuses = make(map[string]*systemd.ServiceStatus)

	mount := &models.MountConfig{Name: "TestMount"}

	// Test unknown status
	status := screen.getMountStatus(mount)
	if !strings.Contains(status, "unknown") {
		t.Errorf("status for unknown mount = %q, should contain 'unknown'", status)
	}

	// Test active status
	screen.statuses["TestMount"] = &systemd.ServiceStatus{Active: true}
	status = screen.getMountStatus(mount)
	if !strings.Contains(status, "running") {
		t.Errorf("status for active mount = %q, should contain 'running'", status)
	}

	// Test inactive status
	screen.statuses["TestMount"] = &systemd.ServiceStatus{Active: false}
	status = screen.getMountStatus(mount)
	if !strings.Contains(status, "stopped") {
		t.Errorf("status for inactive mount = %q, should contain 'stopped'", status)
	}
}

// Tests for renderMountDetails

func TestMountsScreen_RenderMountDetails(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 0
	screen.statuses = make(map[string]*systemd.ServiceStatus)
	screen.statuses["Google Drive"] = &systemd.ServiceStatus{Active: true}

	details := screen.renderMountDetails()

	if !strings.Contains(details, "Google Drive") {
		t.Error("renderMountDetails should contain mount name")
	}

	if !strings.Contains(details, "gdrive") {
		t.Error("renderMountDetails should contain remote name")
	}

	if !strings.Contains(details, "/mnt/gdrive") {
		t.Error("renderMountDetails should contain mount point")
	}
}

func TestMountsScreen_StartCreateForm_NilRclone(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.rclone = nil

	model, cmd := screen.startCreateForm()

	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
	if screen.err == nil {
		t.Error("error should be set when rclone client is nil")
	}
	if !strings.Contains(screen.err.Error(), "rclone client not initialized") {
		t.Errorf("error = %q, should contain 'rclone client not initialized'", screen.err.Error())
	}
	if cmd != nil {
		t.Error("startCreateForm should return nil command when rclone client is nil")
	}
	if model == nil {
		t.Error("startCreateForm should return a model")
	}
}

func TestMountsScreen_StartEditForm_NilRclone(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 0
	screen.rclone = nil

	model, cmd := screen.startEditForm()

	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
	if screen.err == nil {
		t.Error("error should be set when rclone client is nil")
	}
	if !strings.Contains(screen.err.Error(), "rclone client not initialized") {
		t.Errorf("error = %q, should contain 'rclone client not initialized'", screen.err.Error())
	}
	if cmd != nil {
		t.Error("startEditForm should return nil command when rclone client is nil")
	}
	if model == nil {
		t.Error("startEditForm should return a model")
	}
}

func TestMountsScreen_ToggleMount_NilServices(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 0
	screen.generator = nil
	screen.manager = nil

	model, cmd := screen.toggleMount()

	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
	if screen.err == nil {
		t.Error("error should be set when services are nil")
	}
	if !strings.Contains(screen.err.Error(), "systemd services not initialized") {
		t.Errorf("error = %q, should contain 'systemd services not initialized'", screen.err.Error())
	}
	if cmd != nil {
		t.Error("toggleMount should return nil command when services are nil")
	}
	if model == nil {
		t.Error("toggleMount should return a model")
	}
}

func TestMountsScreen_ToggleMount_NilGenerator(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 0
	screen.manager = &systemd.Manager{}
	screen.generator = nil

	model, cmd := screen.toggleMount()

	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
	if screen.err == nil {
		t.Error("error should be set when generator is nil")
	}
	if cmd != nil {
		t.Error("toggleMount should return nil command when generator is nil")
	}
	if model == nil {
		t.Error("toggleMount should return a model")
	}
}

func TestMountsScreen_ToggleMount_NilManager(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 0
	screen.generator = &systemd.Generator{}
	screen.manager = nil

	model, cmd := screen.toggleMount()

	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
	if screen.err == nil {
		t.Error("error should be set when manager is nil")
	}
	if cmd != nil {
		t.Error("toggleMount should return nil command when manager is nil")
	}
	if model == nil {
		t.Error("toggleMount should return a model")
	}
}

func TestMountsScreen_StartMount_NilServices(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 0
	screen.generator = nil
	screen.manager = nil

	model, cmd := screen.startMount()

	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
	if screen.err == nil {
		t.Error("error should be set when services are nil")
	}
	if !strings.Contains(screen.err.Error(), "systemd services not initialized") {
		t.Errorf("error = %q, should contain 'systemd services not initialized'", screen.err.Error())
	}
	if cmd != nil {
		t.Error("startMount should return nil command when services are nil")
	}
	if model == nil {
		t.Error("startMount should return a model")
	}
}

func TestMountsScreen_StartMount_WithServices(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 0
	screen.generator = &systemd.Generator{}
	screen.manager = &systemd.Manager{}

	model, cmd := screen.startMount()

	if screen.err != nil {
		t.Errorf("unexpected error: %v", screen.err)
	}
	if model == nil {
		t.Error("startMount should return a model")
	}
	if cmd == nil {
		t.Error("startMount should return a command with services set")
	}
}

func TestMountsScreen_StopMount_NilServices(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 0
	screen.generator = nil
	screen.manager = nil

	model, cmd := screen.stopMount()

	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
	if screen.err == nil {
		t.Error("error should be set when services are nil")
	}
	if !strings.Contains(screen.err.Error(), "systemd services not initialized") {
		t.Errorf("error = %q, should contain 'systemd services not initialized'", screen.err.Error())
	}
	if cmd != nil {
		t.Error("stopMount should return nil command when services are nil")
	}
	if model == nil {
		t.Error("stopMount should return a model")
	}
}

func TestMountsScreen_StopMount_WithServices(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 0
	screen.generator = &systemd.Generator{}
	screen.manager = &systemd.Manager{}

	model, cmd := screen.stopMount()

	if screen.err != nil {
		t.Errorf("unexpected error: %v", screen.err)
	}
	if model == nil {
		t.Error("stopMount should return a model")
	}
	if cmd == nil {
		t.Error("stopMount should return a command with services set")
	}
}

func TestMountsScreen_UpdateForm_WithForm(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	cfg := createTestConfigWithMounts()
	remotes := []rclone.Remote{{Name: "gdrive", Type: "drive"}}
	screen.form = NewMountForm(nil, remotes, cfg, nil, nil, nil, false)
	screen.mode = MountsModeCreate

	_ = screen.form.Init()

	model, _ := screen.Update(tea.KeyMsg{Type: tea.KeyDown})

	if screen.mode != MountsModeCreate {
		t.Errorf("mode = %d, want %d (MountsModeCreate)", screen.mode, MountsModeCreate)
	}
	if model == nil {
		t.Error("Update should return a model")
	}
}

func TestMountsScreen_UpdateForm_FormDone(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	cfg := createTestConfigWithMounts()
	remotes := []rclone.Remote{{Name: "gdrive", Type: "drive"}}
	screen.form = NewMountForm(nil, remotes, cfg, nil, nil, nil, false)
	screen.form.done = true
	screen.mode = MountsModeCreate

	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
	if screen.form != nil {
		t.Error("form should be nil after form is done")
	}
}

func TestMountsScreen_UpdateDelete_WithDelete(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.delete = NewDeleteConfirm(screen.mounts[0])
	screen.mode = MountsModeDelete

	model, _ := screen.Update(tea.KeyMsg{Type: tea.KeyRight})

	if screen.mode != MountsModeDelete {
		t.Errorf("mode = %d, want %d (MountsModeDelete)", screen.mode, MountsModeDelete)
	}
	if screen.delete.cursor != 1 {
		t.Errorf("delete cursor = %d, want 1", screen.delete.cursor)
	}
	if model == nil {
		t.Error("Update should return a model")
	}
}

func TestMountsScreen_UpdateDelete_DeleteDone(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.delete = NewDeleteConfirm(screen.mounts[0])
	screen.delete.done = true
	screen.mode = MountsModeDelete

	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
	if screen.delete != nil {
		t.Error("delete should be nil after delete is done")
	}
}

func TestMountsScreen_UpdateDetails_WithDetails(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.generator = &systemd.Generator{}
	screen.manager = &systemd.Manager{}
	screen.details = NewMountDetails(screen.mounts[0], screen.manager, screen.generator)
	screen.mode = MountsModeDetails

	model, _ := screen.Update(tea.KeyMsg{Type: tea.KeyTab})

	if screen.mode != MountsModeDetails {
		t.Errorf("mode = %d, want %d (MountsModeDetails)", screen.mode, MountsModeDetails)
	}
	if screen.details.tab != 1 {
		t.Errorf("details tab = %d, want 1", screen.details.tab)
	}
	if model == nil {
		t.Error("Update should return a model")
	}
}

func TestMountsScreen_UpdateDetails_DetailsDone(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.generator = &systemd.Generator{}
	screen.manager = &systemd.Manager{}
	screen.details = NewMountDetails(screen.mounts[0], screen.manager, screen.generator)
	screen.details.done = true
	screen.mode = MountsModeDetails

	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
	if screen.details != nil {
		t.Error("details should be nil after details is done")
	}
}

func TestMountsScreen_StartMountKey(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 0
	screen.generator = &systemd.Generator{}
	screen.manager = &systemd.Manager{}

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	if cmd == nil {
		t.Error("Update should return a command for start mount")
	}
}

func TestMountsScreen_StartMountKey_NoMounts(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = []models.MountConfig{}
	screen.generator = &systemd.Generator{}
	screen.manager = &systemd.Manager{}

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	if cmd != nil {
		t.Error("Update should not return a command when no mounts")
	}
}

func TestMountsScreen_StopMountKey(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 0
	screen.generator = &systemd.Generator{}
	screen.manager = &systemd.Manager{}

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if cmd == nil {
		t.Error("Update should return a command for stop mount")
	}
}

func TestMountsScreen_ToggleMountKey(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 0
	screen.generator = &systemd.Generator{}
	screen.manager = nil

	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})

	if screen.err == nil {
		t.Error("error should be set when manager is nil")
	}
}

func TestMountsScreen_ToggleMountKey_NoMounts(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = []models.MountConfig{}
	screen.generator = &systemd.Generator{}
	screen.manager = &systemd.Manager{}

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})

	if cmd != nil {
		t.Error("Update should not return a command when no mounts")
	}
}

func TestMountsScreen_AddMountKey_NoRclone(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.rclone = nil

	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	if screen.err == nil {
		t.Error("error should be set when rclone is nil")
	}
	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
}

func TestMountsScreen_EditKey_NoRclone(t *testing.T) {
	screen := NewMountsScreen()
	screen.SetSize(80, 24)
	screen.mounts = createTestMounts()
	screen.cursor = 0
	screen.rclone = nil

	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})

	if screen.err == nil {
		t.Error("error should be set when rclone is nil")
	}
	if screen.mode != MountsModeList {
		t.Errorf("mode = %d, want %d (MountsModeList)", screen.mode, MountsModeList)
	}
}

func TestDeleteConfirm_DeleteServiceOnly_NilManager(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)
	dialog.manager = nil
	dialog.generator = &systemd.Generator{}

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	cmd := dialog.deleteServiceOnly()
	if cmd == nil {
		t.Error("deleteServiceOnly should return a command even with nil manager")
		return
	}

	msg := cmd()
	_ = msg
}

func TestDeleteConfirm_DeleteServiceOnly_NilGenerator(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)
	dialog.manager = &systemd.Manager{}
	dialog.generator = nil

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	cmd := dialog.deleteServiceOnly()
	if cmd == nil {
		t.Error("deleteServiceOnly should return a command even with nil generator")
		return
	}

	msg := cmd()
	_ = msg
}

func TestDeleteConfirm_DeleteServiceOnly_WithServices(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)
	dialog.manager = &systemd.Manager{}
	dialog.generator = &systemd.Generator{}

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	cmd := dialog.deleteServiceOnly()
	if cmd == nil {
		t.Fatal("deleteServiceOnly should return a command")
	}

	msg := cmd()
	_ = msg
}

func TestDeleteConfirm_DeleteServiceAndConfig_NilManager(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)
	dialog.manager = nil
	dialog.generator = &systemd.Generator{}
	dialog.config = createTestConfigWithMounts()

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	cmd := dialog.deleteServiceAndConfig()
	if cmd == nil {
		t.Error("deleteServiceAndConfig should return a command even with nil manager")
		return
	}

	msg := cmd()
	_ = msg
}

func TestDeleteConfirm_DeleteServiceAndConfig_NilGenerator(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)
	dialog.manager = &systemd.Manager{}
	dialog.generator = nil
	dialog.config = createTestConfigWithMounts()

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	cmd := dialog.deleteServiceAndConfig()
	if cmd == nil {
		t.Error("deleteServiceAndConfig should return a command even with nil generator")
		return
	}

	msg := cmd()
	_ = msg
}

func TestDeleteConfirm_DeleteServiceAndConfig_NilConfig(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)
	dialog.manager = &systemd.Manager{}
	dialog.generator = &systemd.Generator{}
	dialog.config = nil

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	cmd := dialog.deleteServiceAndConfig()
	if cmd == nil {
		t.Error("deleteServiceAndConfig should return a command even with nil config")
		return
	}

	msg := cmd()
	_ = msg
}

func TestDeleteConfirm_DeleteServiceAndConfig_WithServices(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)
	dialog.manager = &systemd.Manager{}
	dialog.generator = &systemd.Generator{}
	dialog.config = createTestConfigWithMounts()

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	cmd := dialog.deleteServiceAndConfig()
	if cmd == nil {
		t.Fatal("deleteServiceAndConfig should return a command")
	}

	msg := cmd()
	_ = msg
}

func TestDeleteConfirm_EnterOnDeleteServiceOnly(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)
	dialog.cursor = 1
	dialog.manager = &systemd.Manager{}
	dialog.generator = &systemd.Generator{}

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	_, cmd := dialog.Update(tea.KeyMsg{Type: tea.KeyEnter})

	_ = cmd
}

func TestDeleteConfirm_EnterOnDeleteServiceAndConfig(t *testing.T) {
	mount := createTestMounts()[0]
	dialog := NewDeleteConfirm(mount)
	dialog.cursor = 2
	dialog.manager = &systemd.Manager{}
	dialog.generator = &systemd.Generator{}
	dialog.config = createTestConfigWithMounts()

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	_, cmd := dialog.Update(tea.KeyMsg{Type: tea.KeyEnter})

	_ = cmd
}

func TestDeleteConfirm_DeleteServiceOnly_ReturnsMountDeletedMsg(t *testing.T) {
	mount := models.MountConfig{
		ID:         "test1234",
		Name:       "TestMount",
		Remote:     "gdrive",
		RemotePath: "/",
		MountPoint: "/mnt/test",
	}
	dialog := NewDeleteConfirm(mount)
	dialog.manager = &systemd.Manager{}
	dialog.generator = &systemd.Generator{}

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	cmd := dialog.deleteServiceOnly()
	msg := cmd()
	_ = msg
}
