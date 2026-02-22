# Contributing to rclone-mount-sync

Thank you for your interest in contributing to rclone-mount-sync! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Features](#suggesting-features)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Coding Standards](#coding-standards)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Pull Request Process](#pull-request-process)
- [Testing Requirements](#testing-requirements)

## Code of Conduct

Be respectful and inclusive. Treat all contributors with courtesy. We welcome contributions from everyone regardless of experience level, background, or identity.

## Reporting Bugs

Before submitting a bug report, please:

1. Check if the issue has already been reported
2. Test with the latest version from `main`
3. Gather relevant information:
   - Go version (`go version`)
   - Rclone version (`rclone version`)
   - Operating system and version
   - Systemd version (if applicable)

When submitting a bug report, include:

- A clear, descriptive title
- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- Relevant logs or error messages
- Your environment details

## Suggesting Features

Feature suggestions are welcome! Please:

1. Check if the feature has already been requested
2. Provide a clear description of the feature
3. Explain the use case and benefits
4. Consider if it fits within the project's scope

## Development Setup

### Prerequisites

- Go 1.23.0 or later
- Make (optional, for Makefile commands)
- golangci-lint (for linting)
- rclone (for integration testing)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/dtg01100/rclone-mount-sync.git
cd rclone-mount-sync

# Install dependencies
make deps

# Build the binary
make build

# Run tests
make test

# Run the application
make run
```

### Available Make Targets

| Target | Description |
|--------|-------------|
| `make deps` | Download and tidy dependencies |
| `make build` | Build the binary to `bin/rclone-mount-sync` |
| `make run` | Build and run the application |
| `make test` | Run all tests |
| `make fmt` | Format code with `go fmt` |
| `make lint` | Run golangci-lint |
| `make clean` | Remove build artifacts |
| `make install` | Install to `$BINDIR` (default: `/usr/local/bin`) |

## Project Structure

```
rclone-mount-sync/
├── cmd/rclone-mount-sync/          # Application entry point
│   ├── main.go                     # Main function and CLI handling
│   └── main_test.go                # Entry point tests
├── internal/                       # Private application code
│   ├── config/                     # Configuration management
│   │   ├── config.go               # Config loading and saving
│   │   └── config_test.go
│   ├── errors/                     # Error handling utilities
│   │   ├── errors.go
│   │   └── errors_test.go
│   ├── models/                     # Core data structures
│   │   ├── models.go               # MountConfig, SyncJobConfig, etc.
│   │   └── models_test.go
│   ├── rclone/                     # Rclone integration
│   │   ├── client.go               # Rclone binary wrapper
│   │   ├── config.go               # Rclone config handling
│   │   ├── remotes.go              # Remote listing/management
│   │   ├── retry.go                # Retry logic
│   │   ├── validation.go           # Pre-flight checks
│   │   └── *_test.go
│   ├── systemd/                    # Systemd unit generation
│   │   ├── generator.go            # Unit file generation
│   │   ├── manager.go              # Service operations
│   │   ├── paths.go                # Path utilities
│   │   ├── reconcile.go            # Orphan unit handling
│   │   ├── templates.go            # Unit file templates
│   │   └── *_test.go
│   └── tui/                        # Terminal UI
│       ├── app.go                  # Main TUI application
│       ├── tui_test.go
│       ├── components/             # Shared UI components
│       │   ├── common.go
│       │   ├── path_helpers.go
│       │   └── components_test.go
│       └── screens/                # Individual screens
│           ├── main_menu.go
│           ├── mounts.go
│           ├── mount_form.go
│           ├── sync_jobs.go
│           ├── sync_job_form.go
│           ├── services.go
│           ├── settings.go
│           ├── rollback.go
│           └── *_test.go
├── pkg/utils/                      # Public utilities
│   ├── utils.go
│   └── utils_test.go
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── .github/
    └── workflows/
        └── ci.yaml                 # GitHub Actions CI
```

### Package Guidelines

- `cmd/` - Contains main applications (entry points only)
- `internal/` - Private code not importable by other projects
- `pkg/` - Public code that could be imported by external projects
- Each package should have a clear, single responsibility
- Test files (`*_test.go`) should be in the same package as the code they test

## Coding Standards

### Go Conventions

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` for formatting (run `make fmt`)
- Run `golangci-lint` before submitting (run `make lint`)
- Use meaningful variable and function names

### Code Style

- Use tabs for indentation (Go standard)
- Exported functions and types should have documentation comments
- Internal functions need not be documented unless complex
- Group related imports: standard library, external packages, local packages

```go
import (
    "fmt"
    "os"

    "github.com/charmbracelet/bubbletea"

    "github.com/dtg01100/rclone-mount-sync/internal/models"
)
```

### Error Handling

- Return errors rather than panicking
- Wrap errors with context using `fmt.Errorf("operation failed: %w", err)`
- Check errors at every level

### Interfaces

- Define interfaces where they are used, not where they are implemented
- Keep interfaces small and focused
- Use interfaces for testability (see `main.go` for examples)

## Commit Message Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

| Type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `test` | Adding or updating tests |
| `refactor` | Code change without fix or feature |
| `perf` | Performance improvement |
| `chore` | Build, CI, or tooling changes |
| `style` | Code style changes (formatting, etc.) |

### Scopes

Use the package or component name:

- `tui` - Terminal UI changes
- `systemd` - Systemd-related changes
- `rclone` - Rclone integration
- `config` - Configuration handling
- `models` - Data structures
- `ci` - CI/CD changes

### Examples

```
feat(tui): add file picker and remote path suggestions

fix(systemd): correct timer unit OnCalendar format

docs: update README with comprehensive feature docs

test(config): add tests for backup and restore functions

refactor(rclone): split monolithic rclone.go into modular files
```

## Pull Request Process

1. **Fork and Branch**: Create a feature branch from `main`

```bash
git checkout -b feat/your-feature-name
```

2. **Make Changes**: Write clean, tested code

3. **Run Checks**: Ensure all checks pass

```bash
make fmt
make lint
make test
```

4. **Commit**: Use conventional commit messages

5. **Push and Create PR**: Push to your fork and open a pull request

### PR Requirements

- All tests must pass
- Code must be formatted (`make fmt`)
- No linting errors (`make lint`)
- New code should have tests
- PR description should explain the change and motivation

### PR Title

Use the same format as commit messages:

```
feat(tui): add keyboard shortcut help overlay
```

### Review Process

1. Maintainers will review your PR
2. Address any feedback or requested changes
3. Once approved, a maintainer will merge your PR

## Testing Requirements

### Test Coverage

All new code should include tests. We aim for meaningful coverage of:

- Business logic in `internal/` packages
- Edge cases and error paths
- Configuration parsing and validation

### Running Tests

```bash
# Run all tests
make test

# Or directly
go test ./...

# Run with verbose output
go test -v ./...

# Run tests for a specific package
go test ./internal/config/...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Style

- Use table-driven tests for multiple test cases
- Use descriptive test names
- Test both success and error paths

```go
func TestAddMount(t *testing.T) {
    tests := []struct {
        name    string
        mount   models.MountConfig
        wantErr bool
    }{
        {
            name: "valid mount",
            mount: models.MountConfig{
                Name:       "test",
                Remote:     "gdrive:",
                MountPoint: "/mnt/test",
            },
            wantErr: false,
        },
        {
            name: "empty name",
            mount: models.MountConfig{
                Remote:     "gdrive:",
                MountPoint: "/mnt/test",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cfg := newConfigWithDefaults()
            err := cfg.AddMount(tt.mount)
            if (err != nil) != tt.wantErr {
                t.Errorf("AddMount() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Test Helpers

- Use `t.Parallel()` for independent tests
- Create helper functions for common setup
- Use `t.TempDir()` for temporary directories (auto-cleaned)

---

Thank you for contributing to rclone-mount-sync!
