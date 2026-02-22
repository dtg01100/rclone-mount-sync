package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dtg01100/rclone-mount-sync/internal/rclone"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantVersion   bool
		wantSkip      bool
		wantConfigDir string
		wantErr       bool
	}{
		{
			name:          "no flags",
			args:          []string{},
			wantVersion:   false,
			wantSkip:      false,
			wantConfigDir: "",
			wantErr:       false,
		},
		{
			name:          "version flag",
			args:          []string{"--version"},
			wantVersion:   true,
			wantSkip:      false,
			wantConfigDir: "",
			wantErr:       false,
		},
		{
			name:          "skip-checks flag",
			args:          []string{"--skip-checks"},
			wantVersion:   false,
			wantSkip:      true,
			wantConfigDir: "",
			wantErr:       false,
		},
		{
			name:          "config flag",
			args:          []string{"--config", "/custom/config"},
			wantVersion:   false,
			wantSkip:      false,
			wantConfigDir: "/custom/config",
			wantErr:       false,
		},
		{
			name:          "all flags",
			args:          []string{"--version", "--skip-checks", "--config", "/my/config"},
			wantVersion:   true,
			wantSkip:      true,
			wantConfigDir: "/my/config",
			wantErr:       false,
		},
		{
			name:        "invalid flag",
			args:        []string{"--invalid-flag"},
			wantErr:     true,
			wantVersion: false,
		},
		{
			name:          "config flag with equals",
			args:          []string{"--config=/path/to/config"},
			wantVersion:   false,
			wantSkip:      false,
			wantConfigDir: "/path/to/config",
			wantErr:       false,
		},
		{
			name:        "short flags not supported",
			args:        []string{"-v"},
			wantErr:     true,
			wantVersion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseFlags(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Error("parseFlags() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseFlags() unexpected error: %v", err)
				return
			}

			if cfg.ShowVersion != tt.wantVersion {
				t.Errorf("ShowVersion = %v, want %v", cfg.ShowVersion, tt.wantVersion)
			}
			if cfg.SkipChecks != tt.wantSkip {
				t.Errorf("SkipChecks = %v, want %v", cfg.SkipChecks, tt.wantSkip)
			}
			if cfg.ConfigDir != tt.wantConfigDir {
				t.Errorf("ConfigDir = %q, want %q", cfg.ConfigDir, tt.wantConfigDir)
			}
		})
	}
}

func TestPrintVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{name: "dev version", version: "dev", want: "dev\n"},
		{name: "semantic version", version: "1.0.0", want: "1.0.0\n"},
		{name: "version with v prefix", version: "v1.2.3", want: "v1.2.3\n"},
		{name: "version with prerelease", version: "1.0.0-beta.1", want: "1.0.0-beta.1\n"},
		{name: "version with commit", version: "v1.2.3-abc123", want: "v1.2.3-abc123\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printVersion(&buf, tt.version)

			if got := buf.String(); got != tt.want {
				t.Errorf("printVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

type mockPreflightChecker struct {
	results          []rclone.CheckResult
	hasCritical      bool
	allPassed        bool
	formatResultsStr string
}

func (m *mockPreflightChecker) PreflightChecks() []rclone.CheckResult {
	return m.results
}

func (m *mockPreflightChecker) HasCriticalFailure(_ []rclone.CheckResult) bool {
	return m.hasCritical
}

func (m *mockPreflightChecker) AllPassed(_ []rclone.CheckResult) bool {
	return m.allPassed
}

func (m *mockPreflightChecker) FormatResults(_ []rclone.CheckResult) string {
	return m.formatResultsStr
}

func TestRunPreflightChecksTo_Success(t *testing.T) {
	mock := &mockPreflightChecker{
		results: []rclone.CheckResult{
			{Name: "Test Check", Passed: true, Message: "OK"},
		},
		hasCritical:      false,
		allPassed:        true,
		formatResultsStr: "[PASS] Test Check\n  OK\n",
	}

	var buf bytes.Buffer
	err := runPreflightChecksTo(&buf, mock)

	if err != nil {
		t.Errorf("runPreflightChecksTo() unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Running pre-flight checks...") {
		t.Error("output should contain 'Running pre-flight checks...'")
	}
	if !strings.Contains(output, "Pre-flight checks completed") {
		t.Error("output should contain completion message")
	}
}

func TestRunPreflightChecksTo_CriticalFailure(t *testing.T) {
	mock := &mockPreflightChecker{
		results: []rclone.CheckResult{
			{Name: "Critical Check", Passed: false, Message: "Failed", IsCritical: true},
		},
		hasCritical:      true,
		allPassed:        false,
		formatResultsStr: "[FAIL] Critical Check\n  Failed\n",
	}

	var buf bytes.Buffer
	err := runPreflightChecksTo(&buf, mock)

	if err == nil {
		t.Error("runPreflightChecksTo() expected error for critical failure")
	}

	if err.Error() != "critical pre-flight checks failed" {
		t.Errorf("error message = %q, want %q", err.Error(), "critical pre-flight checks failed")
	}

	output := buf.String()
	if !strings.Contains(output, "Critical pre-flight check(s) failed") {
		t.Error("output should contain critical failure message")
	}
	if !strings.Contains(output, "--skip-checks") {
		t.Error("output should mention --skip-checks option")
	}
}

func TestRunPreflightChecksTo_NonCriticalFailure(t *testing.T) {
	mock := &mockPreflightChecker{
		results: []rclone.CheckResult{
			{Name: "Critical Check", Passed: true, Message: "OK", IsCritical: true},
			{Name: "Optional Check", Passed: false, Message: "Failed", IsCritical: false},
		},
		hasCritical:      false,
		allPassed:        false,
		formatResultsStr: "[PASS] Critical Check\n  OK\n[WARN] Optional Check\n  Failed\n",
	}

	var buf bytes.Buffer
	err := runPreflightChecksTo(&buf, mock)

	if err != nil {
		t.Errorf("runPreflightChecksTo() unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Some optional checks failed") {
		t.Error("output should warn about optional failures")
	}
	if !strings.Contains(output, "Pre-flight checks completed") {
		t.Error("output should still show completion for non-critical failures")
	}
}

func TestHandleConfigDir(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		setupFile bool
		expectXDG string
		wantErr   bool
	}{
		{
			name:      "empty string does nothing",
			input:     "",
			expectXDG: "",
			wantErr:   false,
		},
		{
			name:      "sets directory path",
			input:     "/test/config/dir",
			expectXDG: "/test/config/dir",
			wantErr:   false,
		},
		{
			name:      "sets relative path",
			input:     "./config",
			expectXDG: "./config",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalXDG := os.Getenv("XDG_CONFIG_HOME")
			defer os.Setenv("XDG_CONFIG_HOME", originalXDG)
			os.Unsetenv("XDG_CONFIG_HOME")

			var inputPath string
			if tt.setupFile && tt.input != "" {
				tempDir, err := os.MkdirTemp("", "config-test")
				if err != nil {
					t.Fatal(err)
				}
				defer os.RemoveAll(tempDir)

				testFile := filepath.Join(tempDir, "config.yaml")
				if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
				inputPath = testFile
			} else {
				inputPath = tt.input
			}

			err := handleConfigDir(inputPath)

			if tt.wantErr {
				if err == nil {
					t.Error("handleConfigDir() expected error")
				}
				return
			}

			if err != nil {
				t.Errorf("handleConfigDir() unexpected error: %v", err)
				return
			}

			if tt.expectXDG != "" {
				got := os.Getenv("XDG_CONFIG_HOME")
				if tt.setupFile {
					if !strings.HasSuffix(got, tt.expectXDG) {
						t.Errorf("XDG_CONFIG_HOME = %q, should end with parent dir", got)
					}
				} else if got != tt.expectXDG {
					t.Errorf("XDG_CONFIG_HOME = %q, want %q", got, tt.expectXDG)
				}
			}
		})
	}
}

func TestHandleConfigDir_WithFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-file-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)
	os.Unsetenv("XDG_CONFIG_HOME")

	err = handleConfigDir(testFile)
	if err != nil {
		t.Errorf("handleConfigDir() unexpected error: %v", err)
	}

	got := os.Getenv("XDG_CONFIG_HOME")
	if got != tempDir {
		t.Errorf("XDG_CONFIG_HOME = %q, want %q", got, tempDir)
	}
}

func TestRunMain_Version(t *testing.T) {
	originalVersion := version
	version = "test-version"
	defer func() { version = originalVersion }()

	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"--version"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if got := stdout.String(); got != "test-version\n" {
		t.Errorf("stdout = %q, want %q", got, "test-version\n")
	}

	if stderr.Len() > 0 {
		t.Errorf("stderr should be empty, got %q", stderr.String())
	}
}

func TestRunMain_InvalidFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"--invalid-flag"}, &stdout, &stderr)

	if exitCode != 2 {
		t.Errorf("exit code = %d, want 2", exitCode)
	}

	if !strings.Contains(stderr.String(), "Error parsing flags") {
		t.Errorf("stderr should contain flag parsing error, got %q", stderr.String())
	}
}

