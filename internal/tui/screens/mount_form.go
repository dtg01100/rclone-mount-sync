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

// MountForm handles mount creation and editing using huh.
type MountForm struct {
	// Form state
	form      *huh.Form
	done      bool
	cancelled bool
	width     int
	height    int

	// Mount being edited (nil for create)
	mount  *models.MountConfig
	isEdit bool

	// Services
	config       *config.Config
	generator    *systemd.Generator
	manager      *systemd.Manager
	rcloneClient *rclone.Client

	// Available remotes
	remotes []rclone.Remote

	// Form data
	name            string
	remote          string
	remotePath      string
	mountPoint      string
	vfsCacheMode    string
	vfsCacheMaxAge  string
	vfsCacheMaxSize string
	vfsWriteBack    string
	bufferSize      string
	allowOther      bool
	allowRoot       bool
	umask           string
	readOnly        bool
	noModtime       bool
	noChecksum      bool
	logLevel        string
	extraArgs       string
	autoStart       bool
	enabled         bool
}

// NewMountForm creates a new mount form.
func NewMountForm(mount *models.MountConfig, remotes []rclone.Remote, cfg *config.Config, gen *systemd.Generator, mgr *systemd.Manager, rcloneClient *rclone.Client, isEdit bool) *MountForm {
	f := &MountForm{
		mount:        mount,
		isEdit:       isEdit,
		config:       cfg,
		generator:    gen,
		manager:      mgr,
		rcloneClient: rcloneClient,
		remotes:      remotes,
	}

	// Set defaults from config
	if cfg != nil {
		f.vfsCacheMode = cfg.Defaults.Mount.VFSCacheMode
		f.bufferSize = cfg.Defaults.Mount.BufferSize
		f.logLevel = cfg.Defaults.Mount.LogLevel
	}

	// If editing, populate with existing values
	if mount != nil {
		f.name = mount.Name
		f.remote = mount.Remote
		f.remotePath = mount.RemotePath
		f.mountPoint = mount.MountPoint
		f.vfsCacheMode = mount.MountOptions.VFSCacheMode
		f.vfsCacheMaxAge = mount.MountOptions.VFSCacheMaxAge
		f.vfsCacheMaxSize = mount.MountOptions.VFSCacheMaxSize
		f.vfsWriteBack = mount.MountOptions.VFSWriteBack
		f.bufferSize = mount.MountOptions.BufferSize
		f.allowOther = mount.MountOptions.AllowOther
		f.allowRoot = mount.MountOptions.AllowRoot
		f.umask = mount.MountOptions.Umask
		f.readOnly = mount.MountOptions.ReadOnly
		f.noModtime = mount.MountOptions.NoModTime
		f.noChecksum = mount.MountOptions.NoChecksum
		f.logLevel = mount.MountOptions.LogLevel
		f.extraArgs = mount.MountOptions.ExtraArgs
		f.autoStart = mount.AutoStart
		f.enabled = mount.Enabled
	}

	// Set default values if empty
	if f.vfsCacheMode == "" {
		f.vfsCacheMode = "full"
	}
	if f.bufferSize == "" {
		f.bufferSize = "16M"
	}
	if f.logLevel == "" {
		f.logLevel = "INFO"
	}
	if f.remotePath == "" {
		f.remotePath = "/"
	}

	f.buildForm()
	return f
}

