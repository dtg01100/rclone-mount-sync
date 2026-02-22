// Package screens provides individual TUI screens for the application.
package screens

import (
	"fmt"
	"os"
	"path/filepath"
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
	manager      *systemd.Manager
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
	scheduleType string
	onCalendar   string
	onBootSec    string
	onBoot       bool

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
func NewSyncJobForm(job *models.SyncJobConfig, remotes []rclone.Remote, cfg *config.Config, gen *systemd.Generator, mgr *systemd.Manager, rcloneClient *rclone.Client, isEdit bool) *SyncJobForm {
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
		f.onBoot = job.Schedule.Type == "onboot"

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
		// Add a placeholder option when no remotes are available
		remoteOptions = append(remoteOptions, huh.NewOption("No remotes available - run 'rclone config'", ""))
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

			huh.NewFilePicker().
				Title("Destination Path").
				Description("Local directory for synced files. Press Enter to browse, Esc to close browser.").
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
				Description("Systemd calendar format (e.g., daily, hourly, *-*-* 02:00:00)").
				Placeholder("daily").
				Value(&f.onCalendar),

			huh.NewInput().
				Title("On Boot Delay").
				Description("Delay after boot before running (e.g., 5min, 1h)").
				Placeholder("5min").
				Value(&f.onBootSec),
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
				Value(&f.maxTransfers),

			huh.NewInput().
				Title("Bandwidth Limit").
				Description("Limit bandwidth (e.g., 10M, 1G)").
				Placeholder("10M").
				Value(&f.bandwidthLimit),

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
		expandedPath := expandSyncJobPath(path)

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

// expandSyncJobPath expands ~ to the user's home directory.
func expandSyncJobPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// getRemotePathSuggestions returns dynamic suggestions for remote paths.
func (f *SyncJobForm) getRemotePathSuggestions() []string {
	staticSuggestions := []string{"/", "/Photos", "/Documents", "/Backup", "/Sync"}

	if f.rcloneClient == nil || f.sourceRemote == "" {
		return staticSuggestions
	}

	if f.sourceRemote == "" {
		return staticSuggestions
	}

	directories, err := f.rcloneClient.ListRootDirectories(f.sourceRemote)
	if err != nil {
		return staticSuggestions
	}

	result := []string{"/"}
	for _, dir := range directories {
		result = append(result, "/"+dir)
	}

	return result
}

// SetSize sets the form size.
func (f *SyncJobForm) SetSize(width, height int) {
	f.width = width
	f.height = height
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
		return SyncJobsErrorMsg{Err: fmt.Errorf("no source remote selected - please configure rclone remotes first")}
	}

	// Build the source path
	source := f.sourceRemote + ":" + f.sourcePath

	// Build the destination path
	var destination string
	if f.destRemote != "" {
		destination = f.destRemote + ":" + f.destPath
	} else {
		destination = expandSyncJobPath(f.destPath)
	}

	// Parse max transfers
	transfers := 4
	if f.maxTransfers != "" {
		if t := strings.TrimSpace(f.maxTransfers); t != "" {
			var err error
			if transfers, err = strconvAtoi(t); err != nil {
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

	// Determine schedule type
	scheduleType := f.scheduleType
	if f.onBoot {
		scheduleType = "onboot"
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
			Type:       scheduleType,
			OnCalendar: f.onCalendar,
			OnBootSec:  f.onBootSec,
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
	if f.generator != nil {
		_, _, err := f.generator.WriteSyncUnits(&job)
		if err != nil {
			return SyncJobsErrorMsg{Err: fmt.Errorf("failed to generate unit files: %w", err)}
		}

		// Reload systemd daemon
		if f.manager != nil {
			_ = f.manager.DaemonReload()

			serviceName := f.generator.ServiceName(job.Name, "sync") + ".service"
			timerName := f.generator.ServiceName(job.Name, "sync") + ".timer"

			// Enable timer if requested
			if job.Enabled {
				_ = f.manager.EnableTimer(timerName)
				_ = f.manager.StartTimer(timerName)
			}

			// Run immediately if requested
			if f.runImmediately {
				_ = f.manager.RunSyncNow(serviceName)
			}
		}
	}

	f.done = true

	if f.isEdit {
		return SyncJobUpdatedMsg{Job: job}
	}
	return SyncJobCreatedMsg{Job: job}
}

// strconvAtoi is a simple wrapper for strconv.Atoi
func strconvAtoi(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
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
