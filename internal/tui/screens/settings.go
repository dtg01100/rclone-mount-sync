// Package screens provides individual TUI screens for the application.
package screens

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/tui/components"
)

// SettingsScreen handles application settings.
type SettingsScreen struct {
	settings []SettingItem
	actions  []ActionItem
	cursor   int
	width    int
	height   int
	goBack   bool
	config   *config.Config

	// Form state
	form        *huh.Form
	editing     bool
	editIndex   int
	message     string
	messageType string // "success" or "error"

	// Action handling state
	showingActions    bool
	actionCursor      int
	importMode        string
	confirmDialog     *huh.Form
	showingImportMode bool
	showingConfirm    bool
	showingFilePicker bool
	pendingImportPath string
	exportPath        string
}

// ActionItem represents an action item in settings.
type ActionItem struct {
	Name        string
	Description string
	Key         string
	actionType  string
}

// SettingItem represents a setting item.
type SettingItem struct {
	Name        string
	Description string
	Value       string
	Key         string
	settingType string // "string", "int", "select"
	selectOpts  []string
	configKey   string // Key path in config (e.g., "defaults.mount.vfs_cache_mode")
}

// NewSettingsScreen creates a new settings screen.
func NewSettingsScreen() *SettingsScreen {
	return &SettingsScreen{
		settings: []SettingItem{
			{
				Name:        "Default VFS Cache Mode",
				Description: "VFS cache mode for new mounts",
				Key:         "v",
				settingType: "select",
				selectOpts:  []string{"off", "writes", "full"},
				configKey:   "defaults.mount.vfs_cache_mode",
			},
			{
				Name:        "Default Buffer Size",
				Description: "Buffer size for rclone operations (e.g., 16M)",
				Key:         "b",
				settingType: "string",
				configKey:   "defaults.mount.buffer_size",
			},
			{
				Name:        "Default Mount Log Level",
				Description: "Logging verbosity for mounts",
				Key:         "l",
				settingType: "select",
				selectOpts:  []string{"ERROR", "NOTICE", "INFO", "DEBUG"},
				configKey:   "defaults.mount.log_level",
			},
			{
				Name:        "Default Sync Log Level",
				Description: "Logging verbosity for sync jobs",
				Key:         "sl",
				settingType: "select",
				selectOpts:  []string{"ERROR", "NOTICE", "INFO", "DEBUG"},
				configKey:   "defaults.sync.log_level",
			},
			{
				Name:        "Default Transfers",
				Description: "Number of parallel transfers for sync jobs",
				Key:         "t",
				settingType: "int",
				configKey:   "defaults.sync.transfers",
			},
			{
				Name:        "Default Checkers",
				Description: "Number of checkers for sync jobs",
				Key:         "c",
				settingType: "int",
				configKey:   "defaults.sync.checkers",
			},
			{
				Name:        "Rclone Binary Path",
				Description: "Path to rclone binary (empty for system default)",
				Key:         "r",
				settingType: "string",
				configKey:   "settings.rclone_binary_path",
			},
			{
				Name:        "Default Mount Directory",
				Description: "Default directory for mount points",
				Key:         "m",
				settingType: "string",
				configKey:   "settings.default_mount_dir",
			},
			{
				Name:        "Editor",
				Description: "Text editor for editing config files",
				Key:         "e",
				settingType: "string",
				configKey:   "settings.editor",
			},
		},
		actions: []ActionItem{
			{
				Name:        "Export Configuration",
				Description: "Save mounts and sync jobs to a file",
				Key:         "x",
				actionType:  "export",
			},
			{
				Name:        "Import Configuration",
				Description: "Load mounts and sync jobs from a file",
				Key:         "i",
				actionType:  "import",
			},
		},
	}
}

// SetConfig sets the configuration for the settings screen.
func (s *SettingsScreen) SetConfig(cfg *config.Config) {
	s.config = cfg
	s.updateSettingValues()
}

// updateSettingValues updates the setting values from the config.
func (s *SettingsScreen) updateSettingValues() {
	if s.config == nil {
		return
	}

	for i := range s.settings {
		s.settings[i].Value = s.getConfigValue(s.settings[i].configKey)
	}
}

