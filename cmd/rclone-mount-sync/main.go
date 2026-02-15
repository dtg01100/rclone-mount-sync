// Package main is the entry point for the rclone-mount-sync TUI application.
package main

import (
	"fmt"
	"os"

	"github.com/dlafreniere/rclone-mount-sync/internal/tui"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	// Set version for TUI
	tui.Version = version

	// Run the TUI application
	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
