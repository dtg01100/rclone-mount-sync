# Test Generation Skill for rclone-mount-sync

## Purpose

This skill generates high-quality test code following project-specific patterns for mocking, error handling, and assertion quality.

---

## When to Use

Use this skill when:
- Creating new test files for packages
- Adding tests for new functionality
- Improving existing tests with better assertions
- Generating table-driven tests for multiple scenarios

---

## Core Principles

### 1. Make Meaningful Assertions

Always verify actual behavior, not just that code doesn't panic:

```go
// ❌ BAD - No real assertion
func TestSomething(t *testing.T) {
    result, err := doSomething()
    // Just checking it doesn't crash
}

// ✅ GOOD - Verify behavior
func TestSomething(t *testing.T) {
    result, err := doSomething()
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if result != expectedValue {
        t.Errorf("expected %v, got %v", expectedValue, result)
    }
}
```

### 2. Test Error Conditions

Verify errors are returned when preconditions fail:

```go
func TestMountList_InvalidConfig(t *testing.T) {
    // Arrange: Mock config loading to fail
    oldLoadConfig := loadConfig
    defer func() { loadConfig = oldLoadConfig }()
    loadConfig = func() (*config.Config, error) {
        return nil, fmt.Errorf("config file not found")
    }
    
    // Act
    err := runMountList(nil, nil)
    
    // Assert
    if err == nil {
        t.Error("expected error when config loading fails")
    }
    if !strings.Contains(err.Error(), "config file not found") {
        t.Errorf("expected config error, got %v", err)
    }
}
```

### 3. Mock External Dependencies

Use function variables for testable code:

```go
// In production code (e.g., cli/mount.go)
var loadConfig = func() (*config.Config, error) {
    return config.Load()
}

// In test code
func TestMountCreate(t *testing.T) {
    // Mock config loading
    oldLoadConfig := loadConfig
    defer func() { loadConfig = oldLoadConfig }()
    
    mockConfig := &config.Config{
        Mounts: []models.MountConfig{},
    }
    loadConfig = func() (*config.Config, error) {
        return mockConfig, nil
    }
    
    // Test continues with controlled state...
}
```

### 4. Use Table-Driven Tests

For testing multiple scenarios:

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        wantErr     bool
        errContains string
    }{
        {"empty input", "", true, "required"},
        {"valid path", "/mnt/test", false, ""},
        {"relative path", "relative/path", true, "absolute"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validatePath(tt.input)
            
            if tt.wantErr {
                if err == nil {
                    t.Error("expected error, got nil")
                } else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
                    t.Errorf("expected error containing %q, got %v", tt.errContains, err)
                }
            } else {
                if err != nil {
                    t.Errorf("unexpected error: %v", err)
                }
            }
        })
    }
}
```

---

## Test Patterns by Layer

### CLI Layer Tests

```go
func TestMountCreate_Success(t *testing.T) {
    // Arrange
    oldLoadConfig := loadConfig
    defer func() { loadConfig = oldLoadConfig }()
    
    cfg := &config.Config{Mounts: []models.MountConfig{}}
    loadConfig = func() (*config.Config, error) { return cfg, nil }
    
    oldSaveConfig := saveConfig
    defer func() { saveConfig = oldSaveConfig }()
    
    var savedConfig *config.Config
    saveConfig = func(c *config.Config) error {
        savedConfig = c
        return nil
    }
    
    // Act
    err := runMountCreate(testMountConfig)
    
    // Assert
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if len(savedConfig.Mounts) != 1 {
        t.Errorf("expected 1 mount, got %d", len(savedConfig.Mounts))
    }
}
```

### TUI Layer Tests

```go
func TestMainMenu_SelectMounts(t *testing.T) {
    // Arrange
    model := screens.NewMainMenu()
    
    // Simulate down arrow then enter
    model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
    model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
    
    // Assert navigation occurred
    if cmd == nil {
        t.Fatal("expected navigation command")
    }
    
    msg := cmd()
    if changeMsg, ok := msg.(tui.ScreenChangeMsg); !ok {
        t.Error("expected ScreenChangeMsg")
    } else if changeMsg.Screen != tui.ScreenMounts {
        t.Errorf("expected ScreenMounts, got %v", changeMsg.Screen)
    }
}
```

### Systemd Layer Tests

```go
func TestGenerateMountService_Template(t *testing.T) {
    // Arrange
    generator := &systemd.Generator{
        RclonePath: "/usr/bin/rclone",
    }
    
    mount := &models.MountConfig{
        ID:         "test123",
        Name:       "Test Mount",
        Remote:     "gdrive:",
        RemotePath: "/backup",
        MountPoint: "~/mount",
    }
    
    // Act
    content, err := generator.GenerateMountService(mount)
    
    // Assert
    if err != nil {
        t.Fatalf("generation failed: %v", err)
    }
    
    // Verify template substitution
    assert.Contains(t, content, "Description=Rclone mount: Test Mount")
    assert.Contains(t, content, "/usr/bin/rclone mount")
    assert.Contains(t, content, "gdrive:/backup")
    assert.Contains(t, content, "~/mount")
}
```

### Config Layer Tests

```go
func TestLoadConfig_FileNotFound(t *testing.T) {
    // Arrange
    oldConfigPath := getConfigPath
    defer func() { getConfigPath = oldConfigPath }()
    
    getConfigPath = func() string {
        return "/nonexistent/path/config.yaml"
    }
    
    // Act
    cfg, err := config.Load()
    
    // Assert
    if err == nil {
        t.Error("expected error for missing config file")
    }
    if cfg != nil {
        t.Error("expected nil config on error")
    }
}
```

---

## Test File Organization

### File Naming

- Co-locate tests: `foo.go` → `foo_test.go`
- Integration tests: `*_integration_test.go`
- Example: `internal/cli/mount_test.go`

### Test Function Naming

```go
func Test<Function>_<Scenario>(t *testing.T)
// Examples:
func TestMountCreate_Success(t *testing.T)
func TestMountCreate_InvalidRemote(t *testing.T)
func TestMountDelete_NotFound(t *testing.T)
```

### Helper Functions

```go
// Common test setup
func createTestConfig(t *testing.T) *config.Config {
    t.Helper()
    return &config.Config{
        Mounts: []models.MountConfig{},
        SyncJobs: []models.SyncJobConfig{},
    }
}