// getConfigValue retrieves a config value by its key path.
func (s *SettingsScreen) getConfigValue(key string) string {
	if s.config == nil {
		return ""
	}

	switch key {
	case "defaults.mount.vfs_cache_mode":
		return s.config.Defaults.Mount.VFSCacheMode
	case "defaults.mount.buffer_size":
		return s.config.Defaults.Mount.BufferSize
	case "defaults.mount.log_level":
		return s.config.Defaults.Mount.LogLevel
	case "defaults.sync.log_level":
		return s.config.Defaults.Sync.LogLevel
	case "defaults.sync.transfers":
		return fmt.Sprintf("%d", s.config.Defaults.Sync.Transfers)
	case "defaults.sync.checkers":
		return fmt.Sprintf("%d", s.config.Defaults.Sync.Checkers)
	case "settings.rclone_binary_path":
		return s.config.Settings.RcloneBinaryPath
	case "settings.default_mount_dir":
		return s.config.Settings.DefaultMountDir
	case "settings.editor":
		return s.config.Settings.Editor
	default:
		return ""
	}
}

// setConfigValue sets a config value by its key path.
func (s *SettingsScreen) setConfigValue(key, value string) error {
	if s.config == nil {
		return fmt.Errorf("config not initialized")
	}

	switch key {
	case "defaults.mount.vfs_cache_mode":
		s.config.Defaults.Mount.VFSCacheMode = value
	case "defaults.mount.buffer_size":
		s.config.Defaults.Mount.BufferSize = value
	case "defaults.mount.log_level":
		s.config.Defaults.Mount.LogLevel = value
	case "defaults.sync.log_level":
		s.config.Defaults.Sync.LogLevel = value
	case "defaults.sync.transfers":
		var transfers int
		if _, err := fmt.Sscanf(value, "%d", &transfers); err != nil {
			return fmt.Errorf("invalid number: %w", err)
		}
		s.config.Defaults.Sync.Transfers = transfers
	case "defaults.sync.checkers":
		var checkers int
		if _, err := fmt.Sscanf(value, "%d", &checkers); err != nil {
			return fmt.Errorf("invalid number: %w", err)
		}
		s.config.Defaults.Sync.Checkers = checkers
	case "settings.rclone_binary_path":
		s.config.Settings.RcloneBinaryPath = value
	case "settings.default_mount_dir":
		s.config.Settings.DefaultMountDir = value
	case "settings.editor":
		s.config.Settings.Editor = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return nil
}

// SetSize sets the screen dimensions.
func (s *SettingsScreen) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// Init initializes the screen.
func (s *SettingsScreen) Init() tea.Cmd {
	return nil
}

// Update handles screen updates.
func (s *SettingsScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.showingConfirm && s.confirmDialog != nil {
		return s.updateConfirmDialog(msg)
	}

	if s.showingImportMode && s.form != nil {
		return s.updateImportModeForm(msg)
	}

	if s.showingFilePicker && s.form != nil {
		return s.updateFilePicker(msg)
	}

	if s.editing && s.form != nil {
		return s.updateForm(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.showingActions {
				if s.actionCursor > 0 {
					s.actionCursor--
				}
			} else {
				if s.cursor > 0 {
					s.cursor--
				}
			}
		case "down", "j":
			if s.showingActions {
				if s.actionCursor < len(s.actions)-1 {
					s.actionCursor++
				}
			} else {
				if s.cursor < len(s.settings)-1 {
					s.cursor++
				}
			}
		case "right", "l":
			if !s.showingActions {
				s.showingActions = true
				s.actionCursor = 0
			}
		case "left", "h":
			if s.showingActions {
				s.showingActions = false
			}
		case "enter":
			if s.showingActions {
				return s.executeAction()
			}
			return s.startEditing()
		case "x":
			return s.startExport()
		case "i":
			return s.startImport()
		case "esc":
			if s.showingActions {
				s.showingActions = false
			} else {
				s.goBack = true
			}
		}
	}

	return s, nil
}

// updateForm handles form updates when editing a setting.
func (s *SettingsScreen) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel editing
			s.editing = false
			s.form = nil
			return s, nil
		}
	}

	// Update the form
	form, cmd := s.form.Update(msg)
	s.form = form.(*huh.Form)

	// Check if form is complete
	if s.form.State == huh.StateCompleted {
		return s.submitForm()
	}

	return s, cmd
}

