// Package tui provides the terminal user interface for rclone-mount-sync.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/rclone"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
	"github.com/dtg01100/rclone-mount-sync/internal/tui/components"
	"github.com/dtg01100/rclone-mount-sync/internal/tui/screens"
)

// Version is set at build time via ldflags.
var Version = "dev"

// Screen represents a TUI screen in the application.
type Screen int

const (
	ScreenMain Screen = iota
	ScreenMounts
	ScreenSyncJobs
	ScreenServices
	ScreenSettings
	ScreenHelp
)

// String returns the string representation of a screen.
func (s Screen) String() string {
	switch s {
	case ScreenMain:
		return "Main Menu"
	case ScreenMounts:
		return "Mount Management"
	case ScreenSyncJobs:
		return "Sync Job Management"
	case ScreenServices:
		return "Service Status"
	case ScreenSettings:
		return "Settings"
	case ScreenHelp:
		return "Help"
	default:
		return "Unknown"
	}
}

// ScreenChangeMsg is sent when the screen should change.
type ScreenChangeMsg struct {
	Screen Screen
}

// LoadingMsg is sent when a loading state starts.
type LoadingMsg struct{}

// LoadingDoneMsg is sent when loading is complete.
type LoadingDoneMsg struct{}

// App is the main TUI application model.
type App struct {
	currentScreen Screen
	previousScreen Screen
	width         int
	height        int
	loading       bool
	showHelp      bool
	initError     error

	// Help screen scroll state
	helpScrollY    int
	helpContentLen int

	// Screen models
	mainMenu   *screens.MainMenuScreen
	mounts     *screens.MountsScreen
	syncJobs   *screens.SyncJobsScreen
	services   *screens.ServicesScreen
	settings   *screens.SettingsScreen

	// Services
	config     *config.Config
	rclone     *rclone.Client
	generator  *systemd.Generator
	manager    *systemd.Manager
}

// NewApp creates a new TUI application.
func NewApp() *App {
	return &App{
		currentScreen: ScreenMain,
		previousScreen: ScreenMain,
		mainMenu:      screens.NewMainMenuScreen(),
		mounts:        screens.NewMountsScreen(),
		syncJobs:      screens.NewSyncJobsScreen(),
		services:      screens.NewServicesScreen(),
		settings:      screens.NewSettingsScreen(),
	}
}

// Init initializes the application.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.mainMenu.Init(),
		a.initializeServices,
	)
}

// initializeServices initializes the application services.
func (a *App) initializeServices() tea.Msg {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return AppInitError{Err: err}
	}
	a.config = cfg

	// Initialize rclone client
	a.rclone = rclone.NewClient()

	// Initialize systemd generator
	gen, err := systemd.NewGenerator()
	if err != nil {
		return AppInitError{Err: err}
	}
	a.generator = gen

	// Initialize systemd manager
	a.manager = systemd.NewManager()

	// Pass services to screens
	a.mounts.SetServices(cfg, a.rclone, gen, a.manager)
	a.services.SetServices(cfg, a.manager)
	a.settings.SetConfig(cfg)

	return AppInitDone{}
}

// AppInitError is sent when app initialization fails.
type AppInitError struct {
	Err error
}

// AppInitDone is sent when app initialization is complete.
type AppInitDone struct{}

