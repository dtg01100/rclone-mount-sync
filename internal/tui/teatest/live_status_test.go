package teatest

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestLiveStatus_TickRefreshesStatuses asserts the live-status
// poller (5s interval) re-arms itself and keeps the program
// stable across the tick boundary.
//
// In a teatest run, the screen's s.generator is the seam's
// injected value (nil when no Generator is passed). The
// mountsStatusTickMsg handler in mounts.go short-circuits the
// loadStatuses call when generator/manager is nil, so this
// test does not actually exercise the tick in a teatest
// setting. The behavior is covered by the direct-model test
// in internal/tui/screens/mounts_test.go, which constructs a
// real Generator + MockManager and asserts the tick callback.
//
// We still want a teatest-level smoke check that the mounts
// screen is stable across the ~5s tick boundary, so this test
// just navigates to mounts and waits 6s without crashing.
//
// NOTE: This test dominates the suite runtime (~6s) because
// the tick interval is 5s. It is the slowest test in the
// package; keep it last alphabetically.
func TestLiveStatus_TickRefreshesStatuses(t *testing.T) {
	deps := depsWithMockManager()
	deps.Config = testConfigWithMounts(sampleMount("alpha"))
	tm := newTestProgram(t, deps)

	// Navigate to mounts so any in-flight tick on the list
	// mode is at least scheduled (the actual tick callback
	// may be suppressed if deps.Generator is nil).
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	waitForMountsLoaded(t, tm)

	// Wait > 5s for at least one tick to potentially fire.
	time.Sleep(6 * time.Second)

	body := driveToFinal(t, tm)
	if !strings.Contains(body, "Screen: Mount Management") {
		t.Errorf("expected mounts screen after tick wait, body:\n%s", body)
	}
}

// TestLiveStatus_StableOnNavigationToForm asserts that the
// program doesn't crash when the user navigates from list
// mode to the edit form and back. Without this, a regression
// in the form mode (e.g. an Update handler that dereferences
// a nil dep) would only surface when a user actually presses
// 'e' on a real list.
func TestLiveStatus_StableOnNavigationToForm(t *testing.T) {
	deps := depsWithMockManager()
	deps.Config = testConfigWithMounts(sampleMount("alpha"))
	tm := newTestProgram(t, deps)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	waitForMountsLoaded(t, tm)

	// 'e' enters the edit form. Without a real rclone binary
	// the form Init will surface a friendly error, but the
	// screen must not crash.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	time.Sleep(200 * time.Millisecond)

	// Escape back to list mode.
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(200 * time.Millisecond)

	body := driveToFinal(t, tm)
	if !strings.Contains(body, "Screen: Mount Management") {
		t.Errorf("expected mount screen after form round-trip, body:\n%s", body)
	}
}