// startEditing starts editing the selected setting.
func (s *SettingsScreen) startEditing() (tea.Model, tea.Cmd) {
	if s.cursor < 0 || s.cursor >= len(s.settings) {
		return s, nil
	}

	// Use a pointer to the slice element, not a copy, so form edits
	// are applied to the original struct.
	setting := &s.settings[s.cursor]
	s.editIndex = s.cursor

	// Build the form based on setting type
	var formField huh.Field

	switch setting.settingType {
	case "select":
		options := make([]huh.Option[string], len(setting.selectOpts))
		for i, opt := range setting.selectOpts {
			options[i] = huh.NewOption(opt, opt)
		}
		selectField := huh.NewSelect[string]().
			Title(setting.Name).
			Description(setting.Description).
			Options(options...).
			Value(&setting.Value)
		formField = selectField

	case "int":
		inputField := huh.NewInput().
			Title(setting.Name).
			Description(setting.Description).
			Placeholder("Enter value").
			Value(&setting.Value).
			Validate(func(v string) error {
				var num int
				if _, err := fmt.Sscanf(v, "%d", &num); err != nil {
					return fmt.Errorf("please enter a valid number")
				}
				if num < 0 {
					return fmt.Errorf("number must be positive")
				}
				return nil
			})
		formField = inputField

	default: // "string"
		inputField := huh.NewInput().
			Title(setting.Name).
			Description(setting.Description).
			Placeholder("Enter value").
			Value(&setting.Value)
		formField = inputField
	}

	// Create the form
	s.form = huh.NewForm(
		huh.NewGroup(formField),
	)
	s.form.WithTheme(huh.ThemeBase16())

	s.editing = true
	return s, s.form.Init()
}

// submitForm submits the form and saves the setting.
func (s *SettingsScreen) submitForm() (tea.Model, tea.Cmd) {
	setting := s.settings[s.editIndex]

	// Update the config
	if err := s.setConfigValue(setting.configKey, setting.Value); err != nil {
		s.message = fmt.Sprintf("Error: %v", err)
		s.messageType = "error"
	} else {
		// Save the config
		if s.config != nil {
			if err := s.config.Save(); err != nil {
				s.message = fmt.Sprintf("Failed to save config: %v", err)
				s.messageType = "error"
			} else {
				s.message = fmt.Sprintf("Setting '%s' updated to '%s'", setting.Name, setting.Value)
				s.messageType = "success"
			}
		}
	}

	s.editing = false
	s.form = nil
	return s, nil
}

// startExport initiates the export configuration flow.
func (s *SettingsScreen) startExport() (tea.Model, tea.Cmd) {
	s.exportPath = ""
	s.form = huh.NewForm(
		huh.NewGroup(
			huh.NewFilePicker().
				Title("Export Configuration").
				Description("Select a directory and enter filename with .yaml or .json extension.").
				DirAllowed(true).
				FileAllowed(true).
				CurrentDirectory(components.ExpandHome("~")).
				Value(&s.exportPath),
		),
	)
	s.form.WithTheme(huh.ThemeBase16())
	s.showingFilePicker = true
	return s, s.form.Init()
}

// startImport initiates the import configuration flow.
func (s *SettingsScreen) startImport() (tea.Model, tea.Cmd) {
	s.pendingImportPath = ""
	s.form = huh.NewForm(
		huh.NewGroup(
			huh.NewFilePicker().
				Title("Import Configuration").
				Description("Select configuration file to import (.yaml or .json)").
				DirAllowed(false).
				FileAllowed(true).
				CurrentDirectory(components.ExpandHome("~")).
				Value(&s.pendingImportPath),
		),
	)
	s.form.WithTheme(huh.ThemeBase16())
	s.showingFilePicker = true
	return s, s.form.Init()
}

