// Package components provides shared UI components for the TUI.
package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette - based on a professional dark theme
var (
	// Primary colors
	ColorPrimary    = lipgloss.Color("62")  // Muted blue
	ColorPrimaryBright = lipgloss.Color("75") // Brighter blue
	ColorAccent     = lipgloss.Color("86")  // Cyan/teal
	ColorBackground = lipgloss.Color("235") // Dark gray background
	ColorSurface    = lipgloss.Color("236") // Slightly lighter surface

	// Text colors
	ColorText       = lipgloss.Color("252") // Light gray text
	ColorTextMuted  = lipgloss.Color("243") // Muted gray
	ColorTextBright = lipgloss.Color("15")  // White

	// Semantic colors
	ColorSuccess    = lipgloss.Color("82")  // Green
	ColorWarning    = lipgloss.Color("214") // Orange
	ColorError      = lipgloss.Color("196") // Red
	ColorInfo       = lipgloss.Color("117") // Light blue
)

// Styles contains common styling for the TUI.
var Styles = struct {
	// Base styles
	Title      lipgloss.Style
	Subtitle   lipgloss.Style
	Normal     lipgloss.Style
	Selected   lipgloss.Style
	Deselected lipgloss.Style

	// Semantic styles
	Error   lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style

	// UI element styles
	Border     lipgloss.Style
	HelpText   lipgloss.Style
	StatusLine lipgloss.Style
	Header     lipgloss.Style
	Box        lipgloss.Style

	// Menu styles
	MenuItem    lipgloss.Style
	MenuSelected lipgloss.Style
	MenuKey     lipgloss.Style

	// Button styles
	Button      lipgloss.Style
	ButtonFocus lipgloss.Style

	// Input styles
	Input       lipgloss.Style
	InputFocus  lipgloss.Style
	InputLabel  lipgloss.Style

	// Status indicator styles
	StatusActive   lipgloss.Style
	StatusInactive lipgloss.Style
	StatusError    lipgloss.Style
}{
	// Base styles
	Title: lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorTextBright).
		Background(ColorPrimary).
		Padding(0, 2),
	Subtitle: lipgloss.NewStyle().
		Italic(true).
		Foreground(ColorTextMuted),
	Normal: lipgloss.NewStyle().
		Foreground(ColorText),
	Selected: lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent),
	Deselected: lipgloss.NewStyle().
		Foreground(ColorTextMuted),

	// Semantic styles
	Error: lipgloss.NewStyle().
		Foreground(ColorError),
	Success: lipgloss.NewStyle().
		Foreground(ColorSuccess),
	Warning: lipgloss.NewStyle().
		Foreground(ColorWarning),
	Info: lipgloss.NewStyle().
		Foreground(ColorInfo),

	// UI element styles
	Border: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1),
	HelpText: lipgloss.NewStyle().
		Italic(true).
		Foreground(ColorTextMuted),
	StatusLine: lipgloss.NewStyle().
		Foreground(ColorTextBright).
		Background(ColorSurface).
		Padding(0, 1),
	Header: lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorTextBright).
		Background(ColorPrimary).
		Padding(0, 1),
	Box: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2),

	// Menu styles
	MenuItem: lipgloss.NewStyle().
		Foreground(ColorText),
	MenuSelected: lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent),
	MenuKey: lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimaryBright),

	// Button styles
	Button: lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorSurface).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorTextMuted),
	ButtonFocus: lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorTextBright).
		Background(ColorPrimary).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorAccent),

	// Input styles
	Input: lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorSurface).
		Padding(0, 1),
	InputFocus: lipgloss.NewStyle().
		Foreground(ColorTextBright).
		Background(ColorPrimary).
		Padding(0, 1),
	InputLabel: lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorText),

	// Status indicator styles
	StatusActive: lipgloss.NewStyle().
		Foreground(ColorSuccess),
	StatusInactive: lipgloss.NewStyle().
		Foreground(ColorTextMuted),
	StatusError: lipgloss.NewStyle().
		Foreground(ColorError),
}

// MenuItem represents a menu item with label, description, and key binding.
type MenuItem struct {
	Label       string
	Description string
	Key         string
}

// Menu represents a navigable menu.
type Menu struct {
	Items    []MenuItem
	Cursor   int
	Width    int
	ShowKeys bool
}

// NewMenu creates a new menu with the given items.
func NewMenu(items []MenuItem) *Menu {
	return &Menu{
		Items:    items,
		Cursor:   0,
		ShowKeys: true,
	}
}

// SetWidth sets the menu width.
func (m *Menu) SetWidth(width int) {
	m.Width = width
}

// Up moves the cursor up.
func (m *Menu) Up() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

// Down moves the cursor down.
func (m *Menu) Down() {
	if m.Cursor < len(m.Items)-1 {
		m.Cursor++
	}
}

