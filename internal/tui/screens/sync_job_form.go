// Package screens provides individual TUI screens for the application.
package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/rclone"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
	"github.com/dtg01100/rclone-mount-sync/internal/tui/components"
	"github.com/google/uuid"
)

// SyncJobForm handles sync job creation and editing using huh.
type SyncJobForm struct {
	// Form state
	form      *huh.Form
	done      bool
	cancelled bool
	width     int
	height    int

	// Sync job being edited (nil for create)
	job    *models.SyncJobConfig
	isEdit bool

	// Services
	config       *config.Config
	generator    *systemd.Generator
	manager      systemd.ServiceManager
	rcloneClient *rclone.Client

	// Available remotes
	remotes []rclone.Remote

	// Form data - Basic Info
	name         string
	sourceRemote string
	sourcePath   string
	destRemote   string
	destPath     string

	// Form data - Sync Options
	direction       string
	deleteMode      string
	createEmptyDirs bool
	dryRun          bool
	trackRenames    bool

	// Form data - Schedule
	scheduleType     string
	onCalendar       string
	onBootSec        string
	requireACPower   bool
	requireUnmetered bool

	// Form data - Filters & Performance
	excludePattern string
	maxTransfers   string
	bandwidthLimit string
	logLevel       string

	// Form data - Service Options
	enabled        bool
	runImmediately bool
}

// NewSyncJobForm creates a new sync job form.
func NewSyncJobForm(job *models.SyncJobConfig, remotes []rclone.Remote, cfg *config.Config, gen *systemd.Generator, mgr systemd.ServiceManager, rcloneClient *rclone.Client, isEdit bool) *SyncJobForm {
	f := &SyncJobForm{
		job:          job,
		isEdit:       isEdit,
		config:       cfg,
		generator:    gen,
		manager:      mgr,
		rcloneClient: rcloneClient,
		remotes:      remotes,
	}

	// Set defaults from config
	if cfg != nil {
		f.logLevel = cfg.Defaults.Sync.LogLevel
		f.maxTransfers = fmt.Sprintf("%d", cfg.Defaults.Sync.Transfers)
	}

	// If editing, populate with existing values
	if job != nil {
		f.name = job.Name

		// Parse source remote and path
		srcRemote, srcPath := parseRemotePath(job.Source)
		f.sourceRemote = srcRemote
		f.sourcePath = srcPath

		// Parse dest remote and path (if remote) or local path
		if strings.Contains(job.Destination, ":") {
			destRemote, destPath := parseRemotePath(job.Destination)
			f.destRemote = destRemote
			f.destPath = destPath
		} else {
			f.destPath = job.Destination
		}

		// Sync options
		f.direction = job.SyncOptions.Direction
		if job.SyncOptions.DeleteAfter {
			f.deleteMode = "after"
		} else if job.SyncOptions.DeleteExtraneous {
			f.deleteMode = "during"
		} else {
			f.deleteMode = "never"
		}
		f.createEmptyDirs = true // Default in generator
		f.dryRun = job.SyncOptions.DryRun

		// Schedule
		f.scheduleType = job.Schedule.Type
		f.onCalendar = job.Schedule.OnCalendar
		f.onBootSec = job.Schedule.OnBootSec
		f.requireACPower = job.Schedule.RequireACPower
		f.requireUnmetered = job.Schedule.RequireUnmetered

		// Filters & Performance
		f.excludePattern = job.SyncOptions.ExcludePattern
		f.maxTransfers = fmt.Sprintf("%d", job.SyncOptions.Transfers)
		f.bandwidthLimit = job.SyncOptions.BandwidthLimit
		f.logLevel = job.SyncOptions.LogLevel

		// Service options
		f.enabled = job.Enabled
	}

	// Set default values if empty
	if f.direction == "" {
		f.direction = "sync"
	}
	if f.deleteMode == "" {
		f.deleteMode = "after"
	}
	if f.logLevel == "" {
		f.logLevel = "INFO"
	}
	if f.maxTransfers == "0" {
		f.maxTransfers = "4"
	}
	if f.scheduleType == "" {
		f.scheduleType = "timer"
	}
	if f.onCalendar == "" {
		f.onCalendar = "daily"
	}

	f.buildForm()
	return f
}

