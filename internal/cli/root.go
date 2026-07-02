package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dtg01100/rclone-mount-sync/internal/config"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/internal/systemd"
	"github.com/dtg01100/rclone-mount-sync/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	outputJSON  bool
	showVersion bool
)

var rootCmd = &cobra.Command{
	Use:   "rclone-mount-sync",
	Short: "Manage rclone mounts and sync jobs via systemd",
	Long: `rclone-mount-sync is a CLI tool for managing rclone mounts and sync jobs
as systemd user services. It provides commands to create, list, start, stop,
and delete mount points and sync jobs.`,
	// SilenceUsage prevents cobra from dumping the full help text on
	// every RunE error; we only want it shown for genuine usage errors
	// (unknown command, bad flag, missing arg), which cobra handles
	// before RunE runs.
	SilenceUsage: true,
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
	rootCmd.Version = v
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}

func ExecuteWithVersion(version string) error {
	SetVersion(version)
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
var loadGenerator = systemd.NewGenerator

// loadManager returns a new systemd manager instance.
// This function is injectable for testing purposes.
var loadManager = func() systemd.ServiceManager {
	return systemd.NewManager()
}

func printJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// printError routes a single error line to the central ErrorSink.
// Tests swap utils.ErrorSink to capture; production defaults to
// os.Stderr via the utils package.
func printError(err error) {
	utils.NoteError("%v", err)
}

// findMountByIDOrName searches for a mount by ID or name in the config.
// Returns nil if not found. The returned pointer is to a copy held under
// the config's read lock, making it safe for concurrent access.
func findMountByIDOrName(cfg *config.Config, idOrName string) *models.MountConfig {
	return cfg.FindMount(idOrName)
}

// findSyncJobByIDOrName searches for a sync job by ID or name in the config.
// Returns nil if not found. The returned pointer is to a copy held under
// the config's read lock, making it safe for concurrent access.
func findSyncJobByIDOrName(cfg *config.Config, idOrName string) *models.SyncJobConfig {
	return cfg.FindSyncJob(idOrName)
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

	systemctlPath := "systemctl"
	if manager != nil {
		systemctlPath = manager.SystemctlPath()
	}

	cmd2 := exec.Command(systemctlPath, "--user", "list-units", "--state=failed", "--no-legend") //nolint:gosec
	output, err := cmd2.Output()
	// `list-units --state=failed` exits non-zero when nothing matches
	// (exit 1 in practice; older versions used 3). Treat both "no output"
	// and the no-match exit code as "nothing to do". Any other error
	// means systemctl itself failed (DBus down, etc.) and we surface it.
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return fmt.Errorf("failed to list failed units: %w", err)
		}
		if strings.TrimSpace(string(output)) == "" {
			fmt.Println("No failed units found.")
			return nil
		}
	}

	lines := strings.Split(string(output), "\n")
	cleaned := 0
	attempted := 0

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		unitName := fields[0]

		if !strings.HasPrefix(unitName, "rclone-mount-") && !strings.HasPrefix(unitName, "rclone-sync-") {
			continue
		}

		unitPath := filepath.Join(generator.GetSystemdDir(), unitName)
		if _, err := os.Stat(unitPath); os.IsNotExist(err) {
			attempted++
			if err := manager.ResetFailed(unitName); err != nil {
				utils.NoteWarning("failed to reset %s: %v", unitName, err)
			} else {
				fmt.Printf("Cleaned up orphaned unit: %s\n", unitName)
				cleaned++
			}
		}
	}

	if attempted == 0 {
		fmt.Println("No orphaned units found.")
	} else {
		fmt.Printf("\nCleaned up %d of %d orphaned unit(s).\n", cleaned, attempted)
	}

	// Return a non-nil error when the command did real work but every
	// ResetFailed call failed. Scripts that wrap `cleanup` can then
	// distinguish "nothing to do" (exit 0) from "all cleanups failed"
	// (non-zero). Partial success is still exit 0.
	if attempted > 0 && cleaned == 0 {
		return fmt.Errorf("cleanup failed for all %d orphaned unit(s)", attempted)
	}

	return nil
}
