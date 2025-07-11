// editor.go - Defines the EditorModal for displaying a message in a read-only editor with cursor and selection support.
// Used for viewing and copying chat messages in a modal overlay.

package input

import (
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// EditorModal displays a message in a read-only editor with cursor and selection support.
type EditorModal struct {
	Content     string // The message content to display
	Cursor      int    // Cursor position
	SelectStart int    // Selection start index, -1 if no selection
	SelectEnd   int    // Selection end index, -1 if no selection
	Focused     bool   // Whether the editor is focused
	Quitting    bool   // Whether the modal is quitting
	Message     string // Optional info/error message
}

// NewEditorModal creates a new EditorModal with the given content.
func NewEditorModal(content string) *EditorModal {
	return &EditorModal{
		Content:     content,
		Cursor:      0,
		SelectStart: -1,
		SelectEnd:   -1,
		Focused:     true,
		Quitting:    false,
	}
}

// Init initializes the editor modal (Bubble Tea compatibility).
func (m *EditorModal) Init() tea.Cmd { return nil }

// Update handles key events for navigation, selection, clipboard, and closing.
func (m *EditorModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.Focused {
		return m, nil
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			// Prompt for confirmation before closing
			m.Quitting = true
			return m, tea.Quit
		case "ctrl+i":
			// Focus input box (handled by parent)
			m.Focused = false
			return m, nil
		case "ctrl+c":
			// Copy selection or all
			if m.SelectStart >= 0 && m.SelectEnd > m.SelectStart {
				_ = clipboard.WriteAll(m.Content[m.SelectStart:m.SelectEnd])
			} else {
				_ = clipboard.WriteAll(m.Content)
			}
			return m, nil
		case "ctrl+x":
			// Cut selection (read-only, so just copy)
			if m.SelectStart >= 0 && m.SelectEnd > m.SelectStart {
				_ = clipboard.WriteAll(m.Content[m.SelectStart:m.SelectEnd])
			} else {
				_ = clipboard.WriteAll(m.Content)
			}
			return m, nil
		case "ctrl+v":
			// Paste is ignored (read-only)
			return m, nil
		case "left":
			if m.Cursor > 0 {
				m.Cursor--
			}
			return m, nil
		case "right":
			if m.Cursor < len(m.Content) {
				m.Cursor++
			}
			return m, nil
		case "home":
			m.Cursor = 0
			return m, nil
		case "end":
			m.Cursor = len(m.Content)
			return m, nil
		}
	}
	return m, nil
}

// View renders the editor modal with the message content and cursor.
func (m *EditorModal) View() string {
	content := m.Content
	if m.Cursor >= 0 && m.Cursor <= len(content) {
		content = content[:m.Cursor] + "|" + content[m.Cursor:]
	}
	return "[Read-Only Message Editor]\n" + content + "\n" + m.Message + "\n(Esc to close, Ctrl+C/X to copy, Ctrl+I to focus input)"
}
