// Package screens provides individual TUI screens for the application.
package screens

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/rclone"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
	"github.com/dtg01100/rclone-mount-sync/internal/tui/components"
)

// SyncJobsScreenMode represents the current mode of the sync jobs screen.
type SyncJobsScreenMode int

const (
	SyncJobsModeList SyncJobsScreenMode = iota
	SyncJobsModeCreate
	SyncJobsModeEdit
	SyncJobsModeDelete
	SyncJobsModeDetails
)

// SyncJobsScreen manages sync job configurations.
type SyncJobsScreen struct {
	// State
	jobs       []models.SyncJobConfig
	statuses   map[string]*models.ServiceStatus
	cursor     int
	width      int
	height     int
	mode       SyncJobsScreenMode
	goBack     bool

	// Sub-screens
	form    *SyncJobForm
	details *SyncJobDetails
	delete  *SyncJobDeleteConfirm

	// Services
	config    *config.Config
	rclone    *rclone.Client
	generator *systemd.Generator
	manager   *systemd.Manager

	// Messages
	err     error
	success string
	loading bool
}

// NewSyncJobsScreen creates a new sync jobs screen.
func NewSyncJobsScreen() *SyncJobsScreen {
	return &SyncJobsScreen{
		mode:     SyncJobsModeList,
		statuses: make(map[string]*models.ServiceStatus),
	}
}

// SetServices sets the required services for the sync jobs screen.
func (s *SyncJobsScreen) SetServices(cfg *config.Config, rcloneClient *rclone.Client, gen *systemd.Generator, mgr *systemd.Manager) {
	s.config = cfg
	s.rclone = rcloneClient
	s.generator = gen
	s.manager = mgr
}

// SetSize sets the screen dimensions.
func (s *SyncJobsScreen) SetSize(width, height int) {
	s.width = width
	s.height = height
	if s.form != nil {
		s.form.SetSize(width, height)
	}
}

// Init initializes the screen.
func (s *SyncJobsScreen) Init() tea.Cmd {
	return s.loadSyncJobs
}

// loadSyncJobs loads sync job configurations and their statuses.
func (s *SyncJobsScreen) loadSyncJobs() tea.Msg {
	if s.config == nil {
		return SyncJobsErrorMsg{Err: fmt.Errorf("config not initialized")}
	}

	// Load sync jobs from config
	s.jobs = s.config.SyncJobs

	// Load statuses for each sync job
	for _, job := range s.jobs {
		serviceName := s.generator.ServiceName(job.Name, "sync") + ".service"
		status, err := s.manager.GetDetailedStatus(serviceName)
		if err == nil {
			s.statuses[job.Name] = status
		}
	}

	return SyncJobsLoadedMsg{Jobs: s.jobs}
}

// Update handles screen updates.
func (s *SyncJobsScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle mode-specific keybindings
		switch s.mode {
		case SyncJobsModeList:
			return s.updateList(msg)
		case SyncJobsModeCreate, SyncJobsModeEdit:
			return s.updateForm(msg)
		case SyncJobsModeDelete:
			return s.updateDelete(msg)
		case SyncJobsModeDetails:
			return s.updateDetails(msg)
		}

	case SyncJobsLoadedMsg:
		s.jobs = msg.Jobs
		s.loading = false

	case SyncJobCreatedMsg:
		s.jobs = append(s.jobs, msg.Job)
		s.success = fmt.Sprintf("Sync job '%s' created successfully", msg.Job.Name)
		s.mode = SyncJobsModeList
		s.err = nil

	case SyncJobUpdatedMsg:
		// Update the job in the list
		for i, j := range s.jobs {
			if j.ID == msg.Job.ID {
				s.jobs[i] = msg.Job
				break
			}
		}
		s.success = fmt.Sprintf("Sync job '%s' updated successfully", msg.Job.Name)
		s.mode = SyncJobsModeList
		s.err = nil

	case SyncJobDeletedMsg:
		// Remove the job from the list
		for i, j := range s.jobs {
			if j.Name == msg.Name {
				s.jobs = append(s.jobs[:i], s.jobs[i+1:]...)
				break
			}
		}
		s.success = fmt.Sprintf("Sync job '%s' deleted successfully", msg.Name)
		s.mode = SyncJobsModeList
		s.cursor = 0
		s.err = nil

	case SyncJobStatusMsg:
		s.statuses[msg.Name] = msg.Status

	case SyncJobsErrorMsg:
		s.err = msg.Err
		s.loading = false

	case SyncJobFormCancelMsg:
		s.mode = SyncJobsModeList
		s.form = nil
		s.err = nil
	}

	return s, tea.Batch(cmds...)
}

