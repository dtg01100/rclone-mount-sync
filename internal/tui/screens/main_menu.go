// Package screens provides individual TUI screens for the application.
package screens

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dtg01100/rclone-mount-sync/internal/tui/components"
)

// MainMenuScreen is the main navigation screen.
type MainMenuScreen struct {
	menu             *components.Menu
	width            int
	height           int
	navigate         bool
	navigationTarget string
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := strings.ToLower(msg.String())
		switch key {
		case "up", "k":
			s.menu.Up()
		case "down", "j":
			s.menu.Down()
		case "enter", " ":
			s.selectCurrent()
		case "m":
			s.navigationTarget = "mounts"
			s.navigate = true
		case "s":
			s.navigationTarget = "sync_jobs"
			s.navigate = true
		case "v":
			s.navigationTarget = "services"
			s.navigate = true
		case "t":
			s.navigationTarget = "settings"
			s.navigate = true
		case "q":
			s.navigationTarget = "quit"
			s.navigate = true
		}
	}

	return s, nil
}

// selectCurrent selects the current menu item.
func (s *MainMenuScreen) selectCurrent() {
	selected := s.menu.Selected()
	switch selected.Key {
	case "M":
		s.navigationTarget = "mounts"
		s.navigate = true
	case "S":
		s.navigationTarget = "sync_jobs"
		s.navigate = true
	case "V":
		s.navigationTarget = "services"
		s.navigate = true
	case "T":
		s.navigationTarget = "settings"
		s.navigate = true
	case "Q":
		s.navigationTarget = "quit"
		s.navigate = true
	}
}

// ShouldNavigate returns true if the screen should navigate to another screen.
func (s *MainMenuScreen) ShouldNavigate() bool {
	return s.navigate
}

// GetNavigationTarget returns the target screen to navigate to.
func (s *MainMenuScreen) GetNavigationTarget() string {
	return s.navigationTarget
}

// ResetNavigation resets the navigation state.
func (s *MainMenuScreen) ResetNavigation() {
	s.navigate = false
	s.navigationTarget = ""
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
