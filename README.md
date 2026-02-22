# rclone-mount-sync

A Terminal User Interface (TUI) application for managing rclone mounts and sync jobs with systemd user unit file generation.

## Overview

rclone-mount-sync provides an intuitive TUI for configuring and managing rclone mounts and sync operations. It automatically generates systemd user unit files, making it easy to have your cloud storage mounted and synced automatically.

## Features

### Mount Management
Configure and manage rclone mount points with extensive customization:
- **VFS Options**: Cache modes (off, minimal, writes, full), buffer sizes, directory cache time, network timeouts
- **FUSE Options**: allow-other, allow-root, umask, uid/gid settings
- **Auto-start**: Automatically mount on login

### Sync Job Management
Set up scheduled sync operations between local and remote storage:
- **Operations**: sync, copy, and move operations
- **Conflict Resolution**: Various strategies for handling conflicts
- **Filtering**: Include/exclude patterns, age-based filtering
- **Performance Tuning**: Parallel transfers, checkers, bandwidth limits
- **Dry-run Mode**: Preview changes before execution

### Systemd Integration
Automatic generation of systemd user service and timer units with proper dependencies and resource limits.

### Pre-flight Checks
Comprehensive validation before operations:
- Rclone binary verification
- Version compatibility check
- Remote validation
- Systemd user session check
- Fusermount availability check

### Service Status
View and control the status of your mounts and sync jobs through an intuitive interface.

## Requirements

- Go 1.23.0 or later
- [rclone](https://rclone.org/) v1.60.0 or later, installed and configured
- Linux with systemd (user session support)
- FUSE (for mount operations)
- fusermount or fusermount3

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/dtg01100/rclone-mount-sync.git
cd rclone-mount-sync

# Build and install
make install
```

### Build Only

```bash
# Download dependencies and build
make deps build

# The binary will be available at bin/rclone-mount-sync
```

## Usage

### Running the Application

```bash
# If installed system-wide
rclone-mount-sync

# Or run directly from the build directory
./bin/rclone-mount-sync

# Print version
rclone-mount-sync --version

# Specify custom config directory (overrides XDG_CONFIG_HOME)
rclone-mount-sync --config /path/to/configdir

# Skip pre-flight validation checks
rclone-mount-sync --skip-checks
```

### Keyboard Navigation

The TUI uses standard keyboard navigation:

| Key | Action |
|-----|--------|
| `↑/k` | Move up |
| `↓/j` | Move down |
| `Enter` | Select |
| `q` | Quit / Go back |
| `?` | Help |
| `Ctrl+C` | Force quit |
| `Esc` | Go back / Cancel |

### Main Menu Quick Keys

| Key | Action |
|-----|--------|
| `M` | Mount Management |
| `S` | Sync Job Management |
| `V` | Service Status |
| `T` | Settings |

### Mount Management Keys

| Key | Action |
|-----|--------|
| `a` | Add new mount |
| `e` | Edit selected mount |
| `d` | Delete selected mount |
| `s` | Start/Stop mount service |
| `x` | Refresh mount list |
| `r` | Refresh service status |

### Sync Job Keys

| Key | Action |
|-----|--------|
| `a/n` | Add new sync job |
| `e` | Edit selected sync job |
| `d` | Delete selected sync job |
| `r` | Refresh job list |
| `t` | Toggle timer |

### Main Menu Options

1. **Mount Management** - Configure rclone mount points
2. **Sync Job Management** - Set up scheduled sync operations
3. **Service Status** - View and control systemd services
4. **Settings** - Configure application defaults

## Configuration

The application stores its configuration in `~/.config/rclone-mount-sync/config.yaml`.

### Example Configuration

```yaml
version: "1.0"

defaults:
  mount:
    log_level: "INFO"
    vfs_cache_mode: "full"
    buffer_size: "16M"
  sync:
    log_level: "INFO"
    transfers: 4
    checkers: 8

settings:
  rclone_binary_path: ""
  default_mount_dir: "~/mnt"
  editor: ""
  recent_paths: []

mounts:
  - id: "google-drive"
    name: "Google Drive"
    description: "My Google Drive"
    remote: "gdrive:"
    remote_path: "/"
    mount_point: "~/mnt/gdrive"
    mount_options:
      vfs_cache_mode: "full"
      buffer_size: "16M"
      allow_other: false
      read_only: false
    auto_start: true
    enabled: true

sync_jobs:
  - id: "photos-backup"
    name: "Photos Backup"
    description: "Backup photos to local"
    source: "gdrive:/Photos"
    destination: "~/Backup/Photos"
    sync_options:
      direction: "sync"
      delete_extraneous: false
      transfers: 4
      dry_run: false
    schedule:
      type: "timer"
      on_calendar: "daily"
      persistent: true
    auto_start: true
    enabled: true
```

## Generated Systemd Units

### Mount Service (`rclone-mount-{name}.service`)

Mount services are generated with the following characteristics:
- **Type**: `notify` - Properly tracks when mount is ready
- **Mount Point**: Auto-created before start
- **Restart**: Auto-restart on failure with appropriate delays
- **Dependencies**: Proper ordering after network and systemd user session

### Sync Service (`rclone-sync-{name}.service`)

Sync services are generated with:
- **Type**: `oneshot` - Runs once per invocation
- **Resource Limits**: Memory and CPU limits to prevent runaway processes
- **Network Dependency**: Waits for network to be available
- **Working Directory**: Set appropriately for the sync operation

### Sync Timer (`rclone-sync-{name}.timer`)

Timer units support:
- **Calendar-based**: Run on specific schedules (daily, weekly, etc.)
- **Boot-based**: Run after system boot with optional delay
- **Persistent**: Catch up on missed runs if system was off
- **Randomized Delay**: Spread load across multiple jobs

## Development

### Prerequisites

- Go 1.23.0+
- Make (optional, for using the Makefile)

### Development Commands

```bash
# Download dependencies
make deps

# Build the binary
make build

# Run the application
make run

# Run tests
make test

# Format code
make fmt

# Clean build artifacts
make clean
```

### Project Structure

```
rclone-mount-sync/
├── cmd/rclone-mount-sync/main.go      # Application entry point
├── internal/
│   ├── config/config.go               # Configuration management
│   ├── models/models.go               # Core data structures
│   ├── rclone/
│   │   ├── client.go                  # Rclone binary wrapper
│   │   ├── remotes.go                 # Remote listing/management
│   │   ├── validation.go              # Pre-flight checks
│   │   └── config.go                  # Rclone config handling
│   ├── systemd/
│   │   ├── generator.go               # Systemd unit file generation
│   │   ├── manager.go                 # Systemd service operations
│   │   ├── templates.go               # Unit file templates
│   │   └── paths.go                   # Path utilities
│   ├── tui/
│   │   ├── app.go                     # Main TUI application
│   │   ├── screens/
│   │   │   ├── main_menu.go           # Main navigation
│   │   │   ├── mounts.go              # Mount management screen
│   │   │   ├── mount_form.go          # Mount creation/edit form
│   │   │   ├── sync_jobs.go           # Sync job management screen
│   │   │   ├── sync_job_form.go       # Sync job creation/edit form
│   │   │   ├── services.go            # Service status screen
│   │   │   └── settings.go            # Settings screen
│   │   └── components/common.go       # Shared UI components
│   └── errors/errors.go               # Error handling
├── pkg/utils/utils.go                 # General utilities
├── go.mod
├── Makefile
└── README.md
```

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

- [rclone](https://rclone.org/) - The underlying tool for cloud storage operations
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Charm](https://charm.sh/) - Excellent TUI libraries and tools
