# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial implementation of rclone-mount-sync TUI application
- Mount management with VFS and FUSE options configuration
- Sync job management with sync, copy, and move operations
- Automatic systemd user unit file generation for mounts and sync jobs
- Timer-based scheduling for sync jobs (calendar and boot-based)
- Pre-flight validation checks (rclone binary, version, remotes, systemd, fusermount)
- Service status view for monitoring and controlling systemd services
- Application settings screen with editable configuration
- File picker and remote path suggestions in forms
- Recent paths tracking for quicker configuration
- Run conditions for sync jobs (AC power and non-metered connection requirements)
- ID-based systemd unit naming for better identification
- Orphan unit detection and reconciliation
- Graceful error handling with initialization checks
- CLI flags: `--version`, `--config`, `--skip-checks`
- GitHub Actions CI workflow for automated testing
- Configurable install paths via Makefile

### Changed

- Refactored rclone package into modular files (client, remotes, validation, config)
- Extracted systemd templates and paths to separate files for maintainability
- Improved HelpBar truncation to preserve ANSI codes

### Fixed

- CI Go version updated to 1.23 to match go.mod requirement
- GitHub repository URL corrected to match go.mod module path

## [0.1.0] - TBD

Initial release planned. See [Unreleased] for current features.

[Unreleased]: https://github.com/dtg01100/rclone-mount-sync/compare/HEAD
[0.1.0]: https://github.com/dtg01100/rclone-mount-sync/releases/tag/v0.1.0