// buildForm builds the huh form.
func (f *MountForm) buildForm() {
	// Build remote options - handle empty remotes gracefully
	remoteOptions := make([]huh.Option[string], 0)
	if len(f.remotes) > 0 {
		for _, r := range f.remotes {
			remoteOptions = append(remoteOptions, huh.NewOption(r.Name+" ("+r.Type+")", r.Name+":"))
		}
	} else {
		// Add a placeholder option when no remotes are available
		remoteOptions = append(remoteOptions, huh.NewOption("No remotes available - run 'rclone config'", ""))
	}

	// VFS Cache Mode options
	vfsCacheOptions := []huh.Option[string]{
		huh.NewOption("Off", "off"),
		huh.NewOption("Writes", "writes"),
		huh.NewOption("Full", "full"),
	}

	// Log Level options
	logLevelOptions := []huh.Option[string]{
		huh.NewOption("Error", "ERROR"),
		huh.NewOption("Notice", "NOTICE"),
		huh.NewOption("Info", "INFO"),
		huh.NewOption("Debug", "DEBUG"),
	}

	// Build form groups
	groups := []*huh.Group{
		// Step 1: Basic Configuration
		huh.NewGroup(
			huh.NewInput().
				Title("Mount Name").
				Description("A unique name for this mount").
				Placeholder("e.g., Google Drive").
				Value(&f.name).
				Validate(f.validateName),

			huh.NewSelect[string]().
				Title("Remote").
				Description("Select the rclone remote to mount").
				Options(remoteOptions...).
				Value(&f.remote),

			huh.NewInput().
				Title("Remote Path").
				Description("Path on the remote (e.g., / or /Photos)").
				Placeholder("/").
				SuggestionsFunc(f.getRemotePathSuggestions, &f.remote).
				Value(&f.remotePath),

			huh.NewFilePicker().
				Title("Mount Point").
				Description("Local directory where the remote will be mounted. Press Enter to browse, Esc to close browser.").
				DirAllowed(true).
				FileAllowed(false).
				CurrentDirectory(components.ExpandHome("~/mnt")).
				Value(&f.mountPoint).
				Validate(f.validateMountPoint),
		).Title("Step 1: Basic Configuration"),

		// Step 2: VFS Options
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("VFS Cache Mode").
				Description("Caching mode for VFS (recommended: full)").
				Options(vfsCacheOptions...).
				Value(&f.vfsCacheMode),

			huh.NewInput().
				Title("VFS Cache Max Size").
				Description("Maximum size of the cache (e.g., 10G)").
				Placeholder("10G").
				Value(&f.vfsCacheMaxSize),

			huh.NewInput().
				Title("VFS Cache Max Age").
				Description("Maximum age of cache items (e.g., 24h)").
				Placeholder("24h").
				Value(&f.vfsCacheMaxAge),

			huh.NewInput().
				Title("VFS Write Back").
				Description("Time to wait before writing files (e.g., 5s)").
				Placeholder("5s").
				Value(&f.vfsWriteBack),

			huh.NewInput().
				Title("Buffer Size").
				Description("Buffer size for reading (e.g., 16M)").
				Placeholder("16M").
				Value(&f.bufferSize),
		).Title("Step 2: VFS Options"),

		// Step 3: FUSE Options
		huh.NewGroup(
			huh.NewConfirm().
				Title("Allow Other").
				Description("Allow other users to access the mount").
				Value(&f.allowOther),

			huh.NewConfirm().
				Title("Allow Root").
				Description("Allow root to access the mount").
				Value(&f.allowRoot),

			huh.NewInput().
				Title("Umask").
				Description("File permission mask (e.g., 002)").
				Placeholder("002").
				Value(&f.umask),

			huh.NewConfirm().
				Title("Read Only").
				Description("Mount the remote as read-only").
				Value(&f.readOnly),
		).Title("Step 3: FUSE Options"),

		// Step 4: Advanced Options
		huh.NewGroup(
			huh.NewConfirm().
				Title("No ModTime").
				Description("Don't read/write modification times").
				Value(&f.noModtime),

			huh.NewConfirm().
				Title("No Checksum").
				Description("Don't verify checksums").
				Value(&f.noChecksum),

			huh.NewSelect[string]().
				Title("Log Level").
				Description("Logging verbosity").
				Options(logLevelOptions...).
				Value(&f.logLevel),

			huh.NewInput().
				Title("Extra Arguments").
				Description("Additional rclone arguments").
				Placeholder("--option value").
				Value(&f.extraArgs),
		).Title("Step 4: Advanced Options"),

		// Step 5: Service Options
		huh.NewGroup(
			huh.NewConfirm().
				Title("Auto Start").
				Description("Start the mount automatically on login").
				Value(&f.autoStart),

			huh.NewConfirm().
				Title("Enable Service").
				Description("Enable the systemd service").
				Value(&f.enabled),
		).Title("Step 5: Service Options"),
	}

	f.form = huh.NewForm(groups...)
	f.form.WithTheme(huh.ThemeBase16())
}

// validateName validates the mount name.
func (f *MountForm) validateName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > 50 {
		return fmt.Errorf("name must be 50 characters or less")
	}
	// Check for duplicate names (only for new mounts)
	if !f.isEdit && f.config != nil {
		for _, m := range f.config.Mounts {
			if m.Name == name {
				return fmt.Errorf("a mount with this name already exists")
			}
		}
	}
	return nil
}

