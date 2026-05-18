// Package screens provides individual TUI screens for the application.
package screens

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dtg01100/rclone-mount-sync/internal/tui/components"
)

// NavigateToMsg is sent when the screen wants to navigate to a specific screen.
// The string value should be one of: "mounts", "sync_jobs", "services", "settings".
type NavigateToMsg struct {
	Target string
}

// GoBackMsg is sent when the screen wants to go back to the main menu.
type GoBackMsg struct{}

// MainMenuScreen is the main navigation screen.
type MainMenuScreen struct {
	menu   *components.Menu
	width  int
	height int
}

// NewMainMenuScreen creates a new main menu screen.
func NewMainMenuScreen() *MainMenuScreen {
	items := []components.MenuItem{
		{
			Label:       "Mount Management",
			Description: "Configure and manage rclone mount points",
			Key:         "M",
		},
		{
			Label:       "Sync Job Management",
			Description: "Configure and schedule rclone sync operations",
			Key:         "S",
		},
		{
			Label:       "Service Status",
			Description: "View and control systemd services",
			Key:         "V",
		},
		{
			Label:       "Settings",
			Description: "Application configuration",
			Key:         "T",
		},
		{
			Label:       "Quit",
			Description: "Exit the application",
			Key:         "Q",
		},
	}

	return &MainMenuScreen{
		menu: components.NewMenu(items),
	}
}

// SetSize sets the screen dimensions.
func (s *MainMenuScreen) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.menu.SetWidth(width - 8)
}

// Init initializes the screen.
func (s *MainMenuScreen) Init() tea.Cmd {
	return nil
}

// Update handles screen updates.
func (s *MainMenuScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		key := strings.ToLower(msg.String())
		switch key {
		case "up", "k":
			s.menu.Up()
		case "down", "j":
			s.menu.Down()
		case "enter", " ":
			return s, s.navigateFromCursor()
		case "m":
			return s, func() tea.Msg { return NavigateToMsg{Target: "mounts"} }
		case "s":
			return s, func() tea.Msg { return NavigateToMsg{Target: "sync_jobs"} }
		case "v":
			return s, func() tea.Msg { return NavigateToMsg{Target: "services"} }
		case "t":
			return s, func() tea.Msg { return NavigateToMsg{Target: "settings"} }
		case "q":
			return s, tea.Quit
		}
	}

	return s, nil
}

// navigateFromCursor returns a command to navigate based on the current cursor position.
func (s *MainMenuScreen) navigateFromCursor() tea.Cmd {
	selected := s.menu.Selected()
	switch selected.Key {
	case "M":
		return func() tea.Msg { return NavigateToMsg{Target: "mounts"} }
	case "S":
		return func() tea.Msg { return NavigateToMsg{Target: "sync_jobs"} }
	case "V":
		return func() tea.Msg { return NavigateToMsg{Target: "services"} }
	case "T":
		return func() tea.Msg { return NavigateToMsg{Target: "settings"} }
	case "Q":
		return tea.Quit
	default:
		return nil
	}
}

// View renders the screen.
func (s *MainMenuScreen) View() string {
	var b strings.Builder

	// Add some top padding
	b.WriteString("\n")

	// Render title
	title := components.Styles.Title.Render("Main Menu")
	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	// Render menu
	menuContent := s.menu.Render()

	// Center the menu
	menuBox := lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(menuContent)
	b.WriteString(menuBox)

	// Add help text at the bottom
	b.WriteString("\n\n")
	helpText := components.HelpBar(s.width, []components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "Enter", Desc: "select"},
		{Key: "M/S/V/T", Desc: "quick jump"},
		{Key: "?", Desc: "help"},
		{Key: "q", Desc: "quit"},
	})
	b.WriteString(helpText)

	return b.String()
}