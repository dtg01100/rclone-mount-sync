// Package main is the entry point for the rclone-mount-sync TUI application.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dtg01100/rclone-mount-sync/internal/tui"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	// CLI flags
	showVersion := flag.Bool("version", false, "Print version and exit")
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

	// Set version for TUI
	tui.Version = version

	// Run the TUI application
	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
