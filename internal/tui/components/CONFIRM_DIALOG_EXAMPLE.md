# Confirmation Dialog - Implementation Example

This document shows how to use the new `ConfirmDialog` component to replace duplicate delete confirmation code.

## Before: Duplicate Code in mounts.go and sync_jobs.go

Both files had nearly identical `DeleteConfirm` and `SyncJobDeleteConfirm` structs with duplicated logic.

## After: Single Reusable Component

### Example 1: Replacing Mount Delete Confirmation

```go
// OLD CODE in mounts.go:
type MountsScreen struct {
    // ...
    delete  *DeleteConfirm  // Custom struct
    // ...
}

func (s *MountsScreen) confirmDelete() tea.Cmd {
    s.delete = NewDeleteConfirm(s.mounts[s.cursor])
    // ...
}

// NEW CODE using ConfirmDialog component:
type MountsScreen struct {
    // ...
    delete  *components.ConfirmDialog  // Reusable component
    // ...
}

func (s *MountsScreen) confirmDelete() tea.Cmd {
    mount := s.mounts[s.cursor]
    s.delete = components.NewDeleteConfirmDialog(mount.Name, "Mount")
    s.delete.SetSize(s.width, s.height)
    return nil
}

// In Update method:
func (s *MountsScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle delete confirmation
    if s.delete != nil {
        model, cmd := s.delete.Update(msg)
        s.delete = model.(*components.ConfirmDialog)
        
        if s.delete.IsDone() {
            action := s.delete.GetSelectedAction()
            s.delete = nil
            
            switch action {
            case 0:
                // Cancel - do nothing
                return s, nil
            case 1:
                // Delete service only
                return s, s.deleteServiceOnly()
            case 2:
                // Delete service and config
                return s, s.deleteServiceAndConfig()
            }
        }
        
        return s, cmd
    }
    
    // ... rest of update logic
}

// In View method:
func (s *MountsScreen) View() string {
    if s.delete != nil {
        return s.delete.View()
    }
    
    // ... rest of view logic
}
```

### Example 2: Simple Yes/No Confirmation

```go
// For simple confirmations like "Are you sure you want to reset all settings?"
func (m *Model) confirmReset() tea.Cmd {
    m.confirmDialog = components.NewSimpleConfirmDialog(
        "Reset Settings",
        "This will reset all settings to defaults. Continue?",
    )
    m.confirmDialog.SetSize(m.width, m.height)
    return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if m.confirmDialog != nil {
        model, cmd := m.confirmDialog.Update(msg)
        m.confirmDialog = model.(*components.ConfirmDialog)
        
        if m.confirmDialog.IsDone() {
            action := m.confirmDialog.GetSelectedAction()
            m.confirmDialog = nil
            
            if action == 1 { // Yes
                return m, m.performReset()
            }
            // No - do nothing
        }
        
        return m, cmd
    }
    
    // ... rest of update logic
}
```

### Example 3: Custom Action Confirmation

```go
// For custom scenarios like error handling
func (m *Model) showErrorDialog(errorMsg string) tea.Cmd {
    options := []components.ConfirmDialogOption{
        {Label: "Abort", Action: 0, IsDestructive: false},
        {Label: "Retry", Action: 1, IsDestructive: false},
        {Label: "Ignore", Action: 2, IsDestructive: true},
    }
    
    m.confirmDialog = components.NewActionConfirmDialog(
        "Operation Failed",
        fmt.Sprintf("Error: %s", errorMsg),
        options,
    )
    m.confirmDialog.SetSize(m.width, m.height)
    return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if m.confirmDialog != nil {
        model, cmd := m.confirmDialog.Update(msg)
        m.confirmDialog = model.(*components.ConfirmDialog)
        
        if m.confirmDialog.IsDone() {
            action := m.confirmDialog.GetSelectedAction()
            m.confirmDialog = nil
            
            switch action {
            case 0:
                return m, m.abortOperation()
            case 1:
                return m, m.retryOperation()
            case 2:
                return m, m.ignoreError()
            }
        }
        
        return m, cmd
    }
    
    // ... rest of update logic
}
```

### Example 4: Fully Customized Dialog

```go
// For complex scenarios with descriptions and multiple destructive options
func (m *Model) confirmBulkDelete(items []string) tea.Cmd {
    config := components.ConfirmDialogConfig{
        Title:       "Bulk Delete",
        Message:     fmt.Sprintf("Delete %d items?", len(items)),
        Description: "This will remove both services and configurations. This action cannot be undone.",
        Options: []components.ConfirmDialogOption{
            {Label: "Cancel", Action: 0, IsDestructive: false},
            {Label: "Delete Services Only", Action: 1, IsDestructive: true},
            {Label: "Delete Everything", Action: 2, IsDestructive: true},
        },
    }
    
    m.confirmDialog = components.NewConfirmDialog(config)
    m.confirmDialog.SetSize(m.width, m.height)
    return nil
}
```

## Benefits

1. **Code Reuse**: Single component instead of duplicate structs
2. **Consistency**: Same UI/UX across all confirmation dialogs
3. **Maintainability**: Fix bugs or add features in one place
4. **Testability**: Comprehensive test coverage for the component
5. **Flexibility**: Easy to customize for different scenarios
6. **Type Safety**: Clear action codes and option configuration

## Migration Checklist

For each existing delete confirmation dialog:

- [ ] Replace custom struct with `*components.ConfirmDialog`
- [ ] Use appropriate helper function or create custom config
- [ ] Update constructor to call `SetSize()`
- [ ] Update `Update()` method to use component's Update
- [ ] Replace custom done check with `IsDone()`
- [ ] Replace cursor check with `GetSelectedAction()`
- [ ] Update `View()` method to call component's View
- [ ] Remove old custom struct definition
- [ ] Remove old custom constructor
- [ ] Add tests for the new integration

## Testing

After migration, ensure all tests pass:

```bash
# Test the component itself
go test ./internal/tui/components/... -run TestConfirmDialog -v

# Test the screens using the component
go test ./internal/tui/screens/... -v

# Run full test suite
make test
```

## Future Enhancements

Potential improvements to the component:

1. Add timeout auto-dismiss option
2. Support for multi-line descriptions
3. Custom key bindings configuration
4. Animated transitions
5. Support for icons/emoji in options
6. Configurable button styles per option
