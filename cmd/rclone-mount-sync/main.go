// Package main is the entry point for the rclone-mount-sync TUI application.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dtg01100/rclone-mount-sync/internal/rclone"
	"github.com/dtg01100/rclone-mount-sync/internal/tui"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	// CLI flags
	showVersion := flag.Bool("version", false, "Print version and exit")
	skipChecks := flag.Bool("skip-checks", false, "Skip pre-flight validation checks")
	configDir := flag.String("config", "", "Custom config directory (overrides XDG_CONFIG_HOME)")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	// If a custom config directory is provided, set XDG_CONFIG_HOME so
	// internal config loading will pick it up via os.UserConfigDir().
	if *configDir != "" {
		// If a file path was passed, use its directory
		if fi, err := os.Stat(*configDir); err == nil && !fi.IsDir() {
			*configDir = filepath.Dir(*configDir)
		}
		os.Setenv("XDG_CONFIG_HOME", *configDir)
	}

	// Run pre-flight checks unless skipped
	if !*skipChecks {
		if err := runPreflightChecks(); err != nil {
			os.Exit(1)
		}
	}

	// Set version for TUI
	tui.Version = version

	// Run the TUI application
	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runPreflightChecks executes all pre-flight validation checks and displays results.
// Returns an error if any critical checks fail.
func runPreflightChecks() error {
	fmt.Println("Running pre-flight checks...")
	fmt.Println()

	// Create rclone client
	client := rclone.NewClient()

	// Run pre-flight checks
	results := rclone.PreflightChecks(client)

	// Display results
	fmt.Print(rclone.FormatResults(results))
	fmt.Println()

	// Check for critical failures
	if rclone.HasCriticalFailure(results) {
		fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
		fmt.Println("║  Critical pre-flight check(s) failed. Cannot start application.  ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
		fmt.Println()
		fmt.Println("Please fix the issues above and try again.")
		fmt.Println("You can skip these checks with --skip-checks (not recommended).")
		return fmt.Errorf("critical pre-flight checks failed")
	}

	// Check for non-critical failures
	if !rclone.AllPassed(results) {
		fmt.Println("⚠ Some optional checks failed. The application will start, but some")
		fmt.Println("  features may not work correctly.")
		fmt.Println()
	}

	fmt.Println("Pre-flight checks completed. Starting application...")
	fmt.Println()

	return nil
}