// validateMountPoint validates the mount point path.
func (f *MountForm) validateMountPoint(path string) error {
	if path == "" {
		return fmt.Errorf("mount point is required")
	}

	// Expand ~ to home directory
	expandedPath := expandPath(path)

	// Check if path is absolute
	if !filepath.IsAbs(expandedPath) {
		return fmt.Errorf("mount point must be an absolute path or start with ~")
	}

	// Check if parent directory exists
	parentDir := filepath.Dir(expandedPath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		return fmt.Errorf("parent directory does not exist: %s", parentDir)
	}

	return nil
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
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
func (f *MountForm) getRemotePathSuggestions() []string {
	staticSuggestions := []string{"/", "/Photos", "/Documents", "/Backup"}

	if f.rcloneClient == nil || f.remote == "" {
		return staticSuggestions
	}

	remoteName := strings.TrimSuffix(f.remote, ":")
	if remoteName == "" {
		return staticSuggestions
	}

	directories, err := f.rcloneClient.ListRootDirectories(remoteName)
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
func (f *MountForm) SetSize(width, height int) {
	f.width = width
	f.height = height
}

// Init initializes the form.
func (f *MountForm) Init() tea.Cmd {
	return f.form.Init()
}

// Update handles form updates.
func (f *MountForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Check if we're at the first field, if so cancel
			f.cancelled = true
			f.done = true
			return f, func() tea.Msg { return MountFormCancelMsg{} }
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

// submitForm submits the form and creates/updates the mount.
func (f *MountForm) submitForm() tea.Msg {
	// Validate that a remote was selected
	if f.remote == "" {
		return MountsErrorMsg{Err: fmt.Errorf("no remote selected - please configure rclone remotes first")}
	}

	// Build the mount configuration
	mount := models.MountConfig{
		Name:       f.name,
		Remote:     strings.TrimSuffix(f.remote, ":"),
		RemotePath: f.remotePath,
		MountPoint: f.mountPoint,
		MountOptions: models.MountOptions{
			VFSCacheMode:    f.vfsCacheMode,
			VFSCacheMaxAge:  f.vfsCacheMaxAge,
			VFSCacheMaxSize: f.vfsCacheMaxSize,
			VFSWriteBack:    f.vfsWriteBack,
			BufferSize:      f.bufferSize,
			AllowOther:      f.allowOther,
			AllowRoot:       f.allowRoot,
			Umask:           f.umask,
			ReadOnly:        f.readOnly,
			NoModTime:       f.noModtime,
			NoChecksum:      f.noChecksum,
			LogLevel:        f.logLevel,
			ExtraArgs:       f.extraArgs,
		},
		AutoStart: f.autoStart,
		Enabled:   f.enabled,
	}

	// Set timestamps
	now := time.Now()
	if f.isEdit && f.mount != nil {
		mount.ID = f.mount.ID
		mount.CreatedAt = f.mount.CreatedAt
	} else {
		mount.ID = uuid.New().String()[:8]
		mount.CreatedAt = now
	}
	mount.ModifiedAt = now

	// Save to config
	if f.config != nil {
		if f.isEdit {
			// Remove old mount and add updated one
			for i, m := range f.config.Mounts {
				if m.ID == mount.ID {
					f.config.Mounts[i] = mount
					break
				}
			}
		} else {
			f.config.Mounts = append(f.config.Mounts, mount)
		}
		if err := f.config.Save(); err != nil {
			return MountsErrorMsg{Err: fmt.Errorf("failed to save config: %w", err)}
		}
		f.config.AddRecentPath(f.mountPoint)
	}

	// Generate systemd service file
	if f.generator != nil {
		_, err := f.generator.WriteMountService(&mount)
		if err != nil {
			return MountsErrorMsg{Err: fmt.Errorf("failed to generate service file: %w", err)}
		}

		// Reload systemd daemon
		if f.manager != nil {
			_ = f.manager.DaemonReload()

			// Enable service if requested
			serviceName := f.generator.ServiceName(mount.Name, "mount") + ".service"
			if mount.Enabled {
				_ = f.manager.Enable(serviceName)
			}

			// Start service if auto-start is enabled
			if mount.AutoStart {
				_ = f.manager.Start(serviceName)
			}
		}
	}

	f.done = true

	if f.isEdit {
		return MountUpdatedMsg{Mount: mount}
	}
	return MountCreatedMsg{Mount: mount}
}

// IsDone returns true if the form is done.
func (f *MountForm) IsDone() bool {
	return f.done
}

// View renders the form.
func (f *MountForm) View() string {
	if f.done {
		return ""
	}

	// Render the form
	formView := f.form.View()

	// Add header
	title := "Create New Mount"
	if f.isEdit {
		title = "Edit Mount: " + f.name
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
