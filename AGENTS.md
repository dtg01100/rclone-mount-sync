# AGENTS.md

**Version:** 2.0
**Date:** 2026-06-02
**Purpose:** Technical reference for rclone-mount-sync development (methodology in `.clio/instructions.md`)

---

## Project Overview

**rclone-mount-sync** is a Terminal User Interface (TUI) application for managing
rclone mounts and sync jobs on Linux, with automatic systemd user unit file
generation.

- **Language:** Go 1.23.0+
- **Module:** `github.com/dtg01100/rclone-mount-sync`
- **Architecture:** Bubble Tea (charmbracelet) MVC TUI + Cobra CLI dispatch
- **Runtime:** Linux with systemd (user session), FUSE, rclone v1.60.0+

---

## Quick Setup

```bash
# Install dependencies (downloads Go modules)
make deps

# Build binary to bin/rclone-mount-sync
make build

# Build and run the TUI
make run

# Run all tests
make test

# Run tests with race + coverage
make test-coverage

# Format and lint
make fmt
make lint   # requires golangci-lint

# Install to /usr/local/bin
make install
make install PREFIX=/opt/rclone-mount-sync   # custom prefix
```

**Key commands:**

```bash
./bin/rclone-mount-sync                  # Run the TUI
./bin/rclone-mount-sync --version        # Print version
./bin/rclone-mount-sync --skip-checks    # Skip pre-flight validation
./bin/rclone-mount-sync --config /path   # Override XDG config dir
./bin/rclone-mount-sync mount list       # CLI mode (cobra subcommands)
```

---

## Architecture

```
                       +---------------------------+
                       |  cmd/rclone-mount-sync/   |
                       |  main.go (entry + DI)     |
                       +-------------+-------------+
                                     |
              +----------------------+----------------------+
              |                                             |
              v                                             v
   +-------------------+                       +-----------------------+
   |     internal/     |                       |     internal/        |
   |       cli/        |  <-- cobra subcmds    |        tui/          |
   |  mount, sync,     |                       |  Bubble Tea MVC     |
   |  services, ...    |                       |  app, screens,       |
   +---------+---------+                       |  components          |
             |                                 +-------+---------------+
             |                                         |
             v                                         v
   +-------------------+    +------------------+   +----------------+
   |  internal/        |    |  internal/       |   |  internal/     |
   |  systemd/         |    |  rclone/         |   |  config/       |
   |  generator,       |    |  client,         |   |  XDG YAML      |
   |  manager,         |    |  remotes,        |   |  load/save     |
   |  templates,       |    |  validation,     |   +----------------+
   |  reconcile        |    |  retry           |
   +-------------------+    +------------------+
             |
             v
   +-------------------+    +------------------+
   |  internal/models/ |    |  internal/errors/|
   |  MountConfig,     |    |  AppError        |
   |  SyncJobConfig    |    |  code/msg/sug    |
   +-------------------+    +------------------+
```

### Key Design Patterns

1. **Dependency Injection** — `main.go` defines `AppDeps` with interface fields
   (`PreflightChecker`, `TUIRunner`, `NewClient`, `ParseFlags`). Tests inject
   fakes by mutating the struct.
2. **Screen Navigation** — Enum-based screen system in `tui/app.go` with
   `ScreenChangeMsg` driving the model. See `internal/tui/state_machine_test.go`.
3. **Testability via function variables** — Top-level `var` functions
   (`loadConfig`, etc.) are reassigned in tests. See `internal/cli/*_test.go`.
4. **Error handling** — `internal/errors.AppError` with `Code/Message/Suggestion`;
   wrap with `fmt.Errorf("...: %w", err)` at boundaries.
5. **Interfaces where used** — `PreflightChecker`, `TUIRunner`, and other
   interfaces are defined in `main.go` next to the consumer, not in the
   implementing package.

---

## Directory Structure

