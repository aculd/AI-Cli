// model.go - ChatViewState for the center chat window in the three-pane layout.
// Implements ViewState for integration with the navigation stack and app orchestration.
// This state is responsible for rendering chat messages, handling streaming, and managing chat-specific logic.

package chat

import (
	"aichat/src/types"

	tea "github.com/charmbracelet/bubbletea"
)

// ViewState interface is imported from types package

// ChatMessage represents a single message in the chat
// IsUser: true if sent by user, false if assistant
// Content: message text
type ChatMessage struct {
	Content string
	IsUser  bool
}

// ChatViewState represents the chat window (center pane) in the three-pane layout.
type ChatViewState struct {
	Messages           []ChatMessage // Chat messages (now using struct)
	Streaming          bool          // Whether a message is currently streaming
	ChatTitle          string        // Title or summary of the chat
	ScrollPos          int           // Current scroll position (index of first visible message)
	SelectedMessageIdx int           // Index of selected message in visible window, -1 if none
	ResponseReceived   bool          // True if a response was just received
	WaitingForResponse bool          // True if waiting for a response
	ControlSet         []struct {
		Name   string
		Key    string
		Action func(c *ChatViewState) bool
	}
}

// Chat window control set: Up/Down to scroll, PageUp/PageDown for fast scroll, Enter to select, Esc to unfocus
var chatControlSet = []struct {
	Name   string
	Key    string
	Action func(c *ChatViewState) bool
}{
	{"Up", "up", func(c *ChatViewState) bool {
		if c.ScrollPos > 0 {
			c.ScrollPos--
			return true
		}
		return false
	}},
	{"Down", "down", func(c *ChatViewState) bool {
		c.ScrollPos++ // TODO: bound by message count
		return true
	}},
	{"PageUp", "pgup", func(c *ChatViewState) bool {
		c.ScrollPos -= 10
		if c.ScrollPos < 0 {
			c.ScrollPos = 0
		}
		return true
	}},
	{"PageDown", "pgdn", func(c *ChatViewState) bool {
		c.ScrollPos += 10 // TODO: bound by message count
		return true
	}},
	{"Enter", "enter", func(c *ChatViewState) bool {
		// TODO: select message
		return true
	}},
	{"Esc", "esc", func(c *ChatViewState) bool {
		// TODO: unfocus logic
		return true
	}},
}

// ViewType returns the view type for this state
func (c *ChatViewState) ViewType() types.ViewType {
	return types.ChatStateType
}

// IsMainMenu returns false; this is not the main menu state.
func (c *ChatViewState) IsMainMenu() bool {
	return false
}

// MarshalState serializes the chat view state
func (c *ChatViewState) MarshalState() ([]byte, error) {
	// TODO: Implement full serialization if needed
	return nil, nil
}

// UnmarshalState deserializes the chat view state
func (c *ChatViewState) UnmarshalState(data []byte) error {
	// TODO: Implement full deserialization if needed
	return nil
}

// View renders the chat window (placeholder implementation).
func (c *ChatViewState) View() string {
	// TODO: Replace with real rendering logic (Lipgloss, message formatting, etc.)
	view := "[Chat Window: " + c.ChatTitle + "]\n"
	for _, msg := range c.Messages {
		view += msg.Content + "\n"
	}
	if c.Streaming {
		view += "[Streaming...]\n"
	}
	return view
}

// Update handles Bubble Tea messages (placeholder for now).
func (c *ChatViewState) Update(msg tea.Msg) (types.ViewState, tea.Cmd) {
	// TODO: Implement message streaming, chat updates, etc.
	return c, nil
}

// GetControlSets returns the chat window's control sets
func (c *ChatViewState) GetControlSets() []types.ControlSet {
	// Convert chatControlSet to types.ControlSet format
	controls := make([]types.ControlSet, 1)
	controlTypes := make([]types.ControlType, len(chatControlSet))

	for i, ctrl := range chatControlSet {
		controlTypes[i] = types.ControlType{
			Name: ctrl.Name,
			Key:  ctrl.Key,
			Action: func() bool {
				return ctrl.Action(c)
			},
		}
	}

	controls[0] = types.ControlSet{
		Controls: controlTypes,
	}

	return controls
}

// GetControlSet returns the chat window's control set (legacy method)
func (c *ChatViewState) GetControlSet() interface{} {
	return chatControlSet
}

// TODO: Integrate with sidebar navigation and input area for full three-pane layout.
// TODO: Support message streaming, scrolling, and chat metadata display.
// TODO: Replace string messages with proper message structs.
