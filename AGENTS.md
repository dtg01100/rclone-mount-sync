# rclone-mount-sync - AI Agent Instructions

## Quick Start

**Build & Run:**
```bash
make deps build run    # Build and run
make test              # Run all tests
make lint              # Run linter
make fmt               # Format code
```

**Key Commands:**
- `./bin/rclone-mount-sync` - Run the TUI
- `./bin/rclone-mount-sync --version` - Show version
- `./bin/rclone-mount-sync --skip-checks` - Skip pre-flight validation
- `./bin/rclone-mount-sync --config /path/to/dir` - Custom config directory

---

## Project Overview

A Go TUI application for managing rclone mounts and sync jobs with automatic systemd user unit file generation.

**Tech Stack:**
- Go 1.23+
- Bubble Tea (charmbracelet/bubbletea) - TUI framework
- Viper - Configuration management
- Cobra - CLI parsing
- Lipgloss - Styling

---

## Architecture

### Component Boundaries

```
cmd/rclone-mount-sync/     # Entry point with dependency injection
internal/
  ├── cli/                 # CLI command handlers (mount, sync, services)
  ├── config/              # YAML config loading/saving (XDG-compliant)
  ├── errors/              # AppError with code/message/suggestion
  ├── models/              # MountConfig, SyncJobConfig data structures
  ├── rclone/              # Rclone binary wrapper, validation, retry logic
  ├── systemd/             # Unit file generation & service management
  └── tui/                 # Bubble Tea MVC implementation
    ├── components/        # Reusable UI components
    └── screens/           # Individual screens (main menu, forms, etc.)
pkg/utils/                 # Public utilities
```

### Key Design Patterns

1. **Dependency Injection** - See `main.go` for `AppDeps` pattern with interfaces
2. **Screen Navigation** - Enum-based screen system with `ScreenChangeMsg`
3. **Testability** - Mock via function variables (e.g., `var loadConfig = func()...`)
4. **Error Handling** - Return errors with context, never panic
5. **Interfaces** - Define where used, not where implemented

---

## Essential Conventions

### Code Style
- Tabs for indentation (Go standard)
- `gofmt` enforced via `make fmt`
- `golangci-lint` via `make lint`
- Group imports: stdlib → external → local

### Error Handling
```go
// Wrap errors with context
return fmt.Errorf("mount failed: %w", err)

// Check errors at every level
if err != nil {
    return fmt.Errorf("validation failed: %w", err)
}
```

### Testing
- Co-locate `*_test.go` with tested code
- Mock external dependencies via function variables
- Test error conditions, not just happy path
- Make meaningful assertions (see `/memories/repo/testing-best-practices.md`)

```go
// Mocking pattern
oldLoadConfig := loadConfig
defer func() { loadConfig = oldLoadConfig }()
loadConfig = func() (*config.Config, error) {
    return nil, fmt.Errorf("failed to load config")
}
```

### Array Bounds & Error Checking
- **Always check slice length before indexing** (see BUGFIXES.md)
- **Never ignore errors** from system commands (systemctl, rclone, etc.)

---

## Configuration

### XDG Compliance
- Config dir: `$XDG_CONFIG_HOME/rclone-mount-sync/` (default: `~/.config/`)
- Config file: `config.yaml`
- Override with `--config /path/to/dir` flag

### Environment Variables
- `RCLONE_BINARY_PATH` - Custom rclone binary location
- `XDG_CONFIG_HOME` - Override config directory

---

## External Dependencies

### Runtime Requirements
- Linux with systemd (user session)
- FUSE + fusermount/fusermount3
- rclone v1.60.0+
- D-Bus user session

### Go Dependencies
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/huh` - Form components
- `github.com/spf13/viper` - Config management
- `github.com/spf13/cobra` - CLI parsing
- `gopkg.in/yaml.v3` - YAML parsing

---

## Common Development Tasks

### Adding a New Screen
1. Create `internal/tui/screens/new_screen.go`
2. Implement Bubble Tea Model interface (Init, Update, View)
3. Add screen enum value in `tui/app.go`
4. Register in navigation switch

### Adding a New CLI Command
1. Add handler in `internal/cli/newcmd.go`
2. Register in command router
3. Add tests with mocked dependencies

### Modifying Systemd Units
1. Update templates in `internal/systemd/templates.go`
2. Ensure generator handles new fields
3. Test unit file generation
4. Verify with `systemctl --user cat <service>`

### Testing Rclone Integration
1. Mock `rclone.Client` for unit tests
2. Use integration tests for actual rclone calls
3. Test retry logic and error handling

---

## Known Pitfalls

### Array Bounds
```go
// ❌ WRONG - can panic
if !results[0].Passed { ... }

// ✅ CORRECT
if len(results) == 0 || !results[0].Passed { ... }
```

### Ignored Errors
```go
// ❌ WRONG
output, _ := cmd.Output()

// ✅ CORRECT
output, err := cmd.Output()
if err != nil {
    return fmt.Errorf("command failed: %w", err)
}
```

### System Dependencies
- Requires systemd user session (`systemctl --user` must work)
- FUSE must be available for mount operations
- Rclone binary must be in PATH or `RCLONE_BINARY_PATH` set

### Thread Safety
- Config loading/saving uses file locking
- Be careful with shared state in TUI updates

---

## Documentation Links

- **README.md** - User-facing features and usage
- **CONTRIBUTING.md** - Development setup and coding standards
- **BUGFIXES.md** - History of bug fixes and patterns to avoid
- **plans/architecture-design.md** - Detailed architecture decisions
- **/memories/repo/testing-best-practices.md** - Testing guidelines

---

## Example Prompts

Once this file is in place, agents can be prompted with:

- "Add a new screen for viewing sync job logs"
- "Implement bandwidth scheduling for sync jobs"
- "Add unit tests for the new mount validation logic"
- "Refactor the systemd generator to support template units"
- "Fix the array bounds issue in remotes.go"

---

## Related Customizations

Consider creating:
1. **`.instructions.md`** for TUI-specific patterns (applyTo: `internal/tui/**/*.go`)
2. **`.instructions.md`** for systemd integration (applyTo: `internal/systemd/**/*.go`)
3. **Custom agent mode** for test generation with project-specific mocking patterns
4. **Skill** for generating Bubble Tea components with proper MVC separation