// Mock command execution
func runCmd(t *testing.T, args []string) (string, error) {
    t.Helper()
    cmd := exec.Command(args[0], args[1:]...)
    output, err := cmd.CombinedOutput()
    return string(output), err
}
```

---

## Anti-Patterns to Avoid

### ❌ Tests Without Assertions

```go
// BAD - What are we testing?
func TestSomething(t *testing.T) {
    result, _ := doSomething()
    // No assertions at all!
}
```

### ❌ Ignoring Errors in Tests

```go
// BAD - Error ignored
result, _ := doSomething()
if result != expected {
    t.Error("wrong result")
}

// GOOD - Check errors first
result, err := doSomething()
if err != nil {
    t.Fatalf("unexpected error: %v", err)
}
if result != expected {
    t.Errorf("wrong result")
}
```

### ❌ Over-Mocking

```go
// BAD - Mocking everything
func TestEverything(t *testing.T) {
    // 20 lines of mocks...
    // Test is harder to understand than the code itself
}

// GOOD - Mock only external dependencies
func TestCoreLogic(t *testing.T) {
    // 2-3 essential mocks
    // Test remains readable
}
```

### ❌ Testing Implementation Details

```go
// BAD - Tests internal state
func TestInternalState(t *testing.T) {
    model.internalCounter = 5  // Accessing private fields
    if model.privateVar != "" {
        t.Error("wrong")
    }
}

// GOOD - Test observable behavior
func TestBehavior(t *testing.T) {
    output := model.Process(input)
    if output != expected {
        t.Error("wrong output")
    }
}
```

---

## Running Tests

### Standard Test Commands

```bash
# Run all tests
make test

# Run specific test
go test -v ./internal/cli -run TestMountCreate

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/systemd/...
```

### Test Flags

```bash
# Verbose output
go test -v

# Show coverage
go test -cover

# Run benchmarks
go test -bench=.

# Parallel execution
go test -parallel 4

# Race detector
go test -race
```

---

## Quality Checklist

Before considering tests complete, verify:

- [ ] Tests make meaningful assertions (not just "doesn't crash")
- [ ] Error conditions are tested
- [ ] External dependencies are mocked appropriately
- [ ] Test names clearly describe the scenario
- [ ] Helper functions use `t.Helper()`
- [ ] Table-driven tests used for multiple scenarios
- [ ] Tests are independent (no ordering dependencies)
- [ ] Cleanup is handled (defer for mocks)
- [ ] Edge cases are covered (empty input, max values, etc.)

---

## Example Usage

### Prompt Examples

Once this skill is available, use prompts like:

- "Generate tests for the new mount validation function"
- "Add table-driven tests for all sync job conflict scenarios"
- "Create integration tests for the systemd generator"
- "Improve test coverage for the TUI screen navigation"
- "Add error condition tests for config loading failures"

### Template Request

```
Please generate tests for [function/screen/package] that:
1. Test the happy path
2. Test error conditions
3. Mock external dependencies
4. Use table-driven tests where appropriate
5. Follow the patterns in /memories/repo/testing-best-practices.md
```

---

## Related Resources

- `/memories/repo/testing-best-practices.md` - Project testing guidelines
- `BUGFIXES.md` - Common issues to test for (array bounds, error handling)
- `internal/cli/*_test.go` - CLI layer test examples
- `internal/tui/*_test.go` - TUI test examples
- `internal/systemd/*_test.go` - Systemd test examples
