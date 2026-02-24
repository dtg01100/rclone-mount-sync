package screens

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
)

func TestNewSettingsScreen(t *testing.T) {
	screen := NewSettingsScreen()

	if screen == nil {
		t.Fatal("NewSettingsScreen() returned nil")
	}

	// Verify settings are initialized
	if len(screen.settings) == 0 {
		t.Fatal("settings should not be empty")
	}

	// Verify initial state
	if screen.cursor != 0 {
		t.Errorf("cursor = %d, want 0", screen.cursor)
	}

	if screen.goBack {
		t.Error("goBack should be false initially")
	}

	if screen.editing {
		t.Error("editing should be false initially")
	}
}

func TestSettingsScreen_SettingItems(t *testing.T) {
	screen := NewSettingsScreen()

	// Define expected settings
	expectedSettings := []struct {
		name        string
		key         string
		settingType string
		configKey   string
	}{
		{"Default VFS Cache Mode", "v", "select", "defaults.mount.vfs_cache_mode"},
		{"Default Buffer Size", "b", "string", "defaults.mount.buffer_size"},
		{"Default Mount Log Level", "l", "select", "defaults.mount.log_level"},
		{"Default Sync Log Level", "sl", "select", "defaults.sync.log_level"},
		{"Default Transfers", "t", "int", "defaults.sync.transfers"},
		{"Default Checkers", "c", "int", "defaults.sync.checkers"},
		{"Rclone Binary Path", "r", "string", "settings.rclone_binary_path"},
		{"Default Mount Directory", "m", "string", "settings.default_mount_dir"},
		{"Editor", "e", "string", "settings.editor"},
	}

	for i, expected := range expectedSettings {
		if i >= len(screen.settings) {
			t.Errorf("missing setting at index %d", i)
			continue
		}

		if screen.settings[i].Name != expected.name {
			t.Errorf("setting %d name = %q, want %q", i, screen.settings[i].Name, expected.name)
		}

		if screen.settings[i].Key != expected.key {
			t.Errorf("setting %d key = %q, want %q", i, screen.settings[i].Key, expected.key)
		}

		if screen.settings[i].settingType != expected.settingType {
			t.Errorf("setting %d settingType = %q, want %q", i, screen.settings[i].settingType, expected.settingType)
		}

		if screen.settings[i].configKey != expected.configKey {
			t.Errorf("setting %d configKey = %q, want %q", i, screen.settings[i].configKey, expected.configKey)
		}
	}
}

func TestSettingsScreen_GetConfigValue(t *testing.T) {
	tests := []struct {
		name          string
		configKey     string
		setupConfig   func(*config.Config)
		expectedValue string
	}{
		{
			name:          "VFS Cache Mode",
			configKey:     "defaults.mount.vfs_cache_mode",
			setupConfig:   func(c *config.Config) { c.Defaults.Mount.VFSCacheMode = "full" },
			expectedValue: "full",
		},
		{
			name:          "Buffer Size",
			configKey:     "defaults.mount.buffer_size",
			setupConfig:   func(c *config.Config) { c.Defaults.Mount.BufferSize = "32M" },
			expectedValue: "32M",
		},
		{
			name:          "Mount Log Level",
			configKey:     "defaults.mount.log_level",
			setupConfig:   func(c *config.Config) { c.Defaults.Mount.LogLevel = "DEBUG" },
			expectedValue: "DEBUG",
		},
		{
			name:          "Sync Log Level",
			configKey:     "defaults.sync.log_level",
			setupConfig:   func(c *config.Config) { c.Defaults.Sync.LogLevel = "ERROR" },
			expectedValue: "ERROR",
		},
		{
			name:          "Transfers",
			configKey:     "defaults.sync.transfers",
			setupConfig:   func(c *config.Config) { c.Defaults.Sync.Transfers = 8 },
			expectedValue: "8",
		},
		{
			name:          "Checkers",
			configKey:     "defaults.sync.checkers",
			setupConfig:   func(c *config.Config) { c.Defaults.Sync.Checkers = 16 },
			expectedValue: "16",
		},
		{
			name:          "Rclone Binary Path",
			configKey:     "settings.rclone_binary_path",
			setupConfig:   func(c *config.Config) { c.Settings.RcloneBinaryPath = "/usr/local/bin/rclone" },
			expectedValue: "/usr/local/bin/rclone",
		},
		{
			name:          "Default Mount Dir",
			configKey:     "settings.default_mount_dir",
			setupConfig:   func(c *config.Config) { c.Settings.DefaultMountDir = "~/mounts" },
			expectedValue: "~/mounts",
		},
		{
			name:          "Editor",
			configKey:     "settings.editor",
			setupConfig:   func(c *config.Config) { c.Settings.Editor = "vim" },
			expectedValue: "vim",
		},
		{
			name:          "Unknown config key",
			configKey:     "unknown.key",
			setupConfig:   func(c *config.Config) {},
			expectedValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewSettingsScreen()
			cfg := &config.Config{
				Defaults: config.DefaultConfig{
					Mount: config.MountDefaults{},
					Sync:  config.SyncDefaults{},
				},
				Settings: config.Settings{},
			}
			tt.setupConfig(cfg)
			screen.SetConfig(cfg)

			value := screen.getConfigValue(tt.configKey)

			if value != tt.expectedValue {
				t.Errorf("getConfigValue(%q) = %q, want %q", tt.configKey, value, tt.expectedValue)
			}
		})
	}
}

