package teatest

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// waitForMountsLoaded sleeps until the mounts screen has had a
// chance to receive the MountsLoadedMsg and re-render. The Cmd
// that triggers the load runs in a goroutine, so the test must
// give the program at least a few hundred ms to process it
// before sending more keys.
// waitForMountsLoaded waits until the mounts screen has rendered
// past the "Loading mounts..." state. The cmd that triggers the
// load runs in a goroutine, so the test polls the program
// output until the loading banner disappears.
func waitForMountsLoaded(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return !strings.Contains(string(bts), "Loading mounts...")
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(30*time.Millisecond))
}

// driveToFinal sends a sequence of keypresses to tm, waits for
// the program to render, then Quit()s. It uses teatest.WaitFor
// to accumulate the full output stream into a buffer so the
// assertions can inspect every frame, then returns the buffer
// as a string.
//
// Note: teatest.WaitFor calls tb.Fatal on timeout, so we use a
// custom drain via a teatest-compatible internal call. The
// teatest.tm.Output() returns a single shared bytes.Buffer
// across calls; once doWaitFor reads from it, subsequent reads
// see no new data. So we capture the buffer in the condition
// closure and tolerate the trailing timeout by reading the
// captured buffer before letting WaitFor fail.
func driveToFinal(t *testing.T, tm *teatest.TestModel, keys ...tea.KeyMsg) string {
	t.Helper()
	for _, k := range keys {
		tm.Send(k)
	}
	// Give the program a moment to render after each keypress.
	time.Sleep(300 * time.Millisecond)
	if err := tm.Quit(); err != nil {
		t.Fatalf("tm.Quit: %v", err)
	}
	// FinalOutput waits for the program to actually exit, then
	// returns the shared output buffer. Once the program has
	// exited, the buffer holds every frame it ever wrote.
	out := tm.FinalOutput(t, teatest.WithFinalTimeout(2*time.Second))
	return readAll(out)
}

// TestMounts_EmptyList renders the mounts screen with no mounts
// and asserts the start+enable/stop+disable help text from the
// recent UX fix is present.
func TestMounts_EmptyList(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	waitForMountsLoaded(t, tm)

	body := driveToFinal(t, tm)
	for _, want := range []string{
		"Screen: Mount Management",
		"start+enable", // UX fix: pairs Start with Enable
		"stop+disable", // UX fix: pairs Stop with Disable
	} {
		if !strings.Contains(body, want) {
			t.Errorf("mounts screen missing %q, full body:\n%s", want, body)
		}
	}
}

// TestMounts_ListWithEntries renders the mounts screen with two
// mounts and asserts both names appear in the rendered body.
func TestMounts_ListWithEntries(t *testing.T) {
	deps := defaultDeps()
	deps.Config = testConfigWithMounts(
		sampleMount("alpha"),
		sampleMount("beta"),
	)
	tm := newTestProgram(t, deps)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	waitForMountsLoaded(t, tm)

	body := driveToFinal(t, tm)
	for _, want := range []string{"Screen: Mount Management", "alpha", "beta"} {
		if !strings.Contains(body, want) {
			t.Errorf("mounts screen missing %q, full body:\n%s", want, body)
		}
	}
}

// TestMounts_CursorNavigation exercises down/up keys; the screen
// must not crash and the status bar must remain Mount Management.
func TestMounts_CursorNavigation(t *testing.T) {
	deps := defaultDeps()
	deps.Config = testConfigWithMounts(
		sampleMount("alpha"),
		sampleMount("beta"),
		sampleMount("gamma"),
	)
	tm := newTestProgram(t, deps)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	waitForMountsLoaded(t, tm)

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")},
	)
	if !strings.Contains(body, "Screen: Mount Management") {
		t.Errorf("mounts screen not present after cursor nav, body:\n%s", body)
	}
}

// TestMounts_HotkeyWithNoMountsIsNoop presses each list-mode
// hotkey on an empty list and asserts no crash and the screen
// stays on Mount Management.
func TestMounts_HotkeyWithNoMountsIsNoop(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	waitForMountsLoaded(t, tm)

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")},
	)
	if !strings.Contains(body, "Screen: Mount Management") {
		t.Errorf("mounts screen not stable after hotkey spam, body:\n%s", body)
	}
}

// TestMounts_DeleteEntersDeleteMode presses 'd' on a non-empty
// list and asserts the delete-confirm screen renders the
// "Are you sure you want to delete this mount?" prompt (the
// post-UX-fix text; the old text was "delete '<name>'").
func TestMounts_DeleteEntersDeleteMode(t *testing.T) {
	deps := defaultDeps()
	deps.Config = testConfigWithMounts(sampleMount("alpha"))
	tm := newTestProgram(t, deps)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	waitForMountsLoaded(t, tm)
	// Small extra yield so any in-flight screen Init cmds
	// have time to settle before the 'd' keypress.
	time.Sleep(300 * time.Millisecond)

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")},
	)
	for _, want := range []string{
		"alpha",
		"Are you sure you want to delete this mount?",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("delete confirm missing %q, body:\n%s", want, body)
		}
	}
}

// TestMounts_EscFromMountsReturnsToMain presses Esc and asserts
// the screen flips back to main.
func TestMounts_EscFromMountsReturnsToMain(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	waitForMountsLoaded(t, tm)

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyEsc},
	)
	if !strings.Contains(body, "Screen: Main Menu") {
		t.Errorf("after Esc, expected main menu, body:\n%s", body)
	}
}

// TestMounts_StableAcrossUnknownKeys exercises a handful of
// keys that are not bound in list mode; the screen must not
// crash and must remain on Mount Management.
func TestMounts_StableAcrossUnknownKeys(t *testing.T) {
	deps := defaultDeps()
	deps.Config = testConfigWithMounts(sampleMount("alpha"))
	tm := newTestProgram(t, deps)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	waitForMountsLoaded(t, tm)

	body := driveToFinal(t, tm,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("@")},
	)
	if !strings.Contains(body, "Screen: Mount Management") {
		t.Errorf("unknown keys destabilized mounts screen, body:\n%s", body)
	}
}
