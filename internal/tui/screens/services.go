// Package screens provides individual TUI screens for the application.
package screens

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
	"github.com/dtg01100/rclone-mount-sync/internal/tui/components"
)

// Screen modes for the services screen
const (
	ServicesModeList    = "list"    // Main service list
	ServicesModeDetails = "details" // Service details
	ServicesModeLogs    = "logs"    // Log viewer
	ServicesModeActions = "actions" // Action menu
)

// Service filter types
const (
	FilterAll      = "all"
	FilterRunning  = "running"
	FilterStopped  = "stopped"
	FilterFailed   = "failed"
	FilterMounts   = "mounts"
	FilterSyncJobs = "sync"
)

// ServicesScreen handles service status and management.
type ServicesScreen struct {
	// Services
	services         []ServiceInfo
	filteredServices []ServiceInfo

	// Systemd manager and generator
	manager   *systemd.Manager
	generator *systemd.Generator

	// Config for service types
	cfg *config.Config

	// UI state
	mode   string
	cursor int
	width  int
	height int
	goBack bool

	// Filter
	filter string

	// Details view
	selectedService *ServiceInfo
	detailedStatus  *models.ServiceStatus

	// Logs view
	logs        string
	logsLoading bool
	logFilter   string // error, warning, info, debug, all

	// Action menu
	showActions  bool
	actionCursor int

	// Bulk operations
	showBulkMenu bool
	bulkCursor   int

	// Status messages
	statusMessage     string
	statusMessageType string // success, error, info

	// Loading state
	loading bool

	// Systemd status panel
	systemdStatus SystemdStatus
}

// SystemdStatus holds overall systemd user manager status.
type SystemdStatus struct {
	Available      bool
	FailedUnits    int
	SessionType    string
	ActiveServices int
	ActiveTimers   int
}

// ServiceInfo represents display information about a service.
type ServiceInfo struct {
	Name        string // ID-based systemd unit name (e.g., "rclone-mount-abc12345")
	DisplayName string // Friendly name for display (e.g., "my-mount")
	Type        string // "mount" or "sync"
	Status      string // active, inactive, failed, activating
	SubState    string // running, dead, exited
	Enabled     bool
	MountPoint  string // For mounts
	Remote      string // For mounts
	Source      string // For sync
	Destination string // For sync
	NextRun     time.Time
	LastRun     time.Time
	TimerActive bool
}

// Messages

// ServicesLoadedMsg is sent when services are loaded.
type ServicesLoadedMsg struct {
	Services []ServiceInfo
}

// ServiceActionMsg is sent to perform an action on a service.
type ServiceActionMsg struct {
	Name   string
	Action string // start, stop, restart, enable, disable
}

// ServiceActionResultMsg is sent after a service action completes.
type ServiceActionResultMsg struct {
	Name    string
	Action  string
	Success bool
	Error   string
}

// ServiceLogsMsg is sent to request logs for a service.
type ServiceLogsMsg struct {
	Name string
}

// ServiceLogsLoadedMsg is sent when logs are loaded.
type ServiceLogsLoadedMsg struct {
	Name string
	Logs string
}

// ServicesErrorMsg is sent when an error occurs.
type ServicesErrorMsg struct {
	Err error
}

// RefreshServicesMsg triggers a refresh of the services list.
type RefreshServicesMsg struct{}

// NewServicesScreen creates a new services screen.
func NewServicesScreen() *ServicesScreen {
	return &ServicesScreen{
		services:          []ServiceInfo{},
		filteredServices:  []ServiceInfo{},
		mode:              ServicesModeList,
		filter:            FilterAll,
		logFilter:         "all",
		statusMessageType: "info",
	}
}

// SetServices sets the required services for the screen.
func (s *ServicesScreen) SetServices(cfg *config.Config, manager *systemd.Manager, generator *systemd.Generator) {
	s.cfg = cfg
	s.manager = manager
	s.generator = generator
}

// Init initializes the screen and loads services.
func (s *ServicesScreen) Init() tea.Cmd {
	return s.loadServices
}