func TestSettingsScreen_GetConfigValueNilConfig(t *testing.T) {
	screen := NewSettingsScreen()
	// Don't set config - it should be nil

	value := screen.getConfigValue("defaults.mount.vfs_cache_mode")

	if value != "" {
		t.Errorf("getConfigValue with nil config = %q, want empty string", value)
	}
}

func TestSettingsScreen_SetConfigValue(t *testing.T) {
	tests := []struct {
		name        string
		configKey   string
		value       string
		checkConfig func(*testing.T, *config.Config)
		expectError bool
	}{
		{
			name:      "Set VFS Cache Mode",
			configKey: "defaults.mount.vfs_cache_mode",
			value:     "writes",
			checkConfig: func(t *testing.T, c *config.Config) {
				if c.Defaults.Mount.VFSCacheMode != "writes" {
					t.Errorf("VFSCacheMode = %q, want 'writes'", c.Defaults.Mount.VFSCacheMode)
				}
			},
		},
		{
			name:      "Set Buffer Size",
			configKey: "defaults.mount.buffer_size",
			value:     "64M",
			checkConfig: func(t *testing.T, c *config.Config) {
				if c.Defaults.Mount.BufferSize != "64M" {
					t.Errorf("BufferSize = %q, want '64M'", c.Defaults.Mount.BufferSize)
				}
			},
		},
		{
			name:      "Set Mount Log Level",
			configKey: "defaults.mount.log_level",
			value:     "NOTICE",
			checkConfig: func(t *testing.T, c *config.Config) {
				if c.Defaults.Mount.LogLevel != "NOTICE" {
					t.Errorf("LogLevel = %q, want 'NOTICE'", c.Defaults.Mount.LogLevel)
				}
			},
		},
		{
			name:      "Set Sync Log Level",
			configKey: "defaults.sync.log_level",
			value:     "DEBUG",
			checkConfig: func(t *testing.T, c *config.Config) {
				if c.Defaults.Sync.LogLevel != "DEBUG" {
					t.Errorf("LogLevel = %q, want 'DEBUG'", c.Defaults.Sync.LogLevel)
				}
			},
		},
		{
			name:      "Set Transfers",
			configKey: "defaults.sync.transfers",
			value:     "12",
			checkConfig: func(t *testing.T, c *config.Config) {
				if c.Defaults.Sync.Transfers != 12 {
					t.Errorf("Transfers = %d, want 12", c.Defaults.Sync.Transfers)
				}
			},
		},
		{
			name:      "Set Checkers",
			configKey: "defaults.sync.checkers",
			value:     "24",
			checkConfig: func(t *testing.T, c *config.Config) {
				if c.Defaults.Sync.Checkers != 24 {
					t.Errorf("Checkers = %d, want 24", c.Defaults.Sync.Checkers)
				}
			},
		},
		{
			name:      "Set Rclone Binary Path",
			configKey: "settings.rclone_binary_path",
			value:     "/opt/rclone/bin/rclone",
			checkConfig: func(t *testing.T, c *config.Config) {
				if c.Settings.RcloneBinaryPath != "/opt/rclone/bin/rclone" {
					t.Errorf("RcloneBinaryPath = %q, want '/opt/rclone/bin/rclone'", c.Settings.RcloneBinaryPath)
				}
			},
		},
		{
			name:      "Set Default Mount Dir",
			configKey: "settings.default_mount_dir",
			value:     "/mnt/remotes",
			checkConfig: func(t *testing.T, c *config.Config) {
				if c.Settings.DefaultMountDir != "/mnt/remotes" {
					t.Errorf("DefaultMountDir = %q, want '/mnt/remotes'", c.Settings.DefaultMountDir)
				}
			},
		},
		{
			name:      "Set Editor",
			configKey: "settings.editor",
			value:     "nano",
			checkConfig: func(t *testing.T, c *config.Config) {
				if c.Settings.Editor != "nano" {
					t.Errorf("Editor = %q, want 'nano'", c.Settings.Editor)
				}
			},
		},
		{
			name:        "Invalid Transfers (non-numeric)",
			configKey:   "defaults.sync.transfers",
			value:       "abc",
			expectError: true,
		},
		{
			name:        "Invalid Checkers (non-numeric)",
			configKey:   "defaults.sync.checkers",
			value:       "xyz",
			expectError: true,
		},
		{
			name:        "Unknown config key",
			configKey:   "unknown.key",
			value:       "test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewSettingsScreen()
			cfg := &config.Config{
				Defaults: config.DefaultConfig{
					Mount: config.MountDefaults{},
					Sync:  config.SyncDefaults{},
				},
				Settings: config.Settings{},
			}
			screen.SetConfig(cfg)

			err := screen.setConfigValue(tt.configKey, tt.value)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.checkConfig != nil {
					tt.checkConfig(t, cfg)
				}
			}
		})
	}
}

