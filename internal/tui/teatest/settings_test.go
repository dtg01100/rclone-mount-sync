package teatest

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// waitForSettingsLoaded sleeps briefly so the settings screen's
// init has a chance to run. The settings screen is mostly
// synchronous and only triggers imports/exports/file-picker
// flows on user input, so a 300ms yield is plenty.
func waitForSettingsLoaded(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	time.Sleep(300 * time.Millisecond)
}

// TestSettings_RendersList opens the settings screen via the 't'
// quick-jump key and asserts the status bar flips to Settings.
func TestSettings_RendersList(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	waitForSettingsLoaded(t, tm)

	body := driveToFinal(t, tm)
	if !strings.Contains(body, "Screen: Settings") {
		t.Errorf("settings screen missing, body:\n%s", body)
	}
}

// TestSettings_EscReturnsToMain presses Esc and asserts the
// screen flips back to main.
func TestSettings_EscReturnsToMain(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	waitForSettingsLoaded(t, tm)

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyEsc},
	)
	if !strings.Contains(body, "Screen: Main Menu") {
		t.Errorf("after Esc, expected main menu, body:\n%s", body)
	}
}

// TestSettings_CursorNavigation exercises down/up keys; the
// settings screen must not crash and must remain on Settings.
func TestSettings_CursorNavigation(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	waitForSettingsLoaded(t, tm)

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")},
	)
	if !strings.Contains(body, "Screen: Settings") {
		t.Errorf("settings screen not present after cursor nav, body:\n%s", body)
	}
}