// loadServices loads all services from systemd.
func (s *ServicesScreen) loadServices() tea.Msg {
	if s.manager == nil {
		return ServicesLoadedMsg{Services: []ServiceInfo{}}
	}

	var services []ServiceInfo

	// Load mount services from config
	if s.cfg != nil {
		for _, mount := range s.cfg.Mounts {
			serviceName := s.generator.ServiceName(mount.ID, "mount")
			status, err := s.manager.Status(serviceName + ".service")
			if err != nil {
				// Service might not exist yet
				services = append(services, ServiceInfo{
					Name:        serviceName,
					DisplayName: mount.Name,
					Type:        "mount",
					Status:      "not-found",
					Enabled:     mount.Enabled,
					MountPoint:  mount.MountPoint,
					Remote:      mount.Remote,
				})
				continue
			}

			services = append(services, ServiceInfo{
				Name:        serviceName,
				DisplayName: mount.Name,
				Type:        "mount",
				Status:      status.State,
				SubState:    status.SubState,
				Enabled:     status.Enabled,
				MountPoint:  mount.MountPoint,
				Remote:      mount.Remote,
			})
		}

		// Load sync job services from config
		for _, job := range s.cfg.SyncJobs {
			serviceName := s.generator.ServiceName(job.ID, "sync")

			// Get service status
			status, err := s.manager.Status(serviceName + ".service")
			if err != nil {
				services = append(services, ServiceInfo{
					Name:        serviceName,
					DisplayName: job.Name,
					Type:        "sync",
					Status:      "not-found",
					Enabled:     job.Enabled,
					Source:      job.Source,
					Destination: job.Destination,
				})
				continue
			}

			// Get timer status for sync jobs
			timerName := serviceName + ".timer"
			timerStatus, _ := s.manager.Status(timerName)
			timerActive := timerStatus != nil && timerStatus.Active

			// Get next run time
			nextRun, _ := s.manager.GetTimerNextRun(timerName)

			services = append(services, ServiceInfo{
				Name:        serviceName,
				DisplayName: job.Name,
				Type:        "sync",
				Status:      status.State,
				SubState:    status.SubState,
				Enabled:     status.Enabled,
				Source:      job.Source,
				Destination: job.Destination,
				NextRun:     nextRun,
				TimerActive: timerActive,
			})
		}
	}

	// Load systemd status
	s.systemdStatus = s.loadSystemdStatus()

	return ServicesLoadedMsg{
		Services: services,
	}
}

// loadSystemdStatus loads the overall systemd user manager status.
func (s *ServicesScreen) loadSystemdStatus() SystemdStatus {
	status := SystemdStatus{
		Available:   s.manager.IsSystemdAvailable(),
		SessionType: "user@.service",
	}

	if !status.Available {
		return status
	}

	// Get failed units count
	cmd := exec.Command("systemctl", "--user", "list-units", "--state=failed", "--no-legend")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		count := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		status.FailedUnits = count
	}

	// Count active services and timers
	cmd = exec.Command("systemctl", "--user", "list-units", "--type=service", "--state=active", "--no-legend")
	output, _ = cmd.Output()
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "rclone-") {
			status.ActiveServices++
		}
	}

	cmd = exec.Command("systemctl", "--user", "list-units", "--type=timer", "--state=active", "--no-legend")
	output, _ = cmd.Output()
	lines = strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "rclone-") {
			status.ActiveTimers++
		}
	}

	return status
}

// SetSize sets the screen dimensions.
func (s *ServicesScreen) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// Update handles screen updates.
func (s *ServicesScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ServicesLoadedMsg:
		s.services = msg.Services
		s.applyFilter()
		s.loading = false

	case ServicesErrorMsg:
		s.statusMessage = fmt.Sprintf("Error: %v", msg.Err)
		s.statusMessageType = "error"
		s.loading = false

	case RefreshServicesMsg:
		s.loading = true
		return s, s.loadServices

	case ServiceActionResultMsg:
		if msg.Success {
			s.statusMessage = fmt.Sprintf("%s: %s completed successfully", msg.Name, msg.Action)
			s.statusMessageType = "success"
		} else {
			s.statusMessage = fmt.Sprintf("%s: %s failed - %s", msg.Name, msg.Action, msg.Error)
			s.statusMessageType = "error"
		}
		// Refresh services after action
		cmds = append(cmds, s.loadServices)

	case ServiceLogsLoadedMsg:
		s.logs = msg.Logs
		s.logsLoading = false

	case tea.KeyMsg:
		switch s.mode {
		case ServicesModeList:
			cmds = append(cmds, s.handleListKeyPress(msg)...)
		case ServicesModeDetails:
			cmds = append(cmds, s.handleDetailsKeyPress(msg)...)
		case ServicesModeLogs:
			cmds = append(cmds, s.handleLogsKeyPress(msg)...)
		case ServicesModeActions:
			cmds = append(cmds, s.handleActionsKeyPress(msg)...)
		}
	}

	return s, tea.Batch(cmds...)
}

