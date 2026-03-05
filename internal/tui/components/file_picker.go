// Package components provides shared UI components for the TUI.
package components

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// FileEntry represents a file or directory entry in the file picker.
type FileEntry struct {
	Name     string
	Path     string
	IsDir    bool
	Selected bool
}

// recentPaths stores recently visited paths for the quick jump feature.
var (
	recentPaths    []string
	recentPathsMu  sync.Mutex
	maxRecentPaths = 10
)

// EnhancedFilePicker provides an improved file browsing experience with:
// - Visual indicators for files and folders
// - Breadcrumb navigation
// - Quick jump shortcuts
// - Recent locations tracking
type EnhancedFilePicker struct {
	// Configuration
	title       string
	description string
	dirAllowed  bool
	fileAllowed bool
	currentDir  string
	showHidden  bool
	validate    func(string) error

	// Internal state
	entries      []FileEntry
	cursor       int
	selectedPath *string
	width        int
	height       int
	focused      bool
	err          error
	accessible   bool
	position     huh.FieldPosition

	// Quick jump state
	showRecentMenu bool
	recentCursor   int

	// Internal file picker
	innerPicker *huh.FilePicker
}

// NewEnhancedFilePicker creates a new enhanced file picker.
func NewEnhancedFilePicker() *EnhancedFilePicker {
	return &EnhancedFilePicker{
		dirAllowed:  true,
		fileAllowed: true,
		showHidden:  false,
		focused:     true,
	}
}

// Title sets the title of the file picker.
func (p *EnhancedFilePicker) Title(title string) *EnhancedFilePicker {
	p.title = title
	return p
}

// Description sets the description of the file picker.
func (p *EnhancedFilePicker) Description(desc string) *EnhancedFilePicker {
	p.description = desc
	return p
}

// DirAllowed sets whether directories can be selected.
func (p *EnhancedFilePicker) DirAllowed(allowed bool) *EnhancedFilePicker {
	p.dirAllowed = allowed
	return p
}

// FileAllowed sets whether files can be selected.
func (p *EnhancedFilePicker) FileAllowed(allowed bool) *EnhancedFilePicker {
	p.fileAllowed = allowed
	return p
}

// CurrentDirectory sets the starting directory.
func (p *EnhancedFilePicker) CurrentDirectory(dir string) *EnhancedFilePicker {
	p.currentDir = ExpandHome(dir)
	return p
}

// Value sets the pointer to store the selected path.
func (p *EnhancedFilePicker) Value(value *string) *EnhancedFilePicker {
	p.selectedPath = value
	return p
}

// Validate sets the validation function for the selected path.
func (p *EnhancedFilePicker) Validate(validate func(string) error) *EnhancedFilePicker {
	p.validate = validate
	return p
}

// ShowHidden sets whether to show hidden files.
func (p *EnhancedFilePicker) ShowHidden(show bool) *EnhancedFilePicker {
	p.showHidden = show
	return p
}

// initInnerPicker initializes the inner huh file picker.
func (p *EnhancedFilePicker) initInnerPicker() {
	// Set default current directory if not set
	if p.currentDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			p.currentDir = "/"
		} else {
			p.currentDir = homeDir
		}
	}

	p.innerPicker = huh.NewFilePicker().
		Title(p.title).
		Description(p.description).
		DirAllowed(p.dirAllowed).
		FileAllowed(p.fileAllowed).
		CurrentDirectory(p.currentDir).
		ShowHidden(p.showHidden)

	if p.selectedPath != nil {
		p.innerPicker.Value(p.selectedPath)
	}
	if p.validate != nil {
		p.innerPicker.Validate(p.validate)
	}
}

// Init initializes the file picker.
func (p *EnhancedFilePicker) Init() tea.Cmd {
	p.initInnerPicker()
	return p.innerPicker.Init()
}

