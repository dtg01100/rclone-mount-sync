# Confirmation Dialog Component

A reusable confirmation dialog component for destructive actions in the TUI.

## Overview

The `ConfirmDialog` component provides a standardized way to confirm destructive actions with customizable options and styling.

## Features

- **Reusable**: Single component replaces duplicate delete confirmation code
- **Customizable**: Configurable title, message, description, and options
- **Destructive Action Styling**: Options can be marked as destructive with red styling
- **Keyboard Navigation**: Arrow keys or h/l for navigation, Enter to confirm, Esc to cancel
- **Helper Functions**: Pre-configured dialogs for common use cases

## Usage

### Basic Usage

```go
import "github.com/dtg01100/rclone-mount-sync/internal/tui/components"

// Create a custom confirmation dialog
dialog := components.NewConfirmDialog(components.ConfirmDialogConfig{
    Title:   "Delete Mount",
    Message: "Are you sure you want to delete 'my-mount'?",
    Description: "This action cannot be undone.",
    Options: []components.ConfirmDialogOption{
        {Label: "Cancel", Action: 0, IsDestructive: false},
        {Label: "Delete", Action: 1, IsDestructive: true},
    },
})

// Set size
dialog.SetSize(width, height)

// In your Update method
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        model, cmd := dialog.Update(msg)
        d := model.(*components.ConfirmDialog)
        
        if d.IsDone() {
            action := d.GetSelectedAction()
            switch action {
            case 0:
                // Cancel
            case 1:
                // Delete
            }
        }
    }
}

// In your View method
func (m *Model) View() string {
    return dialog.View()
}
```

### Helper Functions

#### Delete Confirmation (3 options)

```go
// Standard delete dialog with options:
// [0] Cancel
// [1] Delete Service Only
// [2] Delete Service and Config
dialog := components.NewDeleteConfirmDialog("mount-name", "Mount")
```

#### Simple Yes/No Confirmation

```go
// Simple yes/no dialog:
// [0] No
// [1] Yes
dialog := components.NewSimpleConfirmDialog("Confirm", "Are you sure?")
```

#### Custom Action Confirmation

```go
// Custom options
options := []components.ConfirmDialogOption{
    {Label: "Abort", Action: 0, IsDestructive: false},
    {Label: "Retry", Action: 1, IsDestructive: false},
    {Label: "Ignore", Action: 2, IsDestructive: true},
}
dialog := components.NewActionConfirmDialog("Error", "Operation failed", options)
```

## Integration Example

### Replacing DeleteConfirm in mounts.go

```go
// Old code:
type MountsScreen struct {
    delete  *DeleteConfirm
    // ...
}

// New code:
type MountsScreen struct {
    delete  *components.ConfirmDialog
    // ...
}

// When creating the dialog:
func (s *MountsScreen) confirmDelete() tea.Cmd {
    s.delete = components.NewDeleteConfirmDialog(
        s.mounts[s.cursor].Name,
        "Mount",
    )
    s.delete.SetSize(s.width, s.height)
    return nil
}

// In Update method:
if s.delete != nil {
    model, cmd := s.delete.Update(msg)
    s.delete = model.(*components.ConfirmDialog)
    
    if s.delete.IsDone() {
        action := s.delete.GetSelectedAction()
        s.delete = nil
        
        switch action {
        case 0:
            // Cancel - do nothing
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

// In View method:
if s.delete != nil {
    return s.delete.View()
}
```

## Configuration Options

### ConfirmDialogConfig

- `Title` (string): Dialog title
- `Message` (string): Main warning message
- `Description` (string, optional): Additional description
- `Options` ([]ConfirmDialogOption): Button options
- `Width` (int): Dialog width

### ConfirmDialogOption

- `Label` (string): Button text
- `Action` (int): Action identifier returned when selected
- `IsDestructive` (bool): If true, uses red error styling

## API Reference

### Constructor

```go
func NewConfirmDialog(config ConfirmDialogConfig) *ConfirmDialog
```

### Methods

```go
func (d *ConfirmDialog) SetSize(width, height int)
func (d *ConfirmDialog) Init() tea.Cmd
func (d *ConfirmDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (d *ConfirmDialog) View() string
func (d *ConfirmDialog) IsDone() bool
func (d *ConfirmDialog) SelectedOption() int
func (d *ConfirmDialog) GetSelectedAction() int
```

### Helper Functions

```go
func NewDeleteConfirmDialog(itemName, itemType string) *ConfirmDialog
func NewSimpleConfirmDialog(title, message string) *ConfirmDialog
func NewActionConfirmDialog(title, message string, options []ConfirmDialogOption) *ConfirmDialog
```

## Testing

The component includes comprehensive tests covering:

- Dialog creation and initialization
- Keyboard navigation (left/right, enter, escape)
- Option selection
- View rendering
- Destructive styling
- Helper functions

Run tests with:
```bash
go test ./internal/tui/components/... -v -run TestConfirmDialog
```

## Migration Guide

To migrate existing delete confirmation dialogs to use the new component:

1. Replace the custom struct with `*components.ConfirmDialog`
2. Use `NewDeleteConfirmDialog()` helper or create custom config
3. Update the Update method to use the component's Update method
4. Check `IsDone()` and `GetSelectedAction()` instead of custom logic
5. Use `View()` method for rendering

Benefits:
- Reduced code duplication
- Consistent UI/UX across the application
- Easier to maintain and update
- Comprehensive test coverage