// updateList handles updates when in list mode.
func (s *SyncJobsScreen) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		if s.cursor < len(s.jobs)-1 {
			s.cursor++
		}
	case "a", "n":
		// Add new sync job
		return s.startCreateForm()
	case "e":
		// Edit selected sync job
		if len(s.jobs) > 0 && s.cursor < len(s.jobs) {
			return s.startEditForm()
		}
	case "d":
		// Delete selected sync job
		if len(s.jobs) > 0 && s.cursor < len(s.jobs) {
			s.mode = SyncJobsModeDelete
			s.delete = NewSyncJobDeleteConfirm(s.jobs[s.cursor])
			if s.config != nil {
				s.delete.SetServices(s.manager, s.generator, s.config)
			}
		}
	case "enter":
		// View details
		if len(s.jobs) > 0 && s.cursor < len(s.jobs) {
			s.mode = SyncJobsModeDetails
			s.details = NewSyncJobDetails(s.jobs[s.cursor], s.manager, s.generator)
		}
	case "r":
		// Run sync job now
		if len(s.jobs) > 0 && s.cursor < len(s.jobs) {
			return s.runSyncJobNow()
		}
	case "t":
		// Toggle timer
		if len(s.jobs) > 0 && s.cursor < len(s.jobs) {
			return s.toggleTimer()
		}
	case "R":
		// Refresh status
		return s, s.loadSyncJobs
	case "esc":
		s.goBack = true
	}

	return s, nil
}

// updateForm handles updates when in form mode.
func (s *SyncJobsScreen) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if s.form == nil {
		s.mode = SyncJobsModeList
		return s, nil
	}

	model, cmd := s.form.Update(msg)
	if f, ok := model.(*SyncJobForm); ok {
		s.form = f
	}

	// Check if form is done
	if s.form.IsDone() {
		s.mode = SyncJobsModeList
		s.form = nil
	}

	return s, cmd
}

// updateDelete handles updates when in delete mode.
func (s *SyncJobsScreen) updateDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if s.delete == nil {
		s.mode = SyncJobsModeList
		return s, nil
	}

	model, cmd := s.delete.Update(msg)
	if d, ok := model.(*SyncJobDeleteConfirm); ok {
		s.delete = d
	}

	// Check if delete is done
	if s.delete.IsDone() {
		s.mode = SyncJobsModeList
		s.delete = nil
	}

	return s, cmd
}

// updateDetails handles updates when in details mode.
func (s *SyncJobsScreen) updateDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if s.details == nil {
		s.mode = SyncJobsModeList
		return s, nil
	}

	model, cmd := s.details.Update(msg)
	if d, ok := model.(*SyncJobDetails); ok {
		s.details = d
	}

	// Check if details view is done
	if s.details.IsDone() {
		s.mode = SyncJobsModeList
		s.details = nil
	}

	return s, cmd
}

// startCreateForm starts the create sync job form.
func (s *SyncJobsScreen) startCreateForm() (tea.Model, tea.Cmd) {
	// Get available remotes
	remotes, err := s.rclone.ListRemotes()
	if err != nil {
		s.err = fmt.Errorf("failed to list remotes: %w", err)
		return s, nil
	}

	s.form = NewSyncJobForm(nil, remotes, s.config, s.generator, s.manager, false)
	s.mode = SyncJobsModeCreate
	s.err = nil
	return s, s.form.Init()
}