// Update handles messages for the file picker.
// This implements the huh.Field interface.
func (p *EnhancedFilePicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle quick jump keys when the picker is active and not in recent menu
	if !p.showRecentMenu {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "~":
				// Jump to home directory
				homeDir, err := os.UserHomeDir()
				if err == nil {
					return p, p.jumpToDirectory(homeDir)
				}
				// Cannot determine home directory, ignore the keypress
				return p, nil
			case "/":
				// Jump to root directory
				return p, p.jumpToDirectory("/")
			case "m":
				// Jump to /mnt or ~/mnt
				if _, err := os.Stat("/mnt"); err == nil {
					return p, p.jumpToDirectory("/mnt")
				}
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return p, nil
				}
				mntDir := filepath.Join(homeDir, "mnt")
				if _, err := os.Stat(mntDir); err == nil {
					return p, p.jumpToDirectory(mntDir)
				}
			case "M":
				// Jump to /media
				if _, err := os.Stat("/media"); err == nil {
					return p, p.jumpToDirectory("/media")
				}
			case "r":
				// Toggle recent locations menu
				if len(GetRecentPaths()) > 0 {
					p.showRecentMenu = true
					p.recentCursor = 0
					return p, nil
				}
			case "backspace":
				// Go to parent directory
				parentDir := GetParentDirectory(p.getCurrentDirectory())
				if parentDir != p.getCurrentDirectory() {
					return p, p.jumpToDirectory(parentDir)
				}
			}
		}
	}

	// Handle recent menu navigation
	if p.showRecentMenu {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			recentPathsList := GetRecentPaths()
			switch keyMsg.String() {
			case "up", "k":
				if p.recentCursor > 0 {
					p.recentCursor--
				}
				return p, nil
			case "down", "j":
				if p.recentCursor < len(recentPathsList)-1 {
					p.recentCursor++
				}
				return p, nil
			case "enter":
				if p.recentCursor >= 0 && p.recentCursor < len(recentPathsList) {
					selectedPath := recentPathsList[p.recentCursor]
					p.showRecentMenu = false
					return p, p.jumpToDirectory(ExpandHome(selectedPath))
				}
			case "esc":
				p.showRecentMenu = false
				return p, nil
			}
		}
		return p, nil
	}

	// Update inner picker
	model, cmd := p.innerPicker.Update(msg)
	if fp, ok := model.(*huh.FilePicker); ok {
		p.innerPicker = fp
	}
	return p, cmd
}

// jumpToDirectory creates a command to jump to a specific directory.
func (p *EnhancedFilePicker) jumpToDirectory(dir string) tea.Cmd {
	p.currentDir = dir
	AddRecentPath(dir)
	p.initInnerPicker()
	return p.innerPicker.Init()
}

// getCurrentDirectory returns the current directory from the inner picker.
func (p *EnhancedFilePicker) getCurrentDirectory() string {
	if p.innerPicker != nil {
		// The inner picker doesn't expose CurrentDirectory as a field,
		// so we use our stored currentDir
		return p.currentDir
	}
	return p.currentDir
}

// View renders the file picker.
func (p *EnhancedFilePicker) View() string {
	// Initialize inner picker if not already done
	if p.innerPicker == nil {
		p.initInnerPicker()
	}

	var b strings.Builder

	// Render breadcrumb bar
	b.WriteString(p.renderBreadcrumb())
	b.WriteString("\n")

	// Render quick jump bar
	b.WriteString(p.renderQuickJumpBar())
	b.WriteString("\n")

	// Render recent menu if active
	if p.showRecentMenu {
		b.WriteString(p.renderRecentMenu())
		b.WriteString("\n")
	}

	// Render the inner file picker
	b.WriteString(p.innerPicker.View())

	// Render help bar
	b.WriteString("\n")
	b.WriteString(p.renderHelpBar())

	return b.String()
}

// renderBreadcrumb renders the breadcrumb navigation bar.
func (p *EnhancedFilePicker) renderBreadcrumb() string {
	currentDir := p.getCurrentDirectory()
	segments := GetBreadcrumbSegments(currentDir)

	var parts []string
	homeIcon := FilePickerStyles.Breadcrumb.Render("🏠")
	parts = append(parts, homeIcon)

	for _, seg := range segments {
		sep := FilePickerStyles.BreadcrumbSep.Render(">")
		part := FilePickerStyles.Breadcrumb.Render(seg)
		parts = append(parts, sep, part)
	}

	breadcrumb := lipgloss.JoinHorizontal(lipgloss.Left, parts...)

	// Create the full breadcrumb bar with background
	bar := FilePickerStyles.BreadcrumbBar.Width(p.width).Render(breadcrumb)
	return bar
}