// parseRemotePath parses a remote:path string into remote and path components.
func parseRemotePath(s string) (remote, path string) {
	if idx := strings.Index(s, ":"); idx != -1 {
		return s[:idx], s[idx+1:]
	}
	return "", s
}

// buildForm builds the huh form.
func (f *SyncJobForm) buildForm() {
	homeDir, _ := os.UserHomeDir()

	// Build remote options - handle empty remotes gracefully
	remoteOptions := make([]huh.Option[string], 0)
	if len(f.remotes) > 0 {
		for _, r := range f.remotes {
			remoteOptions = append(remoteOptions, huh.NewOption(r.Name+" ("+r.Type+")", r.Name))
		}
	} else {
		remoteOptions = append(remoteOptions, huh.NewOption("⚠ No remotes - run 'rclone config' first", ""))
	}

	// Direction options
	directionOptions := []huh.Option[string]{
		huh.NewOption("Sync (mirror)", "sync"),
		huh.NewOption("Copy", "copy"),
		huh.NewOption("Move", "move"),
	}

	// Delete mode options
	deleteModeOptions := []huh.Option[string]{
		huh.NewOption("After sync", "after"),
		huh.NewOption("During sync", "during"),
		huh.NewOption("Never", "never"),
	}

	// Schedule type options
	scheduleTypeOptions := []huh.Option[string]{
		huh.NewOption("Timer (scheduled)", "timer"),
		huh.NewOption("On Boot", "onboot"),
		huh.NewOption("Manual only", "manual"),
	}

	// Log level options
	logLevelOptions := []huh.Option[string]{
		huh.NewOption("Error", "ERROR"),
		huh.NewOption("Notice", "NOTICE"),
		huh.NewOption("Info", "INFO"),
		huh.NewOption("Debug", "DEBUG"),
	}

	// Build form groups
	groups := []*huh.Group{
		// Step 1: Basic Info
		huh.NewGroup(
			huh.NewInput().
				Title("Sync Job Name").
				Description("A unique name for this sync job").
				Placeholder("e.g., Photos Backup").
				Value(&f.name).
				Validate(f.validateName),

			huh.NewSelect[string]().
				Title("Source Remote").
				Description("Select the source rclone remote").
				Options(remoteOptions...).
				Value(&f.sourceRemote),

			huh.NewInput().
				Title("Source Path").
				Description("Path on the source remote (e.g., /Photos)").
				Placeholder("/").
				Value(&f.sourcePath).
				SuggestionsFunc(f.getRemotePathSuggestions, &f.sourceRemote),

			components.NewEnhancedFilePicker().
				Title("Destination Path").
				Description("Local directory for synced files. Use quick jump keys: ~ (home), / (root), m (mnt), M (media), r (recent), Backspace (parent).").
				DirAllowed(true).
				FileAllowed(false).
				CurrentDirectory(homeDir).
				Value(&f.destPath).
				Validate(f.validateDestPath),
		).Title("Step 1: Basic Info"),

		// Step 2: Sync Options
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Sync Direction").
				Description("What operation to perform").
				Options(directionOptions...).
				Value(&f.direction),

			huh.NewSelect[string]().
				Title("Delete Mode").
				Description("When to delete extraneous files").
				Options(deleteModeOptions...).
				Value(&f.deleteMode),

			huh.NewConfirm().
				Title("Create Empty Source Dirs").
				Description("Create empty directories from source").
				Value(&f.createEmptyDirs),

			huh.NewConfirm().
				Title("Dry Run").
				Description("Simulate the sync without making changes").
				Value(&f.dryRun),

			huh.NewConfirm().
				Title("Track Renames").
				Description("Track file renames for efficient syncing").
				Value(&f.trackRenames),
		).Title("Step 2: Sync Options"),

		// Step 3: Schedule
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Schedule Type").
				Description("When to run the sync job").
				Options(scheduleTypeOptions...).
				Value(&f.scheduleType),

			huh.NewInput().
				Title("Calendar Schedule").
				Description("Systemd calendar format (only used when Schedule Type is 'Timer')").
				Placeholder("daily").
				Value(&f.onCalendar).
				Validate(f.validateOnCalendar),

			huh.NewInput().
				Title("On Boot Delay").
				Description("Delay after boot before running (only used when Schedule Type is 'On Boot')").
				Placeholder("5min").
				Value(&f.onBootSec),

			huh.NewConfirm().
				Title("Require AC Power").
				Description("Only run when connected to AC power (not on battery)").
				Value(&f.requireACPower),

			huh.NewConfirm().
				Title("Require Unmetered Connection").
				Description("Only run on non-metered internet connections").
				Value(&f.requireUnmetered),
		).Title("Step 3: Schedule"),

		// Step 4: Filters & Performance
		huh.NewGroup(
			huh.NewInput().
				Title("Exclude Patterns").
				Description("Comma-separated patterns to exclude").
				Placeholder("*.tmp, .git/*, node_modules/*").
				Value(&f.excludePattern),

			huh.NewInput().
				Title("Max Transfers").
				Description("Maximum number of parallel transfers").
				Placeholder("4").
				Value(&f.maxTransfers).
				Validate(f.validateMaxTransfers),

			huh.NewInput().
				Title("Bandwidth Limit").
				Description("Limit bandwidth (e.g., 10M, 1G)").
				Placeholder("10M").
				Value(&f.bandwidthLimit).
				Validate(components.ValidateBandwidthLimit),

			huh.NewSelect[string]().
				Title("Log Level").
				Description("Logging verbosity").
				Options(logLevelOptions...).
				Value(&f.logLevel),
		).Title("Step 4: Filters & Performance"),

		// Step 5: Service Options
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable Timer").
				Description("Enable the systemd timer for scheduled runs").
				Value(&f.enabled),

			huh.NewConfirm().
				Title("Run Immediately").
				Description("Run the sync job immediately after creation").
				Value(&f.runImmediately),
		).Title("Step 5: Service Options"),
	}

	f.form = huh.NewForm(groups...)
	f.form.WithTheme(huh.ThemeBase16())
}