// startEditForm starts the edit sync job form.
func (s *SyncJobsScreen) startEditForm() (tea.Model, tea.Cmd) {
	job := s.jobs[s.cursor]

	// Stop timer if running before editing
	timerName := s.generator.ServiceName(job.Name, "sync") + ".timer"
	_ = s.manager.StopTimer(timerName)
	_ = s.manager.DisableTimer(timerName)

	// Get available remotes
	remotes, err := s.rclone.ListRemotes()
	if err != nil {
		s.err = fmt.Errorf("failed to list remotes: %w", err)
		return s, nil
	}

	s.form = NewSyncJobForm(&job, remotes, s.config, s.generator, s.manager, true)
	s.mode = SyncJobsModeEdit
	s.err = nil
	return s, s.form.Init()
}

// runSyncJobNow runs the selected sync job immediately.
func (s *SyncJobsScreen) runSyncJobNow() (tea.Model, tea.Cmd) {
	job := s.jobs[s.cursor]
	serviceName := s.generator.ServiceName(job.Name, "sync") + ".service"

	return s, func() tea.Msg {
		if err := s.manager.RunSyncNow(serviceName); err != nil {
			return SyncJobsErrorMsg{Err: fmt.Errorf("failed to run sync job: %w", err)}
		}
		return SyncJobRunNowMsg{Name: job.Name}
	}
}

// toggleTimer toggles the sync job timer on/off.
func (s *SyncJobsScreen) toggleTimer() (tea.Model, tea.Cmd) {
	job := s.jobs[s.cursor]
	timerName := s.generator.ServiceName(job.Name, "sync") + ".timer"

	// Check if timer is currently active
	isActive, _ := s.manager.IsActive(timerName)

	if isActive {
		// Stop and disable timer
		_ = s.manager.StopTimer(timerName)
		_ = s.manager.DisableTimer(timerName)
	} else {
		// Enable and start timer
		_ = s.manager.EnableTimer(timerName)
		_ = s.manager.StartTimer(timerName)
	}

	// Refresh status
	return s, s.loadSyncJobs
}

// ShouldGoBack returns true if the screen should go back to the main menu.
func (s *SyncJobsScreen) ShouldGoBack() bool {
	return s.goBack
}

// ResetGoBack resets the go back state.
func (s *SyncJobsScreen) ResetGoBack() {
	s.goBack = false
}

// View renders the screen.
func (s *SyncJobsScreen) View() string {
	switch s.mode {
	case SyncJobsModeCreate, SyncJobsModeEdit:
		if s.form != nil {
			return s.form.View()
		}
	case SyncJobsModeDelete:
		if s.delete != nil {
			return s.delete.View()
		}
	case SyncJobsModeDetails:
		if s.details != nil {
			return s.details.View()
		}
	}

	return s.renderList()
}

// renderList renders the sync job list view.
func (s *SyncJobsScreen) renderList() string {
	var b strings.Builder

	// Title
	title := components.Styles.Title.Render("Sync Job Management")
	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	// Show error if any
	if s.err != nil {
		b.WriteString(components.RenderError(s.err.Error()))
		b.WriteString("\n\n")
	}

	// Show success message if any
	if s.success != "" {
		b.WriteString(components.RenderSuccess(s.success))
		b.WriteString("\n\n")
		s.success = ""
	}

	if s.loading {
		b.WriteString(lipgloss.NewStyle().
			Width(s.width).
			Align(lipgloss.Center).
			Render("Loading sync jobs..."))
	} else if len(s.jobs) == 0 {
		// Empty state
		emptyMsg := components.Styles.Subtitle.Render("No sync jobs configured.")
		addHint := components.Styles.HelpText.Render("Press 'a' or 'n' to add a new sync job.")

		b.WriteString(lipgloss.NewStyle().
			Width(s.width).
			Align(lipgloss.Center).
			Render(emptyMsg))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().
			Width(s.width).
			Align(lipgloss.Center).
			Render(addHint))
	} else {
		// Sync job list
		b.WriteString(s.renderJobList())
		b.WriteString("\n")

		// Selected item details
		if s.cursor >= 0 && s.cursor < len(s.jobs) {
			b.WriteString(s.renderJobDetails())
		}
	}

	// Help bar
	b.WriteString("\n")
	helpText := components.HelpBar(s.width, []components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "a/n", Desc: "add"},
		{Key: "e", Desc: "edit"},
		{Key: "d", Desc: "delete"},
		{Key: "r", Desc: "run now"},
		{Key: "t", Desc: "toggle timer"},
		{Key: "Enter", Desc: "details"},
		{Key: "R", Desc: "refresh"},
		{Key: "Esc", Desc: "back"},
	})
	b.WriteString(helpText)

	return b.String()
}

