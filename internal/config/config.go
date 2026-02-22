// Package config provides configuration management for the rclone-mount-sync application.
// It uses Viper for configuration file handling and supports YAML format.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/pkg/utils"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ImportMode defines how configuration should be imported.
type ImportMode int

const (
	// ImportModeMerge merges imported config with existing config.
	// Existing items with the same name are skipped.
	ImportModeMerge ImportMode = iota
	// ImportModeReplace replaces the entire configuration with imported config.
	ImportModeReplace
)

// ExportData represents the data structure for exported configuration.
type ExportData struct {
	Version  string                 `json:"version" yaml:"version"`
	Mounts   []models.MountConfig   `json:"mounts" yaml:"mounts"`
	SyncJobs []models.SyncJobConfig `json:"sync_jobs" yaml:"sync_jobs"`
	Exported string                 `json:"exported" yaml:"exported"`
}

// Config represents the application configuration.
type Config struct {
	Version  string                 `mapstructure:"version"`
	Mounts   []models.MountConfig   `mapstructure:"mounts"`
	SyncJobs []models.SyncJobConfig `mapstructure:"sync_jobs"`
	Settings Settings               `mapstructure:"settings"`
	Defaults DefaultConfig          `mapstructure:"defaults"`
}

// Settings holds application-wide settings.
type Settings struct {
	RcloneBinaryPath string   `mapstructure:"rclone_binary_path"`
	DefaultMountDir  string   `mapstructure:"default_mount_dir"`
	Editor           string   `mapstructure:"editor"`
	RecentPaths      []string `mapstructure:"recent_paths"`
}

// DefaultConfig holds default settings for mounts and sync jobs.
type DefaultConfig struct {
	Mount MountDefaults `mapstructure:"mount"`
	Sync  SyncDefaults  `mapstructure:"sync"`
}

