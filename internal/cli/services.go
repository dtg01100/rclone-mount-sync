package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/spf13/cobra"
)

var servicesCmd = &cobra.Command{
	Use:   "services",
	Short: "Manage rclone systemd services",
	Long:  `List, check status, and view logs for rclone systemd services.`,
}

var servicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all rclone services",
	RunE:  runServicesList,
}

var servicesStatusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show service status",
	Long: `Show detailed status for a rclone service.

The name can be the service name (e.g., rclone-mount-abc123.service) or
a shortened version (e.g., rclone-mount-abc123).`,
	Args: cobra.ExactArgs(1),
	RunE: runServicesStatus,
}

var servicesLogsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Show service logs",
	Long: `Show journal logs for a rclone service.

The name can be the service name (e.g., rclone-mount-abc123.service) or
a shortened version (e.g., rclone-mount-abc123).`,
	Args: cobra.ExactArgs(1),
	RunE: runServicesLogs,
}

var (
	logsLines  int
	logsFollow bool
)

func init() {
	rootCmd.AddCommand(servicesCmd)
	servicesCmd.AddCommand(servicesListCmd)
	servicesCmd.AddCommand(servicesStatusCmd)
	servicesCmd.AddCommand(servicesLogsCmd)

	servicesLogsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "number of lines to show")
	servicesLogsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "follow log output")
}

func runServicesList(cmd *cobra.Command, args []string) error {
	manager := loadManager()

	services, err := manager.ListServices()
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	if outputJSON {
		return printJSON(services)
	}

	if len(services) == 0 {
		fmt.Println("No rclone services found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATE\tENABLED")

	for _, s := range services {
		state := s.State
		if s.Active {
			state = "running"
		}
		fmt.Fprintf(w, "%s\t%s\t%v\n", s.Name, state, s.Enabled)
	}

	return w.Flush()
}

func runServicesStatus(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !strings.HasSuffix(name, ".service") {
		name = name + ".service"
	}

	manager := loadManager()

	status, err := manager.GetDetailedStatus(name)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if outputJSON {
		return printJSON(status)
	}

	printServiceStatus(status)
	return nil
}

func printServiceStatus(status *models.ServiceStatus) {
	fmt.Printf("Name: %s\n", status.Name)
	fmt.Printf("Type: %s\n", status.Type)
	fmt.Printf("Load State: %s\n", status.LoadState)
	fmt.Printf("Active State: %s\n", status.ActiveState)
	fmt.Printf("Sub State: %s\n", status.SubState)
	fmt.Printf("Enabled: %v\n", status.Enabled)

	if status.MainPID > 0 {
		fmt.Printf("Main PID: %d\n", status.MainPID)
	}

	if status.ExitCode > 0 {
		fmt.Printf("Exit Code: %d\n", status.ExitCode)
	}

	if !status.ActivatedAt.IsZero() {
		fmt.Printf("Activated: %s\n", status.ActivatedAt.Format(time.RFC3339))
	}

	if !status.InactiveAt.IsZero() {
		fmt.Printf("Inactive: %s\n", status.InactiveAt.Format(time.RFC3339))
	}

	if status.Type == "mount" {
		fmt.Printf("Mount Point: %s\n", status.MountPoint)
		fmt.Printf("Is Mounted: %v\n", status.IsMounted)
	}

	if status.Type == "sync" {
		fmt.Printf("Timer Active: %v\n", status.TimerActive)
		if !status.LastRun.IsZero() {
			fmt.Printf("Last Run: %s\n", status.LastRun.Format(time.RFC3339))
		}
		if !status.NextRun.IsZero() {
			fmt.Printf("Next Run: %s\n", status.NextRun.Format(time.RFC3339))
		}
	}
}

func runServicesLogs(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !strings.HasSuffix(name, ".service") {
		name = name + ".service"
	}

	manager := loadManager()

	if logsFollow {
		fmt.Println("Follow mode is not supported in this context.")
		fmt.Println("Use: journalctl --user -u " + name + " -f")
		return nil
	}

	logs, err := manager.GetLogs(name, logsLines)
	if err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}

	fmt.Print(logs)
	return nil
}