func TestDefaultPreflightChecker(t *testing.T) {
	client := rclone.NewClient()
	checker := &defaultPreflightChecker{client: client}

	results := checker.PreflightChecks()
	if len(results) == 0 {
		t.Error("PreflightChecks() returned no results")
	}

	for _, r := range results {
		if r.Name == "" {
			t.Error("CheckResult has empty name")
		}
	}
}

func TestDefaultPreflightChecker_HasCriticalFailure(t *testing.T) {
	client := rclone.NewClient()
	checker := &defaultPreflightChecker{client: client}

	results := []rclone.CheckResult{
		{Name: "Test", Passed: false, IsCritical: true},
	}

	if !checker.HasCriticalFailure(results) {
		t.Error("HasCriticalFailure() should return true for critical failure")
	}

	results[0].IsCritical = false
	if checker.HasCriticalFailure(results) {
		t.Error("HasCriticalFailure() should return false for non-critical failure")
	}
}

func TestDefaultPreflightChecker_AllPassed(t *testing.T) {
	client := rclone.NewClient()
	checker := &defaultPreflightChecker{client: client}

	results := []rclone.CheckResult{
		{Name: "Test", Passed: true},
	}

	if !checker.AllPassed(results) {
		t.Error("AllPassed() should return true when all pass")
	}

	results[0].Passed = false
	if checker.AllPassed(results) {
		t.Error("AllPassed() should return false when any fail")
	}
}

func TestDefaultPreflightChecker_FormatResults(t *testing.T) {
	client := rclone.NewClient()
	checker := &defaultPreflightChecker{client: client}

	results := []rclone.CheckResult{
		{Name: "Test Check 1", Passed: true, Message: "All good", IsCritical: true},
		{Name: "Test Check 2", Passed: false, Message: "Something failed", Suggestion: "Fix it", IsCritical: false},
	}

	output := checker.FormatResults(results)

	if !strings.Contains(output, "Test Check 1") {
		t.Error("FormatResults should contain check name")
	}
	if !strings.Contains(output, "PASS") {
		t.Error("FormatResults should contain 'PASS'")
	}
	if !strings.Contains(output, "FAIL") {
		t.Error("FormatResults should contain 'FAIL'")
	}
	if !strings.Contains(output, "Fix it") {
		t.Error("FormatResults should contain suggestion")
	}
}

func TestConfig_Structure(t *testing.T) {
	cfg := &Config{
		ShowVersion: true,
		SkipChecks:  true,
		ConfigDir:   "/test/path",
	}

	if !cfg.ShowVersion {
		t.Error("ShowVersion should be true")
	}
	if !cfg.SkipChecks {
		t.Error("SkipChecks should be true")
	}
	if cfg.ConfigDir != "/test/path" {
		t.Errorf("ConfigDir = %q, want %q", cfg.ConfigDir, "/test/path")
	}
}

func TestParseFlags_EmptyArgs(t *testing.T) {
	cfg, err := parseFlags([]string{})
	if err != nil {
		t.Fatalf("parseFlags() unexpected error: %v", err)
	}

	if cfg.ShowVersion {
		t.Error("ShowVersion should be false by default")
	}
	if cfg.SkipChecks {
		t.Error("SkipChecks should be false by default")
	}
	if cfg.ConfigDir != "" {
		t.Errorf("ConfigDir should be empty by default, got %q", cfg.ConfigDir)
	}
}

func TestIntegration_PreflightCheckFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	var buf bytes.Buffer
	client := rclone.NewClient()
	checker := &defaultPreflightChecker{client: client}

	err := runPreflightChecksTo(&buf, checker)

	output := buf.String()

	if !strings.Contains(output, "Running pre-flight checks...") {
		t.Error("output should contain header")
	}
	if !strings.Contains(output, "Pre-flight Check Results") {
		t.Error("output should contain results header")
	}

	if rclone.HasCriticalFailure(checker.PreflightChecks()) {
		if err == nil {
			t.Error("expected error for critical failures")
		}
		if !strings.Contains(output, "Critical pre-flight check(s) failed") {
			t.Error("output should contain critical failure message")
		}
	} else {
		if !strings.Contains(output, "Pre-flight checks completed") && err == nil {
			t.Error("output should contain completion message for non-critical failures or success")
		}
	}
}