// Update handles application updates.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle global keybindings
		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "up", "k":
			// Handle scrolling in help screen
			if a.showHelp && a.helpScrollY > 0 {
				a.helpScrollY--
				return a, nil
			}
		case "down", "j":
			// Handle scrolling in help screen
			if a.showHelp {
				maxScroll := a.helpContentLen - (a.height - 6)
				if maxScroll > 0 && a.helpScrollY < maxScroll {
					a.helpScrollY++
				}
				return a, nil
			}
		case "q":
			// Q quits from main menu, goes back from other screens
			if a.currentScreen == ScreenMain {
				return a, tea.Quit
			}
			// Go back to previous screen or main menu
			if a.currentScreen == ScreenHelp {
				a.currentScreen = a.previousScreen
				a.showHelp = false
			} else {
				a.currentScreen = ScreenMain
			}
			return a, nil
		case "esc":
			// Escape goes back or closes help
			if a.showHelp {
				a.currentScreen = a.previousScreen
				a.showHelp = false
				return a, nil
			}
			if a.currentScreen != ScreenMain {
				a.currentScreen = ScreenMain
				return a, nil
			}
		case "?":
			// Toggle help
			if !a.showHelp {
				a.previousScreen = a.currentScreen
				a.currentScreen = ScreenHelp
				a.showHelp = true
				a.helpScrollY = 0 // Reset scroll position
			}
			return a, nil
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Propagate size to all screens
		a.mainMenu.SetSize(a.width, a.height)
		a.mounts.SetSize(a.width, a.height)
		a.syncJobs.SetSize(a.width, a.height)
		a.services.SetSize(a.width, a.height)
		a.settings.SetSize(a.width, a.height)

	case ScreenChangeMsg:
		a.currentScreen = msg.Screen
		a.showHelp = false
		return a, nil

	case AppInitError:
		// Store the error and show it to the user
		a.initError = msg.Err
		a.loading = false

	case AppInitDone:
		// Services initialized, now initialize the mounts screen
		cmds = append(cmds, a.mounts.Init())
	}

	// Update the current screen
	switch a.currentScreen {
	case ScreenMain:
		model, cmd := a.mainMenu.Update(msg)
		if m, ok := model.(*screens.MainMenuScreen); ok {
			a.mainMenu = m
		}
		cmds = append(cmds, cmd)

		// Check if main menu wants to navigate
		if a.mainMenu.ShouldNavigate() {
			target := a.mainMenu.GetNavigationTarget()
			a.mainMenu.ResetNavigation()
			switch target {
			case "mounts":
				a.currentScreen = ScreenMounts
			case "sync_jobs":
				a.currentScreen = ScreenSyncJobs
			case "services":
				a.currentScreen = ScreenServices
			case "settings":
				a.currentScreen = ScreenSettings
			case "quit":
				return a, tea.Quit
			}
		}

	case ScreenMounts:
		model, cmd := a.mounts.Update(msg)
		if m, ok := model.(*screens.MountsScreen); ok {
			a.mounts = m
		}
		cmds = append(cmds, cmd)

		// Check if mounts screen wants to go back
		if a.mounts.ShouldGoBack() {
			a.mounts.ResetGoBack()
			a.currentScreen = ScreenMain
		}

	case ScreenSyncJobs:
		model, cmd := a.syncJobs.Update(msg)
		if m, ok := model.(*screens.SyncJobsScreen); ok {
			a.syncJobs = m
		}
		cmds = append(cmds, cmd)

		// Check if sync jobs screen wants to go back
		if a.syncJobs.ShouldGoBack() {
			a.syncJobs.ResetGoBack()
			a.currentScreen = ScreenMain
		}

	case ScreenServices:
		model, cmd := a.services.Update(msg)
		if m, ok := model.(*screens.ServicesScreen); ok {
			a.services = m
		}
		cmds = append(cmds, cmd)

		// Check if services screen wants to go back
		if a.services.ShouldGoBack() {
			a.services.ResetGoBack()
			a.currentScreen = ScreenMain
		}

	case ScreenSettings:
		model, cmd := a.settings.Update(msg)
		if m, ok := model.(*screens.SettingsScreen); ok {
			a.settings = m
		}
		cmds = append(cmds, cmd)

		// Check if settings screen wants to go back
		if a.settings.ShouldGoBack() {
			a.settings.ResetGoBack()
			a.currentScreen = ScreenMain
		}
	}

	return a, tea.Batch(cmds...)
}

