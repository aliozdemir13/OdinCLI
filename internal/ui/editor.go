package ui

import (
	"fmt"
	"strings"

	"github.com/aliozdemir13/odincli/internal"
	"github.com/aliozdemir13/odincli/internal/models"
	"github.com/aliozdemir13/odincli/internal/style"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type EditorModel struct {
	Textarea textarea.Model
	Content  string
	Aborted  bool

	isSearching bool
	searchQuery string
	suggestion  models.JiraUser // The best match from API
	ghostText   string          // The "dimmed" part of the name
}

func InitialModel() EditorModel {
	ti := textarea.New()
	ti.Placeholder = "Write your comment... (Use Markdown!)\nCtrl+S to save, Ctrl+C to cancel"
	ti.Focus()
	ti.SetWidth(60)
	ti.SetHeight(10)

	return EditorModel{
		Textarea: ti,
	}
}

func (m EditorModel) Init() tea.Cmd {
	return textarea.Blink
}

/*func (m EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlS:
			m.Content = m.Textarea.Value()
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Aborted = true
			return m, tea.Quit

		case tea.KeyTab:
			// IF we have a suggestion, Tab completes it
			if m.ghostText != "" {
				m.insertMention()
				m.resetSearch()
				return m, nil
			}

		case tea.KeyRunes:
			// Check if we just started a mention
			input := string(msg.Runes)
			if input == "@" {
				m.isSearching = true
				m.searchQuery = ""
			} else if m.isSearching {
				m.searchQuery += input
				// Trigger a search (In a real app, you'd debounce this)
				return m, m.lookupUser(m.searchQuery)
			}

		case tea.KeyBackspace:
			if m.isSearching {
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					return m, m.lookupUser(m.searchQuery)
				} else {
					m.resetSearch()
				}
			}
		}

	case userSearchResultMsg:
		if m.isSearching && len(msg.users) > 0 {
			m.suggestion = msg.users[0]
			m.calculateGhostText()
		} else {
			m.ghostText = ""
		}
	}

	m.Textarea, cmd = m.Textarea.Update(msg)
	return m, cmd
}*/

func (m EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlS:
			m.Content = m.Textarea.Value()
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Aborted = true
			return m, tea.Quit

		case tea.KeyTab:
			if m.ghostText != "" {
				m.insertMention()
				m.resetSearch()
				return m, nil
			}

		case tea.KeyRunes:
			input := string(msg.Runes)

			// ALWAYS let the textarea update first so the character appears
			m.Textarea, cmd = m.Textarea.Update(msg)

			if input == "@" {
				m.isSearching = true
				m.searchQuery = ""
				return m, cmd
			} else if m.isSearching {
				m.searchQuery += input
				// Trigger the search via the API
				return m, tea.Batch(cmd, m.lookupUser(m.searchQuery))
			}
			return m, cmd

		case tea.KeyBackspace:
			if m.isSearching {
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					// If we deleted the '@', stop searching
				} else {
					m.resetSearch()
				}
			}
		}

	case userSearchResultMsg:
		if m.isSearching && len(msg.users) > 0 {
			m.suggestion = msg.users[0]
			m.calculateGhostText()
		} else {
			m.ghostText = ""
		}
	}

	m.Textarea, cmd = m.Textarea.Update(msg)
	return m, cmd
}

func (m *EditorModel) calculateGhostText() {
	fullName := m.suggestion.DisplayName
	if strings.HasPrefix(strings.ToLower(fullName), strings.ToLower(m.searchQuery)) {
		m.ghostText = fullName[len(m.searchQuery):]
	}
}

func (m *EditorModel) insertMention() {
	// 1. Prepare your special tag
	mentionTag := fmt.Sprintf("[[%s|%s]] ", m.suggestion.AccountId, m.suggestion.DisplayName)

	// 2. Calculate the "Rewind" length
	// Example: "@ali" -> length 4
	rewindLength := len(m.searchQuery) + 1

	// 3. Simulate Backspace key presses
	// We call the Textarea's Update method manually, passing in a Backspace key event.
	// This "tricks" the component into deleting the characters for us.
	for i := 0; i < rewindLength; i++ {
		m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{
			Type: tea.KeyBackspace,
		})
	}

	// 4. Use the built-in InsertString to add the tag
	// This method exists in textarea and works perfectly
	m.Textarea.InsertString(mentionTag)
}

func (m *EditorModel) resetSearch() {
	m.isSearching = false
	m.searchQuery = ""
	m.ghostText = ""
}

type userSearchResultMsg struct{ users []models.JiraUser }

func (m EditorModel) lookupUser(q string) tea.Cmd {
	return func() tea.Msg {
		if q == "" {
			return userSearchResultMsg{}
		}
		// Use your existing SearchUsers function
		users, _ := internal.SearchUsers(q)
		return userSearchResultMsg{users: users}
	}
}

func (m EditorModel) View() string {
	view := fmt.Sprintf(
		"\n%s\n\n%s",
		style.StyleIndigo(" Odin Markup Editor "),
		m.Textarea.View(),
	)

	// Display Ghost Text Suggestion in the footer
	if m.ghostText != "" {
		view += style.StyleDim("\n  Suggestion: " + m.searchQuery + style.StyleBold(m.ghostText) + " (Press Tab to complete)")
	}

	return view + "\n\n"
}