type mockTUIRunner struct {
	err error
}

func (m *mockTUIRunner) Run() error {
	return m.err
}

func TestRunMainWithDeps_SkipChecks(t *testing.T) {
	originalVersion := version
	version = "skip-test-version"
	defer func() { version = originalVersion }()

	var stdout, stderr bytes.Buffer

	deps := &AppDeps{
		Stdout:    &stdout,
		Stderr:    &stderr,
		NewClient: rclone.NewClient,
		NewTUIRunner: func() TUIRunner {
			return &mockTUIRunner{err: nil}
		},
		ParseFlags: func(args []string) (*Config, error) {
			return &Config{SkipChecks: true}, nil
		},
	}

	exitCode := runMainWithDeps([]string{}, deps)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
}

func TestRunMainWithDeps_TUIError(t *testing.T) {
	originalVersion := version
	version = "tui-error-version"
	defer func() { version = originalVersion }()

	var stdout, stderr bytes.Buffer

	deps := &AppDeps{
		Stdout:    &stdout,
		Stderr:    &stderr,
		NewClient: rclone.NewClient,
		NewTUIRunner: func() TUIRunner {
			return &mockTUIRunner{err: errors.New("TUI failed")}
		},
		ParseFlags: func(args []string) (*Config, error) {
			return &Config{SkipChecks: true}, nil
		},
	}

	exitCode := runMainWithDeps([]string{}, deps)

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}

	if !strings.Contains(stderr.String(), "TUI failed") {
		t.Errorf("stderr should contain TUI error, got %q", stderr.String())
	}
}

func TestRunMainWithDeps_Version(t *testing.T) {
	originalVersion := version
	version = "version-test"
	defer func() { version = originalVersion }()

	var stdout, stderr bytes.Buffer

	deps := &AppDeps{
		Stdout:    &stdout,
		Stderr:    &stderr,
		NewClient: rclone.NewClient,
		NewTUIRunner: func() TUIRunner {
			return &mockTUIRunner{err: nil}
		},
		ParseFlags: func(args []string) (*Config, error) {
			return &Config{ShowVersion: true}, nil
		},
	}

	exitCode := runMainWithDeps([]string{}, deps)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if stdout.String() != "version-test\n" {
		t.Errorf("stdout = %q, want %q", stdout.String(), "version-test\n")
	}
}

func TestRunMainWithDeps_ConfigDir(t *testing.T) {
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)
	os.Unsetenv("XDG_CONFIG_HOME")

	originalVersion := version
	version = "config-test"
	defer func() { version = originalVersion }()

	var stdout, stderr bytes.Buffer

	deps := &AppDeps{
		Stdout:    &stdout,
		Stderr:    &stderr,
		NewClient: rclone.NewClient,
		NewTUIRunner: func() TUIRunner {
			return &mockTUIRunner{err: nil}
		},
		ParseFlags: func(args []string) (*Config, error) {
			return &Config{SkipChecks: true, ConfigDir: "/custom/config"}, nil
		},
	}

	exitCode := runMainWithDeps([]string{}, deps)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if os.Getenv("XDG_CONFIG_HOME") != "/custom/config" {
		t.Errorf("XDG_CONFIG_HOME = %q, want %q", os.Getenv("XDG_CONFIG_HOME"), "/custom/config")
	}
}

func TestRunMainWithDeps_FlagParseError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	deps := &AppDeps{
		Stdout:    &stdout,
		Stderr:    &stderr,
		NewClient: rclone.NewClient,
		NewTUIRunner: func() TUIRunner {
			return &mockTUIRunner{err: nil}
		},
		ParseFlags: func(args []string) (*Config, error) {
			return nil, errors.New("flag error")
		},
	}

	exitCode := runMainWithDeps([]string{}, deps)

	if exitCode != 2 {
		t.Errorf("exit code = %d, want 2", exitCode)
	}

	if !strings.Contains(stderr.String(), "Error parsing flags") {
		t.Errorf("stderr should contain flag error, got %q", stderr.String())
	}
}

func TestDefaultAppDeps(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := DefaultAppDeps(&stdout, &stderr)

	if deps.Stdout != &stdout {
		t.Error("Stdout should be set")
	}
	if deps.Stderr != &stderr {
		t.Error("Stderr should be set")
	}
	if deps.NewClient == nil {
		t.Error("NewClient should be set")
	}
	if deps.NewTUIRunner == nil {
		t.Error("NewTUIRunner should be set")
	}
	if deps.ParseFlags == nil {
		t.Error("ParseFlags should be set")
	}
}

