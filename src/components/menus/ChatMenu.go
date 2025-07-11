// ChatMenu.go - Contains all logic for the Chats menu and its flows in the menu system.
// This includes actions for listing chats, favoriting/unfavoriting, renaming, previewing, and launching chats.

package menus

import (
	"aichat/src/types"
)

// State represents the current and previous state for menu/modal navigation.
type State struct {
	Current string
	Prev    string
}

// FlowType manages a stack of modals and the parent state for a flow.
type FlowType struct {
	Modals      []Modal
	ParentState string
}

// Modal is a generic interface for all modals in a flow.
type Modal interface {
	View() string
	Update(input string) (Modal, bool) // returns updated modal and whether to advance
	GetPrev() string
}

// InputPromptModal prompts the user for input (e.g., renaming a chat).
type InputPromptModal struct {
	Prompt      string
	Input       string
	ControlInfo string
	ActionInfo  string
	Prev        string
}

func (m *InputPromptModal) View() string {
	// TODO: Render ASCII art, prompt, input box, control info centered
	return ""
}

func (m *InputPromptModal) Update(input string) (Modal, bool) {
	// TODO: Handle input, return NoticeModal on Enter, or self on edit
	return m, false
}

func (m *InputPromptModal) GetPrev() string { return m.Prev }

// NoticeModal displays a notice message (e.g., rename result).
type NoticeModal struct {
	Message string
	Prev    string
}

func (m *NoticeModal) View() string {
	// TODO: Render centered notice message
	return ""
}

func (m *NoticeModal) Update(input string) (Modal, bool) {
	// On Enter or Esc, signal to pop modal
	return m, true
}

func (m *NoticeModal) GetPrev() string { return m.Prev }

// ListChatsAction displays the list of chats, allows navigation, selection, favoriting, renaming, and previewing.
func ListChatsAction() error {
	// TODO: Implement chat listing, navigation, and key handling (enter, f, r, p)
	return nil
}

// FavoriteChatAction toggles favorite status for the selected chat.
func FavoriteChatAction(chatName string) error {
	// TODO: Implement favorite/unfavorite logic
	return nil
}

// RenameChatAction launches the renaming flow, managing state and modals.
func RenameChatAction(state State, chatName string) State {
	// TODO: Implement flow logic for renaming (manage modals, transitions, and state)
	return state
}

// PreviewChatAction shows the last 3 messages of the selected chat in a modal popup.
func PreviewChatAction(chatName string) error {
	// TODO: Implement preview logic (open EditorModal with last 3 messages)
	return nil
}

// Helper: Load all chat names and metadata
func loadAllChats() ([]string, error) {
	// TODO: Implement chat loading logic
	return nil, nil
}

// Helper: Load chat metadata
func loadChatMetadata(chatName string) (*types.ChatMetadata, error) {
	// TODO: Implement metadata loading
	return nil, nil
}