// renderJobList renders the list of sync jobs.
func (s *SyncJobsScreen) renderJobList() string {
	var b strings.Builder

	// Header
	header := fmt.Sprintf("  %-20s %-25s %-15s %-12s",
		"Name", "Source → Destination", "Schedule", "Status")
	b.WriteString(components.Styles.Subtitle.Render(header) + "\n")
	b.WriteString(components.Styles.Subtitle.Render(strings.Repeat("─", s.width-4)) + "\n")

	// Jobs
	for i, job := range s.jobs {
		var line string
		status := s.getJobStatus(&job)

		source := job.Source
		if len(source) > 25 {
			source = source[:22] + "..."
		}

		dest := job.Destination
		if len(dest) > 25 {
			dest = dest[:22] + "..."
		}

		sourceDest := source + " → " + dest
		schedule := getScheduleDisplay(&job)

		if i == s.cursor {
			line = fmt.Sprintf("▸ %-20s %-25s %-15s %s",
				components.Styles.Selected.Render(job.Name),
				components.Styles.Normal.Render(sourceDest),
				components.Styles.Normal.Render(schedule),
				status)
		} else {
			line = fmt.Sprintf("  %-20s %-25s %-15s %s",
				components.Styles.Normal.Render(job.Name),
				components.Styles.Normal.Render(sourceDest),
				components.Styles.Normal.Render(schedule),
				status)
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

// getJobStatus returns a formatted status string for a sync job.
func (s *SyncJobsScreen) getJobStatus(job *models.SyncJobConfig) string {
	status, ok := s.statuses[job.Name]
	if !ok {
		return components.StatusIndicator("unknown") + " unknown"
	}

	if status.TimerActive {
		return components.StatusIndicator("active") + " " + components.Styles.Success.Render("scheduled")
	}
	if status.ActiveState == "active" {
		return components.StatusIndicator("active") + " " + components.Styles.Success.Render("running")
	}
	if status.ActiveState == "failed" {
		return components.StatusIndicator("failed") + " " + components.Styles.Error.Render("failed")
	}
	return components.StatusIndicator("inactive") + " " + components.Styles.StatusInactive.Render("inactive")
}

// getScheduleDisplay returns a human-readable schedule string.
func getScheduleDisplay(job *models.SyncJobConfig) string {
	switch job.Schedule.Type {
	case "manual":
		return "Manual"
	case "timer":
		if job.Schedule.OnCalendar != "" {
			return job.Schedule.OnCalendar
		}
		return "Timer"
	case "onboot":
		if job.Schedule.OnBootSec != "" {
			return "On Boot: " + job.Schedule.OnBootSec
		}
		return "On Boot"
	default:
		return "Manual"
	}
}

// renderJobDetails renders the details of the selected sync job.
func (s *SyncJobsScreen) renderJobDetails() string {
	job := s.jobs[s.cursor]

	var b strings.Builder
	b.WriteString("\n")

	// Get status info
	statusStr := "unknown"
	if status, ok := s.statuses[job.Name]; ok {
		if status.TimerActive {
			statusStr = "scheduled"
		} else if status.ActiveState == "active" {
			statusStr = "running"
		} else if status.ActiveState == "failed" {
			statusStr = "failed"
		} else {
			statusStr = "inactive"
		}
	}

	schedule := getScheduleDisplay(&job)

	// Details box
	details := fmt.Sprintf(
		"  Selected: %s\n\n  Source: %s\n  Destination: %s\n  Schedule: %s\n  Status: %s\n  Enabled: %t\n\n  [E] Edit  [D] Delete  [R] Run Now  [T] Toggle Timer  [Enter] Details",
		components.Styles.Selected.Render(job.Name),
		job.Source,
		job.Destination,
		schedule,
		statusStr,
		job.Enabled,
	)

	box := components.Styles.Border.
		Width(s.width - 8).
		Render(details)

	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(box))

	// Show next run time if timer is active
	if status, ok := s.statuses[job.Name]; ok && status != nil {
		if status.TimerActive && !status.NextRun.IsZero() {
			nextRun := status.NextRun.Format("2006-01-02 15:04:05")
			b.WriteString("\n")
			b.WriteString(components.Styles.Info.Render("  Next run: " + nextRun))
		}
	}

	return b.String()
}

// Messages for sync job operations

// SyncJobsLoadedMsg is sent when sync jobs are loaded.
type SyncJobsLoadedMsg struct {
	Jobs []models.SyncJobConfig
}

// SyncJobCreatedMsg is sent when a sync job is created.
type SyncJobCreatedMsg struct {
	Job models.SyncJobConfig
}

// SyncJobUpdatedMsg is sent when a sync job is updated.
type SyncJobUpdatedMsg struct {
	Job models.SyncJobConfig
}

// SyncJobDeletedMsg is sent when a sync job is deleted.
type SyncJobDeletedMsg struct {
	Name string
}

// SyncJobStatusMsg is sent when sync job status is updated.
type SyncJobStatusMsg struct {
	Name   string
	Status *models.ServiceStatus
}

// SyncJobRunNowMsg is sent when a sync job is run.
type SyncJobRunNowMsg struct {
	Name string
}

// SyncJobsErrorMsg is sent when an error occurs.
type SyncJobsErrorMsg struct {
	Err error
}

// SyncJobFormCancelMsg is sent when the form is cancelled.
type SyncJobFormCancelMsg struct{}

// SyncJobDetails displays detailed sync job information.
type SyncJobDetails struct {
	job       models.SyncJobConfig
	status    *models.ServiceStatus
	timerNext string
	logs      string
	manager   *systemd.Manager
	generator *systemd.Generator
	done      bool
	width     int
	height    int
	tab       int // 0: details, 1: logs
}

// NewSyncJobDetails creates a new sync job details view.
func NewSyncJobDetails(job models.SyncJobConfig, manager *systemd.Manager, generator *systemd.Generator) *SyncJobDetails {
	d := &SyncJobDetails{
		job:       job,
		manager:   manager,
		generator: generator,
		tab:       0,
	}
	d.loadStatus()
	d.loadLogs()
	return d
}

// loadStatus loads the service and timer status.
func (d *SyncJobDetails) loadStatus() {
	serviceName := d.generator.ServiceName(d.job.Name, "sync") + ".service"
	status, err := d.manager.GetDetailedStatus(serviceName)
	if err == nil {
		d.status = status
		if !status.NextRun.IsZero() {
			d.timerNext = status.NextRun.Format("2006-01-02 15:04:05")
		}
	}
}

// loadLogs loads the service logs.
func (d *SyncJobDetails) loadLogs() {
	serviceName := d.generator.ServiceName(d.job.Name, "sync") + ".service"
	logs, err := d.manager.GetLogs(serviceName, 30)
	if err == nil {
		d.logs = logs
	} else {
		d.logs = "Failed to load logs"
	}
}

// SetSize sets the size.
func (d *SyncJobDetails) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Init initializes the view.
func (d *SyncJobDetails) Init() tea.Cmd {
	return nil
}

// Update handles updates.
func (d *SyncJobDetails) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			d.done = true
		case "tab":
			d.tab = (d.tab + 1) % 2
		case "r":
			// Run sync job now
			serviceName := d.generator.ServiceName(d.job.Name, "sync") + ".service"
			_ = d.manager.RunSyncNow(serviceName)
			d.loadStatus()
			d.loadLogs()
		case "t":
			// Toggle timer
			timerName := d.generator.ServiceName(d.job.Name, "sync") + ".timer"
			isActive, _ := d.manager.IsActive(timerName)
			if isActive {
				_ = d.manager.StopTimer(timerName)
				_ = d.manager.DisableTimer(timerName)
			} else {
				_ = d.manager.EnableTimer(timerName)
				_ = d.manager.StartTimer(timerName)
			}
			d.loadStatus()
		case "e":
			// Enable timer
			timerName := d.generator.ServiceName(d.job.Name, "sync") + ".timer"
			_ = d.manager.EnableTimer(timerName)
			_ = d.manager.StartTimer(timerName)
			d.loadStatus()
		case "d":
			// Disable timer
			timerName := d.generator.ServiceName(d.job.Name, "sync") + ".timer"
			_ = d.manager.StopTimer(timerName)
			_ = d.manager.DisableTimer(timerName)
			d.loadStatus()
		case "R":
			// Refresh
			d.loadStatus()
			d.loadLogs()
		}
	}

	return d, nil
}

