package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Manage sync jobs",
	Long:  `Create, list, delete, and run rclone sync jobs.`,
}

var syncListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sync jobs",
	RunE:  runSyncList,
}

var syncCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new sync job",
	Long: `Create a new rclone sync job with systemd service and timer.

The sync job will be created with default options. Use flags to customize.`,
	RunE: runSyncCreate,
}

var syncDeleteCmd = &cobra.Command{
	Use:   "delete <name-or-id>",
	Short: "Delete a sync job",
	Long: `Delete a sync job configuration and its systemd units.

This will stop and disable the timer and service before removal.`,
	Args: cobra.ExactArgs(1),
	RunE: runSyncDelete,
}

var syncRunCmd = &cobra.Command{
	Use:   "run <name-or-id>",
	Short: "Run a sync job immediately",
	Long: `Trigger an immediate sync job run.

This starts the systemd service regardless of the timer schedule.`,
	Args: cobra.ExactArgs(1),
	RunE: runSyncRun,
}

var (
	syncCreateName        string
	syncCreateSource      string
	syncCreateDestination string
	syncCreateSchedule    string
	syncCreateEnabled     bool
)

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.AddCommand(syncListCmd)
	syncCmd.AddCommand(syncCreateCmd)
	syncCmd.AddCommand(syncDeleteCmd)
	syncCmd.AddCommand(syncRunCmd)

	syncCreateCmd.Flags().StringVar(&syncCreateName, "name", "", "sync job name (required)")
	syncCreateCmd.Flags().StringVarP(&syncCreateSource, "source", "s", "", "source path (required, e.g., gdrive:/Photos)")
	syncCreateCmd.Flags().StringVarP(&syncCreateDestination, "destination", "d", "", "destination path (required)")
	syncCreateCmd.Flags().StringVar(&syncCreateSchedule, "schedule", "daily", "schedule (e.g., daily, hourly, '*-*-* 02:00:00')")
	syncCreateCmd.Flags().BoolVar(&syncCreateEnabled, "enabled", true, "enable the timer")

	_ = syncCreateCmd.MarkFlagRequired("name")
	_ = syncCreateCmd.MarkFlagRequired("source")
	_ = syncCreateCmd.MarkFlagRequired("destination")
}

func runSyncList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if outputJSON {
		return printJSON(cfg.SyncJobs)
	}

	if len(cfg.SyncJobs) == 0 {
		fmt.Println("No sync jobs configured.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tSOURCE\tDESTINATION\tSCHEDULE\tENABLED")

	for _, j := range cfg.SyncJobs {
		schedule := j.Schedule.OnCalendar
		if schedule == "" {
			schedule = j.Schedule.Type
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%v\n",
			j.ID, j.Name, j.Source, j.Destination, schedule, j.Enabled)
	}

	return w.Flush()
}