// handleListKeyPress handles key presses in list mode.
func (s *ServicesScreen) handleListKeyPress(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	switch msg.String() {
	case "up", "k":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		if s.cursor < len(s.filteredServices)-1 {
			s.cursor++
		}
	case "enter":
		// View details
		if len(s.filteredServices) > 0 && s.cursor < len(s.filteredServices) {
			s.selectedService = &s.filteredServices[s.cursor]
			s.mode = ServicesModeDetails
			s.loadDetailedStatus()
		}
	case "s":
		// Start service
		if len(s.filteredServices) > 0 {
			service := s.filteredServices[s.cursor]
			cmds = append(cmds, s.doServiceAction(service.Name+".service", "start"))
		}
	case "x":
		// Stop service
		if len(s.filteredServices) > 0 {
			service := s.filteredServices[s.cursor]
			cmds = append(cmds, s.doServiceAction(service.Name+".service", "stop"))
		}
	case "r":
		// Restart service
		if len(s.filteredServices) > 0 {
			service := s.filteredServices[s.cursor]
			cmds = append(cmds, s.doServiceAction(service.Name+".service", "restart"))
		}
	case "e":
		// Enable service
		if len(s.filteredServices) > 0 {
			service := s.filteredServices[s.cursor]
			unitName := service.Name
			if service.Type == "sync" {
				unitName += ".timer"
			} else {
				unitName += ".service"
			}
			cmds = append(cmds, s.doServiceAction(unitName, "enable"))
		}
	case "d":
		// Disable service
		if len(s.filteredServices) > 0 {
			service := s.filteredServices[s.cursor]
			unitName := service.Name
			if service.Type == "sync" {
				unitName += ".timer"
			} else {
				unitName += ".service"
			}
			cmds = append(cmds, s.doServiceAction(unitName, "disable"))
		}
	case "l":
		// View logs
		if len(s.filteredServices) > 0 {
			service := s.filteredServices[s.cursor]
			s.mode = ServicesModeLogs
			s.logsLoading = true
			cmds = append(cmds, s.loadServiceLogs(service.Name+".service"))
		}
	case "a":
		// Show actions menu
		if len(s.filteredServices) > 0 {
			s.selectedService = &s.filteredServices[s.cursor]
			s.showActions = true
			s.mode = ServicesModeActions
			s.actionCursor = 0
		}
	case "f":
		// Cycle through filters
		s.cycleFilter()
	case "ctrl+r", "R":
		// Refresh
		s.loading = true
		cmds = append(cmds, s.loadServices)
	case "esc":
		s.goBack = true
	}

	return cmds
}

