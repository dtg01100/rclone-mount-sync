// Package config provides configuration management for the rclone-mount-sync application.
// It uses Viper for configuration file handling and supports YAML format.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/pkg/utils"
)

// Config represents the application configuration.
type Config struct {
	Version  string                   `mapstructure:"version"`
	Mounts   []models.MountConfig     `mapstructure:"mounts"`
	SyncJobs []models.SyncJobConfig   `mapstructure:"sync_jobs"`
	Settings Settings                 `mapstructure:"settings"`
	Defaults DefaultConfig            `mapstructure:"defaults"`
}

// Settings holds application-wide settings.
type Settings struct {
	RcloneBinaryPath string `mapstructure:"rclone_binary_path"`
	DefaultMountDir  string `mapstructure:"default_mount_dir"`
	Editor           string `mapstructure:"editor"`
}

// DefaultConfig holds default settings for mounts and sync jobs.
type DefaultConfig struct {
	Mount MountDefaults `mapstructure:"mount"`
	Sync  SyncDefaults  `mapstructure:"sync"`
}

// MountDefaults holds default mount settings.
type MountDefaults struct {
	LogLevel      string `mapstructure:"log_level"`
	VFSCacheMode  string `mapstructure:"vfs_cache_mode"`
	BufferSize    string `mapstructure:"buffer_size"`
}

// SyncDefaults holds default sync job settings.
type SyncDefaults struct {
	LogLevel  string `mapstructure:"log_level"`
	Transfers int    `mapstructure:"transfers"`
	Checkers  int    `mapstructure:"checkers"`
}

// AppConfigDir returns the application configuration directory.
const appName = "rclone-mount-sync"

// Load reads the configuration from the default config file location.
// If the config file doesn't exist, it returns a new Config with defaults.
func Load() (*Config, error) {
	v := viper.New()

	// Set default config file location
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)
	v.AddConfigPath(".")

	// Set defaults
	setDefaults(v)

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, create a new one with defaults
		return newConfigWithDefaults(), nil
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to the default config file location.
func (c *Config) Save() error {
	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	// Ensure config directory exists
	if err := utils.EnsureDir(configDir); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)

	configPath := filepath.Join(configDir, "config.yaml")
	v.SetConfigFile(configPath)

	// Set all config values
	v.Set("version", c.Version)
	v.Set("mounts", c.Mounts)
	v.Set("sync_jobs", c.SyncJobs)
	v.Set("settings", c.Settings)
	v.Set("defaults", c.Defaults)

	// Write config file
	if err := v.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AddMount adds a new mount configuration.
func (c *Config) AddMount(mount models.MountConfig) error {
	// Generate ID if not provided
	if mount.ID == "" {
		mount.ID = generateID()
	}

	// Set timestamps
	now := time.Now()
	mount.CreatedAt = now
	mount.ModifiedAt = now

	// Check for duplicate name
	for _, m := range c.Mounts {
		if m.Name == mount.Name {
			return fmt.Errorf("mount with name %q already exists", mount.Name)
		}
	}

	c.Mounts = append(c.Mounts, mount)
	return nil
}

// RemoveMount removes a mount configuration by name.
func (c *Config) RemoveMount(name string) error {
	for i, m := range c.Mounts {
		if m.Name == name {
			c.Mounts = append(c.Mounts[:i], c.Mounts[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("mount %q not found", name)
}

// GetMount returns a mount configuration by name.
func (c *Config) GetMount(name string) *models.MountConfig {
	for i := range c.Mounts {
		if c.Mounts[i].Name == name {
			return &c.Mounts[i]
		}
	}
	return nil
}

// AddSyncJob adds a new sync job configuration.
func (c *Config) AddSyncJob(job models.SyncJobConfig) error {
	// Generate ID if not provided
	if job.ID == "" {
		job.ID = generateID()
	}

	// Set timestamps
	now := time.Now()
	job.CreatedAt = now
	job.ModifiedAt = now

	// Check for duplicate name
	for _, j := range c.SyncJobs {
		if j.Name == job.Name {
			return fmt.Errorf("sync job with name %q already exists", job.Name)
		}
	}

	c.SyncJobs = append(c.SyncJobs, job)
	return nil
}

// RemoveSyncJob removes a sync job configuration by name.
func (c *Config) RemoveSyncJob(name string) error {
	for i, j := range c.SyncJobs {
		if j.Name == name {
			c.SyncJobs = append(c.SyncJobs[:i], c.SyncJobs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("sync job %q not found", name)
}

// GetSyncJob returns a sync job configuration by name.
func (c *Config) GetSyncJob(name string) *models.SyncJobConfig {
	for i := range c.SyncJobs {
		if c.SyncJobs[i].Name == name {
			return &c.SyncJobs[i]
		}
	}
	return nil
}

// getConfigDir returns the configuration directory path.
func getConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, appName), nil
}

// setDefaults sets default values in viper.
func setDefaults(v *viper.Viper) {
	v.SetDefault("version", "1.0")
	v.SetDefault("settings.rclone_binary_path", "")
	v.SetDefault("settings.default_mount_dir", "~/mnt")
	v.SetDefault("settings.editor", "")
	v.SetDefault("defaults.mount.log_level", "INFO")
	v.SetDefault("defaults.mount.vfs_cache_mode", "full")
	v.SetDefault("defaults.mount.buffer_size", "16M")
	v.SetDefault("defaults.sync.log_level", "INFO")
	v.SetDefault("defaults.sync.transfers", 4)
	v.SetDefault("defaults.sync.checkers", 8)
}

// newConfigWithDefaults creates a new Config with default values.
func newConfigWithDefaults() *Config {
	return &Config{
		Version: "1.0",
		Mounts:  []models.MountConfig{},
		SyncJobs: []models.SyncJobConfig{},
		Settings: Settings{
			RcloneBinaryPath: "",
			DefaultMountDir:  "~/mnt",
			Editor:           "",
		},
		Defaults: DefaultConfig{
			Mount: MountDefaults{
				LogLevel:      "INFO",
				VFSCacheMode:  "full",
				BufferSize:    "16M",
			},
			Sync: SyncDefaults{
				LogLevel:  "INFO",
				Transfers: 4,
				Checkers:  8,
			},
		},
	}
}

// generateID generates a unique ID for mounts and sync jobs.
func generateID() string {
	return uuid.New().String()[:8]
}