func TestSettingsScreen_SetConfigValueNilConfig(t *testing.T) {
	screen := NewSettingsScreen()
	// Don't set config - it should be nil

	err := screen.setConfigValue("defaults.mount.vfs_cache_mode", "test")

	if err == nil {
		t.Error("expected error with nil config, got nil")
	}
}

func TestSettingsScreen_CursorNavigation(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

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
	for i := 0; i < len(screen.settings)-1; i++ {
		screen.Update(tea.KeyMsg{Type: tea.KeyDown})
		expected := i + 1
		if screen.cursor != expected {
			t.Errorf("cursor after down %d times = %d, want %d", i+1, screen.cursor, expected)
		}
	}

	// Try to move down past last item - should stay at last
	lastIndex := len(screen.settings) - 1
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

func TestSettingsScreen_VimNavigation(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

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

func TestSettingsScreen_EscapeKey(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Press escape
	screen.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if !screen.ShouldGoBack() {
		t.Error("ShouldGoBack() = false, want true")
	}
}

func TestSettingsScreen_ResetGoBack(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Trigger go back
	screen.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if !screen.ShouldGoBack() {
		t.Fatal("ShouldGoBack() = false before reset")
	}

	// Reset
	screen.ResetGoBack()

	if screen.ShouldGoBack() {
		t.Error("ShouldGoBack() = true after reset, want false")
	}
}

func TestSettingsScreen_SetSize(t *testing.T) {
	screen := NewSettingsScreen()

	screen.SetSize(100, 30)

	if screen.width != 100 {
		t.Errorf("width = %d, want 100", screen.width)
	}

	if screen.height != 30 {
		t.Errorf("height = %d, want 30", screen.height)
	}
}

func TestSettingsScreen_Init(t *testing.T) {
	screen := NewSettingsScreen()

	cmd := screen.Init()

	if cmd != nil {
		t.Error("Init() should return nil command")
	}
}

func TestSettingsScreen_View(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	view := screen.View()

	// Check title is rendered
	if !strings.Contains(view, "Settings") {
		t.Error("View() should contain 'Settings' title")
	}

	// Check some settings are rendered
	expectedSettings := []string{
		"Default VFS Cache Mode",
		"Default Buffer Size",
		"Rclone Binary Path",
	}

	for _, setting := range expectedSettings {
		if !strings.Contains(view, setting) {
			t.Errorf("View() should contain setting '%s'", setting)
		}
	}

	// Check help text is present
	if !strings.Contains(view, "navigate") {
		t.Error("View() should contain help text for navigation")
	}

	if !strings.Contains(view, "edit") {
		t.Error("View() should contain help text for edit")
	}

	// Ensure selection marker present
	if !strings.Contains(view, "▸") {
		t.Error("View() should contain selection marker '▸'")
	}
}

func TestSettingsScreen_ViewWithConfig(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(160, 24)

	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				VFSCacheMode: "full",
				BufferSize:   "16M",
				LogLevel:     "INFO",
			},
			Sync: config.SyncDefaults{
				LogLevel:  "INFO",
				Transfers: 4,
				Checkers:  8,
			},
		},
		Settings: config.Settings{
			RcloneBinaryPath: "/usr/bin/rclone",
			DefaultMountDir:  "~/mnt",
			Editor:           "vim",
		},
	}
	screen.SetConfig(cfg)

	view := screen.View()

	if !strings.Contains(view, "full") {
		t.Error("View() should contain VFS cache mode value 'full'")
	}

	if !strings.Contains(view, "16M") {
		t.Error("View() should contain buffer size value '16M'")
	}

	if !strings.Contains(view, "Rclone Binary Path") {
		t.Error("View() should contain 'Rclone Binary Path' setting name")
	}

	if !strings.Contains(view, "/usr") {
		t.Error("View() should contain beginning of the rclone binary path")
	}
}

