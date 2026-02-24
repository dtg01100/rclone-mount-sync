package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/spf13/cobra"
)

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Manage rclone mounts",
	Long:  `Create, list, delete, start, and stop rclone mount services.`,
}

var mountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all mounts",
	RunE:  runMountList,
}

var mountCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new mount",
	Long: `Create a new rclone mount with systemd service.

The mount will be created with default options. Use flags to customize.`,
	RunE: runMountCreate,
}

var mountDeleteCmd = &cobra.Command{
	Use:   "delete <name-or-id>",
	Short: "Delete a mount",
	Long: `Delete a mount configuration and its systemd service.

This will stop and disable the service before removal.`,
	Args: cobra.ExactArgs(1),
	RunE: runMountDelete,
}

var mountStartCmd = &cobra.Command{
	Use:   "start <name-or-id>",
	Short: "Start a mount service",
	Args:  cobra.ExactArgs(1),
	RunE:  runMountStart,
}

var mountStopCmd = &cobra.Command{
	Use:   "stop <name-or-id>",
	Short: "Stop a mount service",
	Args:  cobra.ExactArgs(1),
	RunE:  runMountStop,
}

var (
	mountCreateName       string
	mountCreateRemote     string
	mountCreateRemotePath string
	mountCreateMountPoint string
	mountCreateEnabled    bool
	mountCreateAutoStart  bool
)

func init() {
	rootCmd.AddCommand(mountCmd)
	mountCmd.AddCommand(mountListCmd)
	mountCmd.AddCommand(mountCreateCmd)
	mountCmd.AddCommand(mountDeleteCmd)
	mountCmd.AddCommand(mountStartCmd)
	mountCmd.AddCommand(mountStopCmd)

	mountCreateCmd.Flags().StringVar(&mountCreateName, "name", "", "mount name (required)")
	mountCreateCmd.Flags().StringVar(&mountCreateRemote, "remote", "", "rclone remote name (required)")
	mountCreateCmd.Flags().StringVarP(&mountCreateRemotePath, "remote-path", "p", "/", "remote path to mount")
	mountCreateCmd.Flags().StringVarP(&mountCreateMountPoint, "mount-point", "m", "", "local mount point (required)")
	mountCreateCmd.Flags().BoolVar(&mountCreateEnabled, "enabled", true, "enable the service")
	mountCreateCmd.Flags().BoolVar(&mountCreateAutoStart, "auto-start", false, "start the service immediately")

	mountCreateCmd.MarkFlagRequired("name")
	mountCreateCmd.MarkFlagRequired("remote")
	mountCreateCmd.MarkFlagRequired("mount-point")
}

func runMountList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if outputJSON {
		return printJSON(cfg.Mounts)
	}

	if len(cfg.Mounts) == 0 {
		fmt.Println("No mounts configured.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tREMOTE\tMOUNT POINT\tENABLED\tAUTO-START")

	for _, m := range cfg.Mounts {
		remote := m.Remote + m.RemotePath
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%v\t%v\n",
			m.ID, m.Name, remote, m.MountPoint, m.Enabled, m.AutoStart)
	}

	return w.Flush()
}

func runMountCreate(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	mount := models.MountConfig{
		Name:       mountCreateName,
		Remote:     mountCreateRemote,
		RemotePath: mountCreateRemotePath,
		MountPoint: mountCreateMountPoint,
		Enabled:    mountCreateEnabled,
		AutoStart:  mountCreateAutoStart,
		MountOptions: models.MountOptions{
			VFSCacheMode: cfg.Defaults.Mount.VFSCacheMode,
			BufferSize:   cfg.Defaults.Mount.BufferSize,
			LogLevel:     cfg.Defaults.Mount.LogLevel,
		},
	}

	if err := cfg.AddMount(mount); err != nil {
		return err
	}

	generator, err := loadGenerator()
	if err != nil {
		return err
	}

	savedMount := cfg.GetMount(mountCreateName)
	if savedMount == nil {
		return fmt.Errorf("failed to retrieve saved mount")
	}

	if _, err := generator.WriteMountService(savedMount); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}

	manager := loadManager()
	if err := manager.DaemonReload(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	serviceName := generator.ServiceName(savedMount.ID, "mount") + ".service"

	if mountCreateEnabled {
		if err := manager.Enable(serviceName); err != nil {
			return fmt.Errorf("failed to enable service: %w", err)
		}
	}

	if mountCreateAutoStart {
		if err := manager.Start(serviceName); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Mount '%s' created successfully (ID: %s)\n", savedMount.Name, savedMount.ID)
	return nil
}

func runMountDelete(cmd *cobra.Command, args []string) error {
	idOrName := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	mount := findMountByIDOrName(cfg, idOrName)
	if mount == nil {
		return fmt.Errorf("mount '%s' not found", idOrName)
	}

	generator, err := loadGenerator()
	if err != nil {
		return err
	}

	manager := loadManager()

	serviceName := generator.ServiceName(mount.ID, "mount") + ".service"

	_ = manager.Stop(serviceName)
	_ = manager.Disable(serviceName)
	_ = manager.ResetFailed(serviceName)

	if err := generator.RemoveUnit(serviceName); err != nil {
		return fmt.Errorf("failed to remove unit file: %w", err)
	}

	if err := manager.DaemonReload(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	if err := cfg.RemoveMount(mount.Name); err != nil {
		return fmt.Errorf("failed to remove from config: %w", err)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Mount '%s' deleted successfully\n", mount.Name)
	return nil
}

func runMountStart(cmd *cobra.Command, args []string) error {
	idOrName := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	mount := findMountByIDOrName(cfg, idOrName)
	if mount == nil {
		return fmt.Errorf("mount '%s' not found", idOrName)
	}

	generator, err := loadGenerator()
	if err != nil {
		return err
	}

	manager := loadManager()
	serviceName := generator.ServiceName(mount.ID, "mount") + ".service"

	if err := manager.Start(serviceName); err != nil {
		return fmt.Errorf("failed to start mount: %w", err)
	}

	fmt.Printf("Mount '%s' started successfully\n", mount.Name)
	return nil
}

func runMountStop(cmd *cobra.Command, args []string) error {
	idOrName := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	mount := findMountByIDOrName(cfg, idOrName)
	if mount == nil {
		return fmt.Errorf("mount '%s' not found", idOrName)
	}

	generator, err := loadGenerator()
	if err != nil {
		return err
	}

	manager := loadManager()
	serviceName := generator.ServiceName(mount.ID, "mount") + ".service"

	if err := manager.Stop(serviceName); err != nil {
		return fmt.Errorf("failed to stop mount: %w", err)
	}

	fmt.Printf("Mount '%s' stopped successfully\n", mount.Name)
	return nil
}
