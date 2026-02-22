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

// MountsScreenMode represents the current mode of the mounts screen.
type MountsScreenMode int

const (
	MountsModeList MountsScreenMode = iota
	MountsModeCreate
	MountsModeEdit
	MountsModeDelete
	MountsModeDetails
)

// MountsScreen manages mount configurations.
type MountsScreen struct {
	// State
	mounts   []models.MountConfig
	statuses map[string]*systemd.ServiceStatus
	cursor   int
	width    int
	height   int
	mode     MountsScreenMode
	goBack   bool

	// Sub-screens
	form    *MountForm
	details *MountDetails
	delete  *DeleteConfirm

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

// NewMountsScreen creates a new mounts screen.
func NewMountsScreen() *MountsScreen {
	return &MountsScreen{
		mode:     MountsModeList,
		statuses: make(map[string]*systemd.ServiceStatus),
	}
}

// SetServices sets the required services for the mounts screen.
func (s *MountsScreen) SetServices(cfg *config.Config, rcloneClient *rclone.Client, gen *systemd.Generator, mgr *systemd.Manager) {
	s.config = cfg
	s.rclone = rcloneClient
	s.generator = gen
	s.manager = mgr
}

// SetSize sets the screen dimensions.
func (s *MountsScreen) SetSize(width, height int) {
	s.width = width
	s.height = height
	if s.form != nil {
		s.form.SetSize(width, height)
	}
}

// Init initializes the screen.
func (s *MountsScreen) Init() tea.Cmd {
	return s.loadMounts
}

// loadMounts loads mount configurations and their statuses.
func (s *MountsScreen) loadMounts() tea.Msg {
	if s.config == nil {
		return MountsErrorMsg{Err: fmt.Errorf("config not initialized")}
	}

	// Load mounts from config
	s.mounts = s.config.Mounts

	// Load statuses for each mount (only if generator and manager are available)
	if s.generator != nil && s.manager != nil {
		for _, mount := range s.mounts {
			serviceName := s.generator.ServiceName(mount.ID, "mount") + ".service"
			status, err := s.manager.Status(serviceName)
			if err == nil {
				s.statuses[mount.Name] = status
			}
		}
	}

	return MountsLoadedMsg{Mounts: s.mounts}
}

// Update handles screen updates.
func (s *MountsScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle mode-specific keybindings
		switch s.mode {
		case MountsModeList:
			return s.updateList(msg)
		case MountsModeCreate, MountsModeEdit:
			return s.updateForm(msg)
		case MountsModeDelete:
			return s.updateDelete(msg)
		case MountsModeDetails:
			return s.updateDetails(msg)
		}

	case MountsLoadedMsg:
		s.mounts = msg.Mounts
		s.loading = false

	case MountCreatedMsg:
		s.mounts = append(s.mounts, msg.Mount)
		s.success = fmt.Sprintf("Mount '%s' created successfully", msg.Mount.Name)
		s.mode = MountsModeList
		s.err = nil

	case MountUpdatedMsg:
		// Update the mount in the list
		for i, m := range s.mounts {
			if m.ID == msg.Mount.ID {
				s.mounts[i] = msg.Mount
				break
			}
		}
		s.success = fmt.Sprintf("Mount '%s' updated successfully", msg.Mount.Name)
		s.mode = MountsModeList
		s.err = nil

	case MountDeletedMsg:
		// Remove the mount from the list
		for i, m := range s.mounts {
			if m.Name == msg.Name {
				s.mounts = append(s.mounts[:i], s.mounts[i+1:]...)
				break
			}
		}
		s.success = fmt.Sprintf("Mount '%s' deleted successfully", msg.Name)
		s.mode = MountsModeList
		s.cursor = 0
		s.err = nil

	case MountStatusMsg:
		s.statuses[msg.Name] = msg.Status

	case MountsErrorMsg:
		s.err = msg.Err
		s.loading = false

	case MountFormCancelMsg:
		s.mode = MountsModeList
		s.form = nil
		s.err = nil

	case MountFormSubmitMsg:
		// Form submitted, handled by form
	}

	return s, tea.Batch(cmds...)
}

