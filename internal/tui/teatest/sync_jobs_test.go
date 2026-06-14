package teatest

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// waitForSyncJobsLoaded sleeps until the sync-jobs screen has had
// a chance to receive the SyncJobsLoadedMsg and re-render. Mirrors
// waitForMountsLoaded; see the comment there.
func waitForSyncJobsLoaded(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	time.Sleep(700 * time.Millisecond)
}

// TestSyncJobs_EmptyList renders the sync-jobs screen with no
// jobs and asserts the screen is on the Sync Job Management view.
func TestSyncJobs_EmptyList(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	waitForSyncJobsLoaded(t, tm)

	body := driveToFinal(t, tm)
	if !strings.Contains(body, "Screen: Sync Job Management") {
		t.Errorf("sync jobs screen missing, body:\n%s", body)
	}
}

// TestSyncJobs_ListWithEntries renders the sync-jobs screen with
// two jobs and asserts both names appear in the rendered body.
func TestSyncJobs_ListWithEntries(t *testing.T) {
	deps := defaultDeps()
	deps.Config = testConfigWithSyncJobs(
		sampleSyncJob("alpha"),
		sampleSyncJob("beta"),
	)
	tm := newTestProgram(t, deps)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	waitForSyncJobsLoaded(t, tm)

	body := driveToFinal(t, tm)
	for _, want := range []string{"Screen: Sync Job Management", "alpha", "beta"} {
		if !strings.Contains(body, want) {
			t.Errorf("sync jobs screen missing %q, body:\n%s", want, body)
		}
	}
}

// TestSyncJobs_HotkeyWithNoJobsIsNoop presses each list-mode
// hotkey on an empty list and asserts no crash.
func TestSyncJobs_HotkeyWithNoJobsIsNoop(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	waitForSyncJobsLoaded(t, tm)

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")},
	)
	if !strings.Contains(body, "Screen: Sync Job Management") {
		t.Errorf("sync jobs screen not stable after hotkey spam, body:\n%s", body)
	}
}

// TestSyncJobs_DeleteEntersDeleteMode presses 'd' on a non-empty
// list and asserts the delete-confirm screen renders. The recent
// UX fix added a per-option consequences block; the prompt copy
// changed from "delete '<name>'" to the unified text.
func TestSyncJobs_DeleteEntersDeleteMode(t *testing.T) {
	deps := defaultDeps()
	deps.Config = testConfigWithSyncJobs(sampleSyncJob("alpha"))
	tm := newTestProgram(t, deps)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	waitForSyncJobsLoaded(t, tm)

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")},
	)
	for _, want := range []string{
		"alpha",
		"Are you sure you want to delete this sync job?",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("delete confirm missing %q, body:\n%s", want, body)
		}
	}
}

// TestSyncJobs_EscReturnsToMain presses Esc and asserts the
// screen flips back to main. The sync-jobs list-mode esc handler
// returns GoBackMsg, which App.Update turns into ScreenMain.
func TestSyncJobs_EscReturnsToMain(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	waitForSyncJobsLoaded(t, tm)

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyEsc},
	)
	if !strings.Contains(body, "Screen: Main Menu") {
		t.Errorf("after Esc, expected main menu, body:\n%s", body)
	}
}

// TestSyncJobs_CursorNavigation exercises down/up keys; the screen
// must not crash and the status bar must remain Sync Job
// Management.
func TestSyncJobs_CursorNavigation(t *testing.T) {
	deps := defaultDeps()
	deps.Config = testConfigWithSyncJobs(
		sampleSyncJob("alpha"),
		sampleSyncJob("beta"),
		sampleSyncJob("gamma"),
	)
	tm := newTestProgram(t, deps)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	waitForSyncJobsLoaded(t, tm)

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")},
	)
	if !strings.Contains(body, "Screen: Sync Job Management") {
		t.Errorf("sync jobs screen not present after cursor nav, body:\n%s", body)
	}
}
