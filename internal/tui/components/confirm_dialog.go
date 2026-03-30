// Package components provides shared UI components for the TUI.
package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmDialogOption represents a button option in the confirmation dialog.
type ConfirmDialogOption struct {
	Label         string
	Action        int
	IsDestructive bool // If true, uses error styling
}

// ConfirmDialogConfig holds the configuration for a confirmation dialog.
type ConfirmDialogConfig struct {
	Title       string
	Message     string
	Description string // Optional additional description
	Options     []ConfirmDialogOption
	Width       int
}

// ConfirmDialog is a reusable confirmation dialog for destructive actions.
type ConfirmDialog struct {
	config   ConfirmDialogConfig
	cursor   int
	done     bool
	selected int // The selected option index
}

// NewConfirmDialog creates a new confirmation dialog.
func NewConfirmDialog(config ConfirmDialogConfig) *ConfirmDialog {
	return &ConfirmDialog{
		config:   config,
		cursor:   0,
		selected: 0,
	}
}

// SetSize sets the dialog dimensions.
func (d *ConfirmDialog) SetSize(width, height int) {
	d.config.Width = width
}

// Init initializes the dialog.
func (d *ConfirmDialog) Init() tea.Cmd {
	return nil
}

// Update handles updates.
func (d *ConfirmDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if d.cursor > 0 {
				d.cursor--
			}
		case "right", "l":
			if d.cursor < len(d.config.Options)-1 {
				d.cursor++
			}
		case "enter":
			d.selected = d.cursor
			d.done = true
		case "esc":
			d.done = true
		}
	}

	return d, nil
}

// IsDone returns true if the dialog is done.
func (d *ConfirmDialog) IsDone() bool {
	return d.done
}

// SelectedOption returns the index of the selected option.
func (d *ConfirmDialog) SelectedOption() int {
	return d.selected
}

// View renders the dialog.
func (d *ConfirmDialog) View() string {
	var b strings.Builder

	// Title
	title := Styles.Title.Render(d.config.Title)
	b.WriteString(lipgloss.NewStyle().
		Width(d.config.Width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	// Warning message
	warning := RenderWarning(d.config.Message)
	b.WriteString(lipgloss.NewStyle().
		Width(d.config.Width).
		Align(lipgloss.Center).
		Render(warning))
	b.WriteString("\n\n")

	// Optional description
	if d.config.Description != "" {
		desc := Styles.Normal.Render(d.config.Description)
		b.WriteString(lipgloss.NewStyle().
			Width(d.config.Width).
			Align(lipgloss.Center).
			Render(desc))
		b.WriteString("\n\n")
	}

	// Options
	var optionStrs []string
	for i, opt := range d.config.Options {
		var style lipgloss.Style
		if i == d.cursor {
			if opt.IsDestructive {
				style = Styles.ButtonFocus.Copy().Background(ColorError)
			} else {
				style = Styles.ButtonFocus
			}
		} else {
			if opt.IsDestructive {
				style = Styles.Button.Copy().Foreground(ColorError)
			} else {
				style = Styles.Button
			}
		}
		optionStrs = append(optionStrs, style.Render(opt.Label))
	}

	optionsLine := strings.Join(optionStrs, "  ")
	b.WriteString(lipgloss.NewStyle().
		Width(d.config.Width).
		Align(lipgloss.Center).
		Render(optionsLine))
	b.WriteString("\n\n")

	// Help
	help := Styles.HelpText.Render("←/→: select option  Enter: confirm  Esc: cancel")
	b.WriteString(lipgloss.NewStyle().
		Width(d.config.Width).
		Align(lipgloss.Center).
		Render(help))

	return b.String()
}

// GetSelectedAction returns the action of the selected option.
func (d *ConfirmDialog) GetSelectedAction() int {
	if d.selected >= 0 && d.selected < len(d.config.Options) {
		return d.config.Options[d.selected].Action
	}
	return -1
}

// Helper functions for common confirmation dialog configurations

// NewDeleteConfirmDialog creates a standard delete confirmation dialog.
// Options: [0] Cancel, [1] Delete Service Only, [2] Delete Service and Config
func NewDeleteConfirmDialog(itemName, itemType string) *ConfirmDialog {
	return NewConfirmDialog(ConfirmDialogConfig{
		Title:   fmt.Sprintf("Delete %s", itemType),
		Message: fmt.Sprintf("Are you sure you want to delete '%s'?", itemName),
		Options: []ConfirmDialogOption{
			{Label: "Cancel", Action: 0, IsDestructive: false},
			{Label: "Delete Service Only", Action: 1, IsDestructive: true},
			{Label: "Delete Service and Config", Action: 2, IsDestructive: true},
		},
	})
}

// NewSimpleConfirmDialog creates a simple yes/no confirmation dialog.
// Options: [0] No, [1] Yes
func NewSimpleConfirmDialog(title, message string) *ConfirmDialog {
	return NewConfirmDialog(ConfirmDialogConfig{
		Title:   title,
		Message: message,
		Options: []ConfirmDialogOption{
			{Label: "No", Action: 0, IsDestructive: false},
			{Label: "Yes", Action: 1, IsDestructive: true},
		},
	})
}

// NewActionConfirmDialog creates a confirmation dialog with custom options.
func NewActionConfirmDialog(title, message string, options []ConfirmDialogOption) *ConfirmDialog {
	return NewConfirmDialog(ConfirmDialogConfig{
		Title:   title,
		Message: message,
		Options: options,
	})
}