// IsDone returns true if the view is done.
func (d *SyncJobDetails) IsDone() bool {
	return d.done
}

// View renders the view.
func (d *SyncJobDetails) View() string {
	var b strings.Builder

	// Title
	title := components.Styles.Title.Render(fmt.Sprintf("Sync Job: %s", d.job.Name))
	b.WriteString(lipgloss.NewStyle().
		Width(d.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	// Tabs
	tabs := []string{"Details", "Logs"}
	var tabStrs []string
	for i, tab := range tabs {
		if i == d.tab {
			tabStrs = append(tabStrs, components.Styles.Selected.Render("["+tab+"]"))
		} else {
			tabStrs = append(tabStrs, components.Styles.Normal.Render("["+tab+"]"))
		}
	}
	b.WriteString(lipgloss.NewStyle().
		Width(d.width).
		Align(lipgloss.Center).
		Render(strings.Join(tabStrs, "  ")))
	b.WriteString("\n\n")

	// Content based on tab
	if d.tab == 0 {
		b.WriteString(d.renderDetails())
	} else {
		b.WriteString(d.renderLogs())
	}

	// Help
	b.WriteString("\n")
	help := components.HelpBar(d.width, []components.HelpItem{
		{Key: "Tab", Desc: "switch tab"},
		{Key: "r", Desc: "run now"},
		{Key: "t", Desc: "toggle timer"},
		{Key: "e", Desc: "enable timer"},
		{Key: "d", Desc: "disable timer"},
		{Key: "R", Desc: "refresh"},
		{Key: "Esc", Desc: "back"},
	})
	b.WriteString(help)

	return b.String()
}

// renderDetails renders the details tab.
func (d *SyncJobDetails) renderDetails() string {
	var b strings.Builder

	// Sync job info
	b.WriteString(fmt.Sprintf("  Name: %s\n", d.job.Name))
	b.WriteString(fmt.Sprintf("  Source: %s\n", d.job.Source))
	b.WriteString(fmt.Sprintf("  Destination: %s\n", d.job.Destination))
	b.WriteString(fmt.Sprintf("  Schedule: %s\n", getScheduleDisplay(&d.job)))

	// Schedule details
	if d.job.Schedule.Type == "timer" && d.job.Schedule.OnCalendar != "" {
		b.WriteString(fmt.Sprintf("  Calendar: %s\n", d.job.Schedule.OnCalendar))
	}
	if d.job.Schedule.Type == "onboot" && d.job.Schedule.OnBootSec != "" {
		b.WriteString(fmt.Sprintf("  Boot Delay: %s\n", d.job.Schedule.OnBootSec))
	}

	b.WriteString(fmt.Sprintf("  Enabled: %t\n", d.job.Enabled))

	// Status
	if d.status != nil {
		b.WriteString("\n  Service Status:\n")
		b.WriteString(fmt.Sprintf("    State: %s\n", d.status.ActiveState))
		b.WriteString(fmt.Sprintf("    SubState: %s\n", d.status.SubState))
		b.WriteString(fmt.Sprintf("    Timer Active: %t\n", d.status.TimerActive))

		if d.timerNext != "" {
			b.WriteString(fmt.Sprintf("    Next Run: %s\n", d.timerNext))
		}

		if !d.status.LastRun.IsZero() {
			b.WriteString(fmt.Sprintf("    Last Run: %s\n", d.status.LastRun.Format("2006-01-02 15:04:05")))
		}
	}

	// Sync options
	b.WriteString("\n  Sync Options:\n")
	if d.job.SyncOptions.Direction != "" {
		b.WriteString(fmt.Sprintf("    Direction: %s\n", d.job.SyncOptions.Direction))
	}
	if d.job.SyncOptions.DryRun {
		b.WriteString("    Dry Run: true\n")
	}
	if d.job.SyncOptions.BandwidthLimit != "" {
		b.WriteString(fmt.Sprintf("    Bandwidth Limit: %s\n", d.job.SyncOptions.BandwidthLimit))
	}
	if d.job.SyncOptions.Transfers > 0 {
		b.WriteString(fmt.Sprintf("    Max Transfers: %d\n", d.job.SyncOptions.Transfers))
	}

	return b.String()
}

// renderLogs renders the logs tab.
func (d *SyncJobDetails) renderLogs() string {
	if d.logs == "" {
		return components.Styles.Subtitle.Render("  No logs available")
	}

	// Truncate logs if too long
	lines := strings.Split(d.logs, "\n")
	if len(lines) > 20 {
		lines = lines[:20]
	}

	return components.Styles.Normal.Render(strings.Join(lines, "\n"))
}

// SyncJobDeleteConfirm handles the delete confirmation dialog.
type SyncJobDeleteConfirm struct {
	job        models.SyncJobConfig
	cursor     int
	done       bool
	deleteType int // 0: cancel, 1: service only, 2: service and config
	manager    *systemd.Manager
	generator  *systemd.Generator
	config     *config.Config
	width      int
}

// NewSyncJobDeleteConfirm creates a new delete confirmation dialog.
func NewSyncJobDeleteConfirm(job models.SyncJobConfig) *SyncJobDeleteConfirm {
	return &SyncJobDeleteConfirm{
		job:        job,
		cursor:     0,
		deleteType: 0,
	}
}

// SetServices sets the services for the delete confirmation.
func (d *SyncJobDeleteConfirm) SetServices(mgr *systemd.Manager, gen *systemd.Generator, cfg *config.Config) {
	d.manager = mgr
	d.generator = gen
	d.config = cfg
}

// SetSize sets the size.
func (d *SyncJobDeleteConfirm) SetSize(width, height int) {
	d.width = width
}

// Init initializes the dialog.
func (d *SyncJobDeleteConfirm) Init() tea.Cmd {
	return nil
}

// Update handles updates.
func (d *SyncJobDeleteConfirm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if d.cursor > 0 {
				d.cursor--
			}
		case "right", "l":
			if d.cursor < 2 {
				d.cursor++
			}
		case "enter":
			return d.confirmDelete()
		case "esc":
			d.done = true
		}
	}

	return d, nil
}

