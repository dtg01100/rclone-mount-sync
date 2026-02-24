package cli

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
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
	testErr := fmt.Errorf("test error message")
	printError(testErr)
}

func TestPrintJSON(t *testing.T) {
	data := map[string]string{"key": "value", "name": "test"}
	err := printJSON(data)
	if err != nil {
		t.Fatalf("printJSON failed: %v", err)
	}
}

func TestPrintJSONArray(t *testing.T) {
	data := []string{"item1", "item2", "item3"}
	err := printJSON(data)
	if err != nil {
		t.Fatalf("printJSON array failed: %v", err)
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