// showCalendar returns true if the calendar field should be shown.
func (f *SyncJobForm) showCalendar() bool {
	return f.scheduleType == "timer"
}

// showOnBoot returns true if the on boot field should be shown.
func (f *SyncJobForm) showOnBoot() bool {
	return f.scheduleType == "onboot"
}

// validateName validates the sync job name.
func (f *SyncJobForm) validateName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > 50 {
		return fmt.Errorf("name must be 50 characters or less")
	}
	// Check for duplicate names (only for new sync jobs)
	if !f.isEdit && f.config != nil {
		for _, j := range f.config.SyncJobs {
			if j.Name == name {
				return fmt.Errorf("a sync job with this name already exists")
			}
		}
	}
	return nil
}

// validateDestPath validates the destination path.
func (f *SyncJobForm) validateDestPath(path string) error {
	if path == "" {
		return fmt.Errorf("destination path is required")
	}

	// Check if it's a local path (doesn't contain colon)
	if !strings.Contains(path, ":") {
		// Expand ~ to home directory
		expandedPath := components.ExpandHome(path)

		// Check if path is absolute or starts with ~
		if !filepath.IsAbs(expandedPath) && !strings.HasPrefix(path, "~/") {
			return fmt.Errorf("local path must be absolute or start with ~")
		}

		// Check if parent directory exists
		parentDir := filepath.Dir(expandedPath)
		if _, err := os.Stat(parentDir); os.IsNotExist(err) {
			return fmt.Errorf("parent directory does not exist: %s", parentDir)
		}
	}

	return nil
}

// validateOnCalendar validates the OnCalendar timer string.
func (f *SyncJobForm) validateOnCalendar(calendar string) error {
	return rclone.ValidateOnCalendar(calendar)
}

// validateMaxTransfers validates the max transfers field.
func (f *SyncJobForm) validateMaxTransfers(value string) error {
	if value == "" {
		return nil
	}
	num, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("must be a valid number")
	}
	if num <= 0 {
		return fmt.Errorf("must be greater than 0")
	}
	return nil
}

// getRemotePathSuggestions returns dynamic suggestions for remote paths.
func (f *SyncJobForm) getRemotePathSuggestions() []string {
	staticSuggestions := []string{"/", "/Photos", "/Documents", "/Backup", "/Sync"}
	if f.rcloneClient == nil {
		return staticSuggestions
	}
	return components.GetRemotePathSuggestions(f.rcloneClient, f.sourceRemote, staticSuggestions)
}

// SetSize sets the form size.
func (f *SyncJobForm) SetSize(width, height int) {
	f.width = width
	f.height = height
	if f.form != nil {
		f.form.WithWidth(width)
	}
}

