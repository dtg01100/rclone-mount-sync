// Package screens provides individual TUI screens for the application.
package screens

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dlafreniere/rclone-mount-sync/internal/tui/components"
)

// SettingsScreen handles application settings.
type SettingsScreen struct {
	settings []SettingItem
	cursor   int
	width    int
	height   int
	goBack   bool
}

// SettingItem represents a setting item.
type SettingItem struct {
	Name        string
	Description string
	Value       string
	Key         string
}

// NewSettingsScreen creates a new settings screen.
func NewSettingsScreen() *SettingsScreen {
	return &SettingsScreen{
		settings: []SettingItem{
			{Name: "Default VFS Cache Mode", Description: "VFS cache mode for new mounts", Value: "full", Key: "v"},
			{Name: "Default Buffer Size", Description: "Buffer size for rclone operations", Value: "16M", Key: "b"},
			{Name: "Default Log Level", Description: "Logging verbosity", Value: "INFO", Key: "l"},
			{Name: "Rclone Config Path", Description: "Path to rclone configuration file", Value: "~/.config/rclone/rclone.conf", Key: "r"},
			{Name: "Systemd Unit Path", Description: "Path for generated systemd units", Value: "~/.config/systemd/user", Key: "s"},
		},
	}
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(s.settings)-1 {
				s.cursor++
			}
		case "enter":
			// TODO: Edit selected setting
		case "esc":
			s.goBack = true
		}
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
	var b strings.Builder

	// Title
	title := components.Styles.Title.Render("Settings")
	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	// Settings list
	b.WriteString(s.renderSettingsList())

	// Help bar
	b.WriteString("\n\n")
	helpText := components.HelpBar(s.width, []components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "Enter", Desc: "edit"},
		{Key: "Esc", Desc: "back"},
	})
	b.WriteString(helpText)

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
