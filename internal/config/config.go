// Package config provides configuration management for the rclone-mount-sync application.
// It uses Viper for configuration file handling and supports YAML format.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dtg01100/rclone-mount-sync/internal/models"
	"github.com/dtg01100/rclone-mount-sync/pkg/utils"
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
	mu       sync.RWMutex
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
	// Only look in the XDG config dir. Adding "." would silently load a
	// config.yaml from the current working directory when the XDG file
	// is absent — a real exploitable surprise if the user runs the TUI
	// from an untrusted directory.
	v.AddConfigPath(configDir)

	// Set defaults
	setDefaults(v)

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		return newConfigWithDefaults(), nil
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Reload re-reads the configuration from disk and updates the Config.
// This allows the application to pick up changes made externally (e.g., via CLI).
func (c *Config) Reload() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			c.Mounts = nil
			c.SyncJobs = nil
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Update the existing config struct
	c.Version = cfg.Version
	c.Mounts = cfg.Mounts
	c.SyncJobs = cfg.SyncJobs
	c.Settings = cfg.Settings
	c.Defaults = cfg.Defaults

	return nil
}

// Save writes the configuration to the default config file location.
// It uses an atomic write pattern: writes to a temp file first, then renames.
// A backup of the existing config is created before overwriting.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

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
		if rmErr := os.Remove(tempPath); rmErr != nil && !os.IsNotExist(rmErr) {
			return fmt.Errorf("failed to write config file: %w; cleanup failed: %w", err, rmErr)
		}
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Lock down permissions before the file becomes visible at its final
	// path. The config may hold references to rclone remotes and paths;
	// world-readable (0644) would expose them.
	if err := os.Chmod(tempPath, 0o600); err != nil {
		if rmErr := os.Remove(tempPath); rmErr != nil && !os.IsNotExist(rmErr) {
			return fmt.Errorf("failed to chmod config: %w; cleanup failed: %w", err, rmErr)
		}
		return fmt.Errorf("failed to chmod config: %w", err)
	}

	if err := os.Rename(tempPath, configPath); err != nil {
		if rmErr := os.Remove(tempPath); rmErr != nil && !os.IsNotExist(rmErr) {
			return fmt.Errorf("failed to rename temp file: %w; cleanup failed: %w", err, rmErr)
		}
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// RestoreFromBackup restores the configuration from the backup file.
// It copies the backup to the config path so the backup is preserved.
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

	src, err := os.Open(backupPath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer func() {
		if cerr := src.Close(); cerr != nil {
			utils.NoteWarning("failed to close backup file: %v", cerr)
		}
	}()

	dst, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer func() {
		if cerr := dst.Close(); cerr != nil {
			utils.NoteWarning("failed to close config file: %v", cerr)
		}
	}()

	if _, err := dst.ReadFrom(src); err != nil {
		return fmt.Errorf("failed to copy backup to config: %w", err)
	}

	if err := dst.Sync(); err != nil {
		return fmt.Errorf("failed to sync config file: %w", err)
	}

	// Force 0600 even if the backup file itself was world-readable.
	if err := os.Chmod(configPath, 0o600); err != nil {
		return fmt.Errorf("failed to chmod restored config: %w", err)
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
// The backup is written with mode 0600 regardless of the source file's
// permissions so that a previously-world-readable config doesn't carry
// that perm forward into the backup.
func createBackup(configPath, backupPath string) error {
	srcFile, err := os.Open(configPath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() {
		if cerr := srcFile.Close(); cerr != nil {
			utils.NoteWarning("failed to close config file: %v", cerr)
		}
	}()

	dstFile, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() {
		if cerr := dstFile.Close(); cerr != nil {
			utils.NoteWarning("failed to close backup file: %v", cerr)
		}
	}()

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
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := mount.Validate(); err != nil {
		return err
	}

	if mount.RemotePath == "" {
		mount.RemotePath = "/"
	}

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
		if m.ID == mount.ID {
			return fmt.Errorf("mount with ID %q already exists", mount.ID)
		}
	}

	c.Mounts = append(c.Mounts, mount)
	return nil
}

// RemoveMount removes a mount configuration by name.
func (c *Config) RemoveMount(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, m := range c.Mounts {
		if m.Name == name {
			c.Mounts = append(c.Mounts[:i], c.Mounts[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("mount %q not found", name)
}

// GetMount returns a copy of a mount configuration by name.
// The returned pointer is to a copy, not the internal slice element,
// making it safe for concurrent access.
func (c *Config) GetMount(name string) *models.MountConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := range c.Mounts {
		if c.Mounts[i].Name == name {
			result := c.Mounts[i] // copy
			return &result
		}
	}
	return nil
}

// FindMount returns a copy of the mount whose ID or name matches idOrName.
// Returns nil if no such mount exists. The returned pointer is to a copy
// and is safe for concurrent access.
func (c *Config) FindMount(idOrName string) *models.MountConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := range c.Mounts {
		if c.Mounts[i].ID == idOrName || c.Mounts[i].Name == idOrName {
			result := c.Mounts[i] // copy
			return &result
		}
	}
	return nil
}

// AddSyncJob adds a new sync job configuration.
func (c *Config) AddSyncJob(job models.SyncJobConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := job.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(job.SyncOptions.Direction) == "" {
		job.SyncOptions.Direction = "sync"
	}

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
		if j.ID == job.ID {
			return fmt.Errorf("sync job with ID %q already exists", job.ID)
		}
	}

	c.SyncJobs = append(c.SyncJobs, job)
	return nil
}

// RemoveSyncJob removes a sync job configuration by name.
func (c *Config) RemoveSyncJob(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, j := range c.SyncJobs {
		if j.Name == name {
			c.SyncJobs = append(c.SyncJobs[:i], c.SyncJobs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("sync job %q not found", name)
}

// SetMounts replaces all mount configurations atomically.
func (c *Config) SetMounts(mounts []models.MountConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Mounts = mounts
}

// UpdateMount updates an existing mount configuration by ID.
// Returns an error if the mount is not found.
func (c *Config) UpdateMount(updated models.MountConfig) error {
	if err := updated.Validate(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for i, m := range c.Mounts {
		if m.ID == updated.ID {
			updated.ModifiedAt = time.Now()
			if updated.CreatedAt.IsZero() {
				updated.CreatedAt = m.CreatedAt
			}
			c.Mounts[i] = updated
			return nil
		}
	}
	return fmt.Errorf("mount with ID %q not found", updated.ID)
}

// SetSyncJobs replaces all sync job configurations atomically.
func (c *Config) SetSyncJobs(jobs []models.SyncJobConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SyncJobs = jobs
}

// UpdateSyncJob updates an existing sync job configuration by ID.
// Returns an error if the sync job is not found.
func (c *Config) UpdateSyncJob(updated models.SyncJobConfig) error {
	if err := updated.Validate(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for i, j := range c.SyncJobs {
		if j.ID == updated.ID {
			updated.ModifiedAt = time.Now()
			if updated.CreatedAt.IsZero() {
				updated.CreatedAt = j.CreatedAt
			}
			c.SyncJobs[i] = updated
			return nil
		}
	}
	return fmt.Errorf("sync job with ID %q not found", updated.ID)
}

// GetSyncJob returns a copy of a sync job configuration by name.
// The returned pointer is to a copy, not the internal slice element,
// making it safe for concurrent access.
func (c *Config) GetSyncJob(name string) *models.SyncJobConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := range c.SyncJobs {
		if c.SyncJobs[i].Name == name {
			result := c.SyncJobs[i] // copy
			return &result
		}
	}
	return nil
}

// FindSyncJob returns a copy of the sync job whose ID or name matches
// idOrName. Returns nil if no such sync job exists. The returned pointer is
// to a copy and is safe for concurrent access.
func (c *Config) FindSyncJob(idOrName string) *models.SyncJobConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := range c.SyncJobs {
		if c.SyncJobs[i].ID == idOrName || c.SyncJobs[i].Name == idOrName {
			result := c.SyncJobs[i] // copy
			return &result
		}
	}
	return nil
}

// AddRecentPath adds a path to the front of the recent paths list,
// removes duplicates, and keeps only the 10 most recent paths.
func (c *Config) AddRecentPath(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

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
	return utils.GenerateID()
}

// ExportConfig exports the current mounts and sync jobs to a file.
// The file format is determined by the file extension (.json or .yaml/.yml).
func (c *Config) ExportConfig(filePath string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

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

	file, err := os.Create(filePath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer func() {
		// fsync before close so the data is durable on disk if the
		// process is killed or the host loses power between Encode and
		// the OS flushing its own buffers. Without this, a crash here
		// could leave a truncated export file at the final path.
		if serr := file.Sync(); serr != nil {
			utils.NoteWarning("failed to sync export file: %v", serr)
		}
		if cerr := file.Close(); cerr != nil {
			utils.NoteWarning("failed to close export file: %v", cerr)
		}
	}()

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

// validateMountForImport checks that a mount has all required fields.
func validateMountForImport(m models.MountConfig) error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("imported mount has empty name")
	}
	if strings.TrimSpace(m.Remote) == "" {
		return fmt.Errorf("imported mount %q has empty remote", m.Name)
	}
	if strings.TrimSpace(m.MountPoint) == "" {
		return fmt.Errorf("imported mount %q has empty mount point", m.Name)
	}
	return nil
}

// validateSyncJobForImport checks that a sync job has all required fields.
func validateSyncJobForImport(j models.SyncJobConfig) error {
	if strings.TrimSpace(j.Name) == "" {
		return fmt.Errorf("imported sync job has empty name")
	}
	if strings.TrimSpace(j.Source) == "" {
		return fmt.Errorf("imported sync job %q has empty source", j.Name)
	}
	if strings.TrimSpace(j.Destination) == "" {
		return fmt.Errorf("imported sync job %q has empty destination", j.Name)
	}
	return nil
}

// ImportConfig imports mounts and sync jobs from a file.
// The import mode determines how conflicts are handled.
func (c *Config) ImportConfig(filePath string, mode ImportMode) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("import file does not exist: %s", filePath)
	}

	file, err := os.Open(filePath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to open import file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			utils.NoteWarning("failed to close import file: %v", cerr)
		}
	}()

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
		for _, m := range data.Mounts {
			if err := validateMountForImport(m); err != nil {
				return err
			}
		}
		for _, j := range data.SyncJobs {
			if err := validateSyncJobForImport(j); err != nil {
				return err
			}
		}
		c.Mounts = data.Mounts
		c.SyncJobs = data.SyncJobs
	case ImportModeMerge:
		if err := c.mergeImport(data); err != nil {
			return err
		}
	}

	return nil
}

// mergeImport merges the imported data with the existing configuration.
// Items with duplicate names are skipped. Returns an error if any item fails validation.
func (c *Config) mergeImport(data ExportData) error {
	// Note: mergeImport is called from ImportConfig, which already holds the lock.
	existingMountNames := make(map[string]bool)
	for _, m := range c.Mounts {
		existingMountNames[m.Name] = true
	}

	for _, mount := range data.Mounts {
		if existingMountNames[mount.Name] {
			continue
		}
		if err := validateMountForImport(mount); err != nil {
			return err
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
		if err := validateSyncJobForImport(job); err != nil {
			return err
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

	return nil
}
