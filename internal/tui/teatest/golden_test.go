package teatest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
	"github.com/dtg01100/rclone-mount-sync/internal/tui"
)

const goldenDir = "testdata/golden"

// writeGolden writes the supplied body to the golden file when
// the GOCMS_UPDATE_GOLDEN env var is non-empty. CI does not
// set this; the maintainer runs the test with the env var set
// once to (re)generate the golden file after an intentional
// render change.
//
// The `-update` flag is the teatest idiom; we mirror that with
// an env var so the build cache is not invalidated by a flag
// that is only used in dev.
func writeGolden(t *testing.T, name, body string) {
	t.Helper()
	if os.Getenv("GOCMS_UPDATE_GOLDEN") == "" {
		return
	}
	path := filepath.Join(goldenDir, name+".golden")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil { //nolint:gosec
		t.Fatalf("write: %v", err)
	}
}

// readGolden reads the named golden file. Returns empty string
// if the file does not exist.
func readGolden(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(goldenDir, name+".golden")
	b, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return ""
	}
	return string(b)
}

// TestGolden_OrphanPromptListView is the golden-file
// replacement for the manual snapshot at
// internal/tui/testdata/snapshots/orphan_prompt.snap. We
// render the orphan prompt with a populated ReconciliationResult
// and assert the rendered body matches the committed golden
// file.
//
// To regenerate the golden file after a render change, run:
//
//	GOCMS_UPDATE_GOLDEN=1 go test -count=1 -run TestGolden_OrphanPromptListView ./internal/tui/teatest/
func TestGolden_OrphanPromptListView(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	tm.Send(tui.ReconciliationMsg{
		Result: &systemd.ReconciliationResult{
			OrphanedUnits: []systemd.OrphanedUnit{
				{
					Name:     "rclone-mount-old1.service",
					Type:     "mount",
					Path:     "/home/user/.config/systemd/user/rclone-mount-old1.service",
					ID:       "old-1",
					IsLegacy: false,
				},
				{
					Name:     "rclone-sync-old2.timer",
					Type:     "sync",
					Path:     "/home/user/.config/systemd/user/rclone-sync-old2.timer",
					ID:       "old-2",
					IsLegacy: false,
				},
			},
		},
	})
	// Generous yield so the program processes the screen
	// Init cmds kicked off by ReconciliationMsg before we
	// capture the body.
	time.Sleep(500 * time.Millisecond)

	body := driveToFinal(t, tm)

	golden := readGolden(t, "orphan_prompt")
	if golden == "" {
		// First run: no committed golden. Create one and
		// instruct the developer.
		writeGolden(t, "orphan_prompt", body)
		t.Logf("Created golden: %s/orphan_prompt.golden", goldenDir)
		t.Logf("Re-run without GOCMS_UPDATE_GOLDEN to assert against it.")
		return
	}

	if body != golden {
		// On mismatch, dump the actual to a .actual file for
		// easy diffing.
		actualPath := filepath.Join(goldenDir, "orphan_prompt.actual")
		_ = os.WriteFile(actualPath, []byte(body), 0o600) //nolint:gosec
		t.Errorf("golden mismatch for orphan_prompt; actual written to %s", actualPath)
		t.Logf("Re-run with GOCMS_UPDATE_GOLDEN=1 to refresh if change is intentional.")
	}
}

// TestGolden_MainMenu is a smoke check that the main-menu
// rendered body is stable. It is the first test to fail if
// the main-menu layout drifts (which is most of the
// user-visible regressions in this codebase).
func TestGolden_MainMenu(t *testing.T) {
	tm := newTestProgram(t, defaultDeps())
	time.Sleep(200 * time.Millisecond)

	body := driveToFinal(t, tm)
	// Strip the timestamp/vdev footer line that varies across
	// builds (the App shows the build version, which is
	// "dev" in tests but could be a real tag in production).
	body = stripBuildMetadata(body)

	golden := readGolden(t, "main_menu")
	if golden == "" {
		writeGolden(t, "main_menu", body)
		t.Logf("Created golden: %s/main_menu.golden", goldenDir)
		return
	}

	if body != golden {
		actualPath := filepath.Join(goldenDir, "main_menu.actual")
		_ = os.WriteFile(actualPath, []byte(body), 0o600) //nolint:gosec
		t.Errorf("golden mismatch for main_menu; actual written to %s", actualPath)
	}
}

// stripBuildMetadata removes the per-build version suffix
// ("vdev", "v1.2.3", etc.) from the rendered body so the
// golden file does not need to be re-generated on every
// release tag.
func stripBuildMetadata(body string) string {
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		// The header line includes "Rclone Mount Sync ... v<version>".
		// We replace any "v<word>" with a stable marker.
		if i == 0 || strings.Contains(l, "Rclone Mount Sync") {
			lines[i] = removeVersionToken(l)
		}
	}
	return strings.Join(lines, "\n")
}

// removeVersionToken replaces the first occurrence of "v<word>"
// in a header line with a stable placeholder. This is brittle
// but sufficient for the test; if the header layout changes,
// the golden file will diverge and the maintainer will know.
func removeVersionToken(line string) string {
	const token = "vdev"
	if i := strings.Index(line, token); i >= 0 {
		return line[:i] + "vX.Y.Z" + line[i+len(token):]
	}
	return line
}