// View renders the application.
func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	// Show initialization error if present
	if a.initError != nil {
		return a.renderInitError()
	}

	// Calculate layout
	headerHeight := 1
	statusHeight := 1
	contentHeight := a.height - headerHeight - statusHeight

	// Render header
	header := a.renderHeader()

	// Render content
	var content string
	switch a.currentScreen {
	case ScreenMain:
		content = a.mainMenu.View()
	case ScreenMounts:
		content = a.mounts.View()
	case ScreenSyncJobs:
		content = a.syncJobs.View()
	case ScreenServices:
		content = a.services.View()
	case ScreenSettings:
		content = a.settings.View()
	case ScreenHelp:
		content = a.renderHelp()
	}

	// Ensure content fits in available space
	contentBox := lipgloss.NewStyle().
		Width(a.width).
		Height(contentHeight).
		Render(content)

	// Render status bar
	status := a.renderStatusBar()

	// Combine all parts
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		contentBox,
		status,
	)
}

// renderHeader renders the top header bar.
func (a *App) renderHeader() string {
	return components.TitleBar(a.width, "Rclone Mount Sync", Version)
}

// renderStatusBar renders the bottom status bar.
func (a *App) renderStatusBar() string {
	var statusText string
	if a.showHelp {
		statusText = "Press Esc or q to close help"
	} else {
		statusText = fmt.Sprintf("Screen: %s | ?: Help | q: Quit", a.currentScreen.String())
	}
	return components.StatusBar(a.width, statusText)
}

