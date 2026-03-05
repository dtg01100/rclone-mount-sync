package cli

import (
	"fmt"
	"os"
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

	syncCreateCmd.MarkFlagRequired("name")
	syncCreateCmd.MarkFlagRequired("source")
	syncCreateCmd.MarkFlagRequired("destination")
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
	fmt.Fprintln(w, "ID\tNAME\tSOURCE\tDESTINATION\tSCHEDULE\tENABLED")

	for _, j := range cfg.SyncJobs {
		schedule := j.Schedule.OnCalendar
		if schedule == "" {
			schedule = j.Schedule.Type
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%v\n",
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

	generator, err := loadGenerator()
	if err != nil {
		return err
	}

	savedJob := cfg.GetSyncJob(syncCreateName)
	if savedJob == nil {
		return fmt.Errorf("failed to retrieve saved sync job")
	}

	if _, _, err := generator.WriteSyncUnits(savedJob); err != nil {
		return fmt.Errorf("failed to write systemd units: %w", err)
	}

	manager := loadManager()
	if err := manager.DaemonReload(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	if syncCreateEnabled && savedJob.Schedule.Type != "manual" {
		timerName := generator.ServiceName(savedJob.ID, "sync") + ".timer"
		if err := manager.Enable(timerName); err != nil {
			return fmt.Errorf("failed to enable timer: %w", err)
		}
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
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

	_ = manager.StopTimer(timerName)
	_ = manager.DisableTimer(timerName)
	_ = manager.Stop(serviceName)
	_ = manager.Disable(serviceName)
	_ = manager.ResetFailed(serviceName)

	if err := generator.RemoveUnit(serviceName); err != nil {
		return fmt.Errorf("failed to remove service unit: %w", err)
	}

	if job.Schedule.Type != "manual" {
		if err := generator.RemoveUnit(timerName); err != nil {
			return fmt.Errorf("failed to remove timer unit: %w", err)
		}
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
