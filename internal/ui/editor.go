// Package ui provides markup text editor module
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

// EditorModel is the data structure used for the markup editor
type EditorModel struct {
	Textarea textarea.Model
	Content  string
	Aborted  bool

	isSearching bool
	searchQuery string
	suggestion  models.JiraUser // The best match from API
	ghostText   string          // The "dimmed" part of the name
}

// InitialModel returns the default state for the editor UI.
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

// Init itializes the editor UI.
func (m EditorModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles the text changes in the editor
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

			m.Textarea, cmd = m.Textarea.Update(msg)

			if input == "@" {
				m.isSearching = true
				m.searchQuery = ""
				return m, cmd
			} else if m.isSearching {
				m.searchQuery += input
				return m, tea.Batch(cmd, m.lookupUser(m.searchQuery))
			}
			return m, cmd

		case tea.KeyBackspace:
			if m.isSearching {
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
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
	mentionTag := fmt.Sprintf("[[%s|%s]] ", m.suggestion.AccountID, m.suggestion.DisplayName)

	rewindLength := len(m.searchQuery) + 1

	for i := 0; i < rewindLength; i++ {
		m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{
			Type: tea.KeyBackspace,
		})
	}
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

// View handles text suggestions
func (m EditorModel) View() string {
	view := fmt.Sprintf(
		"\n%s\n\n%s",
		style.Indigo(" Odin Markup Editor "),
		m.Textarea.View(),
	)

	// Display Ghost Text Suggestion in the footer
	if m.ghostText != "" {
		view += style.Dim("\n  Suggestion: " + m.searchQuery + style.Bold(m.ghostText) + " (Press Tab to complete)")
	}

	return view + "\n\n"
}
