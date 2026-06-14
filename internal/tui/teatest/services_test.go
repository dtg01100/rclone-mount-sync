package teatest

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// waitForServicesLoaded sleeps until the services screen has had
// a chance to receive the ServicesLoadedMsg and re-render.
// Mirrors waitForMountsLoaded.
func waitForServicesLoaded(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	time.Sleep(700 * time.Millisecond)
}

// TestServices_EmptyList renders the services screen and asserts
// the status bar flips to Service Status.
func TestServices_EmptyList(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	waitForServicesLoaded(t, tm)

	body := driveToFinal(t, tm)
	if !strings.Contains(body, "Screen: Service Status") {
		t.Errorf("services screen missing, body:\n%s", body)
	}
}

// TestServices_EscReturnsToMain presses Esc and asserts the
// screen flips back to main.
func TestServices_EscReturnsToMain(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	waitForServicesLoaded(t, tm)

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyEsc},
	)
	if !strings.Contains(body, "Screen: Main Menu") {
		t.Errorf("after Esc, expected main menu, body:\n%s", body)
	}
}

// TestServices_StableAcrossHotkeys exercises a handful of
// services-list hotkeys against an empty service list and
// asserts no crash. The screen guards actions on
// len(s.filteredServices) > 0.
func TestServices_StableAcrossHotkeys(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	waitForServicesLoaded(t, tm)

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")},
	)
	if !strings.Contains(body, "Screen: Service Status") {
		t.Errorf("services screen not stable after hotkey spam, body:\n%s", body)
	}
}