func runSyncCreate(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	job := models.SyncJobConfig{
		Name:        syncCreateName,
		Source:      syncCreateSource,
		Destination: syncCreateDestination,
		Enabled:     syncCreateEnabled,
		SyncOptions: models.SyncOptions{
			Direction: "sync",
			LogLevel:  cfg.Defaults.Sync.LogLevel,
			Transfers: cfg.Defaults.Sync.Transfers,
			Checkers:  cfg.Defaults.Sync.Checkers,
		},
		Schedule: models.ScheduleConfig{
			Type:       "timer",
			OnCalendar: syncCreateSchedule,
		},
	}

	if err := cfg.AddSyncJob(job); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	generator, err := loadGenerator()
	if err != nil {
		_ = cfg.RemoveSyncJob(job.Name)
		_ = cfg.Save()
		return err
	}

	savedJob := cfg.GetSyncJob(syncCreateName)
	if savedJob == nil {
		return fmt.Errorf("failed to retrieve saved sync job")
	}

	servicePath, _, err := generator.WriteSyncUnits(savedJob)
	if err != nil {
		// WriteSyncUnits writes the service file first, then the timer.
		// If it returns an error, the service file may already be on
		// disk; remove it so we don't leave a half-configured unit
		// behind for the next daemon-reload to load.
		if servicePath != "" {
			serviceName := filepath.Base(servicePath)
			if remErr := generator.RemoveUnit(serviceName); remErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove partial service unit %s: %v\n", serviceName, remErr)
			}
		}
		_ = cfg.RemoveSyncJob(job.Name)
		_ = cfg.Save()
		return fmt.Errorf("failed to write systemd units: %w", err)
	}

	manager := loadManager()
	if err := manager.DaemonReload(); err != nil {
		serviceName := generator.ServiceName(savedJob.ID, "sync") + ".service"
		timerName := generator.ServiceName(savedJob.ID, "sync") + ".timer"
		if remErr := generator.RemoveUnit(serviceName); remErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove service %s: %v\n", serviceName, remErr)
		}
		if remErr := generator.RemoveUnit(timerName); remErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove timer %s: %v\n", timerName, remErr)
		}
		if remErr := cfg.RemoveSyncJob(job.Name); remErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove sync job from config: %v\n", remErr)
		}
		if saveErr := cfg.Save(); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", saveErr)
		}
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	if syncCreateEnabled && savedJob.Schedule.Type != "manual" {
		timerName := generator.ServiceName(savedJob.ID, "sync") + ".timer"
		if err := manager.Enable(timerName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to enable timer: %v\n", err)
		}
	}

	fmt.Printf("Sync job '%s' created successfully (ID: %s)\n", savedJob.Name, savedJob.ID)
	return nil
}

func runSyncDelete(cmd *cobra.Command, args []string) error {
	idOrName := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	job := findSyncJobByIDOrName(cfg, idOrName)
	if job == nil {
		return fmt.Errorf("sync job '%s' not found", idOrName)
	}

	generator, err := loadGenerator()
	if err != nil {
		return err
	}

	manager := loadManager()

	serviceName := generator.ServiceName(job.ID, "sync") + ".service"
	timerName := generator.ServiceName(job.ID, "sync") + ".timer"

	// Attempt to stop and disable, but don't fail if service doesn't exist
	if err := manager.StopTimer(timerName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to stop timer %s: %v\n", timerName, err)
	}
	if err := manager.DisableTimer(timerName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to disable timer %s: %v\n", timerName, err)
	}
	if err := manager.Stop(serviceName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to stop %s: %v\n", serviceName, err)
	}
	if err := manager.Disable(serviceName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to disable %s: %v\n", serviceName, err)
	}
	if err := manager.ResetFailed(serviceName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to reset failed state for %s: %v\n", serviceName, err)
	}

	// Remove the config entry first so a subsequent unit-removal failure
	// doesn't leave the config referencing a job whose unit is already
	// gone. If RemoveUnit fails we still try to clean up the other unit
	// rather than leaving the system in a half-deleted state.
	removedService := false
	removedTimer := false
	if err := generator.RemoveUnit(serviceName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove service unit %s: %v\n", serviceName, err)
	} else {
		removedService = true
	}

	if job.Schedule.Type != "manual" {
		if err := generator.RemoveUnit(timerName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove timer unit %s: %v\n", timerName, err)
		} else {
			removedTimer = true
		}
	} else {
		removedTimer = true
	}

	if err := manager.DaemonReload(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	if err := cfg.RemoveSyncJob(job.Name); err != nil {
		return fmt.Errorf("failed to remove from config: %w", err)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	if !removedService || !removedTimer {
		return fmt.Errorf("sync job %q removed from config but some unit files could not be removed", job.Name)
	}

	fmt.Printf("Sync job '%s' deleted successfully\n", job.Name)
	return nil
}

func runSyncRun(cmd *cobra.Command, args []string) error {
	idOrName := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	job := findSyncJobByIDOrName(cfg, idOrName)
	if job == nil {
		return fmt.Errorf("sync job '%s' not found", idOrName)
	}

	generator, err := loadGenerator()
	if err != nil {
		return err
	}

	manager := loadManager()
	serviceName := generator.ServiceName(job.ID, "sync") + ".service"

	if err := manager.RunSyncNow(serviceName); err != nil {
		return fmt.Errorf("failed to run sync job: %w", err)
	}

	fmt.Printf("Sync job '%s' started\n", job.Name)
	return nil
}