| Path | Purpose |
|------|---------|
| `cmd/rclone-mount-sync/` | Entry point, flag parsing, DI wiring, TUI-vs-CLI dispatch |
| `internal/cli/` | Cobra command handlers: `mount`, `sync`, `services`, `config`, `remote`, `reconcile`, `doctor`, `cleanup` |
| `internal/config/` | XDG-compliant YAML config load/save, backup, import/export |
| `internal/errors/` | `AppError` type with code/message/suggestion |
| `internal/models/` | `MountConfig`, `SyncJobConfig`, `Defaults`, `Settings` data structures |
| `internal/rclone/` | Rclone binary wrapper (`client.go`), remote listing (`remotes.go`), pre-flight checks (`validation.go`), retry logic (`retry.go`) |
| `internal/systemd/` | Unit file generation (`generator.go`, `templates.go`), service control (`manager.go`, `paths.go`), orphan reconciliation (`reconcile.go`) |
| `internal/tui/` | Bubble Tea MVC: `app.go`, `components/`, `screens/` |
| `internal/tui/components/` | Reusable UI primitives (path helpers, common widgets) |
| `internal/tui/screens/` | One file per screen: `main_menu`, `mounts`, `mount_form`, `sync_jobs`, `sync_job_form`, `services`, `settings`, `rollback` |
| `internal/testutil/` | Shared test helpers (mocks, fixtures) |
| `pkg/utils/` | Public utility functions (importable by external projects) |
| `plans/` | Architecture and design documents |
| `docs/` | (currently empty — see `README.md` and `CONTRIBUTING.md`) |
| `.github/workflows/ci.yaml` | CI: `go vet` + `go test -v -count=1 ./...` on Go 1.23 |

**Key entry points:**

- `cmd/rclone-mount-sync/main.go` — binary entry, routes args to TUI or CLI
- `internal/cli/root.go` — cobra root command
- `internal/tui/app.go` — Bubble Tea root model
- `internal/rclone/validation.go` — pre-flight checks called at startup

---

## Code Style

**Go Conventions (gofmt-enforced):**

- **Tabs** for indentation (Go standard)
- `gofmt` via `make fmt`
- `golangci-lint` via `make lint` — enabled linters: `errcheck`, `unused`,
  `govet`, `ineffassign`, `staticcheck`, `intrange`, `misspell`, `nilnil`,
  `errorlint`, `gosec`, `gocritic`
- Group imports: **stdlib -> external -> local** (separated by blank lines)
- Exported identifiers need doc comments; internal helpers do not

**Module/Type Conventions:**

- One package per subdirectory under `internal/`
- One file per logical responsibility (`client.go`, `remotes.go`, `retry.go`)
- Test files co-located as `*_test.go` in the same package
- Interfaces defined in the consumer, not the producer (see
  `cmd/.../main.go` for the `PreflightChecker` pattern)

**Error Handling:**

```go
// Always wrap with context
return fmt.Errorf("mount failed: %w", err)

// Check errors at every level — never ignore
if err != nil {
    return fmt.Errorf("validation failed: %w", err)
}
```

**Test Mocks via Function Variables:**

```go
// Production code
var loadConfig = func() (*config.Config, error) { return config.Load() }

// Test code
oldLoadConfig := loadConfig
defer func() { loadConfig = oldLoadConfig }()
loadConfig = func() (*config.Config, error) {
    return nil, fmt.Errorf("failed to load config")
}
```

---

## Testing

**Test count:** 32 production `.go` files, 34 `*_test.go` files (roughly 1:1).
Tests use stdlib `testing` only — no testify, no gomock.

**Before Committing:**

```bash
# Format
make fmt

# Lint (requires golangci-lint)
make lint

# All tests
make test

# Verbose + race detector + coverage HTML
make test-coverage

# Single package
go test ./internal/systemd/...

# Single test
go test -run TestGenerator ./internal/systemd/...

# Coverage summary
go tool cover -func=coverage.out | tail -20
```

**Test Locations:**

- `*_test.go` files co-located with the code they test (same package)
- `internal/testutil/` for shared fixtures

**Test Patterns:**

- Table-driven tests for multiple cases
- Mock external dependencies via function variables (see Code Style above)
- Use `t.TempDir()` for filesystem-touching tests (auto-cleaned)
- Test error paths, not just happy path
- `t.Parallel()` for independent tests

**CI Workflow** (`.github/workflows/ci.yaml`):

```yaml
- go vet ./...
- go test -v -count=1 ./...
```

Runs on every push to `main` and every PR. Linux only (uses systemd/FUSE
implicitly in some tests via `t.TempDir()` paths — tests that need a real
rclone binary or systemd user session should be guarded with build tags or
skipped via `t.Skip`).

---

## Commit Format

