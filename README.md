# rclone-mount-sync

A Terminal User Interface (TUI) application for managing rclone mounts and sync jobs with systemd user unit file generation.

## Overview

rclone-mount-sync provides an intuitive TUI for configuring and managing rclone mounts and sync operations. It automatically generates systemd user unit files, making it easy to have your cloud storage mounted and synced automatically.

## Features

- **Mount Management**: Configure and manage rclone mount points
- **Sync Job Management**: Set up scheduled sync operations between local and remote storage
- **Systemd Integration**: Automatic generation of systemd user service and timer units
- **Service Status**: View and control the status of your mounts and sync jobs
- **User-Friendly TUI**: Navigate and configure everything through an intuitive terminal interface

## Requirements

- Go 1.21 or later
- [rclone](https://rclone.org/) installed and configured
- Linux with systemd (user session support)
- FUSE (for mount operations)

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/dlafreniere/rclone-mount-sync.git
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
```

### Keyboard Navigation

The TUI uses standard keyboard navigation:

- `↑/k` - Move up
- `↓/j` - Move down
- `Enter` - Select
- `q` - Quit / Go back
- `?` - Help

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

mounts:
  - id: "google-drive"
    name: "Google Drive"
    remote: "gdrive:"
    remote_path: "/"
    mount_point: "~/mnt/gdrive"
    auto_start: true
    enabled: true

sync_jobs:
  - id: "photos-backup"
    name: "Photos Backup"
    source: "gdrive:/Photos"
    destination: "~/Backup/Photos"
    schedule:
      type: "timer"
      on_calendar: "daily"
    auto_start: true
    enabled: true
```

## Development

### Prerequisites

- Go 1.21+
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
├── cmd/
│   └── rclone-mount-sync/
│       └── main.go          # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── models/
│   │   └── models.go        # Data models
│   ├── rclone/
│   │   └── rclone.go        # Rclone client wrapper
│   ├── systemd/
│   │   ├── manager.go       # Systemd service management
│   │   └── generator.go     # Unit file generation
│   └── tui/
│       ├── app.go           # Main TUI application
│       ├── screens/
│       │   ├── main_menu.go
│       │   ├── mounts.go
│       │   ├── sync_jobs.go
│       │   ├── services.go
│       │   └── settings.go
│       └── components/
│           └── common.go    # Shared UI components
├── pkg/
│   └── utils/
│       └── utils.go         # Utility functions
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