// updateList handles updates when in list mode.
func (s *MountsScreen) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		if s.cursor < len(s.mounts)-1 {
			s.cursor++
		}
	case "a":
		// Add new mount
		return s.startCreateForm()
	case "e":
		// Edit selected mount
		if len(s.mounts) > 0 && s.cursor < len(s.mounts) {
			return s.startEditForm()
		}
	case "d":
		// Delete selected mount
		if len(s.mounts) > 0 && s.cursor < len(s.mounts) {
			s.mode = MountsModeDelete
			s.delete = NewDeleteConfirm(s.mounts[s.cursor])
		}
	case "enter":
		// View details
		if len(s.mounts) > 0 && s.cursor < len(s.mounts) {
			s.mode = MountsModeDetails
			s.details = NewMountDetails(s.mounts[s.cursor], s.manager, s.generator)
		}
	case "t":
		// Toggle mount service
		if len(s.mounts) > 0 && s.cursor < len(s.mounts) {
			return s.toggleMount()
		}
	case "s":
		// Start mount
		if len(s.mounts) > 0 && s.cursor < len(s.mounts) {
			return s.startMount()
		}
	case "x":
		// Stop mount
		if len(s.mounts) > 0 && s.cursor < len(s.mounts) {
			return s.stopMount()
		}
	case "r":
		// Refresh status
		return s, s.loadMounts
	case "esc":
		s.goBack = true
	}

	return s, nil
}

// updateForm handles updates when in form mode.
func (s *MountsScreen) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if s.form == nil {
		s.mode = MountsModeList
		return s, nil
	}

	model, cmd := s.form.Update(msg)
	if f, ok := model.(*MountForm); ok {
		s.form = f
	}

	// Check if form is done
	if s.form.IsDone() {
		s.mode = MountsModeList
		s.form = nil
	}

	return s, cmd
}

// updateDelete handles updates when in delete mode.
func (s *MountsScreen) updateDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if s.delete == nil {
		s.mode = MountsModeList
		return s, nil
	}

	model, cmd := s.delete.Update(msg)
	if d, ok := model.(*DeleteConfirm); ok {
		s.delete = d
	}

	// Check if delete is done
	if s.delete.IsDone() {
		s.mode = MountsModeList
		s.delete = nil
	}

	return s, cmd
}

// updateDetails handles updates when in details mode.
func (s *MountsScreen) updateDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if s.details == nil {
		s.mode = MountsModeList
		return s, nil
	}

	model, cmd := s.details.Update(msg)
	if d, ok := model.(*MountDetails); ok {
		s.details = d
	}

	// Check if details view is done
	if s.details.IsDone() {
		s.mode = MountsModeList
		s.details = nil
	}

	return s, cmd
}

// startCreateForm starts the create mount form.
func (s *MountsScreen) startCreateForm() (tea.Model, tea.Cmd) {
	// Check if rclone client is available
	if s.rclone == nil {
		s.err = fmt.Errorf("rclone client not initialized - please ensure rclone is installed")
		return s, nil
	}

	// Check if rclone is installed
	if !s.rclone.IsInstalled() {
		s.err = fmt.Errorf("rclone binary not found - please install rclone first")
		return s, nil
	}

	// Get available remotes
	remotes, err := s.rclone.ListRemotes()
	if err != nil {
		s.err = fmt.Errorf("failed to list remotes: %w", err)
		return s, nil
	}

	// Check if any remotes are configured
	if len(remotes) == 0 {
		s.err = fmt.Errorf("no rclone remotes configured - run 'rclone config' to set up a remote")
		return s, nil
	}

	s.form = NewMountForm(nil, remotes, s.config, s.generator, s.manager, s.rclone, false)
	s.mode = MountsModeCreate
	s.err = nil
	return s, s.form.Init()
}

// startEditForm starts the edit mount form.
func (s *MountsScreen) startEditForm() (tea.Model, tea.Cmd) {
	mount := s.mounts[s.cursor]

	// Check if rclone client is available
	if s.rclone == nil {
		s.err = fmt.Errorf("rclone client not initialized - please ensure rclone is installed")
		return s, nil
	}

	// Check if rclone is installed
	if !s.rclone.IsInstalled() {
		s.err = fmt.Errorf("rclone binary not found - please install rclone first")
		return s, nil
	}

	// Get available remotes
	remotes, err := s.rclone.ListRemotes()
	if err != nil {
		s.err = fmt.Errorf("failed to list remotes: %w", err)
		return s, nil
	}

	// Check if any remotes are configured
	if len(remotes) == 0 {
		s.err = fmt.Errorf("no rclone remotes configured - run 'rclone config' to set up a remote")
		return s, nil
	}

	s.form = NewMountForm(&mount, remotes, s.config, s.generator, s.manager, s.rclone, true)
	s.mode = MountsModeEdit
	s.err = nil
	return s, s.form.Init()
}