// handleDetailsKeyPress handles key presses in details mode.
func (s *ServicesScreen) handleDetailsKeyPress(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	switch msg.String() {
	case "s":
		// Start service
		if s.selectedService != nil {
			cmds = append(cmds, s.doServiceAction(s.selectedService.Name+".service", "start"))
		}
	case "x":
		// Stop service
		if s.selectedService != nil {
			cmds = append(cmds, s.doServiceAction(s.selectedService.Name+".service", "stop"))
		}
	case "r":
		// Restart service
		if s.selectedService != nil {
			cmds = append(cmds, s.doServiceAction(s.selectedService.Name+".service", "restart"))
		}
	case "e":
		// Enable service
		if s.selectedService != nil {
			unitName := s.selectedService.Name
			if s.selectedService.Type == "sync" {
				unitName += ".timer"
			} else {
				unitName += ".service"
			}
			cmds = append(cmds, s.doServiceAction(unitName, "enable"))
		}
	case "d":
		// Disable service
		if s.selectedService != nil {
			unitName := s.selectedService.Name
			if s.selectedService.Type == "sync" {
				unitName += ".timer"
			} else {
				unitName += ".service"
			}
			cmds = append(cmds, s.doServiceAction(unitName, "disable"))
		}
	case "l":
		// View logs
		if s.selectedService != nil {
			s.logsLoading = true
			cmds = append(cmds, s.loadServiceLogs(s.selectedService.Name+".service"))
		}
	case "ctrl+r", "R":
		// Refresh
		s.loading = true
		cmds = append(cmds, s.loadServices)
	case "esc":
		// Go back to list
		s.mode = ServicesModeList
		s.detailedStatus = nil
	}

	return cmds
}

// handleLogsKeyPress handles key presses in logs mode.
func (s *ServicesScreen) handleLogsKeyPress(msg tea.KeyMsg) []tea.Cmd {
	switch msg.String() {
	case "esc":
		// Go back to details
		s.mode = ServicesModeDetails
	case "f":
		// Cycle log filter
		s.cycleLogFilter()
		// Reload logs with filter
		if s.selectedService != nil {
			s.logsLoading = true
			return []tea.Cmd{s.loadServiceLogs(s.selectedService.Name + ".service")}
		}
	}

	return nil
}

// handleActionsKeyPress handles key presses in actions menu.
func (s *ServicesScreen) handleActionsKeyPress(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	actions := []string{"Start", "Stop", "Restart", "Enable", "Disable", "View Logs", "Back"}

	switch msg.String() {
	case "up", "k":
		if s.actionCursor > 0 {
			s.actionCursor--
		}
	case "down", "j":
		if s.actionCursor < len(actions)-1 {
			s.actionCursor++
		}
	case "enter":
		if s.selectedService != nil {
			action := actions[s.actionCursor]
			switch action {
			case "Start":
				cmds = append(cmds, s.doServiceAction(s.selectedService.Name+".service", "start"))
			case "Stop":
				cmds = append(cmds, s.doServiceAction(s.selectedService.Name+".service", "stop"))
			case "Restart":
				cmds = append(cmds, s.doServiceAction(s.selectedService.Name+".service", "restart"))
			case "Enable":
				unitName := s.selectedService.Name
				if s.selectedService.Type == "sync" {
					unitName += ".timer"
				} else {
					unitName += ".service"
				}
				cmds = append(cmds, s.doServiceAction(unitName, "enable"))
			case "Disable":
				unitName := s.selectedService.Name
				if s.selectedService.Type == "sync" {
					unitName += ".timer"
				} else {
					unitName += ".service"
				}
				cmds = append(cmds, s.doServiceAction(unitName, "disable"))
			case "View Logs":
				s.logsLoading = true
				cmds = append(cmds, s.loadServiceLogs(s.selectedService.Name+".service"))
			case "Back":
				s.mode = ServicesModeList
			}
			s.showActions = false
		}
	case "esc":
		s.showActions = false
		s.mode = ServicesModeList
	}

	return cmds
}

// doServiceAction performs an action on a service.
func (s *ServicesScreen) doServiceAction(name, action string) tea.Cmd {
	return func() tea.Msg {
		// Check if manager is available
		if s.manager == nil {
			return ServiceActionResultMsg{
				Name:    name,
				Action:  action,
				Success: false,
				Error:   "systemd manager not initialized",
			}
		}

		var err error

		switch action {
		case "start":
			err = s.manager.Start(name)
		case "stop":
			err = s.manager.Stop(name)
		case "restart":
			err = s.manager.Restart(name)
		case "enable":
			err = s.manager.Enable(name)
		case "disable":
			err = s.manager.Disable(name)
		}

		if err != nil {
			return ServiceActionResultMsg{
				Name:    name,
				Action:  action,
				Success: false,
				Error:   err.Error(),
			}
		}

		return ServiceActionResultMsg{
			Name:    name,
			Action:  action,
			Success: true,
		}
	}
}

