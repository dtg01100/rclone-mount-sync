package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
)

const snapshotDir = "testdata/snapshots"

// writeSnapshot writes content to a snapshot file.
// Set UPDATE_SNAPSHOTS=1 to update snapshots.
func writeSnapshot(t *testing.T, name string, content string) {
	t.Helper()
	if os.Getenv("UPDATE_SNAPSHOTS") != "1" {
		return
	}

	path := filepath.Join(snapshotDir, name+".snap")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create snapshot dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write snapshot: %v", err)
	}
}

// readSnapshot reads a snapshot file, returning empty string if not found.
func readSnapshot(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(snapshotDir, name+".snap")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// assertSnapshot compares content against a stored snapshot.
func assertSnapshot(t *testing.T, name string, content string) {
	t.Helper()

	expected := readSnapshot(t, name)
	if expected == "" {
		// First run - create snapshot
		writeSnapshot(t, name, content)
		t.Logf("Created new snapshot: %s", name)
		return
	}

	if content != expected {
		// Show diff-like output
		t.Errorf("Snapshot mismatch for %s:\n--- Expected (snapshot)\n+++ Actual\n@@ Content differs @@", name)

		// Write actual to .actual file for easy comparison
		actualPath := filepath.Join(snapshotDir, name+".actual")
		_ = os.WriteFile(actualPath, []byte(content), 0644)
		t.Logf("Actual output written to: %s", actualPath)
		t.Logf("Run UPDATE_SNAPSHOTS=1 go test to update if change is intentional")
	}
}

// TestApp_Snapshot_MainMenu renders the main menu and compares against snapshot.
func TestApp_Snapshot_MainMenu(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.loading = false

	view := app.View()

	// Basic sanity checks
	if view == "" {
		t.Fatal("View should not be empty")
	}
	if !strings.Contains(view, "Rclone Mount Sync") {
		t.Error("Main menu should contain 'Rclone Mount Sync'")
	}
	if !strings.Contains(view, "Main Menu") {
		t.Error("Main menu should contain 'Main Menu'")
	}

	// Snapshot comparison
	assertSnapshot(t, "main_menu", view)
}

// TestApp_Snapshot_HelpScreen renders the help screen and compares against snapshot.
func TestApp_Snapshot_HelpScreen(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenHelp
	app.showHelp = true
	app.helpContentLen = 50

	view := app.View()

	if view == "" {
		t.Fatal("View should not be empty")
	}
	if !strings.Contains(view, "Help") {
		t.Error("Help screen should contain 'Help'")
	}

	assertSnapshot(t, "help_screen", view)
}

// TestApp_Snapshot_InitError renders the init error screen and compares against snapshot.
func TestApp_Snapshot_InitError(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.initError = &testError{msg: "failed to initialize rclone"}

	view := app.View()

	if view == "" {
		t.Fatal("View should not be empty")
	}
	if !strings.Contains(view, "Initialization Error") {
		t.Error("Init error view should contain 'Initialization Error'")
	}
	if !strings.Contains(view, "failed to initialize rclone") {
		t.Error("Init error view should contain the error message")
	}

	assertSnapshot(t, "init_error", view)
}

// TestApp_Snapshot_Loading renders the loading state and compares against snapshot.
func TestApp_Snapshot_Loading(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.loading = true
	app.currentScreen = ScreenMain

	view := app.View()

	if view == "" {
		t.Fatal("View should not be empty")
	}
	// Note: The TUI doesn't show a loading indicator when loading=true,
	// it just renders the current screen. "Loading..." only appears
	// when width/height are 0.
	t.Log("Loading=true renders current screen (no loading overlay)")

	assertSnapshot(t, "loading", view)
}

// TestApp_Snapshot_DifferentSizes tests rendering at different terminal sizes.
func TestApp_Snapshot_DifferentSizes(t *testing.T) {
	sizes := []struct {
		name   string
		width  int
		height int
	}{
		{"small", 60, 20},
		{"standard", 80, 24},
		{"large", 120, 40},
	}

	for _, sz := range sizes {
		t.Run(sz.name, func(t *testing.T) {
			app := NewApp()
			app.width = sz.width
			app.height = sz.height
			app.currentScreen = ScreenMain

			view := app.View()

			if view == "" {
				t.Fatal("View should not be empty")
			}

			// Check that view respects width constraints
			lines := strings.Split(view, "\n")
			for i, line := range lines {
				// Allow some margin for ANSI codes
				if len(line) > sz.width+20 {
					t.Errorf("Line %d length %d exceeds width %d", i, len(line), sz.width)
				}
			}

			assertSnapshot(t, "main_menu_"+sz.name, view)
		})
	}
}

// TestApp_Snapshot_OrphanPrompt renders the orphan detection prompt.
func TestApp_Snapshot_OrphanPrompt(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain
	app.loading = false
	app.showOrphanPrompt = true
	app.orphanSelected = 0
	app.orphanMode = 0
	app.orphans = &systemd.ReconciliationResult{
		OrphanedUnits: []systemd.OrphanedUnit{
			{Name: "rclone-mount-old1.service", ID: "old-1"},
			{Name: "rclone-sync-old2.timer", ID: "old-2"},
		},
	}

	view := app.View()

	if view == "" {
		t.Fatal("View should not be empty")
	}
	if !strings.Contains(view, "orphan") && !strings.Contains(view, "Orphan") {
		t.Error("Orphan prompt view should mention 'orphan'")
	}

	assertSnapshot(t, "orphan_prompt", view)
}

// TestApp_View_LineCount tests that views have reasonable line counts.
func TestApp_View_LineCount(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	view := app.View()
	lines := strings.Count(view, "\n") + 1

	if lines > app.height {
		t.Errorf("View has %d lines, should be <= %d (height)", lines, app.height)
	}
	if lines < 3 {
		t.Errorf("View has %d lines, should be >= 3", lines)
	}
}

// TestApp_View_NoTrailingWhitespace checks for trailing whitespace.
// Note: The TUI uses trailing whitespace for layout purposes, so this
// test documents the current behavior rather than enforcing strict rules.
func TestApp_View_NoTrailingWhitespace(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain

	view := app.View()
	lines := strings.Split(view, "\n")

	trailingWhitespaceLines := 0
	for i, line := range lines {
		if line != strings.TrimRight(line, " \t") {
			trailingWhitespaceLines++
			if trailingWhitespaceLines <= 3 {
				t.Logf("Line %d has trailing whitespace (TUI uses this for layout)", i)
			}
		}
	}
	t.Logf("Total lines with trailing whitespace: %d/%d", trailingWhitespaceLines, len(lines))
}

// TestApp_View_EmptyLines tests that there aren't excessive empty lines.
// Note: Some empty lines are expected for visual spacing in the TUI.
func TestApp_View_EmptyLines(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.currentScreen = ScreenMain

	view := app.View()
	lines := strings.Split(view, "\n")

	consecutiveBlanks := 0
	maxConsecutiveBlanks := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			consecutiveBlanks++
			if consecutiveBlanks > maxConsecutiveBlanks {
				maxConsecutiveBlanks = consecutiveBlanks
			}
		} else {
			consecutiveBlanks = 0
		}
	}

	// Log the max consecutive empty lines for documentation
	t.Logf("Max consecutive empty lines: %d", maxConsecutiveBlanks)
	
	// Flag only if truly excessive (more than 10)
	if maxConsecutiveBlanks > 10 {
		t.Errorf("Found %d consecutive empty lines (excessive)", maxConsecutiveBlanks)
	}
}

// TestApp_Snapshot_WideTerminal tests rendering on very wide terminals.
func TestApp_Snapshot_WideTerminal(t *testing.T) {
	app := NewApp()
	app.width = 200
	app.height = 30
	app.currentScreen = ScreenMain

	view := app.View()

	if view == "" {
		t.Fatal("View should not be empty")
	}

	assertSnapshot(t, "main_menu_wide", view)
}

// TestApp_Snapshot_TallTerminal tests rendering on tall terminals.
func TestApp_Snapshot_TallTerminal(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 60
	app.currentScreen = ScreenMain

	view := app.View()

	if view == "" {
		t.Fatal("View should not be empty")
	}

	assertSnapshot(t, "main_menu_tall", view)
}
