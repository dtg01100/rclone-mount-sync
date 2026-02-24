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
	fmt.Fprintln(w, v)
}

func handleConfigDir(configDir string) error {
	if configDir == "" {
		return nil
	}

	resolvedDir := configDir
	if fi, err := os.Stat(configDir); err == nil && !fi.IsDir() {
		resolvedDir = filepath.Dir(configDir)
	}

	return os.Setenv("XDG_CONFIG_HOME", resolvedDir)
}

func runPreflightChecksTo(w io.Writer, checker PreflightChecker) error {
	fmt.Fprintln(w, "Running pre-flight checks...")
	fmt.Fprintln(w)

	results := checker.PreflightChecks()

	fmt.Fprint(w, checker.FormatResults(results))
	fmt.Fprintln(w)

	if checker.HasCriticalFailure(results) {
		fmt.Fprintln(w, "╔══════════════════════════════════════════════════════════════════╗")
		fmt.Fprintln(w, "║  Critical pre-flight check(s) failed. Cannot start application.  ║")
		fmt.Fprintln(w, "╚══════════════════════════════════════════════════════════════════╝")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Please fix the issues above and try again.")
		fmt.Fprintln(w, "You can skip these checks with --skip-checks (not recommended).")
		return fmt.Errorf("critical pre-flight checks failed")
	}

	if !checker.AllPassed(results) {
		fmt.Fprintln(w, "⚠ Some optional checks failed. The application will start, but some")
		fmt.Fprintln(w, "  features may not work correctly.")
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "Pre-flight checks completed. Starting application...")
	fmt.Fprintln(w)

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
		fmt.Fprintf(deps.Stderr, "Error parsing flags: %v\n", err)
		return 2
	}

	if cfg.ShowVersion {
		printVersion(deps.Stdout, version)
		return 0
	}

	if err := handleConfigDir(cfg.ConfigDir); err != nil {
		fmt.Fprintf(deps.Stderr, "Error handling config directory: %v\n", err)
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
		fmt.Fprintf(deps.Stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

func runMain(args []string, stdout, stderr io.Writer) int {
	return runMainWithDeps(args, DefaultAppDeps(stdout, stderr))
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		os.Exit(runMain(args, os.Stdout, os.Stderr))
	}

	// Handle --version flag
	for _, arg := range args {
		if arg == "--version" || arg == "-v" {
			printVersion(os.Stdout, version)
			os.Exit(0)
		}
	}

	cliCommands := map[string]bool{
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

	// Route to CLI if first arg is a known command or a flag (like --help, -h)
	firstArg := args[0]
	if cliCommands[firstArg] || strings.HasPrefix(firstArg, "-") {
		cli.SetVersion(version)
		if err := cli.Execute(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Otherwise, TUI mode (supports old flags like --skip-checks, --config)
	os.Exit(runMain(args, os.Stdout, os.Stderr))
}