// loadServiceLogs loads logs for a service.
func (s *ServicesScreen) loadServiceLogs(name string) tea.Cmd {
	return func() tea.Msg {
		// Check if manager is available
		if s.manager == nil {
			return ServiceLogsLoadedMsg{
				Name: name,
				Logs: "Error: systemd manager not initialized",
			}
		}

		logs, err := s.manager.GetLogs(name, 50)
		if err != nil {
			return ServiceLogsLoadedMsg{
				Name: name,
				Logs: fmt.Sprintf("Error loading logs: %v", err),
			}
		}
		return ServiceLogsLoadedMsg{
			Name: name,
			Logs: logs,
		}
	}
}

// loadDetailedStatus loads detailed status for the selected service.
func (s *ServicesScreen) loadDetailedStatus() {
	if s.manager == nil || s.selectedService == nil {
		return
	}

	status, err := s.manager.GetDetailedStatus(s.selectedService.Name + ".service")
	if err == nil {
		s.detailedStatus = status
	}
}

// applyFilter applies the current filter to the services list.
func (s *ServicesScreen) applyFilter() {
	s.filteredServices = []ServiceInfo{}

	for _, service := range s.services {
		switch s.filter {
		case FilterRunning:
			if service.Status == "active" {
				s.filteredServices = append(s.filteredServices, service)
			}
		case FilterStopped:
			if service.Status == "inactive" {
				s.filteredServices = append(s.filteredServices, service)
			}
		case FilterFailed:
			if service.Status == "failed" {
				s.filteredServices = append(s.filteredServices, service)
			}
		case FilterMounts:
			if service.Type == "mount" {
				s.filteredServices = append(s.filteredServices, service)
			}
		case FilterSyncJobs:
			if service.Type == "sync" {
				s.filteredServices = append(s.filteredServices, service)
			}
		default:
			s.filteredServices = append(s.filteredServices, service)
		}
	}

	// Reset cursor if out of bounds
	if s.cursor >= len(s.filteredServices) {
		s.cursor = len(s.filteredServices) - 1
		if s.cursor < 0 {
			s.cursor = 0
		}
	}
}

// cycleFilter cycles through the available filters.
func (s *ServicesScreen) cycleFilter() {
	switch s.filter {
	case FilterAll:
		s.filter = FilterRunning
	case FilterRunning:
		s.filter = FilterStopped
	case FilterStopped:
		s.filter = FilterFailed
	case FilterFailed:
		s.filter = FilterMounts
	case FilterMounts:
		s.filter = FilterSyncJobs
	case FilterSyncJobs:
		s.filter = FilterAll
	}
	s.applyFilter()
}

// cycleLogFilter cycles through log level filters.
func (s *ServicesScreen) cycleLogFilter() {
	switch s.logFilter {
	case "all":
		s.logFilter = "error"
	case "error":
		s.logFilter = "warning"
	case "warning":
		s.logFilter = "info"
	case "info":
		s.logFilter = "debug"
	case "debug":
		s.logFilter = "all"
	}
}

// filterLogs filters the logs based on the current log filter.
func (s *ServicesScreen) filterLogs() string {
	if s.logFilter == "all" || s.logs == "" {
		return s.logs
	}

	lines := strings.Split(s.logs, "\n")
	var filtered []string

	levelKeywords := map[string][]string{
		"error":   {"ERROR", "Err", "Failed", "failure"},
		"warning": {"WARN", "Warning"},
		"info":    {"INFO", "info"},
		"debug":   {"DEBUG", "debug"},
	}

	keywords, ok := levelKeywords[s.logFilter]
	if !ok {
		return s.logs
	}

	for _, line := range lines {
		lower := strings.ToLower(line)
		for _, kw := range keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				filtered = append(filtered, line)
				break
			}
		}
	}

	return strings.Join(filtered, "\n")
}

// ShouldGoBack returns true if the screen should go back to the main menu.
func (s *ServicesScreen) ShouldGoBack() bool {
	return s.goBack
}

// ResetGoBack resets the go back state.
func (s *ServicesScreen) ResetGoBack() {
	s.goBack = false
}

