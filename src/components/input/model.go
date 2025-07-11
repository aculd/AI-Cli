// model.go - Defines the InputModel struct for the chat input box, supporting focus, clipboard, shortcuts, and cursor management.
// Integrates with AppModel and ChatModel for message sending and input focus control.

package input

import (
	"strings"

	"aichat/src/types"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// InputModel manages the chat input box state and logic.
type InputModel struct {
	Buffer   string           // Current input buffer
	Cursor   int              // Cursor position in buffer
	Focused  bool             // Whether the input box is focused
	Quitting bool             // Whether input is quitting (for modal/exit)
	Width    int              // Render width
	Height   int              // Render height
	Message  string           // Optional error/info message
	Controls types.ControlSet // ControlSet for this input
}

// NewInputModel constructs a new InputModel.
func NewInputModel(width, height int) *InputModel {
	m := &InputModel{
		Buffer:  "",
		Cursor:  0,
		Focused: true,
		Width:   width,
		Height:  height,
	}
	m.Controls = types.ControlSet{
		Controls: []types.ControlType{
			{
				Name: "Left", Key: "left",
				Action: func() bool {
					if m.Cursor > 0 {
						m.Cursor--
					}
					return true
				},
			},
			{
				Name: "Right", Key: "right",
				Action: func() bool {
					if m.Cursor < len(m.Buffer) {
						m.Cursor++
					}
					return true
				},
			},
			{
				Name: "Backspace", Key: "backspace",
				Action: func() bool {
					if m.Cursor > 0 && len(m.Buffer) > 0 {
						m.Buffer = m.Buffer[:m.Cursor-1] + m.Buffer[m.Cursor:]
						m.Cursor--
					}
					return true
				},
			},
			{
				Name: "Ctrl+C", Key: "ctrl+c",
				Action: func() bool { _ = clipboard.WriteAll(m.Buffer); return true },
			},
			{
				Name: "Ctrl+V", Key: "ctrl+v",
				Action: func() bool {
					paste, err := clipboard.ReadAll()
					if err == nil && paste != "" {
						m.Buffer = m.Buffer[:m.Cursor] + paste + m.Buffer[m.Cursor:]
						m.Cursor += len(paste)
					}
					return true
				},
			},
			{
				Name: "Enter", Key: "enter",
				Action: func() bool { m.Focused = false; return true },
			},
			{
				Name: "Esc", Key: "esc",
				Action: func() bool { m.Quitting = true; return true },
			},
		},
	}
	return m
}

// Init initializes the input model (Bubble Tea compatibility).
func (m *InputModel) Init() tea.Cmd { return nil }

// Update handles Bubble Tea messages for input, clipboard, and focus.
func (m *InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.Focused {
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "ctrl+i" {
			m.Focused = true
		}
		return m, nil
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		for _, ctrl := range m.Controls.Controls {
			if keyMsg.String() == ctrl.Key && ctrl.Action != nil {
				if ctrl.Action() {
					return m, nil
				}
			}
		}
		switch keyMsg.String() {
		case "ctrl+q", "esc":
			// Trigger chat close confirmation (handled by parent)
			return m, tea.Quit
		case "ctrl+c":
			// Copy: if selection is implemented, copy selected text; else copy all
			_ = clipboard.WriteAll(m.Buffer) // TODO: support selection
			return m, nil
		case "ctrl+x":
			// Cut: if selection is implemented, cut selected text; else cut all
			_ = clipboard.WriteAll(m.Buffer) // TODO: support selection
			m.Buffer = ""
			m.Cursor = 0
			return m, nil
		case "ctrl+v":
			paste, err := clipboard.ReadAll()
			if err == nil && paste != "" {
				m.Buffer = m.Buffer[:m.Cursor] + paste + m.Buffer[m.Cursor:]
				m.Cursor += len(paste)
			}
			return m, nil
		case "ctrl+i":
			m.Focused = true
			return m, nil
		case "enter":
			if keyMsg.String() == "shift+enter" || keyMsg.String() == "alt+enter" {
				// Shift+Enter or Alt+Enter: insert newline
				m.Buffer = m.Buffer[:m.Cursor] + "\n" + m.Buffer[m.Cursor:]
				m.Cursor++
				return m, nil
			}
			// Enter: submit (handled by parent)
			return m, tea.Quit
		case "backspace":
			if m.Cursor > 0 && len(m.Buffer) > 0 {
				m.Buffer = m.Buffer[:m.Cursor-1] + m.Buffer[m.Cursor:]
				m.Cursor--
			}
			return m, nil
		case "left":
			if m.Cursor > 0 {
				m.Cursor--
			}
			return m, nil
		case "right":
			if m.Cursor < len(m.Buffer) {
				m.Cursor++
			}
			return m, nil
		case "home":
			m.Cursor = 0
			return m, nil
		case "end":
			m.Cursor = len(m.Buffer)
			return m, nil
		default:
			if len(keyMsg.String()) == 1 && keyMsg.Type == tea.KeyRunes {
				m.Buffer = m.Buffer[:m.Cursor] + keyMsg.String() + m.Buffer[m.Cursor:]
				m.Cursor++
			}
			return m, nil
		}
	}
	return m, nil
}

// View renders the input box UI as a string.
func (m *InputModel) View() string {
	input := m.Buffer
	if m.Cursor >= 0 && m.Cursor <= len(input) {
		input = input[:m.Cursor] + "|" + input[m.Cursor:]
	}
	return "Input: " + strings.ReplaceAll(input, "\n", "\\n") + m.Message
}

// GetControlSet returns the current control set for the input model
func (m *InputModel) GetControlSet() interface{} {
	return m.Controls
}
