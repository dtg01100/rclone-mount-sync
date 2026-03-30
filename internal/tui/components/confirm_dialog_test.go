package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConfirmDialog_NewConfirmDialog(t *testing.T) {
	config := ConfirmDialogConfig{
		Title:   "Test Dialog",
		Message: "Are you sure?",
		Options: []ConfirmDialogOption{
			{Label: "Cancel", Action: 0},
			{Label: "Confirm", Action: 1},
		},
	}

	dialog := NewConfirmDialog(config)

	if dialog == nil {
		t.Fatal("Expected dialog to be created")
	}
	if dialog.cursor != 0 {
		t.Errorf("Expected cursor to be 0, got %d", dialog.cursor)
	}
	if dialog.done {
		t.Error("Expected dialog to not be done initially")
	}
	if dialog.config.Title != "Test Dialog" {
		t.Errorf("Expected title 'Test Dialog', got '%s'", dialog.config.Title)
	}
}

func TestConfirmDialog_SetSize(t *testing.T) {
	dialog := NewConfirmDialog(ConfirmDialogConfig{
		Title:   "Test",
		Message: "Test",
		Options: []ConfirmDialogOption{{Label: "OK", Action: 0}},
	})

	dialog.SetSize(80, 24)

	if dialog.config.Width != 80 {
		t.Errorf("Expected width to be 80, got %d", dialog.config.Width)
	}
}

func TestConfirmDialog_Init(t *testing.T) {
	dialog := NewConfirmDialog(ConfirmDialogConfig{
		Title:   "Test",
		Message: "Test",
		Options: []ConfirmDialogOption{{Label: "OK", Action: 0}},
	})

	cmd := dialog.Init()

	if cmd != nil {
		t.Error("Expected Init to return nil command")
	}
}

func TestConfirmDialog_Update_LeftRight(t *testing.T) {
	dialog := NewConfirmDialog(ConfirmDialogConfig{
		Title:   "Test",
		Message: "Test",
		Options: []ConfirmDialogOption{
			{Label: "Option 1", Action: 0},
			{Label: "Option 2", Action: 1},
			{Label: "Option 3", Action: 2},
		},
	})

	// Move right
	model, _ := dialog.Update(tea.KeyMsg{Type: tea.KeyRight})
	d := model.(*ConfirmDialog)
	if d.cursor != 1 {
		t.Errorf("Expected cursor to be 1 after moving right, got %d", d.cursor)
	}

	// Move right again
	model, _ = d.Update(tea.KeyMsg{Type: tea.KeyRight})
	d = model.(*ConfirmDialog)
	if d.cursor != 2 {
		t.Errorf("Expected cursor to be 2 after moving right again, got %d", d.cursor)
	}

	// Try to move right past the end
	model, _ = d.Update(tea.KeyMsg{Type: tea.KeyRight})
	d = model.(*ConfirmDialog)
	if d.cursor != 2 {
		t.Errorf("Expected cursor to stay at 2, got %d", d.cursor)
	}

	// Move left
	model, _ = d.Update(tea.KeyMsg{Type: tea.KeyLeft})
	d = model.(*ConfirmDialog)
	if d.cursor != 1 {
		t.Errorf("Expected cursor to be 1 after moving left, got %d", d.cursor)
	}
}

func TestConfirmDialog_Update_Enter(t *testing.T) {
	dialog := NewConfirmDialog(ConfirmDialogConfig{
		Title:   "Test",
		Message: "Test",
		Options: []ConfirmDialogOption{
			{Label: "Cancel", Action: 0},
			{Label: "Confirm", Action: 1},
		},
	})

	// Move to second option
	dialog.cursor = 1

	// Press enter
	model, _ := dialog.Update(tea.KeyMsg{Type: tea.KeyEnter})
	d := model.(*ConfirmDialog)

	if !d.done {
		t.Error("Expected dialog to be done after pressing enter")
	}
	if d.selected != 1 {
		t.Errorf("Expected selected to be 1, got %d", d.selected)
	}
}

func TestConfirmDialog_Update_Escape(t *testing.T) {
	dialog := NewConfirmDialog(ConfirmDialogConfig{
		Title:   "Test",
		Message: "Test",
		Options: []ConfirmDialogOption{
			{Label: "Cancel", Action: 0},
			{Label: "Confirm", Action: 1},
		},
	})

	dialog.cursor = 1

	// Press escape
	model, _ := dialog.Update(tea.KeyMsg{Type: tea.KeyEsc})
	d := model.(*ConfirmDialog)

	if !d.done {
		t.Error("Expected dialog to be done after pressing escape")
	}
	if d.selected != 0 {
		t.Errorf("Expected selected to stay at 0 after escape, got %d", d.selected)
	}
}

func TestConfirmDialog_IsDone(t *testing.T) {
	dialog := NewConfirmDialog(ConfirmDialogConfig{
		Title:   "Test",
		Message: "Test",
		Options: []ConfirmDialogOption{{Label: "OK", Action: 0}},
	})

	if dialog.IsDone() {
		t.Error("Expected dialog to not be done initially")
	}

	dialog.done = true

	if !dialog.IsDone() {
		t.Error("Expected dialog to be done after setting done to true")
	}
}

func TestConfirmDialog_SelectedOption(t *testing.T) {
	dialog := NewConfirmDialog(ConfirmDialogConfig{
		Title:   "Test",
		Message: "Test",
		Options: []ConfirmDialogOption{
			{Label: "Option 1", Action: 0},
			{Label: "Option 2", Action: 1},
		},
	})

	if dialog.SelectedOption() != 0 {
		t.Errorf("Expected selected option to be 0, got %d", dialog.SelectedOption())
	}

	dialog.selected = 1

	if dialog.SelectedOption() != 1 {
		t.Errorf("Expected selected option to be 1, got %d", dialog.SelectedOption())
	}
}

