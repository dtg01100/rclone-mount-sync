// Package main is the entry point for the rclone-mount-sync TUI application.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dtg01100/rclone-mount-sync/internal/cli"
	"github.com/dtg01100/rclone-mount-sync/internal/rclone"
	"github.com/dtg01100/rclone-mount-sync/internal/tui"
)

var version = "dev"

type Config struct {
	ShowVersion bool
	SkipChecks  bool
	ConfigDir   string
}

type PreflightChecker interface {
	PreflightChecks() []rclone.CheckResult
	HasCriticalFailure([]rclone.CheckResult) bool
	AllPassed([]rclone.CheckResult) bool
	FormatResults([]rclone.CheckResult) string
}

type defaultPreflightChecker struct {
	client *rclone.Client
}

func (d *defaultPreflightChecker) PreflightChecks() []rclone.CheckResult {
	return rclone.PreflightChecks(d.client)
}

func (d *defaultPreflightChecker) HasCriticalFailure(results []rclone.CheckResult) bool {
	return rclone.HasCriticalFailure(results)
}

func (d *defaultPreflightChecker) AllPassed(results []rclone.CheckResult) bool {
	return rclone.AllPassed(results)
}

func (d *defaultPreflightChecker) FormatResults(results []rclone.CheckResult) string {
	return rclone.FormatResults(results)
}

type TUIRunner interface {
	Run() error
}

type defaultTUIRunner struct{}

func (d *defaultTUIRunner) Run() error {
	return tui.Run()
}

func parseFlags(args []string) (*Config, error) {
	fs := flag.NewFlagSet("rclone-mount-sync", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	showVersion := fs.Bool("version", false, "Print version and exit")
	fs.BoolVar(showVersion, "v", false, "Print version and exit (shorthand)")
	skipChecks := fs.Bool("skip-checks", false, "Skip pre-flight validation checks")
	configDir := fs.String("config", "", "Custom config directory (overrides XDG_CONFIG_HOME)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return &Config{
		ShowVersion: *showVersion,
		SkipChecks:  *skipChecks,
		ConfigDir:   *configDir,
	}, nil
}

func printVersion(w io.Writer, v string) {
	_, _ = fmt.Fprintln(w, v)
}

func handleConfigDir(configDir string) error {
	if configDir == "" {
		return nil
	}

	resolvedDir := configDir
	fi, err := os.Stat(configDir)
	if err == nil && !fi.IsDir() {
		resolvedDir = filepath.Dir(configDir)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot access config directory %q: %w", configDir, err)
	}

	return os.Setenv("XDG_CONFIG_HOME", resolvedDir)
}

func runPreflightChecksTo(w io.Writer, checker PreflightChecker) error {
	_, _ = fmt.Fprintln(w, "Running pre-flight checks...")
	_, _ = fmt.Fprintln(w)

	results := checker.PreflightChecks()

	_, _ = fmt.Fprint(w, checker.FormatResults(results))
	_, _ = fmt.Fprintln(w)

	if checker.HasCriticalFailure(results) {
		_, _ = fmt.Fprintln(w, "╔══════════════════════════════════════════════════════════════════╗")
		_, _ = fmt.Fprintln(w, "║  Critical pre-flight check(s) failed. Cannot start application.  ║")
		_, _ = fmt.Fprintln(w, "╚══════════════════════════════════════════════════════════════════╝")
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Please fix the issues above and try again.")
		_, _ = fmt.Fprintln(w, "You can skip these checks with --skip-checks (not recommended).")
		return fmt.Errorf("critical pre-flight checks failed")
	}

	if !checker.AllPassed(results) {
		_, _ = fmt.Fprintln(w, "⚠ Some optional checks failed. The application will start, but some")
		_, _ = fmt.Fprintln(w, "  features may not work correctly.")
		_, _ = fmt.Fprintln(w)
	}

	_, _ = fmt.Fprintln(w, "Pre-flight checks completed. Starting application...")
	_, _ = fmt.Fprintln(w)

	return nil
}

func runPreflightChecks() error {
	client := rclone.NewClient()
	checker := &defaultPreflightChecker{client: client}
	return runPreflightChecksTo(os.Stdout, checker)
}

type AppDeps struct {
	Stdout       io.Writer
	Stderr       io.Writer
	NewClient    func() *rclone.Client
	NewTUIRunner func() TUIRunner
	ParseFlags   func(args []string) (*Config, error)
}

func DefaultAppDeps(stdout, stderr io.Writer) *AppDeps {
	return &AppDeps{
		Stdout:    stdout,
		Stderr:    stderr,
		NewClient: rclone.NewClient,
		NewTUIRunner: func() TUIRunner {
			return &defaultTUIRunner{}
		},
		ParseFlags: parseFlags,
	}
}

func runMainWithDeps(args []string, deps *AppDeps) int {
	cfg, err := deps.ParseFlags(args)
	if err != nil {
		// flag.ContinueOnError + io.Discard output means ParseFlags only
		// returns errors for actual parse failures (bad value, unknown
		// flag). Those are usage errors → exit 2 per POSIX convention.
		_, _ = fmt.Fprintf(deps.Stderr, "Error parsing flags: %v\n", err)
		return 2
	}

	if cfg.ShowVersion {
		printVersion(deps.Stdout, version)
		return 0
	}

	if err := handleConfigDir(cfg.ConfigDir); err != nil {
		_, _ = fmt.Fprintf(deps.Stderr, "Error handling config directory: %v\n", err)
		return 1
	}

	if !cfg.SkipChecks {
		client := deps.NewClient()
		checker := &defaultPreflightChecker{client: client}

		if err := runPreflightChecksTo(deps.Stdout, checker); err != nil {
			return 1
		}
	}

	tui.Version = version

	runner := deps.NewTUIRunner()
	if err := runner.Run(); err != nil {
		_, _ = fmt.Fprintf(deps.Stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

func runMain(args []string, stdout, stderr io.Writer) int {
	return runMainWithDeps(args, DefaultAppDeps(stdout, stderr))
}

// CLICommands returns the set of subcommand names that are routed to the
// CLI dispatcher (cobra) rather than the TUI. Exported so tests can verify
// the routing set matches what main actually uses, rather than the previous
// tautological pattern of hardcoding the same list in the test.
func CLICommands() map[string]bool {
	return map[string]bool{
		"mount":      true,
		"sync":       true,
		"services":   true,
		"config":     true,
		"remote":     true,
		"reconcile":  true,
		"doctor":     true,
		"cleanup":    true,
		"help":       true,
		"completion": true,
	}
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		os.Exit(runMain(args, os.Stdout, os.Stderr))
	}

	cliCommands := CLICommands()

	// Route to CLI if first arg is a known command
	firstArg := args[0]
	if cliCommands[firstArg] {
		cli.SetVersion(version)
		if err := cli.Execute(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// TUI mode flags (--skip-checks, --config, --version)
	tuiFlags := map[string]bool{
		"--skip-checks": true,
		"--config":      true,
		"--version":     true,
		"-v":            true,
	}
	isTUIArg := tuiFlags[firstArg] || strings.HasPrefix(firstArg, "--config=")
	if isTUIArg {
		// Check if any arg is a CLI subcommand (e.g., --config /path mount list)
		for _, arg := range args {
			if cliCommands[arg] {
				cli.SetVersion(version)
				if err := cli.Execute(); err != nil {
					os.Exit(1)
				}
				os.Exit(0)
			}
		}
		os.Exit(runMain(args, os.Stdout, os.Stderr))
	}

	// Unknown non-flag args route to CLI for help
	cli.SetVersion(version)
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