// toggleMount toggles the mount service on/off.
func (s *MountsScreen) toggleMount() (tea.Model, tea.Cmd) {
	// Check if generator and manager are available
	if s.generator == nil || s.manager == nil {
		s.err = fmt.Errorf("systemd services not initialized")
		return s, nil
	}

	mount := s.mounts[s.cursor]
	serviceName := s.generator.ServiceName(mount.ID, "mount") + ".service"

	// Check current status
	status, err := s.manager.Status(serviceName)
	if err != nil {
		s.err = fmt.Errorf("failed to get status: %w", err)
		return s, nil
	}

	if status.Active {
		// Stop and disable
		return s, tea.Sequence(
			func() tea.Msg {
				if err := s.manager.Stop(serviceName); err != nil {
					return MountsErrorMsg{Err: fmt.Errorf("failed to stop mount: %w", err)}
				}
				return MountStatusMsg{Name: mount.Name, Status: &systemd.ServiceStatus{Active: false}}
			},
		)
	} else {
		// Start and enable
		return s, tea.Sequence(
			func() tea.Msg {
				if err := s.manager.Start(serviceName); err != nil {
					return MountsErrorMsg{Err: fmt.Errorf("failed to start mount: %w", err)}
				}
				return MountStatusMsg{Name: mount.Name, Status: &systemd.ServiceStatus{Active: true}}
			},
		)
	}
}

// startMount starts the mount service.
func (s *MountsScreen) startMount() (tea.Model, tea.Cmd) {
	// Check if generator and manager are available
	if s.generator == nil || s.manager == nil {
		s.err = fmt.Errorf("systemd services not initialized")
		return s, nil
	}

	mount := s.mounts[s.cursor]
	serviceName := s.generator.ServiceName(mount.ID, "mount") + ".service"

	return s, func() tea.Msg {
		if err := s.manager.Start(serviceName); err != nil {
			return MountsErrorMsg{Err: fmt.Errorf("failed to start mount: %w", err)}
		}
		return MountStatusMsg{Name: mount.Name, Status: &systemd.ServiceStatus{Active: true}}
	}
}

// stopMount stops the mount service.
func (s *MountsScreen) stopMount() (tea.Model, tea.Cmd) {
	// Check if generator and manager are available
	if s.generator == nil || s.manager == nil {
		s.err = fmt.Errorf("systemd services not initialized")
		return s, nil
	}

	mount := s.mounts[s.cursor]
	serviceName := s.generator.ServiceName(mount.ID, "mount") + ".service"

	return s, func() tea.Msg {
		if err := s.manager.Stop(serviceName); err != nil {
			return MountsErrorMsg{Err: fmt.Errorf("failed to stop mount: %w", err)}
		}
		return MountStatusMsg{Name: mount.Name, Status: &systemd.ServiceStatus{Active: false}}
	}
}

// ShouldGoBack returns true if the screen should go back to the main menu.
func (s *MountsScreen) ShouldGoBack() bool {
	return s.goBack
}

// ResetGoBack resets the go back state.
func (s *MountsScreen) ResetGoBack() {
	s.goBack = false
}

// View renders the screen.
func (s *MountsScreen) View() string {
	switch s.mode {
	case MountsModeCreate, MountsModeEdit:
		if s.form != nil {
			return s.form.View()
		}
	case MountsModeDelete:
		if s.delete != nil {
			return s.delete.View()
		}
	case MountsModeDetails:
		if s.details != nil {
			return s.details.View()
		}
	}

	return s.renderList()
}