func TestConfirmDialog_GetSelectedAction(t *testing.T) {
	dialog := NewConfirmDialog(ConfirmDialogConfig{
		Title:   "Test",
		Message: "Test",
		Options: []ConfirmDialogOption{
			{Label: "Cancel", Action: 0},
			{Label: "Confirm", Action: 1},
			{Label: "Delete", Action: 2},
		},
	})

	if dialog.GetSelectedAction() != 0 {
		t.Errorf("Expected selected action to be 0, got %d", dialog.GetSelectedAction())
	}

	dialog.selected = 2

	if dialog.GetSelectedAction() != 2 {
		t.Errorf("Expected selected action to be 2, got %d", dialog.GetSelectedAction())
	}

	// Test out of bounds
	dialog.selected = 5
	if dialog.GetSelectedAction() != -1 {
		t.Errorf("Expected selected action to be -1 for out of bounds, got %d", dialog.GetSelectedAction())
	}
}

func TestConfirmDialog_View(t *testing.T) {
	dialog := NewConfirmDialog(ConfirmDialogConfig{
		Title:   "Delete Item",
		Message: "Are you sure?",
		Options: []ConfirmDialogOption{
			{Label: "Cancel", Action: 0},
			{Label: "Delete", Action: 1, IsDestructive: true},
		},
		Width: 80,
	})

	view := dialog.View()

	if view == "" {
		t.Error("Expected view to not be empty")
	}

	// Check that title is rendered
	if !containsString(view, "Delete Item") {
		t.Error("Expected view to contain title 'Delete Item'")
	}

	// Check that message is rendered
	if !containsString(view, "Are you sure?") {
		t.Error("Expected view to contain message 'Are you sure?'")
	}

	// Check that options are rendered
	if !containsString(view, "Cancel") {
		t.Error("Expected view to contain 'Cancel' option")
	}
	if !containsString(view, "Delete") {
		t.Error("Expected view to contain 'Delete' option")
	}

	// Check that help is rendered
	if !containsString(view, "Enter: confirm") {
		t.Error("Expected view to contain help text")
	}
}

func TestNewDeleteConfirmDialog(t *testing.T) {
	dialog := NewDeleteConfirmDialog("test-mount", "Mount")

	if dialog.config.Title != "Delete Mount" {
		t.Errorf("Expected title 'Delete Mount', got '%s'", dialog.config.Title)
	}
	if len(dialog.config.Options) != 3 {
		t.Errorf("Expected 3 options, got %d", len(dialog.config.Options))
	}

	// Check option labels
	expectedLabels := []string{"Cancel", "Delete Service Only", "Delete Service and Config"}
	for i, expected := range expectedLabels {
		if dialog.config.Options[i].Label != expected {
			t.Errorf("Expected option %d to be '%s', got '%s'", i, expected, dialog.config.Options[i].Label)
		}
	}
}

func TestNewSimpleConfirmDialog(t *testing.T) {
	dialog := NewSimpleConfirmDialog("Confirm Action", "Are you sure?")

	if dialog.config.Title != "Confirm Action" {
		t.Errorf("Expected title 'Confirm Action', got '%s'", dialog.config.Title)
	}
	if len(dialog.config.Options) != 2 {
		t.Errorf("Expected 2 options, got %d", len(dialog.config.Options))
	}

	// Check option labels
	if dialog.config.Options[0].Label != "No" {
		t.Errorf("Expected first option to be 'No', got '%s'", dialog.config.Options[0].Label)
	}
	if dialog.config.Options[1].Label != "Yes" {
		t.Errorf("Expected second option to be 'Yes', got '%s'", dialog.config.Options[1].Label)
	}
}

func TestNewActionConfirmDialog(t *testing.T) {
	customOptions := []ConfirmDialogOption{
		{Label: "Abort", Action: 0},
		{Label: "Retry", Action: 1},
		{Label: "Ignore", Action: 2},
	}

	dialog := NewActionConfirmDialog("Error", "Something went wrong", customOptions)

	if dialog.config.Title != "Error" {
		t.Errorf("Expected title 'Error', got '%s'", dialog.config.Title)
	}
	if len(dialog.config.Options) != 3 {
		t.Errorf("Expected 3 options, got %d", len(dialog.config.Options))
	}
	if dialog.config.Options[0].Label != "Abort" {
		t.Errorf("Expected first option to be 'Abort', got '%s'", dialog.config.Options[0].Label)
	}
}

func TestConfirmDialog_DestructiveStyling(t *testing.T) {
	dialog := NewConfirmDialog(ConfirmDialogConfig{
		Title:   "Test",
		Message: "Test",
		Options: []ConfirmDialogOption{
			{Label: "Safe", Action: 0, IsDestructive: false},
			{Label: "Danger", Action: 1, IsDestructive: true},
		},
		Width: 80,
	})

	view := dialog.View()

	// The view should be rendered without errors
	if view == "" {
		t.Error("Expected view to not be empty")
	}
}

func TestConfirmDialog_Description(t *testing.T) {
	dialog := NewConfirmDialog(ConfirmDialogConfig{
		Title:       "Test",
		Message:     "Test",
		Description: "Additional information",
		Options:     []ConfirmDialogOption{{Label: "OK", Action: 0}},
		Width:       80,
	})

	view := dialog.View()

	if !containsString(view, "Additional information") {
		t.Error("Expected view to contain description")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