// confirmDelete performs the delete action.
func (d *SyncJobDeleteConfirm) confirmDelete() (tea.Model, tea.Cmd) {
	switch d.cursor {
	case 0:
		// Cancel
		d.done = true
		return d, nil
	case 1:
		// Delete service only
		return d, d.deleteServiceOnly()
	case 2:
		// Delete service and config
		return d, d.deleteServiceAndConfig()
	}
	return d, nil
}

// deleteServiceOnly deletes only the systemd service and timer.
func (d *SyncJobDeleteConfirm) deleteServiceOnly() tea.Cmd {
	return func() tea.Msg {
		serviceName := d.generator.ServiceName(d.job.Name, "sync") + ".service"
		timerName := d.generator.ServiceName(d.job.Name, "sync") + ".timer"

		// Stop the service if running
		_ = d.manager.Stop(serviceName)

		// Stop and disable the timer if running
		_ = d.manager.StopTimer(timerName)
		_ = d.manager.DisableTimer(timerName)

		// Disable the service
		_ = d.manager.Disable(serviceName)

		// Remove the unit files
		_ = d.generator.RemoveUnit(serviceName + ".service")
		_ = d.generator.RemoveUnit(timerName + ".timer")

		// Reload daemon
		_ = d.manager.DaemonReload()

		return SyncJobDeletedMsg{Name: d.job.Name}
	}
}

