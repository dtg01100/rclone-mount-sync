package teatest

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
	"github.com/dtg01100/rclone-mount-sync/internal/tui"
)

// sendOrphans posts a ReconciliationMsg to the program so the
// App's Update handler flips showOrphanPrompt and renders the
// prompt. The dep-injected path (NewAppWithDeps) does not run
// initializeServices, so the test must inject the message
// directly to exercise the orphan-prompt flow.
func sendOrphans(tm *teatest.TestModel, units ...systemd.OrphanedUnit) {
	tm.Send(tui.ReconciliationMsg{
		Result: &systemd.ReconciliationResult{OrphanedUnits: units},
	})
}

// waitForOrphanPromptReady waits until the orphan prompt has
// been rendered. We detect readiness via the presence of the
// prompt banner "Orphaned Units Detected" in the output. The
// ReconciliationMsg handler in app.go also kicks off screen
// Init cmds that produce a flurry of async messages, so the
// wait is generous.
func waitForOrphanPromptReady(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Orphaned Units Detected")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(20*time.Millisecond))
	// A short extra yield so the keys sent immediately after
	// are processed in the right frame.
	time.Sleep(200 * time.Millisecond)
}

// TestOrphanPrompt_EnterThenImportList first posts an orphan
// list, then presses Enter to enter action mode, then Enter
// again to attempt import. Without a real generator/manager,
// the import is gated (returns OrphanActionMsg with
// "services not initialized" error). The test asserts the
// error gate in the prompt view is rendered.
func TestOrphanPrompt_EnterThenImportErrorGate(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())

	sendOrphans(tm,
		systemd.OrphanedUnit{
			Name: "rclone-mount-orphan.service", Type: "mount",
			Path: "/home/user/.config/systemd/user/rclone-mount-orphan.service",
			ID:   "orphan1", IsLegacy: false,
		},
	)
	waitForOrphanPromptReady(t, tm)
	// Generous yield between keypresses so the second Enter
	// sees orphanMode=1 (action menu) rather than orphanMode=0
	// (list) — otherwise it falls through to importSelectedOrphan
	// before the action-menu state is committed.
	time.Sleep(500 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // orphanMode 0 -> 1
	time.Sleep(500 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // triggers import
	// Generous final yield so the async Cmd's OrphanActionMsg
	// lands in the program and the error overlay renders.
	time.Sleep(1 * time.Second)

	body := driveToFinal(t, tm)
	// Assert the error message is rendered (the title is
	// scrolled out of the visible buffer by repeated cursor
	// moves, so we focus on the user-facing error).
	for _, want := range []string{
		"rclone-mount-orphan.service",
		"services not initialized",
		"Suggestions:",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("orphan prompt missing %q, body:\n%s", want, body)
		}
	}
}

// TestOrphanPrompt_EnterThenCleanupErrorGate is the cleanup
// counterpart to EnterThenImportErrorGate: Enter twice to enter
// the action menu, then 'c' to attempt cleanup.
func TestOrphanPrompt_EnterThenCleanupErrorGate(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())

	sendOrphans(tm,
		systemd.OrphanedUnit{
			Name: "rclone-mount-orphan.service", Type: "mount",
			Path: "/home/user/.config/systemd/user/rclone-mount-orphan.service",
			ID:   "orphan1", IsLegacy: false,
		},
	)
	waitForOrphanPromptReady(t, tm)
	time.Sleep(500 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // orphanMode 0 -> 1
	time.Sleep(500 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	time.Sleep(1 * time.Second)

	body := driveToFinal(t, tm)
	for _, want := range []string{
		"services not initialized",
		"Suggestions:",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("orphan prompt cleanup missing %q, body:\n%s", want, body)
		}
	}
}

// TestOrphanPrompt_SkipRemovesFromList presses 's' on the
// selected orphan and asserts the list shrinks. With two
// orphans and the cursor at 0, skipping removes the first
// one and the second shifts to position 0.
func TestOrphanPrompt_SkipRemovesFromList(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())

	sendOrphans(tm,
		systemd.OrphanedUnit{
			Name: "rclone-mount-orphan.service", Type: "mount",
			Path: "/home/user/.config/systemd/user/rclone-mount-orphan.service",
			ID:   "orphan1", IsLegacy: false,
		},
		systemd.OrphanedUnit{
			Name: "rclone-sync-orphan.timer", Type: "sync",
			Path: "/home/user/.config/systemd/user/rclone-sync-orphan.timer",
			ID:   "orphan2", IsLegacy: false,
		},
	)
	waitForOrphanPromptReady(t, tm)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	time.Sleep(500 * time.Millisecond)

	body := driveToFinal(t, tm)
	// After skipping the first orphan, the mount orphan
	// should be gone, the sync orphan should remain.
	if strings.Contains(body, "rclone-mount-orphan.service") {
		t.Errorf("orphan list should not still contain the skipped mount orphan, body:\n%s", body)
	}
	if !strings.Contains(body, "rclone-sync-orphan.timer") {
		t.Errorf("remaining sync orphan should still be in the list, body:\n%s", body)
	}
}

// TestOrphanPrompt_DismissAll presses 'd' to dismiss the
// entire prompt regardless of which orphan is selected.
func TestOrphanPrompt_DismissAll(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())

	sendOrphans(tm,
		systemd.OrphanedUnit{
			Name: "rclone-mount-orphan.service", Type: "mount",
			Path: "/home/user/.config/systemd/user/rclone-mount-orphan.service",
			ID:   "orphan1", IsLegacy: false,
		},
	)
	waitForOrphanPromptReady(t, tm)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	time.Sleep(500 * time.Millisecond)

	body := driveToFinal(t, tm)
	// After dismiss, the orphan prompt should be gone. The
	// "Orphaned Units Detected" banner should not be rendered.
	if strings.Contains(body, "Orphaned Units Detected") {
		t.Errorf("orphan prompt should be dismissed after 'd', body:\n%s", body)
	}
	// And the underlying screen (main menu) should be visible.
	if !strings.Contains(body, "Screen: Main Menu") {
		t.Errorf("expected main menu after dismiss, body:\n%s", body)
	}
}

// TestOrphanPrompt_EscClosesPromptFromListMode presses Esc
// from the orphan list (orphanMode=0) and asserts the prompt
// closes and the underlying main menu is shown.
func TestOrphanPrompt_EscClosesPromptFromListMode(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())

	sendOrphans(tm,
		systemd.OrphanedUnit{
			Name: "rclone-mount-orphan.service", Type: "mount",
			Path: "/home/user/.config/systemd/user/rclone-mount-orphan.service",
			ID:   "orphan1", IsLegacy: false,
		},
	)
	waitForOrphanPromptReady(t, tm)

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(500 * time.Millisecond)

	body := driveToFinal(t, tm)
	if strings.Contains(body, "Orphaned Units Detected") {
		t.Errorf("orphan prompt should be dismissed after Esc, body:\n%s", body)
	}
	if !strings.Contains(body, "Screen: Main Menu") {
		t.Errorf("expected main menu after Esc, body:\n%s", body)
	}
}