func TestSettingsScreen_UpdateSettingValues(t *testing.T) {
	screen := NewSettingsScreen()

	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				VFSCacheMode: "writes",
				BufferSize:   "32M",
				LogLevel:     "DEBUG",
			},
			Sync: config.SyncDefaults{
				LogLevel:  "ERROR",
				Transfers: 8,
				Checkers:  16,
			},
		},
		Settings: config.Settings{
			RcloneBinaryPath: "/custom/rclone",
			DefaultMountDir:  "/custom/mnt",
			Editor:           "emacs",
		},
	}

	screen.SetConfig(cfg)

	// Verify all settings have been updated with config values
	for _, setting := range screen.settings {
		if setting.Value == "" && setting.configKey != "settings.rclone_binary_path" {
			// rclone_binary_path can be empty by default
			t.Errorf("setting %q has empty value after SetConfig", setting.Name)
		}
	}
}

func TestSettingsScreen_SelectTypeOptions(t *testing.T) {
	screen := NewSettingsScreen()

	// Find VFS Cache Mode setting (it's a select type)
	var vfsSetting *SettingItem
	for i := range screen.settings {
		if screen.settings[i].configKey == "defaults.mount.vfs_cache_mode" {
			vfsSetting = &screen.settings[i]
			break
		}
	}

	if vfsSetting == nil {
		t.Fatal("VFS Cache Mode setting not found")
	}

	// Verify it's a select type
	if vfsSetting.settingType != "select" {
		t.Errorf("VFS Cache Mode setting type = %q, want 'select'", vfsSetting.settingType)
	}

	// Verify select options
	expectedOpts := []string{"off", "writes", "full"}
	for i, opt := range expectedOpts {
		if i >= len(vfsSetting.selectOpts) {
			t.Errorf("missing select option at index %d", i)
			continue
		}
		if vfsSetting.selectOpts[i] != opt {
			t.Errorf("select option %d = %q, want %q", i, vfsSetting.selectOpts[i], opt)
		}
	}
}

func TestSettingsScreen_EnterStartsEditing(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Press enter to start editing
	_, _ = screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should be in editing mode
	if !screen.editing {
		t.Error("editing should be true after pressing Enter")
	}

	if screen.form == nil {
		t.Error("form should be initialized when editing")
	}
}

func TestSettingsScreen_EscapeDuringEditing(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Start editing
	_, _ = screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !screen.editing {
		t.Fatal("editing should be true")
	}

	// Press escape to cancel editing
	_, _ = screen.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Should no longer be editing
	if screen.editing {
		t.Error("editing should be false after pressing Escape")
	}

	if screen.form != nil {
		t.Error("form should be nil after canceling edit")
	}
}

func TestSettingsScreen_EditIndexSet(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Move cursor to index 2
	screen.cursor = 2

	// Press enter to start editing
	_, _ = screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// editIndex should match cursor
	if screen.editIndex != 2 {
		t.Errorf("editIndex = %d, want 2", screen.editIndex)
	}
}

func TestSettingsScreen_SubmitFormSuccess(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				VFSCacheMode: "full",
			},
		},
	}
	screen.SetConfig(cfg)
	screen.cursor = 0 // VFS Cache Mode setting

	// Start editing
	screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify form is created
	if screen.form == nil {
		t.Fatal("form should be created")
	}

	// Manually set the value and submit
	screen.settings[0].Value = "writes"

	// Submit the form
	screen.submitForm()

	// Verify the config was updated
	if cfg.Defaults.Mount.VFSCacheMode != "writes" {
		t.Errorf("VFSCacheMode = %q, want 'writes'", cfg.Defaults.Mount.VFSCacheMode)
	}

	// Verify success message
	if !strings.Contains(screen.message, "updated") {
		t.Errorf("message = %q, should contain 'updated'", screen.message)
	}

	if screen.messageType != "success" {
		t.Errorf("messageType = %q, want 'success'", screen.messageType)
	}
}

func TestSettingsScreen_EditSettingWithIntType(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Sync: config.SyncDefaults{
				Transfers: 4,
			},
		},
	}
	screen.SetConfig(cfg)

	// Find the Transfers setting (index 4)
	screen.cursor = 4

	// Start editing
	screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !screen.editing {
		t.Error("should be in editing mode")
	}

	if screen.form == nil {
		t.Error("form should be created")
	}
}

func TestSettingsScreen_EditSettingWithSelectType(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	cfg := &config.Config{
		Defaults: config.DefaultConfig{
			Mount: config.MountDefaults{
				VFSCacheMode: "full",
			},
		},
	}
	screen.SetConfig(cfg)

	// VFS Cache Mode is a select type (index 0)
	screen.cursor = 0

	// Start editing
	screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !screen.editing {
		t.Error("should be in editing mode")
	}

	if screen.form == nil {
		t.Error("form should be created")
	}
}