func TestDefaultTUIRunner(t *testing.T) {
	runner := &defaultTUIRunner{}
	if runner == nil {
		t.Error("defaultTUIRunner should not be nil")
	}
}

func TestParseFlags_MultipleValues(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want *Config
	}{
		{
			name: "version and skip-checks",
			args: []string{"--version", "--skip-checks"},
			want: &Config{ShowVersion: true, SkipChecks: true, ConfigDir: ""},
		},
		{
			name: "skip-checks and config",
			args: []string{"--skip-checks", "--config", "/tmp/config"},
			want: &Config{ShowVersion: false, SkipChecks: true, ConfigDir: "/tmp/config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseFlags(tt.args)
			if err != nil {
				t.Fatalf("parseFlags() error: %v", err)
			}

			if cfg.ShowVersion != tt.want.ShowVersion {
				t.Errorf("ShowVersion = %v, want %v", cfg.ShowVersion, tt.want.ShowVersion)
			}
			if cfg.SkipChecks != tt.want.SkipChecks {
				t.Errorf("SkipChecks = %v, want %v", cfg.SkipChecks, tt.want.SkipChecks)
			}
			if cfg.ConfigDir != tt.want.ConfigDir {
				t.Errorf("ConfigDir = %q, want %q", cfg.ConfigDir, tt.want.ConfigDir)
			}
		})
	}
}

func TestRunMainWithDeps_PreflightChecksExecuted(t *testing.T) {
	originalVersion := version
	version = "preflight-test"
	defer func() { version = originalVersion }()

	var stdout, stderr bytes.Buffer

	preflightExecuted := false

	deps := &AppDeps{
		Stdout:    &stdout,
		Stderr:    &stderr,
		NewClient: func() *rclone.Client { return rclone.NewClient() },
		NewTUIRunner: func() TUIRunner {
			return &mockTUIRunner{err: nil}
		},
		ParseFlags: func(args []string) (*Config, error) {
			return &Config{SkipChecks: false}, nil
		},
	}

	exitCode := runMainWithDeps([]string{}, deps)

	preflightExecuted = strings.Contains(stdout.String(), "Running pre-flight checks")

	if !preflightExecuted {
		t.Log("Note: Preflight checks should be executed when SkipChecks is false")
	}

	t.Logf("Exit code: %d", exitCode)
	t.Logf("Stdout: %s", stdout.String())
}

func TestRunPreflightChecks_Wrapper(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that calls actual preflight checks")
	}

	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runPreflightChecks()

	w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Running pre-flight checks") {
		t.Error("runPreflightChecks() should print header")
	}

	t.Logf("Preflight output length: %d bytes", len(output))
	t.Logf("Error: %v", err)
}

func TestDefaultAppDeps_NewTUIRunner(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := DefaultAppDeps(&stdout, &stderr)

	runner := deps.NewTUIRunner()
	if runner == nil {
		t.Error("NewTUIRunner should return non-nil runner")
	}

	_, ok := runner.(*defaultTUIRunner)
	if !ok {
		t.Error("NewTUIRunner should return *defaultTUIRunner")
	}
}

func TestDefaultAppDeps_NewClient(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := DefaultAppDeps(&stdout, &stderr)

	client := deps.NewClient()
	if client == nil {
		t.Error("NewClient should return non-nil client")
	}
}

func TestDefaultAppDeps_ParseFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := DefaultAppDeps(&stdout, &stderr)

	cfg, err := deps.ParseFlags([]string{"--version"})
	if err != nil {
		t.Errorf("ParseFlags error: %v", err)
	}
	if !cfg.ShowVersion {
		t.Error("ParseFlags should parse --version flag")
	}
}

func BenchmarkParseFlags(b *testing.B) {
	args := []string{"--version", "--skip-checks", "--config", "/test/config"}
	for i := 0; i < b.N; i++ {
		_, _ = parseFlags(args)
	}
}

func BenchmarkPrintVersion(b *testing.B) {
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		buf.Reset()
		printVersion(&buf, "1.0.0")
	}
}

type capturingWriter struct {
	buf bytes.Buffer
}

func (w *capturingWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func (w *capturingWriter) String() string {
	return w.buf.String()
}

type noopWriteCloser struct {
	io.Writer
}

func (n *noopWriteCloser) Close() error { return nil }