// deleteServiceAndConfig deletes both the service and config entry.
func (d *SyncJobDeleteConfirm) deleteServiceAndConfig() tea.Cmd {
	return func() tea.Msg {
		serviceName := d.generator.ServiceName(d.job.Name, "sync") + ".service"
		timerName := d.generator.ServiceName(d.job.Name, "sync") + ".timer"

		// Stop the service if running
		_ = d.manager.Stop(serviceName)

		// Stop and disable the timer if running
		_ = d.manager.StopTimer(timerName)
		_ = d.manager.DisableTimer(timerName)

		// Disable the service
		_ = d.manager.Disable(serviceName)

		// Remove the unit files
		_ = d.generator.RemoveUnit(serviceName + ".service")
		_ = d.generator.RemoveUnit(timerName + ".timer")

		// Reload daemon
		_ = d.manager.DaemonReload()

		// Remove from config
		if d.config != nil {
			if err := d.config.RemoveSyncJob(d.job.Name); err == nil {
				_ = d.config.Save()
			}
		}

		return SyncJobDeletedMsg{Name: d.job.Name}
	}
}

// IsDone returns true if the dialog is done.
func (d *SyncJobDeleteConfirm) IsDone() bool {
	return d.done
}

// View renders the dialog.
func (d *SyncJobDeleteConfirm) View() string {
	var b strings.Builder

	// Title
	title := components.Styles.Title.Render("Delete Sync Job")
	b.WriteString(lipgloss.NewStyle().
		Width(d.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	// Warning message
	warning := fmt.Sprintf("Are you sure you want to delete '%s'?", d.job.Name)
	b.WriteString(lipgloss.NewStyle().
		Width(d.width).
		Align(lipgloss.Center).
		Render(components.RenderWarning(warning)))
	b.WriteString("\n\n")

	// Options
	options := []string{"Cancel", "Delete Service Only", "Delete Service and Config"}
	var optionStrs []string
	for i, opt := range options {
		if i == d.cursor {
			optionStrs = append(optionStrs, components.Styles.ButtonFocus.Render(opt))
		} else {
			optionStrs = append(optionStrs, components.Styles.Button.Render(opt))
		}
	}

	optionsLine := strings.Join(optionStrs, "  ")
	b.WriteString(lipgloss.NewStyle().
		Width(d.width).
		Align(lipgloss.Center).
		Render(optionsLine))
	b.WriteString("\n\n")

	// Help
	help := components.Styles.HelpText.Render("←/→: select option  Enter: confirm  Esc: cancel")
	b.WriteString(lipgloss.NewStyle().
		Width(d.width).
		Align(lipgloss.Center).
		Render(help))

	return b.String()
}

// Helper function to get current time
func syncJobNow() time.Time {
	return time.Now()
}