// renderList renders the mount list view.
func (s *MountsScreen) renderList() string {
	var b strings.Builder

	// Title
	title := components.Styles.Title.Render("Mount Management")
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
			Render("Loading mounts..."))
	} else if len(s.mounts) == 0 {
		// Empty state
		emptyMsg := components.Styles.Subtitle.Render("No mounts configured.")
		addHint := components.Styles.HelpText.Render("Press 'a' to add a new mount.")

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
		// Mount list
		b.WriteString(s.renderMountList())
		b.WriteString("\n")

		// Selected item details
		if s.cursor >= 0 && s.cursor < len(s.mounts) {
			b.WriteString(s.renderMountDetails())
		}
	}

	// Help bar
	b.WriteString("\n")
	helpText := components.HelpBar(s.width, []components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "a", Desc: "add"},
		{Key: "e", Desc: "edit"},
		{Key: "d", Desc: "delete"},
		{Key: "s", Desc: "start"},
		{Key: "x", Desc: "stop"},
		{Key: "Enter", Desc: "details"},
		{Key: "Esc", Desc: "back"},
	})
	b.WriteString(helpText)

	return b.String()
}

// renderMountList renders the list of mounts.
func (s *MountsScreen) renderMountList() string {
	var b strings.Builder

	// Header
	header := fmt.Sprintf("  %-20s %-20s %-25s %-10s",
		"Name", "Remote", "Mount Point", "Status")
	b.WriteString(components.Styles.Subtitle.Render(header) + "\n")
	b.WriteString(components.Styles.Subtitle.Render(strings.Repeat("─", s.width-4)) + "\n")

	// Mounts
	for i, mount := range s.mounts {
		var line string
		status := s.getMountStatus(&mount)

		if i == s.cursor {
			line = fmt.Sprintf("▸ %-20s %-20s %-25s %s",
				components.Styles.Selected.Render(mount.Name),
				components.Styles.Normal.Render(mount.Remote+mount.RemotePath),
				components.Styles.Normal.Render(mount.MountPoint),
				status)
		} else {
			line = fmt.Sprintf("  %-20s %-20s %-25s %s",
				components.Styles.Normal.Render(mount.Name),
				components.Styles.Normal.Render(mount.Remote+mount.RemotePath),
				components.Styles.Normal.Render(mount.MountPoint),
				status)
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

// getMountStatus returns a formatted status string for a mount.
func (s *MountsScreen) getMountStatus(mount *models.MountConfig) string {
	status, ok := s.statuses[mount.Name]
	if !ok {
		return components.StatusIndicator("unknown") + " unknown"
	}

	if status.Active {
		return components.StatusIndicator("active") + " " + components.Styles.Success.Render("running")
	}
	return components.StatusIndicator("inactive") + " " + components.Styles.StatusInactive.Render("stopped")
}

// renderMountDetails renders the details of the selected mount.
func (s *MountsScreen) renderMountDetails() string {
	mount := s.mounts[s.cursor]

	var b strings.Builder
	b.WriteString("\n")

	// Get status info
	statusStr := "unknown"
	if status, ok := s.statuses[mount.Name]; ok {
		if status.Active {
			statusStr = "running"
		} else {
			statusStr = "stopped"
		}
	}

	// Details box
	details := fmt.Sprintf(
		"  Selected: %s\n\n  Remote: %s\n  Remote Path: %s\n  Mount Point: %s\n  Status: %s\n  Enabled: %t\n\n  [E] Edit  [D] Delete  [S] Start  [X] Stop  [Enter] Details",
		components.Styles.Selected.Render(mount.Name),
		mount.Remote,
		mount.RemotePath,
		mount.MountPoint,
		statusStr,
		mount.Enabled,
	)

	box := components.Styles.Border.
		Width(s.width - 8).
		Render(details)

	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(box))

	return b.String()
}

// Messages for mount operations

// MountsLoadedMsg is sent when mounts are loaded.
type MountsLoadedMsg struct {
	Mounts []models.MountConfig
}

// MountCreatedMsg is sent when a mount is created.
type MountCreatedMsg struct {
	Mount models.MountConfig
}

// MountUpdatedMsg is sent when a mount is updated.
type MountUpdatedMsg struct {
	Mount models.MountConfig
}

// MountDeletedMsg is sent when a mount is deleted.
type MountDeletedMsg struct {
	Name string
}

// MountStatusMsg is sent when mount status is updated.
type MountStatusMsg struct {
	Name   string
	Status *systemd.ServiceStatus
}

// MountsErrorMsg is sent when an error occurs.
type MountsErrorMsg struct {
	Err error
}

// MountFormCancelMsg is sent when the form is cancelled.
type MountFormCancelMsg struct{}

// MountFormSubmitMsg is sent when the form is submitted.
type MountFormSubmitMsg struct {
	Mount models.MountConfig
	Edit  bool
}

// DeleteConfirm handles the delete confirmation dialog.
type DeleteConfirm struct {
	mount      models.MountConfig
	cursor     int
	done       bool
	deleteType int // 0: cancel, 1: service only, 2: service and config
	manager    *systemd.Manager
	generator  *systemd.Generator
	config     *config.Config
	width      int
}

// NewDeleteConfirm creates a new delete confirmation dialog.
func NewDeleteConfirm(mount models.MountConfig) *DeleteConfirm {
	return &DeleteConfirm{
		mount:      mount,
		cursor:     0,
		deleteType: 0,
	}
}

// SetServices sets the services for the delete confirmation.
func (d *DeleteConfirm) SetServices(mgr *systemd.Manager, gen *systemd.Generator, cfg *config.Config) {
	d.manager = mgr
	d.generator = gen
	d.config = cfg
}

// SetSize sets the size.
func (d *DeleteConfirm) SetSize(width, height int) {
	d.width = width
}

// Init initializes the dialog.
func (d *DeleteConfirm) Init() tea.Cmd {
	return nil
}

// Update handles updates.
func (d *DeleteConfirm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
func (d *DeleteConfirm) confirmDelete() (tea.Model, tea.Cmd) {
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

// deleteServiceOnly deletes only the systemd service.
func (d *DeleteConfirm) deleteServiceOnly() tea.Cmd {
	return func() tea.Msg {
		serviceName := d.generator.ServiceName(d.mount.ID, "mount") + ".service"

		// Stop the service if running
		_ = d.manager.Stop(serviceName)

		// Disable the service
		_ = d.manager.Disable(serviceName)

		// Remove the unit file
		_ = d.generator.RemoveUnit(serviceName)

		// Reload daemon
		_ = d.manager.DaemonReload()

		return MountDeletedMsg{Name: d.mount.Name}
	}
}

// deleteServiceAndConfig deletes both the service and config entry.
func (d *DeleteConfirm) deleteServiceAndConfig() tea.Cmd {
	return func() tea.Msg {
		serviceName := d.generator.ServiceName(d.mount.ID, "mount") + ".service"

		// Stop the service if running
		_ = d.manager.Stop(serviceName)

		// Disable the service
		_ = d.manager.Disable(serviceName)

		// Remove the unit file
		_ = d.generator.RemoveUnit(serviceName)

		// Reload daemon
		_ = d.manager.DaemonReload()

		// Remove from config
		if err := d.config.RemoveMount(d.mount.Name); err == nil {
			_ = d.config.Save()
		}

		return MountDeletedMsg{Name: d.mount.Name}
	}
}

// IsDone returns true if the dialog is done.
func (d *DeleteConfirm) IsDone() bool {
	return d.done
}

// View renders the dialog.
func (d *DeleteConfirm) View() string {
	var b strings.Builder

	// Title
	title := components.Styles.Title.Render("Delete Mount")
	b.WriteString(lipgloss.NewStyle().
		Width(d.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	// Warning message
	warning := fmt.Sprintf("Are you sure you want to delete '%s'?", d.mount.Name)
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

// MountDetails displays detailed mount information.
type MountDetails struct {
	mount     models.MountConfig
	status    *systemd.ServiceStatus
	logs      string
	manager   *systemd.Manager
	generator *systemd.Generator
	done      bool
	width     int
	height    int
	tab       int // 0: details, 1: logs
}

// NewMountDetails creates a new mount details view.
func NewMountDetails(mount models.MountConfig, manager *systemd.Manager, generator *systemd.Generator) *MountDetails {
	d := &MountDetails{
		mount:     mount,
		manager:   manager,
		generator: generator,
		tab:       0,
	}
	d.loadStatus()
	d.loadLogs()
	return d
}

// loadStatus loads the service status.
func (d *MountDetails) loadStatus() {
	serviceName := d.generator.ServiceName(d.mount.ID, "mount") + ".service"
	status, err := d.manager.Status(serviceName)
	if err == nil {
		d.status = status
	}
}

// loadLogs loads the service logs.
func (d *MountDetails) loadLogs() {
	serviceName := d.generator.ServiceName(d.mount.ID, "mount") + ".service"
	logs, err := d.manager.GetLogs(serviceName, 20)
	if err == nil {
		d.logs = logs
	} else {
		d.logs = "Failed to load logs"
	}
}

// SetSize sets the size.
func (d *MountDetails) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Init initializes the view.
func (d *MountDetails) Init() tea.Cmd {
	return nil
}

// Update handles updates.
func (d *MountDetails) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			d.done = true
		case "tab":
			d.tab = (d.tab + 1) % 2
		case "s":
			// Start service
			serviceName := d.generator.ServiceName(d.mount.ID, "mount") + ".service"
			_ = d.manager.Start(serviceName)
			d.loadStatus()
		case "x":
			// Stop service
			serviceName := d.generator.ServiceName(d.mount.ID, "mount") + ".service"
			_ = d.manager.Stop(serviceName)
			d.loadStatus()
		case "e":
			// Enable service
			serviceName := d.generator.ServiceName(d.mount.ID, "mount") + ".service"
			_ = d.manager.Enable(serviceName)
			d.loadStatus()
		case "d":
			// Disable service
			serviceName := d.generator.ServiceName(d.mount.ID, "mount") + ".service"
			_ = d.manager.Disable(serviceName)
			d.loadStatus()
		case "r":
			// Refresh
			d.loadStatus()
			d.loadLogs()
		}
	}

	return d, nil
}

// IsDone returns true if the view is done.
func (d *MountDetails) IsDone() bool {
	return d.done
}

// View renders the view.
func (d *MountDetails) View() string {
	var b strings.Builder

	// Title
	title := components.Styles.Title.Render(fmt.Sprintf("Mount: %s", d.mount.Name))
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
		{Key: "s", Desc: "start"},
		{Key: "x", Desc: "stop"},
		{Key: "e", Desc: "enable"},
		{Key: "d", Desc: "disable"},
		{Key: "r", Desc: "refresh"},
		{Key: "Esc", Desc: "back"},
	})
	b.WriteString(help)

	return b.String()
}

// renderDetails renders the details tab.
func (d *MountDetails) renderDetails() string {
	var b strings.Builder

	// Mount info
	b.WriteString(fmt.Sprintf("  Name: %s\n", d.mount.Name))
	b.WriteString(fmt.Sprintf("  Remote: %s\n", d.mount.Remote))
	b.WriteString(fmt.Sprintf("  Remote Path: %s\n", d.mount.RemotePath))
	b.WriteString(fmt.Sprintf("  Mount Point: %s\n", d.mount.MountPoint))
	b.WriteString(fmt.Sprintf("  Auto Start: %t\n", d.mount.AutoStart))
	b.WriteString(fmt.Sprintf("  Enabled: %t\n", d.mount.Enabled))

	// Status
	if d.status != nil {
		b.WriteString("\n  Service Status:\n")
		b.WriteString(fmt.Sprintf("    State: %s\n", d.status.State))
		b.WriteString(fmt.Sprintf("    SubState: %s\n", d.status.SubState))
		b.WriteString(fmt.Sprintf("    Enabled: %t\n", d.status.Enabled))
	}

	// Mount options
	b.WriteString("\n  Mount Options:\n")
	if d.mount.MountOptions.VFSCacheMode != "" {
		b.WriteString(fmt.Sprintf("    VFS Cache Mode: %s\n", d.mount.MountOptions.VFSCacheMode))
	}
	if d.mount.MountOptions.BufferSize != "" {
		b.WriteString(fmt.Sprintf("    Buffer Size: %s\n", d.mount.MountOptions.BufferSize))
	}
	if d.mount.MountOptions.ReadOnly {
		b.WriteString("    Read Only: true\n")
	}

	return b.String()
}

// renderLogs renders the logs tab.
func (d *MountDetails) renderLogs() string {
	if d.logs == "" {
		return components.Styles.Subtitle.Render("  No logs available")
	}

	// Truncate logs if too long
	lines := strings.Split(d.logs, "\n")
	if len(lines) > 15 {
		lines = lines[:15]
	}

	return components.Styles.Normal.Render(strings.Join(lines, "\n"))
}

// Helper function to get current time
func now() time.Time {
	return time.Now()
}