// View renders the screen.
func (s *ServicesScreen) View() string {
	switch s.mode {
	case ServicesModeList:
		return s.renderListView()
	case ServicesModeDetails:
		return s.renderDetailsView()
	case ServicesModeLogs:
		return s.renderLogsView()
	case ServicesModeActions:
		return s.renderActionsView()
	default:
		return s.renderListView()
	}
}

// renderListView renders the main services list.
func (s *ServicesScreen) renderListView() string {
	var b strings.Builder

	// Title with filter indicator
	filterDesc := getFilterDescription(s.filter)
	title := fmt.Sprintf("Service Status [%s]", filterDesc)
	b.WriteString(components.Styles.Title.Render(title))
	b.WriteString("\n\n")

	// Systemd status panel
	b.WriteString(s.renderSystemdStatus())
	b.WriteString("\n")

	// Loading indicator
	if s.loading {
		b.WriteString(components.Styles.Info.Render("Loading services..."))
		b.WriteString("\n\n")
	}

	// Status message
	if s.statusMessage != "" {
		switch s.statusMessageType {
		case "success":
			b.WriteString(components.RenderSuccess(s.statusMessage))
		case "error":
			b.WriteString(components.RenderError(s.statusMessage))
		default:
			b.WriteString(components.RenderInfo(s.statusMessage))
		}
		b.WriteString("\n\n")
		s.statusMessage = "" // Clear after displaying
	}

	if len(s.filteredServices) == 0 {
		// Empty state
		emptyMsg := components.Styles.Subtitle.Render("No services match the current filter.")
		hint := components.Styles.HelpText.Render("Add mounts or sync jobs to see services here.")

		b.WriteString(lipgloss.NewStyle().
			Width(s.width).
			Align(lipgloss.Center).
			Render(emptyMsg))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().
			Width(s.width).
			Align(lipgloss.Center).
			Render(hint))
	} else {
		// Service list
		b.WriteString(s.renderServiceList())
	}

	// Help bar
	b.WriteString("\n")
	helpText := components.HelpBar(s.width, []components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "Enter", Desc: "details"},
		{Key: "s", Desc: "start"},
		{Key: "x", Desc: "stop"},
		{Key: "r", Desc: "restart"},
		{Key: "e", Desc: "enable"},
		{Key: "d", Desc: "disable"},
		{Key: "l", Desc: "logs"},
		{Key: "a", Desc: "actions"},
		{Key: "f", Desc: "filter"},
		{Key: "Ctrl+R", Desc: "refresh"},
		{Key: "Esc", Desc: "back"},
	})
	b.WriteString(helpText)

	return b.String()
}

// renderSystemdStatus renders the systemd status panel.
func (s *ServicesScreen) renderSystemdStatus() string {
	var b strings.Builder

	status := s.systemdStatus

	// Status indicators
	available := "Available"
	if !status.Available {
		available = "Unavailable"
	}

	failedUnits := fmt.Sprintf("%d", status.FailedUnits)
	if status.FailedUnits == 0 {
		failedUnits = components.Styles.Success.Render("0")
	} else {
		failedUnits = components.Styles.Error.Render(failedUnits)
	}

	activeServices := fmt.Sprintf("%d", status.ActiveServices)
	activeTimers := fmt.Sprintf("%d", status.ActiveTimers)

	// Build status line
	statusLine := fmt.Sprintf("Systemd: %s  |  Failed: %s  |  Active Services: %s  |  Active Timers: %s",
		components.Styles.Info.Render(available),
		failedUnits,
		components.Styles.Success.Render(activeServices),
		components.Styles.Success.Render(activeTimers),
	)

	b.WriteString(components.Styles.Subtitle.Render(statusLine))

	return b.String()
}

// getFilterDescription returns a human-readable description of the filter.
func getFilterDescription(filter string) string {
	switch filter {
	case FilterAll:
		return "All"
	case FilterRunning:
		return "Running"
	case FilterStopped:
		return "Stopped"
	case FilterFailed:
		return "Failed"
	case FilterMounts:
		return "Mounts"
	case FilterSyncJobs:
		return "Sync Jobs"
	default:
		return "All"
	}
}

