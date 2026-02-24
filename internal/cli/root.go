package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/rclone"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	outputJSON  bool
	showVersion bool
	cliVersion  = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "rclone-mount-sync",
	Short: "Manage rclone mounts and sync jobs via systemd",
	Long: `rclone-mount-sync is a CLI tool for managing rclone mounts and sync jobs
as systemd user services. It provides commands to create, list, start, stop,
and delete mount points and sync jobs.`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config directory (default is $XDG_CONFIG_HOME/rclone-mount-sync)")
	rootCmd.PersistentFlags().BoolVarP(&outputJSON, "json", "j", false, "output in JSON format")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print version and exit")
	rootCmd.AddCommand(cleanupCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

func SetVersion(v string) {
	cliVersion = v
	rootCmd.Version = v
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}

func ExecuteWithVersion(version string) error {
	cliVersion = version
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	return rootCmd.Execute()
}

// loadConfig returns the application configuration, using the --config flag
// if provided. This function is injectable for testing purposes.
var loadConfig = func() (*config.Config, error) {
	if cfgFile != "" {
		if err := os.Setenv("XDG_CONFIG_HOME", cfgFile); err != nil {
			return nil, fmt.Errorf("failed to set config directory: %w", err)
		}
	}
	return config.Load()
}

// loadGenerator returns a new systemd generator instance.
// This function is injectable for testing purposes.
var loadGenerator = func() (*systemd.Generator, error) {
	return systemd.NewGenerator()
}

// loadManager returns a new systemd manager instance.
// This function is injectable for testing purposes.
var loadManager = func() systemd.ServiceManager {
	return systemd.NewManager()
}

// loadRcloneClient returns a new rclone client instance.
// This function is injectable for testing purposes.
var loadRcloneClient = func() *rclone.Client {
	return rclone.NewClient()
}

func printJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func printError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

// findMountByIDOrName searches for a mount by ID or name in the config.
// Returns nil if not found.
func findMountByIDOrName(cfg *config.Config, idOrName string) *models.MountConfig {
	for i := range cfg.Mounts {
		if cfg.Mounts[i].ID == idOrName || cfg.Mounts[i].Name == idOrName {
			return &cfg.Mounts[i]
		}
	}
	return nil
}

// findSyncJobByIDOrName searches for a sync job by ID or name in the config.
// Returns nil if not found.
func findSyncJobByIDOrName(cfg *config.Config, idOrName string) *models.SyncJobConfig {
	for i := range cfg.SyncJobs {
		if cfg.SyncJobs[i].ID == idOrName || cfg.SyncJobs[i].Name == idOrName {
			return &cfg.SyncJobs[i]
		}
	}
	return nil
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up orphaned systemd units",
	Long: `Remove failed rclone units from systemd that no longer have unit files.

This can happen if mounts/sync jobs were deleted improperly or if unit files
were manually removed. The command will:
1. Find all failed rclone units
2. Check if they have corresponding unit files
3. Reset the failed state for units without files`,
	RunE: runCleanup,
}

func runCleanup(cmd *cobra.Command, args []string) error {
	manager := loadManager()
	generator, err := loadGenerator()
	if err != nil {
		return err
	}

	cmd2 := exec.Command("systemctl", "--user", "list-units", "--state=failed", "--no-legend")
	output, err := cmd2.Output()
	if err != nil {
		return fmt.Errorf("failed to list failed units: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	cleaned := 0

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		unitName := fields[1]

		if !strings.HasPrefix(unitName, "rclone-mount-") && !strings.HasPrefix(unitName, "rclone-sync-") {
			continue
		}

		unitPath := filepath.Join(generator.GetSystemdDir(), unitName)
		if _, err := os.Stat(unitPath); os.IsNotExist(err) {
			if err := manager.ResetFailed(unitName); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to reset %s: %v\n", unitName, err)
			} else {
				fmt.Printf("Cleaned up orphaned unit: %s\n", unitName)
				cleaned++
			}
		}
	}

	if cleaned == 0 {
		fmt.Println("No orphaned units found.")
	} else {
		fmt.Printf("\nCleaned up %d orphaned unit(s).\n", cleaned)
	}

	return nil
}
