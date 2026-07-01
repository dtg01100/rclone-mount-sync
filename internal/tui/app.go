// Package tui provides the terminal user interface for rclone-mount-sync.
package tui

import (
	"fmt"
	"os"
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

// newSystemdGenerator is a package-level function variable that
// initializeServices calls to create the systemd Generator. It is
// reassigned in tests so the failure path of initializeServices
// (systemd generator error) can be exercised without a real
// filesystem. Production behavior is identical to calling
// systemd.NewGenerator directly.
var newSystemdGenerator = systemd.NewGenerator

// newSystemdManager is the equivalent for the systemd Manager. It
// is also reassigned in tests; production behavior is identical to
// calling systemd.NewManager directly.
var newSystemdManager = systemd.NewManager

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

// LoadingMsg and LoadingDoneMsg are signal messages for async UI
// updates. They are handled by the cases below in the Update
// switch. The tests in this file assert that they actually flip
// a.loading as expected.

// LoadingMsg is sent when a loading state starts.
type LoadingMsg struct{}

// LoadingDoneMsg is sent when loading is complete.
type LoadingDoneMsg struct{}

// ReconciliationMsg is sent when orphan detection is complete.
type ReconciliationMsg struct {
	Result *systemd.ReconciliationResult
}

// App is the main TUI application model.
type App struct {
	version        string // Build version, set via NewApp/Run
	currentScreen  Screen
	previousScreen Screen
	width          int
	height         int
	loading        bool
	showHelp       bool
	initError      error

	// Help screen scroll state
	helpScrollY    int
	helpContentLen int

	// Screen models
	mainMenu *screens.MainMenuScreen
	mounts   *screens.MountsScreen
	syncJobs *screens.SyncJobsScreen
	services *screens.ServicesScreen
	settings *screens.SettingsScreen

	// Services
	config    *config.Config
	rclone    *rclone.Client
	generator *systemd.Generator
	manager   *systemd.Manager

	// skipInitializeServices is set by NewAppWithDeps. When
	// true, App.Init returns only the main-menu init (no
	// initializeServices Cmd), so the teatest seam can run a
	// pre-wired App without the production side-effects of
	// loading real config, spawning a real rclone client, or
	// constructing a real systemd Generator/Manager. It is the
	// test-only escape hatch that makes the dependency-injection
	// seam actually inert.
	skipInitializeServices bool

	// Orphan detection
	orphans          *systemd.ReconciliationResult
	showOrphanPrompt bool
	orphanSelected   int
	orphanMode       int
	orphanError      error
}

// NewApp creates a new TUI application with the given version.
func NewApp(version string) *App {
	return &App{
		version:        version,
		currentScreen:  ScreenMain,
		previousScreen: ScreenMain,
		mainMenu:       screens.NewMainMenuScreen(),
		mounts:         screens.NewMountsScreen(),
		syncJobs:       screens.NewSyncJobsScreen(),
		services:       screens.NewServicesScreen(),
		settings:       screens.NewSettingsScreen(),
	}
}

// AppDeps is the bundle of injected dependencies NewAppWithDeps
// expects. Any field may be nil; screens fall back to error-only
// modes for nil services, mirroring the existing direct-model tests
// that use &rclone.Client{} / a no-op MockManager.
//
// This is the test seam: production code paths (NewApp +
// initializeServices, and main.go) are unchanged. Tests construct an
// App pre-wired with mocks so tea.Program's Init() can run without
// touching the real filesystem, a real rclone binary, or a real
// systemd user session.
type AppDeps struct {
	// Config is the loaded configuration. Required by most screens;
	// nil is allowed but screen commands that read config will fail
	// gracefully.
	Config *config.Config

	// Rclone is the rclone client. May be nil; screens that need
	// remote listing will surface a friendly error rather than
	// panicking.
	Rclone *rclone.Client

	// Generator creates systemd unit files. May be nil.
	Generator *systemd.Generator

	// Manager controls systemd user services. May be nil; production
	// code typically passes a *systemd.Manager, tests can pass
	// *systemd.MockManager.
	Manager systemd.ServiceManager
}

// NewAppWithDeps creates an App whose services are pre-wired from
// the provided dependencies. It is the test seam used by the
// teatest suite in internal/tui/teatest/.
//
// Unlike NewApp (which returns a bare App and lets initializeServices
// load config / build a real client asynchronously), NewAppWithDeps
// wires dependencies synchronously: the returned App can be handed
// to tea.NewProgram and Init() will succeed without a real config
// directory, a real rclone binary, or a real systemd user session.
//
// The screen's Init cmds (e.g. mounts.Init, services.Init) still run
// in the background, but they no-op cleanly when their underlying
// dependencies are nil.
func NewAppWithDeps(version string, deps AppDeps) *App {
	app := NewApp(version)
	app.config = deps.Config
	app.rclone = deps.Rclone
	app.generator = deps.Generator
	// app.manager stays as the concrete *systemd.Manager field; for
	// the teatest seam, tests typically pass a *systemd.MockManager
	// (or nil) and screens that take a systemd.ServiceManager
	// interface get the mock via SetServices below. The orphan
	// import/cleanup paths require the concrete *Manager and are
	// not exercised through the PT (the orphan prompt is tested
	// against a pre-populated ReconciliationResult with no
	// real-systemd side effects).
	app.skipInitializeServices = true

	// Mirror initializeServices' SetServices calls so screens see
	// the same pre-wired state they would after a real init.
	if deps.Config != nil {
		app.mounts.SetServices(deps.Config, deps.Rclone, deps.Generator, deps.Manager)
		app.syncJobs.SetServices(deps.Config, deps.Rclone, deps.Generator, deps.Manager)
		app.services.SetServices(deps.Config, deps.Manager, deps.Generator)
		app.settings.SetConfig(deps.Config)
	}

	return app
}

// Init initializes the application.
func (a *App) Init() tea.Cmd {
	// In production, the screens' Init() cmds are kicked off
	// after the async initializeServices Cmd posts
	// AppInitDone / ReconciliationMsg (see the case statements
	// below). For the teatest seam (NewAppWithDeps), the
	// services are already wired, so we don't kick off any
	// screen loads here — they would run in a goroutine and
	// race with the render loop reading s.mounts/s.jobs (the
	// screens don't lock their state). The screens' inits
	// are fired when the user actually navigates to them
	// (see the NavigateToMsg and ScreenChangeMsg handlers
	// in Update), and tests that need a pre-loaded list can
	// use AppDeps.Config.
	if a.skipInitializeServices {
		return a.mainMenu.Init()
	}
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
	gen, err := newSystemdGenerator()
	if err != nil {
		return AppInitError{Err: err}
	}
	a.generator = gen

	// Initialize systemd manager
	a.manager = newSystemdManager()

	// Pass services to screens
	a.mounts.SetServices(cfg, a.rclone, gen, a.manager)
	a.syncJobs.SetServices(cfg, a.rclone, gen, a.manager)
	a.services.SetServices(cfg, a.manager, gen)
	a.settings.SetConfig(cfg)

	// Run reconciliation to detect orphaned units
	reconciler := systemd.NewReconciler(gen, a.manager)

	// Build sets of valid IDs
	mountIDs := make(map[string]bool)
	for _, m := range cfg.Mounts {
		mountIDs[m.ID] = true
	}
	syncIDs := make(map[string]bool)
	for _, j := range cfg.SyncJobs {
		syncIDs[j.ID] = true
	}

	result, err := reconciler.ScanForOrphans(mountIDs, syncIDs)
	if err != nil {
		return AppInitError{Err: fmt.Errorf("failed to scan for orphaned units: %w", err)}
	}

	if len(result.OrphanedUnits) > 0 {
		return ReconciliationMsg{Result: result}
	}

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
		if a.showOrphanPrompt {
			return a.updateOrphanPrompt(msg)
		}

		// Handle global keybindings
		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "up", "k":
			// Handle scrolling in help screen
			if a.showHelp && a.helpScrollY > 0 && a.helpContentLen > 0 {
				a.helpScrollY--
				return a, nil
			}
		case "down", "j":
			// Handle scrolling in help screen
			if a.showHelp && a.helpContentLen > 0 {
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
				// Recompute the help content length up front so View()
				// doesn't have to mutate state during rendering.
				a.helpContentLen = a.computeHelpContentLen()
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
		// If the help screen is open, a width change may affect line
		// wrapping, so recompute the help content length.
		if a.showHelp {
			a.helpContentLen = a.computeHelpContentLen()
		}

	case ScreenChangeMsg:
		a.currentScreen = msg.Screen
		a.showHelp = false
		// When the seam is active, the screen inits (other
		// than a.mounts) haven't been kicked off yet — see
		// App.Init. Trigger them here so each screen has
		// populated its list by the time the user navigates
		// to it.
		if a.skipInitializeServices {
			switch msg.Screen {
			case ScreenSyncJobs:
				cmds = append(cmds, a.syncJobs.Init())
			case ScreenServices:
				cmds = append(cmds, a.services.Init())
			}
		}
		return a, tea.Batch(cmds...)

	case AppInitError:
		a.initError = msg.Err
		a.loading = false

	case ReconciliationMsg:
		a.orphans = msg.Result
		a.showOrphanPrompt = len(msg.Result.OrphanedUnits) > 0
		cmds = append(cmds, a.mounts.Init(), a.syncJobs.Init(), a.services.Init())

	case AppInitDone:
		cmds = append(cmds, a.mounts.Init(), a.syncJobs.Init(), a.services.Init())

	case LoadingMsg:
		// Loading state is now signal-driven. A producer sends
		// LoadingMsg when it kicks off async work; LoadingDoneMsg
		// flips it back. This lets screens and async commands
		// (orphan import, config save) participate in the same
		// loading UI without reaching into App state directly.
		a.loading = true

	case LoadingDoneMsg:
		a.loading = false

	case OrphanActionMsg:
		a.loading = false
		if msg.Err != nil {
			a.orphanError = msg.Err
		} else {
			if a.orphans != nil && msg.Index >= 0 && msg.Index < len(a.orphans.OrphanedUnits) {
				a.orphans.OrphanedUnits = append(
					a.orphans.OrphanedUnits[:msg.Index],
					a.orphans.OrphanedUnits[msg.Index+1:]...,
				)
			}

			if len(a.orphans.OrphanedUnits) == 0 {
				a.orphanSelected = -1
				a.showOrphanPrompt = false
			} else if a.orphanSelected >= len(a.orphans.OrphanedUnits) {
				a.orphanSelected = len(a.orphans.OrphanedUnits) - 1
			}
			a.orphanMode = 0

			cmds = append(cmds, a.mounts.Init(), a.syncJobs.Init(), a.services.Init())
		}
	}

	// Update the current screen. Screens may return a tea.Cmd that
	// produces a NavigateToMsg or GoBackMsg; we invoke the cmd inline
	// (in this same Update cycle) so a single keypress navigates in
	// one cycle. To keep this the single source of truth for screen
	// transitions, the duplicate switch that used to live at the
	// bottom of this function has been removed.
	var screenCmd tea.Cmd
	switch a.currentScreen {
	case ScreenMain:
		model, cmd := a.mainMenu.Update(msg)
		if m, ok := model.(*screens.MainMenuScreen); ok {
			a.mainMenu = m
		}
		cmds = append(cmds, cmd)
		screenCmd = cmd

	case ScreenMounts:
		model, cmd := a.mounts.Update(msg)
		if m, ok := model.(*screens.MountsScreen); ok {
			a.mounts = m
		}
		cmds = append(cmds, cmd)
		screenCmd = cmd

	case ScreenSyncJobs:
		model, cmd := a.syncJobs.Update(msg)
		if m, ok := model.(*screens.SyncJobsScreen); ok {
			a.syncJobs = m
		}
		cmds = append(cmds, cmd)
		screenCmd = cmd

	case ScreenServices:
		model, cmd := a.services.Update(msg)
		if m, ok := model.(*screens.ServicesScreen); ok {
			a.services = m
		}
		cmds = append(cmds, cmd)
		screenCmd = cmd

	case ScreenSettings:
		model, cmd := a.settings.Update(msg)
		if m, ok := model.(*screens.SettingsScreen); ok {
			a.settings = m
		}
		cmds = append(cmds, cmd)
		screenCmd = cmd
	}

	// Apply navigation results inline so a single keypress changes
	// the screen in the same Update.
	//
	// We invoke the cmd synchronously here (not via tea.Batch below)
	// because the navigation messages are produced by the screen's
	// own Update handler and are already fully resolved — they don't
	// schedule any further async work. Calling them inline keeps a
	// single keypress in a single Update cycle and avoids the user
	// having to press Enter twice when navigating from the main menu.
	// This is safe: screenCmd was just returned from the screen's
	// Update on the same goroutine, and producing a non-nil tea.Msg
	// is a pure value operation (no side effects, no I/O).
	if screenCmd != nil {
		if navMsg := screenCmd(); navMsg != nil {
			switch m := navMsg.(type) {
			case screens.NavigateToMsg:
				switch m.Target {
				case "mounts":
					a.currentScreen = ScreenMounts
					if a.skipInitializeServices {
						cmds = append(cmds, a.mounts.Init())
					}
				case "sync_jobs":
					a.currentScreen = ScreenSyncJobs
					if a.skipInitializeServices {
						cmds = append(cmds, a.syncJobs.Init())
					}
				case "services":
					a.currentScreen = ScreenServices
					if a.skipInitializeServices {
						cmds = append(cmds, a.services.Init())
					}
				case "settings":
					a.currentScreen = ScreenSettings
				}
			case screens.GoBackMsg:
				a.currentScreen = ScreenMain
			}
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
	// Clamp to a sane minimum. A very small window (e.g. terminal
	// resized to 2 lines tall) would produce a negative contentHeight
	// that lipgloss then renders as zero/garbage. This mirrors the
	// defensive clamp in renderHelp.
	if contentHeight < 1 {
		contentHeight = 1
	}

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
	view := lipgloss.JoinVertical(lipgloss.Left,
		header,
		contentBox,
		status,
	)

	// Show orphan prompt overlay if needed
	if a.showOrphanPrompt && a.orphans != nil {
		view = a.renderOrphanPrompt(view)
	}

	return view
}

// renderHeader renders the top header bar.
func (a *App) renderHeader() string {
	return components.TitleBar(a.width, "Rclone Mount Sync", a.version)
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
// helpSections returns the help-screen content as a single string. It is
// the single source of truth for what renderHelp shows and how many lines
// it spans; computeHelpContentLen counts the lines in this string.
func (a *App) helpSections() string {
	groups := []struct {
		title string
		items []components.HelpItem
	}{
		{
			title: "Global Keybindings",
			items: []components.HelpItem{
				{Key: "↑/k", Desc: "Move up"},
				{Key: "↓/j", Desc: "Move down"},
				{Key: "Enter", Desc: "Select/confirm"},
				{Key: "Esc", Desc: "Go back/cancel"},
				{Key: "q", Desc: "Quit (from main menu) or go back"},
				{Key: "Ctrl+C", Desc: "Force quit"},
				{Key: "?", Desc: "Toggle this help screen"},
			},
		},
		{
			title: "Screen Navigation",
			items: []components.HelpItem{
				{Key: "M", Desc: "Mount Management"},
				{Key: "S", Desc: "Sync Job Management"},
				{Key: "V", Desc: "Service Status"},
				{Key: "T", Desc: "Settings"},
			},
		},
		{
			title: "Mount Management",
			items: []components.HelpItem{
				{Key: "a", Desc: "Add new mount"},
				{Key: "e", Desc: "Edit selected mount"},
				{Key: "d", Desc: "Delete selected mount"},
				{Key: "s", Desc: "Start mount"},
				{Key: "x", Desc: "Stop mount"},
				{Key: "Enter", Desc: "View details"},
				{Key: "r", Desc: "Refresh status"},
			},
		},
		{
			title: "Sync Job Management",
			items: []components.HelpItem{
				{Key: "a", Desc: "Add new sync job"},
				{Key: "e", Desc: "Edit selected sync job"},
				{Key: "d", Desc: "Delete selected sync job"},
				{Key: "r", Desc: "Run sync job now"},
				{Key: "t", Desc: "Toggle timer"},
			},
		},
		{
			title: "Service Status",
			items: []components.HelpItem{
				{Key: "s", Desc: "Start service"},
				{Key: "x", Desc: "Stop service"},
				{Key: "e", Desc: "Enable service"},
				{Key: "d", Desc: "Disable service"},
				{Key: "l", Desc: "View logs"},
				{Key: "r", Desc: "Refresh status"},
			},
		},
	}

	var b strings.Builder
	b.WriteString(components.Styles.Title.Render("Help & Keybindings") + "\n\n")
	for _, g := range groups {
		b.WriteString(components.Styles.Subtitle.Render(g.title) + "\n")
		for _, item := range g.items {
			fmt.Fprintf(&b, "  %s  %s\n",
				components.Styles.MenuKey.Render(item.Key),
				components.Styles.Normal.Render(item.Desc))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// computeHelpContentLen returns the number of lines the help screen will
// produce for the current state. Called from Update (when help is opened
// or the window is resized) so View() doesn't have to mutate state.
func (a *App) computeHelpContentLen() int {
	return strings.Count(a.helpSections(), "\n")
}

func (a *App) renderHelp() string {
	fullContent := a.helpSections()
	lines := strings.Split(fullContent, "\n")

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

func (a *App) updateOrphanPrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.loading {
		return a, nil
	}
	if a.orphans == nil {
		return a, nil
	}

	// While an error is displayed, ignore all keys except those that
	// dismiss the prompt. Without this gate, pressing enter or 'c'
	// re-runs the failing import/cleanup in a tight loop with no way
	// to acknowledge the error short of quitting.
	if a.orphanError != nil {
		switch msg.String() {
		case "esc", "q", "d":
			a.orphanError = nil
			a.orphanMode = 0
			a.showOrphanPrompt = false
		}
		return a, nil
	}

	orphans := a.orphans.OrphanedUnits

	switch msg.String() {
	case "up", "k":
		if a.orphanMode == 0 && a.orphanSelected > 0 {
			a.orphanSelected--
		}
	case "down", "j":
		if a.orphanMode == 0 && a.orphanSelected < len(orphans)-1 {
			a.orphanSelected++
		}
	case "enter":
		if a.orphanMode == 0 {
			a.orphanMode = 1
		} else {
			return a.importSelectedOrphan()
		}
	case "c":
		if a.orphanMode == 1 {
			return a.cleanupSelectedOrphan()
		}
	case "s":
		// Skip: remove from local list without touching the unit
		// file. Useful when the user wants to handle it manually
		// later. The unit file remains on disk and will be detected
		// again on next reconcile.
		if a.orphanMode == 0 && len(orphans) > 0 && a.orphanSelected < len(orphans) {
			a.orphans.OrphanedUnits = append(
				a.orphans.OrphanedUnits[:a.orphanSelected],
				a.orphans.OrphanedUnits[a.orphanSelected+1:]...,
			)
			if len(a.orphans.OrphanedUnits) == 0 {
				a.orphanSelected = -1
				a.showOrphanPrompt = false
			} else if a.orphanSelected >= len(a.orphans.OrphanedUnits) {
				a.orphanSelected = len(a.orphans.OrphanedUnits) - 1
			}
		}
	case "esc", "q":
		if a.orphanMode == 1 {
			a.orphanMode = 0
		} else {
			a.showOrphanPrompt = false
		}
	case "d":
		a.showOrphanPrompt = false
	}
	return a, nil
}

func (a *App) importSelectedOrphan() (tea.Model, tea.Cmd) {
	if a.orphans == nil || a.orphanSelected < 0 || a.orphanSelected >= len(a.orphans.OrphanedUnits) {
		return a, func() tea.Msg {
			return OrphanActionMsg{Err: fmt.Errorf("invalid orphan selection")}
		}
	}
	if a.generator == nil || a.manager == nil {
		return a, func() tea.Msg {
			return OrphanActionMsg{Err: fmt.Errorf("services not initialized")}
		}
	}
	orphan := a.orphans.OrphanedUnits[a.orphanSelected]

	a.loading = true
	a.orphanError = nil

	// Return a command to handle the heavy lifting
	return a, func() tea.Msg {
		reconciler := systemd.NewReconciler(a.generator, a.manager)
		imported, err := reconciler.Import(orphan)
		if err != nil {
			return OrphanActionMsg{Err: fmt.Errorf("failed to import orphan: %w", err)}
		}

		// We need to access config here, but it's safe since this is a command (async)
		// and we added mutexes to Config.
		if imported.Mount != nil {
			if err := a.config.AddMount(*imported.Mount); err != nil {
				return OrphanActionMsg{Err: fmt.Errorf("failed to add mount: %w", err)}
			}
		} else if imported.SyncJob != nil {
			if err := a.config.AddSyncJob(*imported.SyncJob); err != nil {
				return OrphanActionMsg{Err: fmt.Errorf("failed to add sync job: %w", err)}
			}
		}

		if err := a.config.Save(); err != nil {
			return OrphanActionMsg{Err: fmt.Errorf("failed to save config: %w", err)}
		}

		var writeErr error
		if imported.Mount != nil {
			_, writeErr = a.generator.WriteMountService(imported.Mount)
		} else if imported.SyncJob != nil {
			_, _, writeErr = a.generator.WriteSyncUnits(imported.SyncJob)
		}

		if writeErr != nil {
			if imported.Mount != nil {
				_ = a.config.RemoveMount(imported.Mount.Name)
			} else if imported.SyncJob != nil {
				_ = a.config.RemoveSyncJob(imported.SyncJob.Name)
			}
			if saveErr := a.config.Save(); saveErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to rollback config: %v\n", saveErr)
			}
			return OrphanActionMsg{Err: fmt.Errorf("failed to write service file: %w", writeErr)}
		}

		if err := reconciler.RemoveOrphan(orphan); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove orphan unit file %s: %v\n", orphan.Name, err)
		}

		return OrphanActionMsg{
			Action: "import",
			Index:  a.orphanSelected,
		}
	}
}

func (a *App) cleanupSelectedOrphan() (tea.Model, tea.Cmd) {
	if a.orphans == nil || a.orphanSelected < 0 || a.orphanSelected >= len(a.orphans.OrphanedUnits) {
		return a, func() tea.Msg {
			return OrphanActionMsg{Err: fmt.Errorf("invalid orphan selection")}
		}
	}
	if a.generator == nil || a.manager == nil {
		return a, func() tea.Msg {
			return OrphanActionMsg{Err: fmt.Errorf("services not initialized")}
		}
	}
	orphan := a.orphans.OrphanedUnits[a.orphanSelected]

	a.loading = true
	a.orphanError = nil

	return a, func() tea.Msg {
		reconciler := systemd.NewReconciler(a.generator, a.manager)
		if err := reconciler.RemoveOrphan(orphan); err != nil {
			return OrphanActionMsg{Err: fmt.Errorf("failed to cleanup orphan: %w", err)}
		}

		return OrphanActionMsg{
			Action: "cleanup",
			Index:  a.orphanSelected,
		}
	}
}

// OrphanActionMsg is sent when an orphan action is completed.
type OrphanActionMsg struct {
	Action string // "import" or "cleanup"
	Index  int
	Err    error
}

// orphanErrorSuggestions returns a short list of suggested next
// steps for a given orphan-action error, matched by substring. The
// goal is to translate raw "failed to X: <system error>" text into
// something a user can act on.
func orphanErrorSuggestions(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "permission denied"):
		return "Suggestions:\n  • Check that you own the unit file (chown $USER:$USER <path>)\n  • Try running without sudo; this app uses your user systemd session"
	case strings.Contains(msg, "name already exists") || strings.Contains(msg, "duplicate"):
		return "Suggestions:\n  • A mount or sync job with this name is already in your config\n  • Edit the existing entry instead of importing this orphan"
	case strings.Contains(msg, "failed to write service file"):
		return "Suggestions:\n  • Ensure ~/.config/systemd/user/ is writable\n  • If the unit file already exists, remove it manually first"
	case strings.Contains(msg, "failed to import") || strings.Contains(msg, "no remote"):
		return "Suggestions:\n  • The unit references a remote that is no longer configured\n  • Run 'rclone config' to recreate the remote, or cleanup (delete) this orphan"
	case strings.Contains(msg, "failed to cleanup") || strings.Contains(msg, "failed to remove orphan"):
		return "Suggestions:\n  • The unit file is in use or owned by another process\n  • Stop the service first, or remove the unit file manually"
	default:
		return "Suggestions:\n  • See the message above for the underlying cause\n  • Re-run 'rclone-mount-sync doctor' to verify your environment"
	}
}

func (a *App) renderOrphanPrompt(baseView string) string {
	var b strings.Builder

	b.WriteString(components.Styles.Warning.Render("Orphaned Units Detected"))
	b.WriteString("\n\n")

	switch {
	case a.loading:
		b.WriteString(components.Styles.Info.Render("Processing..."))
	case a.orphanError != nil:
		b.WriteString(components.RenderError(a.orphanError.Error()))
		b.WriteString("\n")
		b.WriteString(components.Styles.HelpText.Render(orphanErrorSuggestions(a.orphanError)))
		b.WriteString("\n")
		b.WriteString(components.Styles.HelpText.Render("Press Esc to dismiss"))
	case a.orphanMode == 0:
		b.WriteString("Select a unit to manage:\n\n")
		for i, orphan := range a.orphans.OrphanedUnits {
			legacyTag := ""
			if orphan.IsLegacy {
				legacyTag = " (legacy)"
			}
			cursor := "  "
			if i == a.orphanSelected {
				cursor = "> "
				fmt.Fprintf(&b, "%s%s [%s%s]\n", cursor, components.Styles.Selected.Render(orphan.Name), orphan.Type, legacyTag)
			} else {
				fmt.Fprintf(&b, "%s%s [%s%s]\n", cursor, orphan.Name, orphan.Type, legacyTag)
			}
		}
		b.WriteString("\n")
		b.WriteString(components.Styles.HelpText.Render("[↑/k↓/j] Navigate  [Enter] Select  [s] Skip  [d] Dismiss all  [q/Esc] Close"))
	default:
		orphan := a.orphans.OrphanedUnits[a.orphanSelected]
		legacyTag := ""
		if orphan.IsLegacy {
			legacyTag = " (legacy)"
		}
		fmt.Fprintf(&b, "Unit: %s\n", orphan.Name)
		fmt.Fprintf(&b, "Type: %s%s\n", orphan.Type, legacyTag)
		fmt.Fprintf(&b, "Path: %s\n\n", orphan.Path)
		b.WriteString(components.Styles.HelpText.Render("[Enter] Import to config  [c] Cleanup (delete)  [Esc] Back"))
	}

	promptContent := b.String()

	boxWidth := a.width - 8
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxWidth > 70 {
		boxWidth = 70
	}

	box := lipgloss.NewStyle().
		Width(boxWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("yellow")).
		Render(promptContent)

	overlay := lipgloss.Place(a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceChars(" "),
	)

	return overlay
}

// Run starts the TUI application.
func Run() error {
	app := NewApp(Version)
	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