This project uses [Conventional Commits](https://www.conventionalcommits.org/).

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:** `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `perf`, `style`

**Scopes:** `tui`, `systemd`, `rclone`, `config`, `models`, `cli`, `ci`, `deps`

**Examples:**

```
feat(tui): add file picker and remote path suggestions

fix(systemd): correct timer unit OnCalendar format

test(config): add tests for backup and restore functions

refactor(rclone): split monolithic rclone.go into modular files
```

**Pre-Commit Checklist:**

- [ ] `make fmt` ran
- [ ] `make test` passes
- [ ] `make lint` passes (when applicable)
- [ ] Conventional commit message
- [ ] No `ai-assisted/` handoff files staged
- [ ] No untracked junk in `bin/`, `coverage.out`, `coverage.html`

---

## Common Patterns

**Mocking a function variable for tests:**

```go
var loadConfig = func() (*config.Config, error) {
    return config.Load()
}

func TestSomething(t *testing.T) {
    old := loadConfig
    defer func() { loadConfig = old }()
    loadConfig = func() (*config.Config, error) {
        return &config.Config{}, nil
    }
    // ... test code
}
```

**Bubble Tea screen integration test pattern** (see
`internal/tui/state_machine_test.go`):

```go
m := newTestModel(t)
m = update(m, navigationMsg{to: screenMounts})
view := m.View()
if !strings.Contains(view, "Mounts") {
    t.Errorf("expected Mounts screen, got %q", view)
}
```

**Systemd unit file generation** (see `internal/systemd/generator.go`):

```go
data := MountUnitData{
    Name:       "gdrive",
    Remote:     "gdrive:",
    MountPoint: "/home/user/mnt/gdrive",
    // ...
}
content, err := GenerateMountUnit(data)
```

**Wrapping `cmd.Output()` calls** (from `BUGFIXES.md`):

```go
// [FAIL] WRONG — error silently ignored
output, _ := cmd.Output()

// [OK] CORRECT
output, err := cmd.Output()
if err != nil {
    return fmt.Errorf("command failed: %w", err)
}
```

**Array bounds before indexing** (from `BUGFIXES.md`):

```go
// [FAIL] WRONG — can panic
if !results[0].Passed { ... }

// [OK] CORRECT
if len(results) == 0 || !results[0].Passed { ... }
```

---

## Configuration

**XDG Compliance:**

- Config dir: `$XDG_CONFIG_HOME/rclone-mount-sync/` (default: `~/.config/`)
- Config file: `config.yaml`
- Override with `--config /path/to/dir` flag (sets `XDG_CONFIG_HOME`)

**Environment Variables:**

| Variable | Purpose |
|----------|---------|
| `RCLONE_BINARY_PATH` | Custom rclone binary location (overrides PATH lookup) |
| `XDG_CONFIG_HOME` | Override config directory base |

**Example config:** see `README.md` -> Configuration section for a full
`config.yaml` example with `mounts:` and `sync_jobs:`.

---

## External Dependencies

**Runtime Requirements:**

- Linux with systemd (user session) — `systemctl --user` must work
- FUSE + `fusermount` / `fusermount3`
- rclone v1.60.0+
- D-Bus user session (for some NetworkManager ExecCondition checks)

**Key Go Dependencies** (from `go.mod`):

| Module | Purpose |
|--------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework (MVC) |
| `github.com/charmbracelet/bubbles` | TUI components (textinput, list, etc.) |
| `github.com/charmbracelet/huh` | Form components (mount/sync job forms) |
| `github.com/charmbracelet/lipgloss` | Styling |
| `github.com/spf13/viper` | Configuration management (YAML + env + defaults) |
| `github.com/spf13/cobra` | CLI subcommand parsing |
| `github.com/google/uuid` | ID generation for mounts/jobs |
| `gopkg.in/yaml.v3` | YAML parsing for `config.yaml` |

---

## Common Development Tasks

### Adding a New Screen

1. Create `internal/tui/screens/new_screen.go` with Bubble Tea `Model` interface (`Init`, `Update`, `View`)
2. Add a screen enum value in `internal/tui/app.go`
3. Register the screen in the navigation switch
4. Add screen-level tests in `internal/tui/screens/new_screen_test.go`

### Adding a New CLI Command

1. Add handler in `internal/cli/newcmd.go` (use cobra patterns from existing commands)
2. Register in `internal/cli/root.go` command tree
3. Add the command name to `CLICommands()` in `cmd/rclone-mount-sync/main.go`
4. Add tests in `internal/cli/newcmd_test.go` with mocked dependencies

### Modifying Systemd Units

1. Update templates in `internal/systemd/templates.go`
2. Update `MountUnitData` / `SyncUnitData` structs in `internal/systemd/generator.go` if adding fields
3. Test unit file generation (`internal/systemd/generator_test.go`)
4. Verify manually: `systemctl --user cat rclone-mount-<name>.service`

### Testing Rclone Integration

1. Unit tests mock `*rclone.Client` via function variables or interfaces
2. Integration tests (gated) call the real `rclone` binary
3. Test retry logic explicitly (`internal/rclone/retry_test.go`)
4. Test pre-flight check results (`internal/rclone/validation_test.go`)

---

## Anti-Patterns (What NOT To Do)

Sourced from `BUGFIXES.md` and project conventions.

| Anti-Pattern | Why It's Wrong | What To Do |
|--------------|----------------|------------|
| `if !results[0].Passed` without length check | Panics on empty slice | `if len(results) == 0 \|\| !results[0].Passed` |
| `output, _ := cmd.Output()` | Hides system command failures | Check `err` and wrap with context |
| `defer srcFile.Close()` ignoring error | Hides I/O failures on close | `defer func() { if cerr := srcFile.Close(); cerr != nil { ... } }()` |
| Bare `panic` in goroutines | Crashes the whole TUI | `defer recover()` and send error to result channel |
| Calling `manager.Stop()` etc. with `_, _ =` | Silent cleanup failures | `if err := manager.Stop(); err != nil { ... }` |
| Hardcoded `cliCommands` list duplicated in tests | Tautological tests fail to catch real drift | Use the exported `CLICommands()` function in tests |
| Untyped error returns | Loses context for users/TUI | Use `internal/errors.AppError{Code, Message, Suggestion}` or wrap with `fmt.Errorf` |
| New screen without updating navigation switch | Screen unreachable from TUI | Add enum value AND register in switch |
| New CLI command without `CLICommands()` update | Routes to TUI mode by default | Update the map in `cmd/rclone-mount-sync/main.go` |
| Stale fields in test fixtures | Tests pass against wrong shape | Audit fixture structs when changing `*UnitData` |
| Running TUI and CLI against same config simultaneously | Config RWMutex is per-process, not cross-process | External coordination only; document if needed |
| Assuming code behavior in unfamiliar files | Silent bugs from upstream changes | Read the file first (`file_operations(read_file)`) |
| Committing with `ai-assisted/` staged | Leaks internal session context | `git reset HEAD ai-assisted/` before `git commit` |

---

## Quick Reference

**Build & Test:**

```bash
make deps build run          # Build and run
make test                    # All tests
make test-coverage           # Tests + race + coverage HTML
make fmt && make lint        # Format and lint
```

**TUI Navigation Cheatsheet** (from `README.md`):

| Key | Action |
|-----|--------|
| `^/k` | Move up |
| `v/j` | Move down |
| `Enter` | Select |
| `q` | Quit / Go back |
| `?` | Help |
| `Ctrl+C` | Force quit |
| `Esc` | Go back / Cancel |

**Main Menu Quick Keys:** `M` Mount, `S` Sync, `V` Services, `T` Settings.

**Find Code:**

```bash
# Symbol search
code_intelligence(operation: "list_usages", symbol_name: "MountConfig")

# Regex in code
file_operations(operation: "grep_search", query: "PreflightChecker", is_regex: true)

# Recent history
git log --oneline --since="1 week ago"
```

**Git:**

```bash
git status
git log --oneline -20
git diff
git add <files> && git commit -m "type(scope): description"
```

---

## Documentation Links

- `README.md` — User-facing features, installation, configuration
- `CONTRIBUTING.md` — Development setup, coding standards, commit format
- `BUGFIXES.md` — History of bug fixes and patterns to avoid
- `plans/architecture-design.md` — Detailed architecture decisions
- `plans/folder-navigation-improvements.md` — TUI navigation refactor notes
- `plans/tui-validation-strategies.md` — Form validation strategy

---

*For project methodology and workflow, see `.clio/instructions.md`.*
*For universal agent behavior, see system prompt.*