func TestSettingsScreen_EditSettingWithStringType(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	cfg := &config.Config{
		Settings: config.Settings{
			Editor: "vim",
		},
	}
	screen.SetConfig(cfg)

	// Editor is a string type (index 8)
	screen.cursor = 8

	// Start editing
	screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !screen.editing {
		t.Error("should be in editing mode")
	}

	if screen.form == nil {
		t.Error("form should be created")
	}
}

func TestSettingsScreen_FormRender(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	cfg := &config.Config{}
	screen.SetConfig(cfg)
	screen.cursor = 0
	screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	view := screen.View()

	// Should show form title
	if !strings.Contains(view, "Edit Setting") {
		t.Error("View() should contain 'Edit Setting' when editing")
	}

	// Should show help
	if !strings.Contains(view, "Enter") {
		t.Error("View() should contain help text for Enter key")
	}
}

func TestSettingsScreen_InvalidCursor(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Set cursor to invalid index
	screen.cursor = 100

	// Try to start editing - should not panic
	_, _ = screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if screen.editing {
		t.Error("should not be editing with invalid cursor")
	}
}

func TestSettingsScreen_MessageRendering(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Set success message
	screen.message = "Setting saved"
	screen.messageType = "success"

	view := screen.View()

	if !strings.Contains(view, "Setting saved") {
		t.Error("View() should contain success message")
	}

	// Set error message
	screen.message = "Error occurred"
	screen.messageType = "error"

	view = screen.View()

	if !strings.Contains(view, "Error occurred") {
		t.Error("View() should contain error message")
	}
}

func TestSettingsScreen_ViewWithMessage(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)
	screen.message = "Test message"
	screen.messageType = "success"

	view := screen.View()

	if !strings.Contains(view, "Test message") {
		t.Error("View() should contain message")
	}
}

func TestSettingsScreen_SaveConfigError(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Create config with invalid path to trigger save error
	cfg := &config.Config{}
	screen.SetConfig(cfg)

	// Set editIndex and settings value
	screen.editIndex = 0
	screen.settings[0].Value = "writes"

	// Manually call submitForm - this should handle save errors gracefully
	screen.submitForm()
	// Just verify it doesn't panic
}

func TestSettingsScreen_StartEditingWithNilConfig(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)
	// Don't set config - it should be nil

	screen.cursor = 0
	_, _ = screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should still be able to start editing
	if !screen.editing {
		t.Error("should be editing even with nil config")
	}
}

func TestSettingsScreen_ValidateIntField(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	cfg := &config.Config{}
	screen.SetConfig(cfg)

	// Edit the Transfers field (int type, index 4)
	screen.cursor = 4
	_, _ = screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// The form should have validation for int type
	if screen.form == nil {
		t.Error("form should be created for int type setting")
	}
}

func TestSettingsScreen_SettingsListTruncation(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Create a setting with a very long name
	screen.settings[0].Name = "This is a very long setting name that should be truncated in the view"

	view := screen.View()

	// View should still render without errors
	if view == "" {
		t.Error("View() should not be empty")
	}
}

// Tests for Import/Export functionality

func TestSettingsScreen_StartExport(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Call startExport
	model, cmd := screen.startExport()

	// Verify state was set correctly
	if !screen.showingFilePicker {
		t.Error("showingFilePicker should be true after startExport")
	}

	if screen.form == nil {
		t.Error("form should be initialized")
	}

	// Verify model returned is the screen
	if model == nil {
		t.Error("model should not be nil")
	}

	// cmd should be the form init command (may not be nil)
	_ = cmd
}

func TestSettingsScreen_StartImport(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Call startImport
	model, cmd := screen.startImport()

	// Verify state was set correctly
	if !screen.showingFilePicker {
		t.Error("showingFilePicker should be true after startImport")
	}

	if screen.form == nil {
		t.Error("form should be initialized")
	}

	// Verify model returned is the screen
	if model == nil {
		t.Error("model should not be nil")
	}

	// cmd should be the form init command
	_ = cmd
}

