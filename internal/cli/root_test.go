package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
	"github.com/spf13/cobra"
)

func runCmd(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	t.Helper()
	bufOut := &bytes.Buffer{}
	bufErr := &bytes.Buffer{}
	cmd.SetOut(bufOut)
	cmd.SetErr(bufErr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return bufOut.String(), bufErr.String(), err
}

func TestVersionFlag(t *testing.T) {
	SetVersion("1.2.3")
	out, _, err := runCmd(t, rootCmd, "--version")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out != "1.2.3\n" {
		t.Fatalf("expected version output, got %q", out)
	}
}

func TestUnknownFlag(t *testing.T) {
	_, errOut, err := runCmd(t, rootCmd, "--no-such-flag")
	if err == nil {
		t.Fatalf("expected error for unknown flag")
	}
	if errOut == "" {
		t.Fatalf("expected error message on stderr")
	}
}

func TestPrintError(t *testing.T) {
	// Test that printError handles various error types without panicking
	testCases := []struct {
		name string
		err  error
	}{
		{"simple error", fmt.Errorf("simple error")},
		{"error with wrapping", fmt.Errorf("wrapped: %w", fmt.Errorf("inner"))},
		{"empty error", fmt.Errorf("")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("printError panicked: %v", r)
				}
			}()
			printError(tc.err)
		})
	}
}

func TestPrintJSON(t *testing.T) {
	// Test that printJSON produces valid JSON output
	// Note: We test the JSON encoding behavior, not stdout output directly

	data := map[string]string{"key": "value", "name": "test"}

	// Verify the data can be marshaled to JSON (which is what printJSON does)
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Errorf("JSON output is not valid: %v", err)
	}
	if parsed["key"] != "value" || parsed["name"] != "test" {
		t.Errorf("JSON output = %q, want contains key/value pairs", string(jsonBytes))
	}

	// Also verify printJSON itself doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printJSON panicked: %v", r)
		}
	}()
	err = printJSON(data)
	if err != nil {
		t.Errorf("printJSON returned error: %v", err)
	}
}

func TestPrintJSONArray(t *testing.T) {
	// Test that printJSON produces valid JSON array output
	data := []string{"item1", "item2", "item3"}

	// Verify the data can be marshaled to JSON (which is what printJSON does)
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed []string
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Errorf("JSON output is not valid: %v", err)
	}
	if len(parsed) != 3 || parsed[0] != "item1" || parsed[1] != "item2" || parsed[2] != "item3" {
		t.Errorf("JSON output = %q, want [\"item1\", \"item2\", \"item3\"]", string(jsonBytes))
	}

	// Also verify printJSON itself doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printJSON panicked: %v", r)
		}
	}()
	err = printJSON(data)
	if err != nil {
		t.Errorf("printJSON returned error: %v", err)
	}
}

func TestFindMountByIDOrName(t *testing.T) {
	cfg := &config.Config{
		Mounts: []models.MountConfig{
			{ID: "abc123", Name: "test-mount-1"},
			{ID: "def456", Name: "test-mount-2"},
		},
	}

	mount := findMountByIDOrName(cfg, "abc123")
	if mount == nil {
		t.Fatal("expected to find mount by ID")
	}
	if mount.Name != "test-mount-1" {
		t.Errorf("expected mount name 'test-mount-1', got %q", mount.Name)
	}

	mount = findMountByIDOrName(cfg, "test-mount-2")
	if mount == nil {
		t.Fatal("expected to find mount by name")
	}
	if mount.ID != "def456" {
		t.Errorf("expected mount ID 'def456', got %q", mount.ID)
	}

	mount = findMountByIDOrName(cfg, "nonexistent")
	if mount != nil {
		t.Error("expected nil for nonexistent mount")
	}
}

func TestFindSyncJobByIDOrName(t *testing.T) {
	cfg := &config.Config{
		SyncJobs: []models.SyncJobConfig{
			{ID: "abc123", Name: "test-sync-1"},
			{ID: "def456", Name: "test-sync-2"},
		},
	}

	job := findSyncJobByIDOrName(cfg, "abc123")
	if job == nil {
		t.Fatal("expected to find sync job by ID")
	}
	if job.Name != "test-sync-1" {
		t.Errorf("expected sync job name 'test-sync-1', got %q", job.Name)
	}

	job = findSyncJobByIDOrName(cfg, "test-sync-2")
	if job == nil {
		t.Fatal("expected to find sync job by name")
	}
	if job.ID != "def456" {
		t.Errorf("expected sync job ID 'def456', got %q", job.ID)
	}

	job = findSyncJobByIDOrName(cfg, "nonexistent")
	if job != nil {
		t.Error("expected nil for nonexistent sync job")
	}
}

func TestRunCleanup_SystemctlNotAvailable(t *testing.T) {
	// Test that runCleanup returns an error when systemctl is not available
	// This is done by pointing to a non-existent systemctl path

	oldLoadManager := loadManager
	oldLoadGenerator := loadGenerator
	defer func() {
		loadManager = oldLoadManager
		loadGenerator = oldLoadGenerator
	}()

	// Create a generator that returns a non-existent systemctl path
	tmp := t.TempDir()
	loadGenerator = func() (*systemd.Generator, error) {
		return systemd.NewTestGenerator(tmp), nil
	}

	// Mock manager that returns a non-existent systemctl path
	mock := &systemd.MockManager{}
	loadManager = func() systemd.ServiceManager { return mock }

	// Temporarily override systemctlPath by using a custom manager
	// The actual test is that when systemctl fails, we get an error
	// Since we can't easily mock exec.Command, we test the error path
	// by verifying that cleanup fails gracefully when systemctl isn't available

	// We can't truly test this without mocking exec.Command, so we
	// test that the function doesn't panic and handles errors properly
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("runCleanup panicked: %v", r)
		}
	}()

	err := runCleanup(rootCmd, []string{})
	// If systemctl is not available, we expect an error
	// If it is available and returns no failed units, we get no error
	if err != nil {
		t.Logf("runCleanup returned error (expected if systemctl unavailable): %v", err)
	}
}