// renderQuickJumpBar renders the quick jump shortcuts bar.
func (p *EnhancedFilePicker) renderQuickJumpBar() string {
	var buttons []string

	// Home button
	buttons = append(buttons, p.renderQuickJumpButton("~", "Home"))

	// Root button
	buttons = append(buttons, p.renderQuickJumpButton("/", "Root"))

	// Mnt button
	buttons = append(buttons, p.renderQuickJumpButton("m", "mnt"))

	// Media button
	buttons = append(buttons, p.renderQuickJumpButton("M", "media"))

	// Recent button (only if there are recent paths)
	if len(GetRecentPaths()) > 0 {
		buttons = append(buttons, p.renderQuickJumpButton("r", "Recent"))
	}

	content := lipgloss.JoinHorizontal(lipgloss.Left, buttons...)
	bar := FilePickerStyles.QuickJumpBar.Width(p.width).Render(content)
	return bar
}

// renderQuickJumpButton renders a single quick jump button.
func (p *EnhancedFilePicker) renderQuickJumpButton(keyStr, label string) string {
	keyStyle := FilePickerStyles.QuickJumpKey.Render("[" + keyStr + "]")
	labelStyle := FilePickerStyles.QuickJumpLabel.Render(label)
	return lipgloss.JoinHorizontal(lipgloss.Left, keyStyle, labelStyle, " ")
}

// renderRecentMenu renders the recent locations dropdown menu.
func (p *EnhancedFilePicker) renderRecentMenu() string {
	recentPathsList := GetRecentPaths()
	if len(recentPathsList) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, FilePickerStyles.RecentMenuHeader.Render("Recent Locations:"))

	for i, path := range recentPathsList {
		// Truncate path if too long
		displayPath := path
		maxLen := p.width - 6
		if maxLen < 20 {
			maxLen = 20
		}
		if len(displayPath) > maxLen {
			displayPath = "..." + displayPath[len(displayPath)-maxLen+3:]
		}

		if i == p.recentCursor {
			line := FilePickerStyles.RecentMenuItemSelected.Render("▸ " + displayPath)
			lines = append(lines, line)
		} else {
			line := FilePickerStyles.RecentMenuItem.Render("  " + displayPath)
			lines = append(lines, line)
		}
	}

	return FilePickerStyles.RecentMenu.Width(p.width).Render(
		strings.Join(lines, "\n"),
	)
}

// renderHelpBar renders the help bar with keybindings.
func (p *EnhancedFilePicker) renderHelpBar() string {
	items := []HelpItem{
		{Key: "~", Desc: "home"},
		{Key: "/", Desc: "root"},
		{Key: "m", Desc: "mnt"},
		{Key: "M", Desc: "media"},
		{Key: "Backspace", Desc: "parent"},
	}

	if len(GetRecentPaths()) > 0 {
		items = append(items, HelpItem{Key: "r", Desc: "recent"})
	}

	items = append(items, HelpItem{Key: "Enter", Desc: "select"})
	items = append(items, HelpItem{Key: "Esc", Desc: "cancel"})

	return HelpBar(p.width, items)
}

// Error returns any error from the file picker.
func (p *EnhancedFilePicker) Error() error {
	return p.err
}

// Skip returns whether this field should be skipped.
func (p *EnhancedFilePicker) Skip() bool {
	return false
}

// Zoom returns whether this field should be zoomed.
func (p *EnhancedFilePicker) Zoom() bool {
	return false
}

// Focus focuses the file picker.
func (p *EnhancedFilePicker) Focus() tea.Cmd {
	p.focused = true
	if p.innerPicker != nil {
		return p.innerPicker.Focus()
	}
	return nil
}

// Blur blurs the file picker.
func (p *EnhancedFilePicker) Blur() tea.Cmd {
	p.focused = false
	if p.innerPicker != nil {
		return p.innerPicker.Blur()
	}
	return nil
}

// KeyBinds returns the key bindings for help display.
func (p *EnhancedFilePicker) KeyBinds() []key.Binding {
	if p.innerPicker != nil {
		return p.innerPicker.KeyBinds()
	}
	return nil
}

// WithTheme applies a theme to the file picker.
func (p *EnhancedFilePicker) WithTheme(theme *huh.Theme) huh.Field {
	if p.innerPicker != nil {
		p.innerPicker.WithTheme(theme)
	}
	return p
}

// WithKeyMap sets the key map for the file picker.
func (p *EnhancedFilePicker) WithKeyMap(keyMap *huh.KeyMap) huh.Field {
	if p.innerPicker != nil {
		p.innerPicker.WithKeyMap(keyMap)
	}
	return p
}