func TestSettingsScreen_UpdateFilePicker_Escape(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Initialize file picker state
	screen.showingFilePicker = true
	screen.exportPath = "/some/path"
	screen.pendingImportPath = "/some/import/path"

	// Create a dummy form
	screen.startExport()

	// Press escape to cancel
	model, _ := screen.updateFilePicker(tea.KeyMsg{Type: tea.KeyEsc})

	// Verify state was reset
	if screen.showingFilePicker {
		t.Error("showingFilePicker should be false after escape")
	}

	if screen.form != nil {
		t.Error("form should be nil after escape")
	}

	if screen.exportPath != "" {
		t.Error("exportPath should be empty after escape")
	}

	if screen.pendingImportPath != "" {
		t.Error("pendingImportPath should be empty after escape")
	}

	// Verify model returned is the screen
	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_CompleteExport_NilConfig(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)
	// Don't set config - it should be nil

	// Call completeExport
	model, _ := screen.completeExport("/tmp/export.yaml")

	// Verify error message was set
	if screen.message == "" {
		t.Error("message should be set when config is nil")
	}

	if screen.messageType != "error" {
		t.Errorf("messageType = %q, want 'error'", screen.messageType)
	}

	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_CompleteExport_Success(t *testing.T) {
	// Create a temp file for export
	tmpDir := t.TempDir()
	exportPath := tmpDir + "/export.yaml"

	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	cfg := &config.Config{
		Version: "1.0",
		Mounts: []models.MountConfig{
			{Name: "test-mount", Remote: "remote:path", MountPoint: "/mnt/test"},
		},
	}
	screen.SetConfig(cfg)

	// Call completeExport
	model, _ := screen.completeExport(exportPath)

	// Verify success message
	if !strings.Contains(screen.message, "exported") {
		t.Errorf("message = %q, should contain 'exported'", screen.message)
	}

	if screen.messageType != "success" {
		t.Errorf("messageType = %q, want 'success'", screen.messageType)
	}

	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_CompleteImportFileSelection_FileNotExist(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Call with non-existent file
	model, _ := screen.completeImportFileSelection("/nonexistent/path.yaml")

	// Verify error message
	if !strings.Contains(screen.message, "does not exist") {
		t.Errorf("message = %q, should contain 'does not exist'", screen.message)
	}

	if screen.messageType != "error" {
		t.Errorf("messageType = %q, want 'error'", screen.messageType)
	}

	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_CompleteImportFileSelection_Success(t *testing.T) {
	// Create a temp file for import
	tmpFile, err := os.CreateTemp("", "import-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write valid export data
	exportData := `version: "1.0"
mounts: []
sync_jobs: []
exported: "2024-01-01T00:00:00Z"
`
	if _, err := tmpFile.WriteString(exportData); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Call with existing file
	model, _ := screen.completeImportFileSelection(tmpFile.Name())

	// Verify pendingImportPath was set
	if screen.pendingImportPath != tmpFile.Name() {
		t.Errorf("pendingImportPath = %q, want %q", screen.pendingImportPath, tmpFile.Name())
	}

	// Verify showing import mode selection
	if !screen.showingImportMode {
		t.Error("showingImportMode should be true")
	}

	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_ShowImportModeSelection(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Call showImportModeSelection
	model, cmd := screen.showImportModeSelection()

	// Verify state
	if !screen.showingImportMode {
		t.Error("showingImportMode should be true")
	}

	if screen.form == nil {
		t.Error("form should be initialized")
	}

	if screen.importMode != "merge" {
		t.Errorf("importMode = %q, want 'merge'", screen.importMode)
	}

	if model == nil {
		t.Error("model should not be nil")
	}

	_ = cmd
}

func TestSettingsScreen_UpdateImportModeForm_Escape(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Initialize import mode form
	screen.showImportModeSelection()
	screen.pendingImportPath = "/some/path.yaml"

	// Press escape to cancel
	model, _ := screen.updateImportModeForm(tea.KeyMsg{Type: tea.KeyEsc})

	// Verify state was reset
	if screen.showingImportMode {
		t.Error("showingImportMode should be false after escape")
	}

	if screen.form != nil {
		t.Error("form should be nil after escape")
	}

	if screen.pendingImportPath != "" {
		t.Error("pendingImportPath should be empty after escape")
	}

	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_ShowReplaceConfirm(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Call showReplaceConfirm
	model, cmd := screen.showReplaceConfirm()

	// Verify state
	if !screen.showingConfirm {
		t.Error("showingConfirm should be true")
	}

	if screen.confirmDialog == nil {
		t.Error("confirmDialog should be initialized")
	}

	if model == nil {
		t.Error("model should not be nil")
	}

	_ = cmd
}

func TestSettingsScreen_UpdateConfirmDialog_Escape(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Initialize confirm dialog
	screen.showReplaceConfirm()
	screen.pendingImportPath = "/some/path.yaml"

	// Press escape to cancel
	model, _ := screen.updateConfirmDialog(tea.KeyMsg{Type: tea.KeyEsc})

	// Verify state was reset
	if screen.showingConfirm {
		t.Error("showingConfirm should be false after escape")
	}

	if screen.confirmDialog != nil {
		t.Error("confirmDialog should be nil after escape")
	}

	if screen.pendingImportPath != "" {
		t.Error("pendingImportPath should be empty after escape")
	}

	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_ExecuteImport_NilConfig(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)
	// Don't set config - it should be nil

	screen.pendingImportPath = "/some/path.yaml"
	screen.importMode = "merge"

	// Call executeImport
	model, _ := screen.executeImport()

	// Verify error message
	if !strings.Contains(screen.message, "No configuration") {
		t.Errorf("message = %q, should contain 'No configuration'", screen.message)
	}

	if screen.messageType != "error" {
		t.Errorf("messageType = %q, want 'error'", screen.messageType)
	}

	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_ExecuteImport_Success(t *testing.T) {
	// Create a temp file for import
	tmpFile, err := os.CreateTemp("", "import-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write valid export data
	exportData := `version: "1.0"
mounts:
  - name: imported-mount
    remote: remote:path
    mount_point: /mnt/imported
sync_jobs:
  - name: imported-sync
    source: remote:src
    destination: /dest
exported: "2024-01-01T00:00:00Z"
`
	if _, err := tmpFile.WriteString(exportData); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	cfg := &config.Config{
		Version: "1.0",
	}
	screen.SetConfig(cfg)
	screen.pendingImportPath = tmpFile.Name()
	screen.importMode = "merge"

	// Call executeImport
	model, _ := screen.executeImport()

	// Verify success message
	if !strings.Contains(screen.message, "imported") {
		t.Errorf("message = %q, should contain 'imported'", screen.message)
	}

	if screen.messageType != "success" {
		t.Errorf("messageType = %q, want 'success'", screen.messageType)
	}

	// Verify pendingImportPath was cleared
	if screen.pendingImportPath != "" {
		t.Error("pendingImportPath should be empty after import")
	}

	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_ExecuteImport_ReplaceMode(t *testing.T) {
	// Create a temp file for import
	tmpFile, err := os.CreateTemp("", "import-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write valid export data
	exportData := `version: "1.0"
mounts:
  - name: replaced-mount
    remote: remote:path
    mount_point: /mnt/replaced
sync_jobs: []
exported: "2024-01-01T00:00:00Z"
`
	if _, err := tmpFile.WriteString(exportData); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	cfg := &config.Config{
		Version: "1.0",
		Mounts: []models.MountConfig{
			{Name: "old-mount", Remote: "old:path", MountPoint: "/mnt/old"},
		},
	}
	screen.SetConfig(cfg)
	screen.pendingImportPath = tmpFile.Name()
	screen.importMode = "replace"

	// Call executeImport
	model, _ := screen.executeImport()

	// Verify success
	if screen.messageType != "success" {
		t.Errorf("messageType = %q, want 'success'", screen.messageType)
	}

	// Verify mounts were replaced
	if len(cfg.Mounts) != 1 || cfg.Mounts[0].Name != "replaced-mount" {
		t.Errorf("mounts should be replaced, got: %v", cfg.Mounts)
	}

	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_ExecuteAction_Export(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Set action cursor to export action (index 0)
	screen.actionCursor = 0
	screen.showingActions = true

	// Call executeAction
	model, _ := screen.executeAction()

	// Verify showingActions was reset
	if screen.showingActions {
		t.Error("showingActions should be false after executeAction")
	}

	// Verify file picker is shown
	if !screen.showingFilePicker {
		t.Error("showingFilePicker should be true for export action")
	}

	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_ExecuteAction_Import(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Set action cursor to import action (index 1)
	screen.actionCursor = 1
	screen.showingActions = true

	// Call executeAction
	model, _ := screen.executeAction()

	// Verify showingActions was reset
	if screen.showingActions {
		t.Error("showingActions should be false after executeAction")
	}

	// Verify file picker is shown
	if !screen.showingFilePicker {
		t.Error("showingFilePicker should be true for import action")
	}

	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_ExecuteAction_InvalidCursor(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Set invalid cursor
	screen.actionCursor = 100
	screen.showingActions = true

	// Call executeAction - should not panic
	model, _ := screen.executeAction()

	// Verify model is returned
	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_RenderFilePicker(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Initialize file picker
	screen.startExport()

	// Render file picker
	view := screen.renderFilePicker()

	// Verify view contains expected elements
	if !strings.Contains(view, "File Selection") {
		t.Error("renderFilePicker should contain 'File Selection' title")
	}

	if !strings.Contains(view, "Enter") {
		t.Error("renderFilePicker should contain help text for Enter key")
	}

	if !strings.Contains(view, "Esc") {
		t.Error("renderFilePicker should contain help text for Esc key")
	}
}

func TestSettingsScreen_RenderImportModeForm(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Initialize import mode form
	screen.showImportModeSelection()

	// Render import mode form
	view := screen.renderImportModeForm()

	// Verify view contains expected elements
	if !strings.Contains(view, "Import Mode") {
		t.Error("renderImportModeForm should contain 'Import Mode' title")
	}

	if !strings.Contains(view, "Enter") {
		t.Error("renderImportModeForm should contain help text for Enter key")
	}

	if !strings.Contains(view, "Esc") {
		t.Error("renderImportModeForm should contain help text for Esc key")
	}
}

func TestSettingsScreen_RenderConfirmDialog(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Initialize confirm dialog
	screen.showReplaceConfirm()

	// Render confirm dialog
	view := screen.renderConfirmDialog()

	// Verify view contains expected elements
	if !strings.Contains(view, "Confirm Import") {
		t.Error("renderConfirmDialog should contain 'Confirm Import' title")
	}

	if !strings.Contains(view, "Enter") {
		t.Error("renderConfirmDialog should contain help text for Enter key")
	}

	if !strings.Contains(view, "Esc") {
		t.Error("renderConfirmDialog should contain help text for Esc key")
	}
}

func TestSettingsScreen_RenderSettingsList(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Render settings list
	view := screen.renderSettingsList()

	// Verify view contains expected elements
	if !strings.Contains(view, "Setting") {
		t.Error("renderSettingsList should contain 'Setting' header")
	}

	// Verify settings are rendered
	for _, setting := range screen.settings {
		if !strings.Contains(view, setting.Name) {
			t.Errorf("renderSettingsList should contain setting name '%s'", setting.Name)
		}
	}
}

func TestSettingsScreen_View_WithFilePicker(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Start file picker
	screen.startExport()

	view := screen.View()

	// Should show file picker, not normal view
	if !strings.Contains(view, "File Selection") {
		t.Error("View() should show file picker when showingFilePicker is true")
	}
}

func TestSettingsScreen_View_WithImportModeForm(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Start import mode form
	screen.showImportModeSelection()

	view := screen.View()

	// Should show import mode form
	if !strings.Contains(view, "Import Mode") {
		t.Error("View() should show import mode form when showingImportMode is true")
	}
}

func TestSettingsScreen_View_WithConfirmDialog(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Start confirm dialog
	screen.showReplaceConfirm()

	view := screen.View()

	// Should show confirm dialog
	if !strings.Contains(view, "Confirm Import") {
		t.Error("View() should show confirm dialog when showingConfirm is true")
	}
}

func TestSettingsScreen_KeyboardShortcuts_Export(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Press 'x' to start export
	model, _ := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	// Verify export started
	if !screen.showingFilePicker {
		t.Error("pressing 'x' should start export (showingFilePicker should be true)")
	}

	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_KeyboardShortcuts_Import(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Press 'i' to start import
	model, _ := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})

	// Verify import started
	if !screen.showingFilePicker {
		t.Error("pressing 'i' should start import (showingFilePicker should be true)")
	}

	if model == nil {
		t.Error("model should not be nil")
	}
}

func TestSettingsScreen_ActionsNavigation(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Switch to actions panel with right arrow
	screen.Update(tea.KeyMsg{Type: tea.KeyRight})

	if !screen.showingActions {
		t.Error("showingActions should be true after pressing right")
	}

	if screen.actionCursor != 0 {
		t.Errorf("actionCursor = %d, want 0", screen.actionCursor)
	}

	// Move down in actions
	screen.Update(tea.KeyMsg{Type: tea.KeyDown})

	if screen.actionCursor != 1 {
		t.Errorf("actionCursor = %d, want 1", screen.actionCursor)
	}

	// Move up in actions
	screen.Update(tea.KeyMsg{Type: tea.KeyUp})

	if screen.actionCursor != 0 {
		t.Errorf("actionCursor = %d, want 0", screen.actionCursor)
	}

	// Switch back to settings with left arrow
	screen.Update(tea.KeyMsg{Type: tea.KeyLeft})

	if screen.showingActions {
		t.Error("showingActions should be false after pressing left")
	}
}

func TestSettingsScreen_ActionsEnterKey(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Switch to actions panel
	screen.showingActions = true
	screen.actionCursor = 0 // Export action

	// Press enter to execute action
	screen.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify export started
	if !screen.showingFilePicker {
		t.Error("pressing Enter on export action should start export")
	}
}

func TestSettingsScreen_EscapeFromActions(t *testing.T) {
	screen := NewSettingsScreen()
	screen.SetSize(80, 24)

	// Switch to actions panel
	screen.showingActions = true

	// Press escape
	screen.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Should go back from actions, not main menu
	if screen.showingActions {
		t.Error("showingActions should be false after escape")
	}

	if screen.ShouldGoBack() {
		t.Error("ShouldGoBack should be false when escaping from actions")
	}
}