// renderServiceList renders the list of services.
func (s *ServicesScreen) renderServiceList() string {
	var b strings.Builder

	// Calculate column widths based on screen width
	serviceWidth := 30
	typeWidth := 8
	statusWidth := 12
	enabledWidth := 8

	// Header
	header := fmt.Sprintf("  %-*s %-*s %-*s %-*s",
		serviceWidth, "Service",
		typeWidth, "Type",
		statusWidth, "Status",
		enabledWidth, "Enabled")
	b.WriteString(components.Styles.Subtitle.Render(header) + "\n")
	b.WriteString(components.Styles.Subtitle.Render(strings.Repeat("─", s.width-4)) + "\n")

	// Services
	for i, service := range s.filteredServices {
		var line string
		status := components.StatusIndicator(service.Status)
		enabled := "no"
		if service.Enabled {
			enabled = "yes"
		}

		// Format type
		typeStr := service.Type
		if service.Type == "sync" && service.TimerActive {
			typeStr = "sync (timer)"
		} else if service.Type == "sync" {
			typeStr = "sync"
		}

		if i == s.cursor {
			line = fmt.Sprintf("▸ %-*s %-*s %s %-*s %-*s",
				serviceWidth-1,
				components.Styles.Selected.Render(components.Truncate(service.DisplayName, serviceWidth-1)),
				typeWidth,
				components.Styles.Selected.Render(typeStr),
				status,
				statusWidth,
				components.Styles.Selected.Render(service.Status),
				enabledWidth,
				enabled)
		} else {
			line = fmt.Sprintf("  %-*s %-*s %s %-*s %-*s",
				serviceWidth-1,
				components.Styles.Normal.Render(components.Truncate(service.DisplayName, serviceWidth-1)),
				typeWidth,
				components.Styles.Normal.Render(typeStr),
				status,
				statusWidth,
				components.Styles.Normal.Render(service.Status),
				enabledWidth,
				components.Styles.Normal.Render(enabled))
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

// renderDetailsView renders the service details view.
func (s *ServicesScreen) renderDetailsView() string {
	var b strings.Builder

	// Title
	title := "Service Details"
	if s.selectedService != nil {
		title = fmt.Sprintf("Service Details - %s", s.selectedService.DisplayName)
	}
	b.WriteString(components.Styles.Title.Render(title))
	b.WriteString("\n\n")

	if s.selectedService == nil {
		b.WriteString(components.Styles.Error.Render("No service selected"))
		return b.String()
	}

	service := s.selectedService

	// Status indicator
	status := components.StatusIndicator(service.Status)
	statusText := fmt.Sprintf("%s %s (%s)", status, service.Status, service.SubState)
	if service.Status == "active" {
		b.WriteString(components.Styles.Success.Render(statusText))
	} else if service.Status == "failed" {
		b.WriteString(components.Styles.Error.Render(statusText))
	} else {
		b.WriteString(components.Styles.Normal.Render(statusText))
	}
	b.WriteString("\n\n")

	// Details box
	enabled := "No"
	if service.Enabled {
		enabled = "Yes"
	}

	details := ""

	if service.Type == "mount" {
		details = fmt.Sprintf(`
  Display Name: %s
  Service: %s
  Type: %s
  Status: %s
  Enabled: %s
  Mount Point: %s
  Remote: %s`,
			service.DisplayName,
			service.Name,
			service.Type,
			service.Status,
			enabled,
			service.MountPoint,
			service.Remote,
		)
	} else {
		nextRun := "Not scheduled"
		if !service.NextRun.IsZero() {
			nextRun = service.NextRun.Format("2006-01-02 15:04:05")
		}

		timerStatus := "Inactive"
		if service.TimerActive {
			timerStatus = "Active"
		}

		details = fmt.Sprintf(`
  Display Name: %s
  Service: %s
  Type: %s
  Status: %s
  Enabled: %s
  Timer: %s
  Source: %s
  Destination: %s
  Next Run: %s`,
			service.DisplayName,
			service.Name,
			service.Type,
			service.Status,
			enabled,
			timerStatus,
			service.Source,
			service.Destination,
			nextRun,
		)
	}

	box := components.Styles.Border.
		Width(s.width - 8).
		Render(details)

	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(box))

	// Action buttons hint
	b.WriteString("\n\n")
	b.WriteString(components.Styles.Subtitle.Render("Actions:"))
	b.WriteString("\n")
	b.WriteString("  [S] Start  [X] Stop  [R] Restart  [E] Enable  [D] Disable  [L] Logs  [Ctrl+R] Refresh  [Esc] Back")

	// Help bar
	b.WriteString("\n")
	helpText := components.HelpBar(s.width, []components.HelpItem{
		{Key: "s", Desc: "start"},
		{Key: "x", Desc: "stop"},
		{Key: "r", Desc: "restart"},
		{Key: "e", Desc: "enable"},
		{Key: "d", Desc: "disable"},
		{Key: "l", Desc: "logs"},
		{Key: "Ctrl+R", Desc: "refresh"},
		{Key: "Esc", Desc: "back"},
	})
	b.WriteString(helpText)

	return b.String()
}

// renderLogsView renders the logs viewer.
func (s *ServicesScreen) renderLogsView() string {
	var b strings.Builder

	// Title
	title := "Service Logs"
	if s.selectedService != nil {
		title = fmt.Sprintf("Logs - %s", s.selectedService.DisplayName)
	}
	b.WriteString(components.Styles.Title.Render(title))
	b.WriteString("\n\n")

	// Filter indicator
	b.WriteString(components.Styles.Subtitle.Render(fmt.Sprintf("Filter: %s", strings.ToUpper(s.logFilter))))
	b.WriteString("\n\n")

	if s.logsLoading {
		b.WriteString(components.Styles.Info.Render("Loading logs..."))
		return b.String()
	}

	// Apply log filter
	logs := s.filterLogs()

	// Render logs with some basic highlighting
	lines := strings.Split(logs, "\n")
	logHeight := s.height - 12

	if len(lines) > logHeight {
		lines = lines[len(lines)-logHeight:]
	}

	for _, line := range lines {
		rendered := s.renderLogLine(line)
		b.WriteString(rendered)
		b.WriteString("\n")
	}

	// Help bar
	b.WriteString("\n")
	helpText := components.HelpBar(s.width, []components.HelpItem{
		{Key: "f", Desc: "filter level"},
		{Key: "Esc", Desc: "back"},
	})
	b.WriteString(helpText)

	return b.String()
}

// renderLogLine renders a single log line with basic syntax highlighting.
func (s *ServicesScreen) renderLogLine(line string) string {
	lower := strings.ToLower(line)

	if strings.Contains(lower, "error") || strings.Contains(lower, "fail") || strings.Contains(lower, "critical") {
		return components.Styles.Error.Render(line)
	}
	if strings.Contains(lower, "warn") {
		return components.Styles.Warning.Render(line)
	}
	if strings.Contains(lower, "info") {
		return components.Styles.Info.Render(line)
	}
	if strings.Contains(lower, "debug") {
		return components.Styles.Subtitle.Render(line)
	}

	return components.Styles.Normal.Render(line)
}

// renderActionsView renders the actions menu.
func (s *ServicesScreen) renderActionsView() string {
	var b strings.Builder

	// Title
	title := "Service Actions"
	if s.selectedService != nil {
		title = fmt.Sprintf("Actions - %s", s.selectedService.DisplayName)
	}
	b.WriteString(components.Styles.Title.Render(title))
	b.WriteString("\n\n")

	actions := []string{"Start", "Stop", "Restart", "Enable", "Disable", "View Logs", "Back"}

	for i, action := range actions {
		if i == s.actionCursor {
			b.WriteString(components.Styles.MenuSelected.Render("▸ " + action))
		} else {
			b.WriteString(components.Styles.MenuItem.Render("  " + action))
		}
		b.WriteString("\n")
	}

	// Help bar
	b.WriteString("\n")
	helpText := components.HelpBar(s.width, []components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "Enter", Desc: "select"},
		{Key: "Esc", Desc: "cancel"},
	})
	b.WriteString(helpText)

	return b.String()
}
