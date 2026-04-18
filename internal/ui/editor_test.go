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

// TestInit ensures Init returns a non-nil command (the blink cursor).
func TestInit(t *testing.T) {
	m := InitialModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("Expected Init to return a non-nil command")
	}
}

// TestEditorAbortion checks if Esc or Ctrl+C sets the Aborted flag
func TestEditorAbortion(t *testing.T) {
	cases := []struct {
		name string
		key  tea.KeyType
	}{
		{"Esc", tea.KeyEsc},
		{"CtrlC", tea.KeyCtrlC},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := InitialModel()
			msg := tea.KeyMsg{Type: tc.key}
			updatedModel, _ := m.Update(msg)

			finalModel := updatedModel.(EditorModel)
			if !finalModel.Aborted {
				t.Errorf("Expected model to be aborted after %s", tc.name)
			}
		})
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

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("@")}
	updatedModel, _ := m.Update(msg)

	finalModel := updatedModel.(EditorModel)
	if !finalModel.isSearching {
		t.Error("Expected isSearching to be true after typing @")
	}

}

// TestBackspaceDuringSearch trims the search query character by character.
func TestBackspaceDuringSearch(t *testing.T) {
	m := InitialModel()
	m.isSearching = true
	m.searchQuery = "ali"

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ := m.Update(msg)

	finalModel := updatedModel.(EditorModel)
	if finalModel.searchQuery != "al" {
		t.Errorf("Expected searchQuery 'al', got %q", finalModel.searchQuery)
	}
	if !finalModel.isSearching {
		t.Error("Expected to still be searching after trimming one char")
	}

}

// TestBackspaceResetsSearchWhenEmpty ensures that backspacing past '@' exits search mode.
func TestBackspaceResetsSearchWhenEmpty(t *testing.T) {
	m := InitialModel()
	m.isSearching = true
	m.searchQuery = ""

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ := m.Update(msg)

	finalModel := updatedModel.(EditorModel)
	if finalModel.isSearching {
		t.Error("Expected search mode to reset when backspacing with empty query")
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

// TestGhostTextNoMatch ensures ghost text stays empty when query doesn't prefix the name.
func TestGhostTextNoMatch(t *testing.T) {
	m := InitialModel()
	m.searchQuery = "zzz"
	m.suggestion = models.JiraUser{DisplayName: "Ali Ozdemir"}

	m.calculateGhostText()

	if m.ghostText != "" {
		t.Errorf("Expected empty ghost text for non-matching prefix, got %q", m.ghostText)
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

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := m.Update(msg)

	finalModel := updatedModel.(EditorModel)

	val := finalModel.Textarea.Value()
	if !strings.Contains(val, "[[12345|Ali Ozdemir]]") {
		t.Errorf("Expected mention tag in textarea, got %q", val)
	}

	if finalModel.isSearching {
		t.Error("Expected search mode to reset after Tab completion")
	}

}

// TestTabWithoutGhostTextIsNoop makes sure Tab does nothing special when no suggestion is active.
func TestTabWithoutGhostTextIsNoop(t *testing.T) {
	m := InitialModel()
	m.Textarea.SetValue("hello")

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := m.Update(msg)
	finalModel := updatedModel.(EditorModel)

	if finalModel.isSearching {
		t.Error("Tab without ghost text should not enter search mode")
	}

}

// TestUserSearchResultMsg simulates the async API response
func TestUserSearchResultMsg(t *testing.T) {
	m := InitialModel()
	m.isSearching = true
	m.searchQuery = "ali"

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

// TestUserSearchResultMsgEmpty ensures ghost text clears when the API returns nothing.
func TestUserSearchResultMsgEmpty(t *testing.T) {
	m := InitialModel()
	m.isSearching = true
	m.searchQuery = "xyz"
	m.ghostText = "stale"

	updatedModel, _ := m.Update(userSearchResultMsg{users: nil})
	finalModel := updatedModel.(EditorModel)

	if finalModel.ghostText != "" {
		t.Errorf("Expected ghost text to clear on empty results, got %q", finalModel.ghostText)
	}

}

// TestLookupUserEmptyQuery covers the early-return branch without touching the network.
func TestLookupUserEmptyQuery(t *testing.T) {
	m := InitialModel()
	cmd := m.lookupUser("")
	if cmd == nil {
		t.Fatal("Expected a command, got nil")
	}

	msg := cmd()
	result, ok := msg.(userSearchResultMsg)
	if !ok {
		t.Fatalf("Expected userSearchResultMsg, got %T", msg)
	}
	if len(result.users) != 0 {
		t.Errorf("Expected no users for empty query, got %d", len(result.users))
	}

}

// TestView covers the default and ghost-text rendering branches.
func TestView(t *testing.T) {
	m := InitialModel()
	out := m.View()
	if !strings.Contains(out, "Odin Markup Editor") {
		t.Error("Expected view to contain the editor title")
	}

	m.searchQuery = "ali"
	m.ghostText = " Ozdemir"
	out = m.View()
	if !strings.Contains(out, "Suggestion:") {
		t.Error("Expected view to show suggestion hint when ghostText is set")
	}

}
