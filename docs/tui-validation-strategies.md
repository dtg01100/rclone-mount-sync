# TUI Validation Strategies

This document describes the validation strategies implemented for the rclone-mount-sync TUI.

## Overview

The TUI validation suite covers four main areas:

1. **State Machine Tests** - Screen navigation and state transitions
2. **Message Handling Tests** - All Bubble Tea message types
3. **Snapshot Tests** - Visual rendering regression detection
4. **Dependency Failure Tests** - External service error handling

## Test Files

### `state_machine_test.go`

Validates screen navigation as a finite state machine.

**Coverage:**
- All valid screen transitions via keyboard shortcuts
- Help screen toggle and close behavior
- Programmatic screen changes via `ScreenChangeMsg`
- All Bubble Tea message type handling
- Invariant validation (dimensions, screen enums, help state consistency)
- Rapid key press resilience
- Help scroll boundary conditions

**Key Tests:**
```bash
go test -run "TestApp_StateMachine"      # Navigation state machine
go test -run "TestApp_MessageHandling"   # All message types
go test -run "TestApp_Invariants"        # State invariants
go test -run "TestApp_RapidKeyPresses"   # Stress testing
go test -run "TestApp_ScrollBoundaries"  # Edge cases
```

**Example - State Transition:**
```go
transitions := []struct {
    name         string
    startScreen  Screen
    key          string
    expectScreen Screen
    expectQuit   bool
}{
    {"main->mounts via m", ScreenMain, "m", ScreenMounts, false},
    {"mounts->main via q", ScreenMounts, "q", ScreenMain, false},
    {"main->quit via ctrl+c", ScreenMain, "ctrl+c", ScreenMain, true},
}
```

### `snapshot_test.go`

Captures rendered TUI output for regression detection.

**Coverage:**
- Main menu rendering
- Help screen layout
- Initialization error display
- Loading state
- Orphan detection prompt
- Different terminal sizes (small, standard, large, wide, tall)
- Line count validation
- Trailing whitespace documentation
- Empty line spacing analysis

**Usage:**
```bash
# Run snapshot tests
go test -run "TestApp_Snapshot"

# Update snapshots after intentional changes
UPDATE_SNAPSHOTS=1 go test -run "TestApp_Snapshot"
```

**Snapshot Management:**
- Snapshots stored in `testdata/snapshots/*.snap`
- Mismatches create `.actual` files for comparison
- Use `UPDATE_SNAPSHOTS=1` to regenerate after intentional changes

### `dependency_failure_test.go`

Tests how the TUI handles external service failures.

**Coverage:**
- Config directory permission errors
- Systemd generator initialization failures
- Rclone binary unavailability
- Empty/minimal config handling
- Config with existing mounts
- Orphan detection and reconciliation
- Orphan action handling (remove, ignore, error)
- Concurrent message processing
- Rclone client initialization

**Key Tests:**
```bash
go test -run "TestApp_InitError"          # Initialization failures
go test -run "TestApp_Reconciliation"     # Orphan detection
go test -run "TestApp_OrphanAction"       # Orphan management
go test -run "TestApp_ConcurrentMessages" # Thread safety
```

**Example - Config Failure:**
```go
func TestApp_InitError_ConfigLoadFailure(t *testing.T) {
    // Create restricted directory
    restrictedDir := filepath.Join(tmpDir, "restricted")
    os.Mkdir(restrictedDir, 0000)
    
    // Point XDG_CONFIG_HOME to restricted dir
    os.Setenv("XDG_CONFIG_HOME", restrictedDir)
    
    app := NewApp()
    msg := app.initializeServices()
    
    // Should handle gracefully
    t.Logf("Message type: %T", msg)
}
```

## Validation Approaches

### 1. State Machine Testing

Models the TUI as a finite state machine with defined transitions.

**Benefits:**
- Documents expected navigation behavior
- Catches broken keybindings
- Validates screen enum consistency
- Ensures quit/escape behavior is correct

**Implementation:**
- Define all valid transitions in table-driven tests
- Test each transition independently
- Verify post-conditions (screen, help state, quit command)

### 2. Message Type Coverage

Ensures all Bubble Tea message types are handled.

**Message Types Tested:**
- `tea.KeyMsg` - Keyboard input
- `tea.WindowSizeMsg` - Terminal resize
- `ScreenChangeMsg` - Programmatic navigation
- `AppInitError` - Initialization failures
- `AppInitDone` - Successful initialization
- `ReconciliationMsg` - Orphan detection results
- `OrphanActionMsg` - User orphan management actions
- `LoadingMsg` / `LoadingDoneMsg` - Loading state (documented as not handled)

### 3. Snapshot Testing

Captures rendered output for visual regression detection.

**When to Update Snapshots:**
- Intentional UI/UX changes
- Layout improvements
- Style/color updates
- New screen additions

**When to Investigate Failures:**
- Unexpected layout changes
- Missing content
- Rendering artifacts
- Width/height constraint violations

### 4. Dependency Failure Modes

Tests graceful degradation when external services fail.

**Failure Scenarios:**
- Config file inaccessible (permissions)
- Systemd generator unavailable (XDG path issues)
- Rclone not in PATH
- Empty configuration
- Service manager failures
- Orphan removal errors

## Running Tests

```bash
# All TUI tests
make test

# Specific test categories
go test ./internal/tui -run "StateMachine"
go test ./internal/tui -run "Snapshot"
go test ./internal/tui -run "Dependency"
go test ./internal/tui -run "Orphan"

# Update snapshots
UPDATE_SNAPSHOTS=1 go test ./internal/tui -run "Snapshot"

# Verbose output
go test ./internal/tui -v -run "TestApp_Invariants"
```

## Test Coverage Metrics

To check coverage:
```bash
go test ./internal/tui -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Future Enhancements

Potential additions to the validation suite:

1. **Property-Based Testing** - Use `pgregory.net/rapid` to generate random key sequences
2. **Integration Testing** - Use `tea.Program` with simulated input streams
3. **Performance Benchmarks** - Validate rendering speed with large configs
4. **Accessibility Testing** - Screen reader compatibility checks
5. **Visual Regression** - Use VHS for terminal recording and comparison

## Design Decisions

### Why Not Mock Everything?

The tests prefer **real filesystem interactions** over mocking where possible:
- More realistic test scenarios
- Catches actual integration issues
- Less brittle to internal refactoring
- Still uses mocks for systemd manager (via `MockManager`)

### Why Document Rather Than Fail?

Some tests document current behavior rather than enforcing strict rules:
- Trailing whitespace is used for TUI layout
- Empty lines provide visual spacing
- Loading state doesn't show overlay (by design)

These observations help future developers understand intentional design choices.

### Snapshot vs Unit Tests

**Use Unit Tests When:**
- Testing logic/state transitions
- Validating error handling
- Checking invariants

**Use Snapshot Tests When:**
- Validating rendered output
- Catching visual regressions
- Documenting UI layout

## Related Files

- `internal/tui/app.go` - Main TUI application
- `internal/tui/screens/*.go` - Individual screen implementations
- `internal/tui/components/*.go` - Reusable UI components
- `internal/systemd/manager.go` - ServiceManager interface and MockManager
- `internal/systemd/reconcile.go` - Orphan detection logic