// WithWidth sets the width of the file picker.
func (p *EnhancedFilePicker) WithWidth(width int) huh.Field {
	p.width = width
	if p.innerPicker != nil {
		p.innerPicker.WithWidth(width)
	}
	return p
}

// WithHeight sets the height of the file picker.
func (p *EnhancedFilePicker) WithHeight(height int) huh.Field {
	p.height = height
	if p.innerPicker != nil {
		p.innerPicker.WithHeight(height)
	}
	return p
}

// WithPosition sets the field position in the form.
func (p *EnhancedFilePicker) WithPosition(pos huh.FieldPosition) huh.Field {
	p.position = pos
	if p.innerPicker != nil {
		p.innerPicker.WithPosition(pos)
	}
	return p
}

// WithAccessible sets whether the field should run in accessible mode.
func (p *EnhancedFilePicker) WithAccessible(accessible bool) huh.Field {
	p.accessible = accessible
	if p.innerPicker != nil {
		p.innerPicker.WithAccessible(accessible)
	}
	return p
}

// GetValue returns the currently selected value.
func (p *EnhancedFilePicker) GetValue() any {
	if p.selectedPath != nil {
		return *p.selectedPath
	}
	return ""
}

// GetKey returns the key for the field.
func (p *EnhancedFilePicker) GetKey() string {
	return ""
}

// Run runs the file picker as a standalone program.
func (p *EnhancedFilePicker) Run() error {
	return huh.NewForm(huh.NewGroup(p)).Run()
}

// RunAccessible runs the field in accessible mode.
func (p *EnhancedFilePicker) RunAccessible(w io.Writer, r io.Reader) error {
	if p.innerPicker != nil {
		return p.innerPicker.RunAccessible(w, r)
	}
	return nil
}

// Ensure EnhancedFilePicker implements huh.Field interface
var _ huh.Field = (*EnhancedFilePicker)(nil)

// Recent path management functions

// GetRecentPaths returns the list of recently visited paths.
func GetRecentPaths() []string {
	recentPathsMu.Lock()
	defer recentPathsMu.Unlock()

	// Return a copy to avoid race conditions
	result := make([]string, len(recentPaths))
	copy(result, recentPaths)
	return result
}

// AddRecentPath adds a path to the recent paths list.
func AddRecentPath(path string) {
	if path == "" {
		return
	}

	// Expand the path
	expandedPath := ExpandHome(path)

	// Contract for display
	displayPath := ContractHome(expandedPath)

	recentPathsMu.Lock()
	defer recentPathsMu.Unlock()

	// Remove if already exists
	for i, rp := range recentPaths {
		if ExpandHome(rp) == expandedPath {
			recentPaths = append(recentPaths[:i], recentPaths[i+1:]...)
			break
		}
	}

	// Add to front
	recentPaths = append([]string{displayPath}, recentPaths...)

	// Trim to max size
	if len(recentPaths) > maxRecentPaths {
		recentPaths = recentPaths[:maxRecentPaths]
	}
}

// ClearRecentPaths clears all recent paths.
func ClearRecentPaths() {
	recentPathsMu.Lock()
	defer recentPathsMu.Unlock()
	recentPaths = nil
}

// SetRecentPaths sets the recent paths list (used for loading from config).
func SetRecentPaths(paths []string) {
	recentPathsMu.Lock()
	defer recentPathsMu.Unlock()
	recentPaths = make([]string, len(paths))
	copy(recentPaths, paths)
}

// FormatPathForDisplay formats a path for display in the UI.
func FormatPathForDisplay(path string) string {
	if path == "" {
		return ""
	}
	return ContractHome(ExpandHome(path))
}

// ValidateDirectoryPath validates that a path is a valid directory.
func ValidateDirectoryPath(path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}

	expandedPath := ExpandHome(path)

	// Check if path exists
	info, err := os.Stat(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", path)
		}
		return fmt.Errorf("cannot access path: %w", err)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	return nil
}

// ValidateFilePath validates that a path is a valid file path (parent exists).
func ValidateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}

	expandedPath := ExpandHome(path)

	// Check if parent directory exists
	parentDir := filepath.Dir(expandedPath)
	if _, err := os.Stat(parentDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("parent directory does not exist: %s", parentDir)
		}
		return fmt.Errorf("cannot access parent directory: %w", err)
	}

	return nil
}
