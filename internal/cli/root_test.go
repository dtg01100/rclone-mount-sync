package cli

import (
	"bytes"
	"github.com/spf13/cobra"
	"testing"
)

// helper to run cobra command and capture output
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