// Selected returns the currently selected menu item.
func (m *Menu) Selected() MenuItem {
	if m.Cursor >= 0 && m.Cursor < len(m.Items) {
		return m.Items[m.Cursor]
	}
	return MenuItem{}
}

// Render renders the menu with styling.
func (m *Menu) Render() string {
	var b strings.Builder

	for i, item := range m.Items {
		var line string
		if i == m.Cursor {
			// Selected item
			cursor := Styles.MenuSelected.Render("▸")
			key := ""
			if m.ShowKeys && item.Key != "" {
				key = Styles.MenuKey.Render("["+item.Key+"] ") 
			}
			label := Styles.MenuSelected.Render(item.Label)
			line = lipgloss.JoinHorizontal(lipgloss.Left, cursor, " ", key, label)
		} else {
			// Unselected item
			cursor := "  "
			key := ""
			if m.ShowKeys && item.Key != "" {
				key = Styles.MenuKey.Render("["+item.Key+"] ")
			}
			label := Styles.MenuItem.Render(item.Label)
			line = lipgloss.JoinHorizontal(lipgloss.Left, cursor, " ", key, label)
		}

		// Add description if present
		if item.Description != "" {
			desc := Styles.Subtitle.Render("    " + item.Description)
			line = line + "\n" + desc
		}

		b.WriteString(line + "\n")
	}

	return b.String()
}

// Button represents a clickable button.
type Button struct {
	Label string
	Focus bool
}

// NewButton creates a new button.
func NewButton(label string) *Button {
	return &Button{Label: label}
}

// Render renders the button with styling.
func (b *Button) Render() string {
	if b.Focus {
		return Styles.ButtonFocus.Render(b.Label)
	}
	return Styles.Button.Render(b.Label)
}

// HelpItem represents a help item with key and description.
type HelpItem struct {
	Key  string
	Desc string
}

// HelpBar renders a help bar showing keybindings.
func HelpBar(width int, items []HelpItem) string {
	var parts []string
	for _, item := range items {
		part := Styles.MenuKey.Render(item.Key) + Styles.HelpText.Render(" "+item.Desc)
		parts = append(parts, part)
	}

	content := strings.Join(parts, Styles.HelpText.Render(" • "))
	
	// Truncate if too wide
	if lipgloss.Width(content) > width {
		content = content[:width-3] + "..."
	}

	return Styles.StatusLine.Width(width).Render(content)
}

// TitleBar renders a title bar with the application name and version.
func TitleBar(width int, title, version string) string {
	left := Styles.Header.Render(title)
	right := Styles.Subtitle.Render("v" + version + "  [?] Help  [q] Quit")
	
	// Calculate padding
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	padding := width - leftWidth - rightWidth
	
	if padding < 0 {
		padding = 0
	}

	return lipgloss.JoinHorizontal(lipgloss.Left,
		left,
		strings.Repeat(" ", padding),
		right,
	)
}

// StatusBar renders a status line at the bottom of the screen.
func StatusBar(width int, text string) string {
	return Styles.StatusLine.Width(width).Render(text)
}

// Box creates a bordered box with content.
func Box(title, content string, width int) string {
	boxStyle := Styles.Box.
		Width(width - 4).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true)

	if title != "" {
		boxStyle = boxStyle.BorderForeground(ColorPrimary)
	}

	return boxStyle.Render(content)
}

// Center centers text within a given width.
func Center(text string, width int) string {
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(text)
}

// PadLeft adds left padding to text.
func PadLeft(text string, padding int) string {
	return lipgloss.NewStyle().PaddingLeft(padding).Render(text)
}

// PadRight adds right padding to text.
func PadRight(text string, padding int) string {
	return lipgloss.NewStyle().PaddingRight(padding).Render(text)
}

// StatusIndicator returns a colored status indicator.
func StatusIndicator(status string) string {
	switch status {
	case "active", "running", "mounted":
		return Styles.StatusActive.Render("●")
	case "inactive", "stopped", "unmounted":
		return Styles.StatusInactive.Render("○")
	case "failed", "error":
		return Styles.StatusError.Render("✗")
	default:
		return Styles.StatusInactive.Render("○")
	}
}

// Truncate truncates text to fit within maxLen characters.
func Truncate(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}

// RenderTitle renders a title with consistent styling.
func RenderTitle(text string) string {
	return Styles.Title.Render(text)
}

// RenderError renders an error message.
func RenderError(text string) string {
	return Styles.Error.Render("✗ " + text)
}

// RenderSuccess renders a success message.
func RenderSuccess(text string) string {
	return Styles.Success.Render("✓ " + text)
}

// RenderWarning renders a warning message.
func RenderWarning(text string) string {
	return Styles.Warning.Render("⚠ " + text)
}

// RenderInfo renders an info message.
func RenderInfo(text string) string {
	return Styles.Info.Render("ℹ " + text)
}