// updateFilePicker handles file picker updates.
func (s *SettingsScreen) updateFilePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			s.form = nil
			s.showingFilePicker = false
			s.exportPath = ""
			s.pendingImportPath = ""
			return s, nil
		}
	}

	form, cmd := s.form.Update(msg)
	s.form = form.(*huh.Form)

	if s.form.State == huh.StateCompleted {
		s.showingFilePicker = false
		if s.exportPath != "" {
			exportPath := s.exportPath
			s.exportPath = ""
			s.form = nil
			return s.completeExport(exportPath)
		}
		importPath := s.pendingImportPath
		s.pendingImportPath = ""
		s.form = nil
		return s.completeImportFileSelection(importPath)
	}

	return s, cmd
}

// completeExport completes the export operation.
func (s *SettingsScreen) completeExport(filePath string) (tea.Model, tea.Cmd) {
	if s.config == nil {
		s.message = "No configuration to export"
		s.messageType = "error"
		return s, nil
	}

	if err := s.config.ExportConfig(filePath); err != nil {
		s.message = fmt.Sprintf("Export failed: %v", err)
		s.messageType = "error"
	} else {
		s.message = fmt.Sprintf("Configuration exported to %s", filePath)
		s.messageType = "success"
	}

	s.exportPath = ""
	return s, nil
}

// completeImportFileSelection handles the file selection for import.
func (s *SettingsScreen) completeImportFileSelection(filePath string) (tea.Model, tea.Cmd) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		s.message = fmt.Sprintf("File does not exist: %s", filePath)
		s.messageType = "error"
		return s, nil
	}

	s.pendingImportPath = filePath
	return s.showImportModeSelection()
}

// showImportModeSelection shows the import mode selection form.
func (s *SettingsScreen) showImportModeSelection() (tea.Model, tea.Cmd) {
	s.importMode = "merge"
	s.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Import Mode").
				Description("How should the imported configuration be merged?").
				Options(
					huh.NewOption("Merge - Add new items, keep existing", "merge"),
					huh.NewOption("Replace - Replace all items with imported", "replace"),
				).
				Value(&s.importMode),
		),
	)
	s.form.WithTheme(huh.ThemeBase16())
	s.showingImportMode = true
	return s, s.form.Init()
}

// updateImportModeForm handles the import mode selection form.
func (s *SettingsScreen) updateImportModeForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			s.showingImportMode = false
			s.form = nil
			s.pendingImportPath = ""
			return s, nil
		}
	}

	form, cmd := s.form.Update(msg)
	s.form = form.(*huh.Form)

	if s.form.State == huh.StateCompleted {
		s.showingImportMode = false
		if s.importMode == "replace" {
			return s.showReplaceConfirm()
		}
		return s.executeImport()
	}

	return s, cmd
}

// showReplaceConfirm shows a confirmation dialog for replace mode.
func (s *SettingsScreen) showReplaceConfirm() (tea.Model, tea.Cmd) {
	confirm := false
	s.confirmDialog = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Replace Configuration?").
				Description("This will replace ALL existing mounts and sync jobs. This action cannot be undone.").
				Value(&confirm),
		),
	)
	s.confirmDialog.WithTheme(huh.ThemeBase16())
	s.showingConfirm = true
	return s, s.confirmDialog.Init()
}

// updateConfirmDialog handles the confirmation dialog.
func (s *SettingsScreen) updateConfirmDialog(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			s.showingConfirm = false
			s.confirmDialog = nil
			s.pendingImportPath = ""
			return s, nil
		}
	}

	form, cmd := s.confirmDialog.Update(msg)
	s.confirmDialog = form.(*huh.Form)

	if s.confirmDialog.State == huh.StateCompleted {
		s.showingConfirm = false
		confirm := s.confirmDialog.GetBool("confirm")
		s.confirmDialog = nil
		if confirm {
			return s.executeImport()
		}
		s.pendingImportPath = ""
		s.message = "Import cancelled"
		s.messageType = "info"
		return s, nil
	}

	return s, cmd
}