// MountDefaults holds default mount settings.
type MountDefaults struct {
	LogLevel     string `mapstructure:"log_level"`
	VFSCacheMode string `mapstructure:"vfs_cache_mode"`
	BufferSize   string `mapstructure:"buffer_size"`
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
// It uses an atomic write pattern: writes to a temp file first, then renames.
// A backup of the existing config is created before overwriting.
func (c *Config) Save() error {
	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	if err := utils.EnsureDir(configDir); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	backupPath := configPath + ".bak"

	if _, err := os.Stat(configPath); err == nil {
		if err := createBackup(configPath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.SetConfigFile(configPath)

	v.Set("version", c.Version)
	v.Set("mounts", c.Mounts)
	v.Set("sync_jobs", c.SyncJobs)
	v.Set("settings.rclone_binary_path", c.Settings.RcloneBinaryPath)
	v.Set("settings.default_mount_dir", c.Settings.DefaultMountDir)
	v.Set("settings.editor", c.Settings.Editor)
	v.Set("settings.recent_paths", c.Settings.RecentPaths)
	v.Set("defaults.mount.log_level", c.Defaults.Mount.LogLevel)
	v.Set("defaults.mount.vfs_cache_mode", c.Defaults.Mount.VFSCacheMode)
	v.Set("defaults.mount.buffer_size", c.Defaults.Mount.BufferSize)
	v.Set("defaults.sync.log_level", c.Defaults.Sync.LogLevel)
	v.Set("defaults.sync.transfers", c.Defaults.Sync.Transfers)
	v.Set("defaults.sync.checkers", c.Defaults.Sync.Checkers)

	tempPath := configPath + ".tmp.yaml"

	if err := v.WriteConfigAs(tempPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if err := os.Rename(tempPath, configPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// RestoreFromBackup restores the configuration from the backup file.
// Returns an error if no backup exists.
func RestoreFromBackup() error {
	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	backupPath := configPath + ".bak"

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup file found")
	}

	if err := os.Rename(backupPath, configPath); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	return nil
}

// HasBackup returns true if a backup file exists.
func HasBackup() (bool, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return false, fmt.Errorf("failed to get config directory: %w", err)
	}

	backupPath := filepath.Join(configDir, "config.yaml.bak")
	_, err = os.Stat(backupPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// createBackup creates a backup of the existing config file.
// It overwrites any existing backup to keep only the most recent one.
func createBackup(configPath, backupPath string) error {
	srcFile, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	dstFile, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dstFile.Close()

	if _, err := srcFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek config file: %w", err)
	}

	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		return fmt.Errorf("failed to copy config to backup: %w", err)
	}

	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync backup file: %w", err)
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

// AddRecentPath adds a path to the front of the recent paths list,
// removes duplicates, and keeps only the 10 most recent paths.
func (c *Config) AddRecentPath(path string) {
	var result []string
	result = append(result, path)
	for _, p := range c.Settings.RecentPaths {
		if p != path {
			result = append(result, p)
		}
	}
	if len(result) > 10 {
		result = result[:10]
	}
	c.Settings.RecentPaths = result
}

// getConfigDir returns the configuration directory path.
var getConfigDir = func() (string, error) {
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
	v.SetDefault("settings.recent_paths", []string{})
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
		Version:  "1.0",
		Mounts:   []models.MountConfig{},
		SyncJobs: []models.SyncJobConfig{},
		Settings: Settings{
			RcloneBinaryPath: "",
			DefaultMountDir:  "~/mnt",
			Editor:           "",
			RecentPaths:      []string{},
		},
		Defaults: DefaultConfig{
			Mount: MountDefaults{
				LogLevel:     "INFO",
				VFSCacheMode: "full",
				BufferSize:   "16M",
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

// ExportConfig exports the current mounts and sync jobs to a file.
// The file format is determined by the file extension (.json or .yaml/.yml).
func (c *Config) ExportConfig(filePath string) error {
	data := ExportData{
		Version:  c.Version,
		Mounts:   c.Mounts,
		SyncJobs: c.SyncJobs,
		Exported: time.Now().Format(time.RFC3339),
	}

	fileDir := filepath.Dir(filePath)
	if fileDir != "" && fileDir != "." {
		if err := utils.EnsureDir(fileDir); err != nil {
			return fmt.Errorf("failed to create export directory: %w", err)
		}
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(data); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	case ".yaml", ".yml":
		encoder := yaml.NewEncoder(file)
		encoder.SetIndent(2)
		if err := encoder.Encode(data); err != nil {
			return fmt.Errorf("failed to encode YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported file format: %s (use .json, .yaml, or .yml)", ext)
	}

	return nil
}

// ImportConfig imports mounts and sync jobs from a file.
// The import mode determines how conflicts are handled.
func (c *Config) ImportConfig(filePath string, mode ImportMode) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("import file does not exist: %s", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open import file: %w", err)
	}
	defer file.Close()

	var data ExportData
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".json":
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&data); err != nil {
			return fmt.Errorf("failed to decode JSON: %w", err)
		}
	case ".yaml", ".yml":
		decoder := yaml.NewDecoder(file)
		if err := decoder.Decode(&data); err != nil {
			return fmt.Errorf("failed to decode YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported file format: %s (use .json, .yaml, or .yml)", ext)
	}

	if data.Version == "" && len(data.Mounts) == 0 && len(data.SyncJobs) == 0 {
		return fmt.Errorf("invalid config file: no valid configuration data found")
	}

	switch mode {
	case ImportModeReplace:
		c.Mounts = data.Mounts
		c.SyncJobs = data.SyncJobs
	case ImportModeMerge:
		c.mergeImport(data)
	}

	return nil
}

// mergeImport merges the imported data with the existing configuration.
// Items with duplicate names are skipped with an error recorded.
func (c *Config) mergeImport(data ExportData) {
	existingMountNames := make(map[string]bool)
	for _, m := range c.Mounts {
		existingMountNames[m.Name] = true
	}

	for _, mount := range data.Mounts {
		if existingMountNames[mount.Name] {
			continue
		}
		if mount.ID == "" {
			mount.ID = generateID()
		}
		if mount.CreatedAt.IsZero() {
			mount.CreatedAt = time.Now()
		}
		if mount.ModifiedAt.IsZero() {
			mount.ModifiedAt = time.Now()
		}
		c.Mounts = append(c.Mounts, mount)
	}

	existingSyncJobNames := make(map[string]bool)
	for _, j := range c.SyncJobs {
		existingSyncJobNames[j.Name] = true
	}

	for _, job := range data.SyncJobs {
		if existingSyncJobNames[job.Name] {
			continue
		}
		if job.ID == "" {
			job.ID = generateID()
		}
		if job.CreatedAt.IsZero() {
			job.CreatedAt = time.Now()
		}
		if job.ModifiedAt.IsZero() {
			job.ModifiedAt = time.Now()
		}
		c.SyncJobs = append(c.SyncJobs, job)
	}
}