// Init initializes the form.
func (f *SyncJobForm) Init() tea.Cmd {
	return f.form.Init()
}

// Update handles form updates.
func (f *SyncJobForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Check if we're at the first field, if so cancel
			f.cancelled = true
			f.done = true
			return f, func() tea.Msg { return SyncJobFormCancelMsg{} }
		}
	}

	// Update the form
	form, cmd := f.form.Update(msg)
	f.form = form.(*huh.Form)
	cmds = append(cmds, cmd)

	// Check if form is complete
	if f.form.State == huh.StateCompleted {
		cmds = append(cmds, f.submitForm)
		return f, tea.Batch(cmds...)
	}

	return f, tea.Batch(cmds...)
}

// submitForm submits the form and creates/updates the sync job.
func (f *SyncJobForm) submitForm() tea.Msg {
	// Validate that a source remote was selected
	if f.sourceRemote == "" {
		return SyncJobsErrorMsg{Err: fmt.Errorf("no source remote selected.\n\nTo add remotes:\n  1. Open a terminal and run: rclone config\n  2. Press 'n' to create a new remote\n  3. Follow the prompts to configure your cloud storage\n  4. Restart this application")}
	}

	// Build the source path
	source := f.sourceRemote + ":" + f.sourcePath

	// Build the destination path
	var destination string
	if f.destRemote != "" {
		destination = f.destRemote + ":" + f.destPath
	} else {
		destination = components.ExpandHome(f.destPath)
	}

	// Parse max transfers
	transfers := 4
	if f.maxTransfers != "" {
		if t := strings.TrimSpace(f.maxTransfers); t != "" {
			var err error
			if transfers, err = strconv.Atoi(t); err != nil {
				transfers = 4
			}
		}
	}

	// Determine delete mode
	deleteAfter := false
	deleteExtraneous := false
	switch f.deleteMode {
	case "after":
		deleteAfter = true
	case "during":
		deleteExtraneous = true
	}

	// Determine schedule type and clear irrelevant schedule fields
	scheduleType := f.scheduleType
	onCalendar := f.onCalendar
	onBootSec := f.onBootSec

	switch scheduleType {
	case "timer":
		onBootSec = ""
	case "onboot":
		onCalendar = ""
	case "manual":
		onCalendar = ""
		onBootSec = ""
	}

	// Build the sync job configuration
	job := models.SyncJobConfig{
		Name:        f.name,
		Source:      source,
		Destination: destination,
		SyncOptions: models.SyncOptions{
			Direction:        f.direction,
			DeleteAfter:      deleteAfter,
			DeleteExtraneous: deleteExtraneous,
			DryRun:           f.dryRun,
			ExcludePattern:   f.excludePattern,
			Transfers:        transfers,
			BandwidthLimit:   f.bandwidthLimit,
			LogLevel:         f.logLevel,
		},
		Schedule: models.ScheduleConfig{
			Type:             scheduleType,
			OnCalendar:       onCalendar,
			OnBootSec:        onBootSec,
			RequireACPower:   f.requireACPower,
			RequireUnmetered: f.requireUnmetered,
		},
		Enabled: f.enabled,
	}

	// Set timestamps
	now := time.Now()
	if f.isEdit && f.job != nil {
		job.ID = f.job.ID
		job.CreatedAt = f.job.CreatedAt
	} else {
		job.ID = uuid.New().String()[:8]
		job.CreatedAt = now
	}
	job.ModifiedAt = now

	op := OperationCreate
	if f.isEdit {
		op = OperationUpdate
	}

	var rollbackData SyncJobRollbackData
	if f.config != nil {
		rollbackMgr := NewRollbackManager(f.config, f.generator, f.manager)
		rollbackData = rollbackMgr.PrepareSyncJobRollback(job.ID, job.Name, op)
	}

	// Save to config
	if f.config != nil {
		if f.isEdit {
			// Remove old job and add updated one
			for i, j := range f.config.SyncJobs {
				if j.ID == job.ID {
					f.config.SyncJobs[i] = job
					break
				}
			}
		} else {
			f.config.SyncJobs = append(f.config.SyncJobs, job)
		}
		if err := f.config.Save(); err != nil {
			return SyncJobsErrorMsg{Err: fmt.Errorf("failed to save config: %w", err)}
		}
		if !strings.Contains(f.destPath, ":") {
			f.config.AddRecentPath(f.destPath)
		}
	}

	// Generate systemd service and timer files
	if f.generator == nil {
		return SyncJobsErrorMsg{Err: fmt.Errorf("systemd generator not initialized - cannot create unit files")}
	}

	_, _, err := f.generator.WriteSyncUnits(&job)
	if err != nil {
		if f.config != nil {
			// Attempt rollback on failure; errors are ignored since we're already
			// in an error path and the primary error is more important to report
			rollbackMgr := NewRollbackManager(f.config, f.generator, f.manager)
			_ = rollbackMgr.RollbackSyncJob(rollbackData, true)
		}
		return SyncJobsErrorMsg{Err: fmt.Errorf("failed to write unit files: %w", err)}
	}

	// Reload systemd daemon
	if f.manager == nil {
		return SyncJobsErrorMsg{Err: fmt.Errorf("systemd manager not initialized - cannot reload daemon")}
	}

	if err := f.manager.DaemonReload(); err != nil {
		if f.config != nil {
			// Attempt rollback on failure; errors are ignored since we're already
			// in an error path and the primary error is more important to report
			rollbackMgr := NewRollbackManager(f.config, f.generator, f.manager)
			_ = rollbackMgr.RollbackSyncJob(rollbackData, true)
		}
		return SyncJobsErrorMsg{Err: fmt.Errorf("failed to reload systemd daemon: %w", err)}
	}

	serviceName := f.generator.ServiceName(job.ID, "sync") + ".service"
	timerName := f.generator.ServiceName(job.ID, "sync") + ".timer"

	// Enable timer if requested
	if job.Enabled {
		if err := f.manager.EnableTimer(timerName); err != nil {
			if f.config != nil {
				// Attempt rollback on failure; errors are ignored since we're already
				// in an error path and the primary error is more important to report
				rollbackMgr := NewRollbackManager(f.config, f.generator, f.manager)
				_ = rollbackMgr.RollbackSyncJob(rollbackData, true)
			}
			return SyncJobsErrorMsg{Err: fmt.Errorf("failed to enable timer: %w", err)}
		}
		if err := f.manager.StartTimer(timerName); err != nil {
			if f.config != nil {
				// Attempt rollback on failure; errors are ignored since we're already
				// in an error path and the primary error is more important to report
				rollbackMgr := NewRollbackManager(f.config, f.generator, f.manager)
				_ = rollbackMgr.RollbackSyncJob(rollbackData, true)
			}
			return SyncJobsErrorMsg{Err: fmt.Errorf("failed to start timer: %w", err)}
		}
	}

	// Run immediately if requested
	if f.runImmediately {
		if err := f.manager.RunSyncNow(serviceName); err != nil {
			if f.config != nil {
				// Attempt rollback on failure; errors are ignored since we're already
				// in an error path and the primary error is more important to report
				rollbackMgr := NewRollbackManager(f.config, f.generator, f.manager)
				_ = rollbackMgr.RollbackSyncJob(rollbackData, true)
			}
			return SyncJobsErrorMsg{Err: fmt.Errorf("failed to run sync job: %w", err)}
		}
	}

	f.done = true

	if f.isEdit {
		return SyncJobUpdatedMsg{Job: job}
	}
	return SyncJobCreatedMsg{Job: job}
}

// IsDone returns true if the form is done.
func (f *SyncJobForm) IsDone() bool {
	return f.done
}

// View renders the form.
func (f *SyncJobForm) View() string {
	if f.done {
		return ""
	}

	// Render the form
	formView := f.form.View()

	// Add header
	title := "Create New Sync Job"
	if f.isEdit {
		title = "Edit Sync Job: " + f.name
	}

	header := components.Styles.Title.Render(title)
	header = lipgloss.NewStyle().
		Width(f.width).
		Align(lipgloss.Center).
		Render(header)

	// Add help text
	help := components.Styles.HelpText.Render("Tab: next field  Shift+Tab: previous field  Enter: confirm/browse  Esc: cancel  Ctrl+E: accept suggestion")
	help = lipgloss.NewStyle().
		Width(f.width).
		Align(lipgloss.Center).
		Render(help)

	// Combine
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		formView,
		"",
		help,
	)
}