// executeImport executes the import operation.
func (s *SettingsScreen) executeImport() (tea.Model, tea.Cmd) {
	if s.config == nil {
		s.message = "No configuration to import into"
		s.messageType = "error"
		s.pendingImportPath = ""
		return s, nil
	}

	var mode config.ImportMode
	if s.importMode == "replace" {
		mode = config.ImportModeReplace
	} else {
		mode = config.ImportModeMerge
	}

	if err := s.config.ImportConfig(s.pendingImportPath, mode); err != nil {
		s.message = fmt.Sprintf("Import failed: %v", err)
		s.messageType = "error"
	} else {
		if err := s.config.Save(); err != nil {
			s.message = fmt.Sprintf("Imported but failed to save: %v", err)
			s.messageType = "error"
		} else {
			s.message = fmt.Sprintf("Configuration imported from %s (%s mode)", s.pendingImportPath, s.importMode)
			s.messageType = "success"
		}
	}

	s.pendingImportPath = ""
	s.importMode = ""
	return s, nil
}

// executeAction executes the selected action.
func (s *SettingsScreen) executeAction() (tea.Model, tea.Cmd) {
	if s.actionCursor < 0 || s.actionCursor >= len(s.actions) {
		return s, nil
	}

	action := s.actions[s.actionCursor]
	s.showingActions = false

	switch action.actionType {
	case "export":
		return s.startExport()
	case "import":
		return s.startImport()
	}

	return s, nil
}

// ShouldGoBack returns true if the screen should go back to the main menu.
func (s *SettingsScreen) ShouldGoBack() bool {
	return s.goBack
}

// ResetGoBack resets the go back state.
func (s *SettingsScreen) ResetGoBack() {
	s.goBack = false
}

// View renders the screen.
func (s *SettingsScreen) View() string {
	if s.showingConfirm && s.confirmDialog != nil {
		return s.renderConfirmDialog()
	}

	if s.showingImportMode && s.form != nil {
		return s.renderImportModeForm()
	}

	if s.showingFilePicker && s.form != nil {
		return s.renderFilePicker()
	}

	if s.editing && s.form != nil {
		return s.renderForm()
	}

	var b strings.Builder

	title := components.Styles.Title.Render("Settings")
	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	if s.message != "" {
		var msgStyle lipgloss.Style
		if s.messageType == "success" {
			msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
		} else {
			msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		}
		msg := msgStyle.Render(s.message)
		b.WriteString(lipgloss.NewStyle().
			Width(s.width).
			Align(lipgloss.Center).
			Render(msg))
		b.WriteString("\n\n")
	}

	leftWidth := s.width/2 - 2
	rightWidth := s.width/2 - 2
	if leftWidth < 30 {
		leftWidth = s.width - 4
		rightWidth = 0
	}

	leftPanel := s.renderSettingsListCompact(leftWidth)

	if rightWidth > 0 {
		rightPanel := s.renderActionsListCompact(rightWidth)
		row := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
		b.WriteString(row)
	} else {
		b.WriteString(leftPanel)
		b.WriteString("\n")
		b.WriteString(s.renderActionsListCompact(leftWidth))
	}

	b.WriteString("\n\n")
	helpItems := []components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "Enter", Desc: "edit/action"},
	}
	if rightWidth > 0 {
		helpItems = append(helpItems, components.HelpItem{Key: "←/→", Desc: "switch panel"})
	}
	helpItems = append(helpItems, components.HelpItem{Key: "x", Desc: "export"})
	helpItems = append(helpItems, components.HelpItem{Key: "i", Desc: "import"})
	helpItems = append(helpItems, components.HelpItem{Key: "Esc", Desc: "back"})
	helpText := components.HelpBar(s.width, helpItems)
	b.WriteString(helpText)

	return b.String()
}

// renderFilePicker renders the file picker form.
func (s *SettingsScreen) renderFilePicker() string {
	var b strings.Builder

	title := components.Styles.Title.Render("File Selection")
	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	b.WriteString(s.form.View())

	b.WriteString("\n\n")
	help := components.Styles.HelpText.Render("Enter: select  Esc: cancel")
	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(help))

	return b.String()
}

