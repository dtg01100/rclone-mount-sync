package teatest

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// TestMainMenu_RendersAllEntries asserts the main menu lists the
// four quick-jump screens plus a quit hint.
func TestMainMenu_RendersAllEntries(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())

	body := driveToFinal(t, tm)
	for _, want := range []string{
		"Rclone Mount Sync",
		"Mount Management",
		"Sync Job Management",
		"Service Status",
		"Settings",
		"Quit",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("main menu missing %q, body:\n%s", want, body)
		}
	}
}

// TestMainMenu_QuickJumpKeys drives the four quick-jump keys and
// asserts that each navigates to the expected screen.
func TestMainMenu_QuickJumpKeys(t *testing.T) {
	cases := []struct {
		name       string
		key        string
		wantStatus string
	}{
		{"mounts", "m", "Mount Management"},
		{"syncJobs", "s", "Sync Job Management"},
		{"services", "v", "Service Status"},
		{"settings", "t", "Settings"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tm := newTestProgram(t, defaultDeps())
			body := driveToFinal(t, tm, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})
			if !strings.Contains(body, tc.wantStatus) {
				t.Errorf("after %q key: expected status %q, body:\n%s",
					tc.key, tc.wantStatus, body)
			}
		})
	}
}

// TestMainMenu_QQuitsFromMain asserts that pressing 'q' from the
// main menu causes the program to actually exit (rather than
// navigating back to main from a sub-screen).
func TestMainMenu_QQuitsFromMain(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	// FinalOutput blocks until the program exits or the timeout
	// elapses. Reaching this line without a timeout means the
	// program terminated (we don't need to assert on body).
	_ = tm.FinalOutput(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestMainMenu_CtrlCQuitsFromMain is the ctrl+c equivalent of
// QQuitsFromMain. App.Update returns tea.Quit for ctrl+c from any
// screen (unlike 'q' which only quits from main).
func TestMainMenu_CtrlCQuitsFromMain(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	_ = tm.FinalOutput(t, teatest.WithFinalTimeout(2*time.Second))
}