// renderHelp renders the help screen.
func (a *App) renderHelp() string {
	var b strings.Builder

	// Title
	title := components.Styles.Title.Render("Help & Keybindings")
	b.WriteString(title + "\n\n")

	// Global keybindings
	b.WriteString(components.Styles.Subtitle.Render("Global Keybindings") + "\n")
	globalKeys := []components.HelpItem{
		{Key: "↑/k", Desc: "Move up"},
		{Key: "↓/j", Desc: "Move down"},
		{Key: "Enter", Desc: "Select/confirm"},
		{Key: "Esc", Desc: "Go back/cancel"},
		{Key: "q", Desc: "Quit (from main menu) or go back"},
		{Key: "Ctrl+C", Desc: "Force quit"},
		{Key: "?", Desc: "Toggle this help screen"},
	}

	for _, item := range globalKeys {
		line := fmt.Sprintf("  %s  %s",
			components.Styles.MenuKey.Render(item.Key),
			components.Styles.Normal.Render(item.Desc))
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")

	// Screen-specific keybindings
	b.WriteString(components.Styles.Subtitle.Render("Screen Navigation") + "\n")
	screenKeys := []components.HelpItem{
		{Key: "M", Desc: "Mount Management"},
		{Key: "S", Desc: "Sync Job Management"},
		{Key: "V", Desc: "Service Status"},
		{Key: "T", Desc: "Settings"},
	}

	for _, item := range screenKeys {
		line := fmt.Sprintf("  %s  %s",
			components.Styles.MenuKey.Render(item.Key),
			components.Styles.Normal.Render(item.Desc))
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")

	// Mount screen keybindings
	b.WriteString(components.Styles.Subtitle.Render("Mount Management") + "\n")
	mountKeys := []components.HelpItem{
		{Key: "a", Desc: "Add new mount"},
		{Key: "e", Desc: "Edit selected mount"},
		{Key: "d", Desc: "Delete selected mount"},
		{Key: "s", Desc: "Start mount"},
		{Key: "x", Desc: "Stop mount"},
		{Key: "Enter", Desc: "View details"},
		{Key: "r", Desc: "Refresh status"},
	}

	for _, item := range mountKeys {
		line := fmt.Sprintf("  %s  %s",
			components.Styles.MenuKey.Render(item.Key),
			components.Styles.Normal.Render(item.Desc))
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")

	// Sync job screen keybindings
	b.WriteString(components.Styles.Subtitle.Render("Sync Job Management") + "\n")
	syncKeys := []components.HelpItem{
		{Key: "a", Desc: "Add new sync job"},
		{Key: "e", Desc: "Edit selected sync job"},
		{Key: "d", Desc: "Delete selected sync job"},
		{Key: "r", Desc: "Run sync job now"},
		{Key: "t", Desc: "Toggle timer"},
	}

	for _, item := range syncKeys {
		line := fmt.Sprintf("  %s  %s",
			components.Styles.MenuKey.Render(item.Key),
			components.Styles.Normal.Render(item.Desc))
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")

	// Services screen keybindings
	b.WriteString(components.Styles.Subtitle.Render("Service Status") + "\n")
	serviceKeys := []components.HelpItem{
		{Key: "s", Desc: "Start service"},
		{Key: "x", Desc: "Stop service"},
		{Key: "e", Desc: "Enable service"},
		{Key: "d", Desc: "Disable service"},
		{Key: "l", Desc: "View logs"},
		{Key: "r", Desc: "Refresh status"},
	}

	for _, item := range serviceKeys {
		line := fmt.Sprintf("  %s  %s",
			components.Styles.MenuKey.Render(item.Key),
			components.Styles.Normal.Render(item.Desc))
		b.WriteString(line + "\n")
	}

	// Get the full content
	fullContent := b.String()
	lines := strings.Split(fullContent, "\n")
	a.helpContentLen = len(lines)

	// Calculate visible area
	availableHeight := a.height - 6 // Account for border and status
	if availableHeight < 1 {
		availableHeight = 1
	}

	// Apply scroll
	startLine := a.helpScrollY
	if startLine < 0 {
		startLine = 0
	}
	endLine := startLine + availableHeight
	if endLine > len(lines) {
		endLine = len(lines)
	}

	// Get visible lines
	visibleLines := lines[startLine:endLine]
	visibleContent := strings.Join(visibleLines, "\n")

	// Add scroll indicator if needed
	maxScroll := len(lines) - availableHeight
	if maxScroll > 0 {
		scrollInfo := fmt.Sprintf("\n\n[%d/%d] ↑/↓ to scroll", startLine+1, maxScroll+1)
		visibleContent += components.Styles.HelpText.Render(scrollInfo)
	}

	// Wrap in a box
	return components.Styles.Border.
		Width(a.width - 4).
		Render(visibleContent)
}

// renderInitError renders the initialization error screen.
func (a *App) renderInitError() string {
	var b strings.Builder

	// Title
	title := components.Styles.Title.Render("Initialization Error")
	b.WriteString(lipgloss.NewStyle().
		Width(a.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	// Error message
	errorMsg := fmt.Sprintf("Failed to initialize application:\n\n%v", a.initError)
	b.WriteString(lipgloss.NewStyle().
		Width(a.width).
		Align(lipgloss.Center).
		Render(components.RenderError(errorMsg)))
	b.WriteString("\n\n")

	// Suggestions
	b.WriteString(lipgloss.NewStyle().
		Width(a.width).
		Align(lipgloss.Center).
		Render(components.Styles.Subtitle.Render("Possible solutions:")))
	b.WriteString("\n\n")

	suggestions := []string{
		"• Ensure rclone is installed and in your PATH",
		"• Run 'rclone config' to configure at least one remote",
		"• Check that systemd user session is available",
		"• Verify you have proper permissions for the config directory",
	}

	for _, suggestion := range suggestions {
		b.WriteString(lipgloss.NewStyle().
			Width(a.width).
			Align(lipgloss.Center).
			Render(suggestion))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Quit hint
	quitHint := components.Styles.HelpText.Render("Press q or Ctrl+C to quit")
	b.WriteString(lipgloss.NewStyle().
		Width(a.width).
		Align(lipgloss.Center).
		Render(quitHint))

	return b.String()
}

// Run starts the TUI application.
func Run() error {
	app := NewApp()
	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