// renderImportModeForm renders the import mode selection form.
func (s *SettingsScreen) renderImportModeForm() string {
	var b strings.Builder

	title := components.Styles.Title.Render("Import Mode")
	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	b.WriteString(s.form.View())

	b.WriteString("\n\n")
	help := components.Styles.HelpText.Render("Enter: confirm  Esc: cancel")
	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(help))

	return b.String()
}

// renderConfirmDialog renders the confirmation dialog.
func (s *SettingsScreen) renderConfirmDialog() string {
	var b strings.Builder

	title := components.Styles.Title.Render("Confirm Import")
	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	b.WriteString(s.confirmDialog.View())

	b.WriteString("\n\n")
	help := components.Styles.HelpText.Render("Enter: confirm  Esc: cancel")
	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(help))

	return b.String()
}

// renderForm renders the editing form.
func (s *SettingsScreen) renderForm() string {
	var b strings.Builder

	// Title
	title := components.Styles.Title.Render("Edit Setting")
	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	// Render the form
	b.WriteString(s.form.View())

	// Help text
	b.WriteString("\n\n")
	help := components.Styles.HelpText.Render("Enter: confirm  Esc: cancel")
	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(help))

	return b.String()
}

// renderSettingsList renders the list of settings.
func (s *SettingsScreen) renderSettingsList() string {
	var b strings.Builder

	// Header
	header := "  Setting                                          Value"
	b.WriteString(components.Styles.Subtitle.Render(header) + "\n")
	b.WriteString(components.Styles.Subtitle.Render(strings.Repeat("─", s.width-4)) + "\n")

	// Settings
	for i, setting := range s.settings {
		var line string

		// Format: Name (description): Value
		name := setting.Name
		if setting.Description != "" {
			name = fmt.Sprintf("%s (%s)", setting.Name, setting.Description)
		}

		// Truncate name if too long
		maxNameLen := 45
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		if i == s.cursor {
			line = fmt.Sprintf("▸ %-48s %s",
				components.Styles.Selected.Render(name),
				components.Styles.Normal.Render(setting.Value))
		} else {
			line = fmt.Sprintf("  %-48s %s",
				components.Styles.Normal.Render(name),
				components.Styles.Normal.Render(setting.Value))
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

// renderSettingsListCompact renders the list of settings in a compact format.
func (s *SettingsScreen) renderSettingsListCompact(width int) string {
	var b strings.Builder

	header := components.Styles.Subtitle.Render("Settings")
	b.WriteString(header + "\n")
	b.WriteString(components.Styles.Subtitle.Render(strings.Repeat("─", width-2)) + "\n")

	for i, setting := range s.settings {
		name := setting.Name
		maxNameLen := width - 15
		if maxNameLen < 10 {
			maxNameLen = 10
		}
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		value := setting.Value
		maxValueLen := width - maxNameLen - 5
		if maxValueLen < 10 {
			maxValueLen = 10
		}
		if len(value) > maxValueLen {
			value = value[:maxValueLen-3] + "..."
		}

		if !s.showingActions && i == s.cursor {
			line := fmt.Sprintf("▸ %-*s %s", maxNameLen, components.Styles.Selected.Render(name), value)
			b.WriteString(line + "\n")
		} else {
			line := fmt.Sprintf("  %-*s %s", maxNameLen, name, value)
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}

// renderActionsListCompact renders the list of actions in a compact format.
func (s *SettingsScreen) renderActionsListCompact(width int) string {
	var b strings.Builder

	header := components.Styles.Subtitle.Render("Actions")
	b.WriteString(header + "\n")
	b.WriteString(components.Styles.Subtitle.Render(strings.Repeat("─", width-2)) + "\n")

	for i, action := range s.actions {
		name := action.Name
		maxNameLen := width - 6
		if maxNameLen < 10 {
			maxNameLen = 10
		}
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		if s.showingActions && i == s.actionCursor {
			line := fmt.Sprintf("▸ %s", components.Styles.Selected.Render(name))
			b.WriteString(line + "\n")
			if len(action.Description) <= maxNameLen {
				b.WriteString(fmt.Sprintf("  %s\n", components.Styles.Subtitle.Render(action.Description)))
			}
		} else {
			line := fmt.Sprintf("  %s", name)
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}
