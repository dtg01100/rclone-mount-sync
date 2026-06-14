package teatest

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestHelp_OpensWithQuestionMark presses '?' from the main menu
// and asserts the help overlay lists each keybinding group
// (Global, Screen Navigation, Mount Management, Sync Job
// Management, Service Status).
func TestHelp_OpensWithQuestionMark(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())

	body := driveToFinal(t, tm, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	for _, want := range []string{
		"Help & Keybindings",
		"Global Keybindings",
		"Screen Navigation",
		"Mount Management",
		"Sync Job Management",
		"Service Status",
		"Ctrl+C", // appears in the help body
	} {
		if !strings.Contains(body, want) {
			t.Errorf("help missing %q, body:\n%s", want, body)
		}
	}
}

// TestHelp_EscClosesHelp opens help, then presses Esc, and
// asserts the help overlay disappears. The main-menu status
// bar must come back.
func TestHelp_EscClosesHelp(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")},
		tea.KeyMsg{Type: tea.KeyEsc},
	)
	if !strings.Contains(body, "Screen: Main Menu") {
		t.Errorf("after Esc, expected main menu, body:\n%s", body)
	}
}

// TestHelp_QClosesHelp opens help, then presses 'q', and
// asserts the overlay is closed.
func TestHelp_QClosesHelp(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")},
	)
	if !strings.Contains(body, "Screen: Main Menu") {
		t.Errorf("after q from help, expected main menu, body:\n%s", body)
	}
}
