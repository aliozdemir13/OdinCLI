package ui

import (
	"strings"
	"testing"

	"github.com/aliozdemir13/odincli/internal/models"
	tea "github.com/charmbracelet/bubbletea"
)

// TestInitialModel checks if the editor starts with correct defaults
func TestInitialModel(t *testing.T) {
	m := InitialModel()

	if !m.Textarea.Focused() {
		t.Error("Expected textarea to be focused on start")
	}

	if m.Aborted {
		t.Error("Expected Aborted to be false initially")
	}
}

// TestEditorAbortion checks if Esc or Ctrl+C sets the Aborted flag
func TestEditorAbortion(t *testing.T) {
	m := InitialModel()

	// Simulate Esc key
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ := m.Update(msg)

	finalModel := updatedModel.(EditorModel)
	if !finalModel.Aborted {
		t.Error("Expected model to be aborted after Esc")
	}
}

// TestEditorSave checks if Ctrl+S captures the content
func TestEditorSave(t *testing.T) {
	m := InitialModel()
	m.Textarea.SetValue("Hello Jira")

	msg := tea.KeyMsg{Type: tea.KeyCtrlS}
	updatedModel, _ := m.Update(msg)

	finalModel := updatedModel.(EditorModel)
	if finalModel.Content != "Hello Jira" {
		t.Errorf("Expected content 'Hello Jira', got %q", finalModel.Content)
	}
}

// TestMentionTrigger checks if typing '@' starts the search mode
func TestMentionTrigger(t *testing.T) {
	m := InitialModel()

	// Simulate typing '@'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("@")}
	updatedModel, _ := m.Update(msg)

	finalModel := updatedModel.(EditorModel)
	if !finalModel.isSearching {
		t.Error("Expected isSearching to be true after typing @")
	}
}

// TestGhostTextCalculation checks the auto-complete suggestion logic
func TestGhostTextCalculation(t *testing.T) {
	m := InitialModel()
	m.isSearching = true
	m.searchQuery = "ali"
	m.suggestion = models.JiraUser{
		DisplayName: "Ali Ozdemir",
		AccountID:   "12345",
	}

	m.calculateGhostText()

	expectedGhost := " Ozdemir"
	if m.ghostText != expectedGhost {
		t.Errorf("Expected ghost text %q, got %q", expectedGhost, m.ghostText)
	}
}

// TestTabCompletion ensures that pressing Tab inserts the mention tag
func TestTabCompletion(t *testing.T) {
	m := InitialModel()
	m.isSearching = true
	m.searchQuery = "ali"
	m.ghostText = " Ozdemir"
	m.suggestion = models.JiraUser{
		DisplayName: "Ali Ozdemir",
		AccountID:   "12345",
	}

	// Simulate Tab key
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := m.Update(msg)

	finalModel := updatedModel.(EditorModel)

	// The textarea value should now contain the Jira mention format
	val := finalModel.Textarea.Value()
	if !strings.Contains(val, "[[12345|Ali Ozdemir]]") {
		t.Errorf("Expected mention tag in textarea, got %q", val)
	}

	if finalModel.isSearching {
		t.Error("Expected search mode to reset after Tab completion")
	}
}

// TestUserSearchResultMsg simulates the async API response
func TestUserSearchResultMsg(t *testing.T) {
	m := InitialModel()
	m.isSearching = true
	m.searchQuery = "ali"

	// Create a mock search result message
	mockResults := userSearchResultMsg{
		users: []models.JiraUser{
			{DisplayName: "Ali Ozdemir", AccountID: "123"},
		},
	}

	updatedModel, _ := m.Update(mockResults)
	finalModel := updatedModel.(EditorModel)

	if finalModel.suggestion.DisplayName != "Ali Ozdemir" {
		t.Errorf("Expected suggestion 'Ali Ozdemir', got %q", finalModel.suggestion.DisplayName)
	}

	if finalModel.ghostText == "" {
		t.Error("Expected ghost text to be calculated after receiving results")
	}
}
