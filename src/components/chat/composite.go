// composite.go - Composite ChatViewState for three-pane chat layout
// Implements ViewState and persists sidebar, chat window, and input subviews for restoration after modals.

package chat

import (
	"aichat/src/components/input"
	"aichat/src/components/sidebar"
	"aichat/src/types"

	"strings"
	"time"

	"aichat/src/components/modals/dialogs"
	"aichat/src/models"
	"aichat/src/services/storage/repositories"
	"context"

	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuViewState represents the state of a menu or submenu
// Can be used for Main Menu, Chats, Prompts, etc.
type MenuViewState struct {
	MenuName    string
	Entries     []string
	Selected    int
	ControlInfo string
}

// IsMainMenu returns true if this is the main menu
func (m *MenuViewState) IsMainMenu() bool { return m.MenuName == "Main Menu" }

// MarshalState serializes the menu state
func (m *MenuViewState) MarshalState() ([]byte, error) { return nil, nil }

// UnmarshalState deserializes the menu state
func (m *MenuViewState) UnmarshalState(data []byte) error { return nil }

// ViewType returns ModalStateType for menus
func (m *MenuViewState) ViewType() types.ViewType { return types.ModalStateType }

// Update method: handle up/down/enter/esc, return new MenuViewState for submenus
func (m *MenuViewState) Update(msg tea.Msg) (types.ViewState, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up":
			if m.Selected > 0 {
				m.Selected--
			} else {
				m.Selected = len(m.Entries) - 1
			}
			return m, nil
		case "down":
			if m.Selected < len(m.Entries)-1 {
				m.Selected++
			} else {
				m.Selected = 0
			}
			return m, nil
		case "enter":
			// Example: handle Chats submenu
			if m.MenuName == "Main Menu" {
				selected := m.Entries[m.Selected]
				if selected == "Chats" {
					chatsMenu := &MenuViewState{
						MenuName:    "Chats",
						Entries:     []string{"New Chat", "List Chats", "List Favorites", "Custom Chat", "Delete Chat"},
						Selected:    0,
						ControlInfo: "Up/Down: Navigate  Enter: Select  Esc: Back",
					}
					return chatsMenu, nil
				}
				// TODO: handle other main menu entries
			}
			if m.MenuName == "Chats" {
				// TODO: handle chats submenu actions
				return m, nil
			}
			return m, nil
		case "esc":
			// Signal back/exit by returning nil (caller should handle stack)
			return nil, nil
		}
	}
	return m, nil
}

// RenderMenuView renders the current menu or submenu using RenderMenuSubmenu
func (m *MenuViewState) RenderMenuView(width, height int) string {
	styledEntries := make([]string, len(m.Entries))
	for i, entry := range m.Entries {
		entryStyle := lipgloss.NewStyle().Padding(0, 1)
		if i == m.Selected {
			entryStyle = entryStyle.Bold(true).Foreground(lipgloss.Color("203")).Background(lipgloss.Color("236"))
		}
		styledEntries[i] = entryStyle.Render(entry)
	}
	menuContent := RenderMenuSubmenu(m.MenuName, styledEntries, m.ControlInfo)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, menuContent)
}

// CompositeChatViewState now supports menu stack navigation
// Add a MenuStack field for stack-based menu/submenu navigation
type CompositeChatViewState struct {
	Sidebar      *sidebar.SidebarModel
	Chats        map[string]*ChatViewState // All open chats by ID/tab name
	ActiveChatID string                    // Currently active chat/tab
	Input        *input.InputModel
	Focus        string                   // "sidebar", "chat", "input"
	CurrentFocus string                   // Tracks the currently focused subview
	Workers      map[string]*StreamWorker // Streaming workers per chat
	Menu         *MenuViewState           // Current menu/submenu (nil if not in menu mode)
	MenuStack    []types.ViewState        // Stack of menus/modals
}

var _ types.ViewState = (*CompositeChatViewState)(nil)

// ViewType returns ChatStateType for navigation stack
func (c *CompositeChatViewState) ViewType() types.ViewType { return types.ChatStateType }

// IsMainMenu returns false; this is not the main menu
func (c *CompositeChatViewState) IsMainMenu() bool { return false }

// MarshalState serializes the composite state (for persistence)
func (c *CompositeChatViewState) MarshalState() ([]byte, error) {
	// TODO: Implement full serialization if needed
	return nil, nil
}

// UnmarshalState deserializes the composite state
func (c *CompositeChatViewState) UnmarshalState(data []byte) error {
	// TODO: Implement full deserialization if needed
	return nil
}

// GetControlSets returns the composite view's control sets
func (c *CompositeChatViewState) GetControlSets() []types.ControlSet {
	controls := []types.ControlSet{
		{
			Controls: []types.ControlType{
				{
					Name: "Tab", Key: "tab",
					Action: func() bool {
						// TODO: switch focus between sidebar, chat, and input
						return true
					},
				},
				{
					Name: "Esc", Key: "esc",
					Action: func() bool {
						// TODO: handle back/cancel
						return true
					},
				},
			},
		},
	}
	return controls
}

// HandleCtrlS stops the stream for the active chat
func (c *CompositeChatViewState) HandleCtrlS() {
	if c.ActiveChatID != "" && c.Workers != nil {
		if worker, ok := c.Workers[c.ActiveChatID]; ok && worker != nil {
			worker.Stop()
		}
	}
}

// HandleStreamCancel removes the last assistant and user message and pre-fills the input box
func (c *CompositeChatViewState) HandleStreamCancel() {
	activeChat := c.GetActiveChat()
	if activeChat == nil || len(activeChat.Messages) < 2 {
		return
	}
	// Remove last assistant and user message
	userMsg := activeChat.Messages[len(activeChat.Messages)-2]
	activeChat.Messages = activeChat.Messages[:len(activeChat.Messages)-2]
	// Prefill input box with user's message content
	if c.Input != nil {
		c.Input.Buffer = userMsg.Content
		c.Input.Cursor = len(userMsg.Content)
	}
}

// PushModal pushes a new modal (menu, input, notice, etc.) onto the stack and sets it as current
func (c *CompositeChatViewState) PushModal(modal types.ViewState) {
	c.MenuStack = append(c.MenuStack, modal)
	if menu, ok := modal.(*MenuViewState); ok {
		c.Menu = menu
	} else {
		c.Menu = nil
	}
}

// PopModal pops the current modal and returns to the previous one, or main menu if stack is empty
func (c *CompositeChatViewState) PopModal() {
	if len(c.MenuStack) > 1 {
		c.MenuStack = c.MenuStack[:len(c.MenuStack)-1]
		if menu, ok := c.MenuStack[len(c.MenuStack)-1].(*MenuViewState); ok {
			c.Menu = menu
		} else {
			c.Menu = nil
		}
	} else {
		mainMenu := CreateMainMenu()
		c.MenuStack = []types.ViewState{mainMenu}
		c.Menu = mainMenu
	}
}

// ResetToMainMenu resets the menu stack and renders the main menu
func (c *CompositeChatViewState) ResetToMainMenu() {
	mainMenu := CreateMainMenu()
	c.MenuStack = []types.ViewState{mainMenu}
	c.Menu = mainMenu
}

// Update handles modal stack navigation and updates
func (c *CompositeChatViewState) Update(msg tea.Msg) (types.ViewState, tea.Cmd) {
	// If any modal is active, handle modal updates
	if len(c.MenuStack) > 0 {
		currentModal := c.MenuStack[len(c.MenuStack)-1]

		// Update the current modal
		newModal, cmd := currentModal.Update(msg)

		// Update the modal in the stack
		c.MenuStack[len(c.MenuStack)-1] = newModal

		// Update the Menu field if it's a MenuViewState
		if menu, ok := newModal.(*MenuViewState); ok {
			c.Menu = menu
		} else {
			c.Menu = nil
		}

		// Handle specific modal logic
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "enter":
				// Handle MenuViewState specific logic
				if menu, ok := currentModal.(*MenuViewState); ok {
					if menu.MenuName == "Main Menu" {
						selected := menu.Entries[menu.Selected]
						if selected == "Chats" {
							chatsMenu := &MenuViewState{
								MenuName:    "Chats",
								Entries:     []string{"New Chat", "List Chats", "List Favorites", "Custom Chat", "Delete Chat"},
								Selected:    0,
								ControlInfo: "Up/Down: Navigate  Enter: Select  Esc: Back",
							}
							c.PushModal(chatsMenu)
							return c, nil
						}
						// TODO: handle other main menu entries
					}
					if menu.MenuName == "Chats" {
						selected := menu.Entries[menu.Selected]
						switch selected {
						case "New Chat":
							// TODO: Implement New Chat flow
							return c, nil
						case "List Chats":
							// TODO: Implement List Chats
							return c, nil
						case "List Favorites":
							// TODO: Implement List Favorites
							return c, nil
						case "Custom Chat":
							c.startCustomChatFlow()
							return c, nil
						case "Delete Chat":
							// TODO: Implement Delete Chat
							return c, nil
						}
					}
				}
				return c, cmd
			case "esc":
				c.PopModal()
				return c, nil
			}
		}

		return c, cmd
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "ctrl+s" {
		c.HandleCtrlS()
		return c, nil
	}
	// Listen for StreamEventCancel from the worker
	if c.ActiveChatID != "" && c.Workers != nil {
		if worker, ok := c.Workers[c.ActiveChatID]; ok && worker != nil {
			select {
			case event := <-worker.EventChan:
				if event.Type == StreamEventCancel {
					c.HandleStreamCancel()
					return c, nil
				}
			default:
			}
		}
	}
	switch c.Focus {
	case "sidebar":
		model, cmd, _ := c.Sidebar.Update(msg)
		// Detect tab switch
		if model.ActiveTab != c.Sidebar.ActiveTab {
			// Tab changed: update ActiveChatID
			if model.ActiveTab >= 0 && model.ActiveTab < len(model.Tabs) {
				c.ActiveChatID = model.Tabs[model.ActiveTab]
			}
		}
		c.Sidebar = model
		return c, cmd
	case "input":
		if newInput, cmd := c.Input.Update(msg); newInput != nil {
			if im, ok := newInput.(*input.InputModel); ok {
				c.Input = im
			}
			return c, cmd
		}
		return c, nil
	default: // "chat" or fallback
		if c.ActiveChatID != "" && c.Chats[c.ActiveChatID] != nil {
			if newChat, cmd := c.Chats[c.ActiveChatID].Update(msg); newChat != nil {
				if cv, ok := newChat.(*ChatViewState); ok {
					c.Chats[c.ActiveChatID] = cv
				}
				return c, cmd
			}
			return c, nil
		}
		return c, nil
	}
}

// GetActiveChat returns the currently active chat view (or nil if not found)
func (c *CompositeChatViewState) GetActiveChat() *ChatViewState {
	if c.Chats == nil || c.ActiveChatID == "" {
		return nil
	}
	return c.Chats[c.ActiveChatID]
}

// GetActiveControlSet returns the control set for the currently focused subview
func (c *CompositeChatViewState) GetActiveControlSet() interface{} {
	switch c.CurrentFocus {
	case "sidebar":
		if c.Sidebar != nil {
			return c.Sidebar.GetControlSet()
		}
	case "input":
		if c.Input != nil {
			return c.Input.GetControlSet()
		}
	case "chat":
		if c.ActiveChatID != "" && c.Chats[c.ActiveChatID] != nil {
			return c.Chats[c.ActiveChatID].GetControlSet()
		}
	}
	return nil
}

// Styles and emoji constants
var (
	activeChatEmoji = "🟢"
	allChatsEmoji   = "💬"
	favoritesEmoji  = "⭐"
	responseEmoji   = "📨"
	statusWaiting   = "⏳ Waiting for response"
	statusStreaming = "💬 Streaming"
	statusInput     = "🟢 Waiting for input"
	statusReceived  = "📨 Response received"

	userTagStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("33")).
			Background(lipgloss.Color("15")).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder())

	assistantTagStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("129")).
				Background(lipgloss.Color("15")).
				Padding(0, 1).
				Border(lipgloss.RoundedBorder())

	msgBoxStyle = lipgloss.NewStyle().
			Margin(1, 0).
			Padding(1, 2).
			Border(lipgloss.NormalBorder()).
			Width(40)
)

// RenderStatusBar renders the top status bar with model (left) and status (right)
func (c *CompositeChatViewState) RenderStatusBar() string {
	modelName := "Model name here"
	activeChat := c.GetActiveChat()
	status := statusInput
	if activeChat != nil {
		if activeChat.Streaming {
			status = statusStreaming
		} else if activeChat.ResponseReceived {
			status = statusReceived + " " + responseEmoji
		} else if activeChat.WaitingForResponse {
			status = statusWaiting
		}
	}
	// Left: model name, Right: status
	left := lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Width(30).Render(modelName)
	right := lipgloss.NewStyle().Align(lipgloss.Right).Width(30).Render(status)
	// The status bar should span the width of the chat window area
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// RenderSidebar renders the sidebar with section headings and emoji
func (c *CompositeChatViewState) RenderSidebar() string {
	var b strings.Builder
	pad := "  "
	b.WriteString(pad + activeChatEmoji + " Active Chats\n")
	b.WriteString(pad + strings.Repeat("-", 18) + "\n")
	for _, chat := range c.Sidebar.Tabs {
		line := pad + chat
		if c.ActiveChatID == chat {
			line = pad + activeChatEmoji + " " + chat
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + pad + allChatsEmoji + " All Chats\n")
	b.WriteString(pad + strings.Repeat("-", 18) + "\n")
	for _, chat := range c.Sidebar.ChatList {
		b.WriteString(pad + chat + "\n")
	}
	b.WriteString("\n" + pad + favoritesEmoji + " Favorites\n")
	b.WriteString(pad + strings.Repeat("-", 18) + "\n")
	for _, fav := range c.Sidebar.Favorites {
		b.WriteString(pad + fav + "\n")
	}
	b.WriteString("\n" + pad + "create new chat\n")
	return lipgloss.NewStyle().Padding(1, 1).Width(24).Render(b.String())
}

// RenderChatWindow renders the chat messages with colorized tags and margin/padding
func (c *CompositeChatViewState) RenderChatWindow() string {
	activeChat := c.GetActiveChat()
	if activeChat == nil {
		return ""
	}
	var msgs []string
	for _, msg := range activeChat.Messages {
		var tag string
		if msg.IsUser {
			tag = userTagStyle.Render("User")
		} else {
			tag = assistantTagStyle.Render("Assistant")
		}
		msgBox := msgBoxStyle.Render(msg.Content)
		msgs = append(msgs, lipgloss.JoinVertical(lipgloss.Left, tag, msgBox))
	}
	return lipgloss.NewStyle().Padding(2, 4).Render(strings.Join(msgs, "\n"))
}

// RenderMenuSubmenu renders a centered menu/submenu view with a title, menu entries, and control info text
func RenderMenuSubmenu(title string, entries []string, controlInfo string) string {
	// Title (centered)
	// titleStyle := lipgloss.NewStyle().Bold(true).Align(lipgloss.Center).MarginBottom(1)
	// titleView := titleStyle.Render(title) // Unused, remove

	// Menu box
	menuBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("245")).
		Padding(1, 4).
		Align(lipgloss.Left)
	menuEntries := ""
	for _, entry := range entries {
		menuEntries += lipgloss.NewStyle().Margin(0, 0, 0, 0).Render(entry) + "\n"
	}
	menuBox := menuBoxStyle.Render(menuEntries)

	// Submenu name (above menu box)
	submenuName := lipgloss.NewStyle().Align(lipgloss.Center).MarginBottom(1).Render(title)

	// Control info (centered below)
	controlInfoView := lipgloss.NewStyle().Align(lipgloss.Center).MarginTop(1).Render(controlInfo)

	// Compose layout (centered vertically and horizontally)
	menuArea := lipgloss.JoinVertical(lipgloss.Center, submenuName, menuBox, controlInfoView)
	centered := lipgloss.Place(60, 20, lipgloss.Center, lipgloss.Center, menuArea)
	// Add app title above
	appTitle := lipgloss.NewStyle().Bold(true).Align(lipgloss.Center).MarginBottom(2).Render("AICHAT")
	return lipgloss.JoinVertical(lipgloss.Center, appTitle, centered)
}

// RenderChatsMenu renders the Chats submenu with best-practice styling
func RenderChatsMenu(entries []string, controlInfo string) string {
	// Menu title
	title := "Chats"
	// Highlight the selected entry (for demonstration, highlight the first)
	styledEntries := make([]string, len(entries))
	for i, entry := range entries {
		entryStyle := lipgloss.NewStyle().Padding(0, 1)
		if i == 0 { // Example: highlight first entry
			entryStyle = entryStyle.Bold(true).Foreground(lipgloss.Color("203")).Background(lipgloss.Color("236"))
		}
		styledEntries[i] = entryStyle.Render(entry)
	}
	// Use the shared RenderMenuSubmenu for layout
	return RenderMenuSubmenu(title, styledEntries, controlInfo)
}

// View arranges the three panes using Lipgloss
func (c *CompositeChatViewState) View() string {
	// If any modal is active (check the modal stack), render only the modal
	if len(c.MenuStack) > 0 {
		currentModal := c.MenuStack[len(c.MenuStack)-1]

		// Handle different modal types
		switch modal := currentModal.(type) {
		case *MenuViewState:
			return modal.RenderMenuView(80, 25) // Use larger size for full-screen menu
		case *DynamicNoticeModal:
			return lipgloss.Place(80, 25, lipgloss.Center, lipgloss.Center, modal.View())
		case *InputPromptModal:
			return lipgloss.Place(80, 25, lipgloss.Center, lipgloss.Center, modal.View())
		case *dialogs.ListModal:
			return modal.ViewRegion(80, 25) // Use the modal's built-in centering
		default:
			// For any other modal type, try to render it centered
			return lipgloss.Place(80, 25, lipgloss.Center, lipgloss.Center, modal.View())
		}
	}

	// Otherwise render the full chat layout
	statusBar := c.RenderStatusBar()
	sidebarView := c.RenderSidebar()
	chatView := c.RenderChatWindow()
	inputView := ""
	if c.Input != nil {
		inputView = c.Input.View()
	}
	mainArea := lipgloss.JoinVertical(lipgloss.Left, statusBar, chatView, inputView)
	layout := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainArea)
	return lipgloss.NewStyle().Padding(1, 2).Render(layout)
}

// DynamicNoticeModal cycles through an array of notices at a set interval
// Used for animated 'Testing...' etc.
type DynamicNoticeModal struct {
	Notices       []string
	Current       int
	Interval      int // in seconds
	TickCount     int
	Message       string
	OnComplete    func(success bool)
	Testing       bool
	Success       bool
	Done          bool
	ResultMessage string
	ResultEmoji   string
}

// IsMainMenu returns false for DynamicNoticeModal
func (m *DynamicNoticeModal) IsMainMenu() bool { return false }

// MarshalState serializes the modal state
func (m *DynamicNoticeModal) MarshalState() ([]byte, error) { return nil, nil }

// UnmarshalState deserializes the modal state
func (m *DynamicNoticeModal) UnmarshalState(data []byte) error { return nil }

// ViewType returns ModalStateType
func (m *DynamicNoticeModal) ViewType() types.ViewType { return types.ModalStateType }

// GetControlSets returns the dynamic notice modal's control sets
func (m *DynamicNoticeModal) GetControlSets() []types.ControlSet {
	controls := []types.ControlSet{
		{
			Controls: []types.ControlType{
				{
					Name: "Esc", Key: "esc",
					Action: func() bool {
						// TODO: handle cancel
						return true
					},
				},
			},
		},
	}
	return controls
}

// Update cycles through notices every Interval seconds using tea.Tick
func (m *DynamicNoticeModal) Update(msg tea.Msg) (types.ViewState, tea.Cmd) {
	if m.Done {
		// Wait for Enter/Esc to confirm if unsuccessful
		if !m.Success {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				switch keyMsg.String() {
				case "enter", "esc":
					if m.OnComplete != nil {
						m.OnComplete(false)
					}
				}
			}
		}
		return m, nil
	}
	if !m.Testing {
		return m, nil
	}
	// Cycle through notices every Interval seconds
	m.Current = (m.Current + 1) % len(m.Notices)
	m.Message = m.Notices[m.Current]
	return m, nil
}

// View renders the current notice or result
func (m *DynamicNoticeModal) View() string {
	if m.Done {
		if m.Success {
			return lipgloss.NewStyle().Bold(true).Align(lipgloss.Center).Render("✅ " + m.ResultMessage)
		} else {
			return lipgloss.NewStyle().Bold(true).Align(lipgloss.Center).Render("❌ " + m.ResultMessage + "\n(Enter or Esc to try another key)")
		}
	}
	return lipgloss.NewStyle().Bold(true).Align(lipgloss.Center).Render(m.Message)
}

// InputPromptModal is a reusable modal for text input with instruction and control text
// Size: 1 for single-line, 3 for multi-line (with wrapping)
type InputPromptModal struct {
	Value           string
	InstructionText string // Text above the input box
	ControlText     string // Hint below the input box
	Size            int    // 1 or 3 lines
	Error           string // Optional error message
	OnSubmit        func(value string)
	OnCancel        func()
}

// IsMainMenu returns false for InputPromptModal
func (m *InputPromptModal) IsMainMenu() bool { return false }

// MarshalState serializes the modal state
func (m *InputPromptModal) MarshalState() ([]byte, error) { return nil, nil }

// UnmarshalState deserializes the modal state
func (m *InputPromptModal) UnmarshalState(data []byte) error { return nil }

// ViewType returns ModalStateType
func (m *InputPromptModal) ViewType() types.ViewType { return types.ModalStateType }

// GetControlSets returns the input prompt modal's control sets
func (m *InputPromptModal) GetControlSets() []types.ControlSet {
	controls := []types.ControlSet{
		{
			Controls: []types.ControlType{
				{
					Name: "Enter", Key: "enter",
					Action: func() bool {
						// TODO: handle submit
						return true
					},
				},
				{
					Name: "Esc", Key: "esc",
					Action: func() bool {
						// TODO: handle cancel
						return true
					},
				},
			},
		},
	}
	return controls
}

// wrapText is a simple helper for wrapping text to a given width
func wrapText(text string, width int) string {
	if len(text) <= width {
		return text
	}
	var out string
	for i := 0; i < len(text); i += width {
		end := i + width
		if end > len(text) {
			end = len(text)
		}
		out += text[i:end] + "\n"
	}
	return out
}

// View renders the prompt, input box, error (if any), and control hints
func (m *InputPromptModal) View() string {
	promptStyle := lipgloss.NewStyle().Align(lipgloss.Center).MarginBottom(1)
	inputBoxStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 2).Width(32)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Align(lipgloss.Center).MarginTop(1)
	controlHintStyle := lipgloss.NewStyle().Align(lipgloss.Center).MarginTop(1).Foreground(lipgloss.Color("244"))

	// Instruction text
	prompt := promptStyle.Render(m.InstructionText)
	// Input box (single or multi-line)
	input := m.Value
	if m.Size > 1 {
		input = lipgloss.NewStyle().Width(32).MaxWidth(32).Render(wrapText(m.Value, 32))
	}
	inputBox := inputBoxStyle.Render(input)
	// Error message (if any)
	errorMsg := ""
	if m.Error != "" {
		errorMsg = errorStyle.Render(m.Error)
	}
	// Control hint
	control := controlHintStyle.Render(m.ControlText)
	// Compose layout (centered)
	return lipgloss.Place(60, 12, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, prompt, inputBox, errorMsg, control),
	)
}

// Ensure InputPromptModal implements Update
func (m *InputPromptModal) Update(msg tea.Msg) (types.ViewState, tea.Cmd) {
	// TODO: Implement text input, Enter, Esc, and multi-line support
	return m, nil
}

// NewCompositeChatView creates a new composite chat view with the given chat as active
func NewCompositeChatView(chat *ChatViewState) *CompositeChatViewState {
	chats := make(map[string]*ChatViewState)
	chats[chat.ChatTitle] = chat // Use ChatTitle as key; replace with unique ID if available
	return &CompositeChatViewState{
		Sidebar:      nil, // Set as needed
		Chats:        chats,
		ActiveChatID: chat.ChatTitle, // Use ChatTitle as key
		Input:        nil,            // Set as needed
		Focus:        "chat",
		CurrentFocus: "chat",
		Workers:      make(map[string]*StreamWorker),
		Menu:         nil,
		MenuStack:    nil,
	}
}

// CreateMainMenu returns a new main menu MenuViewState
func CreateMainMenu() *MenuViewState {
	return &MenuViewState{
		MenuName:    "Main Menu",
		Entries:     []string{"Chats", "Prompts", "Models", "API Keys", "Help", "Exit"},
		Selected:    0,
		ControlInfo: "Up/Down: Navigate  Enter: Select  Esc: Back",
	}
}

// View returns a placeholder string for now
var defaultMenuView = "[MenuViewState placeholder view]"

func (m *MenuViewState) View() string { return defaultMenuView }

// GetControlSets returns the menu's control sets
func (m *MenuViewState) GetControlSets() []types.ControlSet {
	controls := []types.ControlSet{
		{
			Controls: []types.ControlType{
				{
					Name: "Up", Key: "up",
					Action: func() bool {
						if m.Selected > 0 {
							m.Selected--
						} else {
							m.Selected = len(m.Entries) - 1
						}
						return true
					},
				},
				{
					Name: "Down", Key: "down",
					Action: func() bool {
						if m.Selected < len(m.Entries)-1 {
							m.Selected++
						} else {
							m.Selected = 0
						}
						return true
					},
				},
				{
					Name: "Enter", Key: "enter",
					Action: func() bool {
						// TODO: handle menu selection
						return true
					},
				},
				{
					Name: "Esc", Key: "esc",
					Action: func() bool {
						// TODO: handle back/cancel
						return true
					},
				},
			},
		},
	}
	return controls
}

func (c *CompositeChatViewState) startCustomChatFlow() {
	// Step 1: Prompt for chat name
	inputModal := &InputPromptModal{
		InstructionText: "Enter name for custom chat:",
		ControlText:     "Enter to confirm, Esc to cancel, leave blank for timestamp",
		Size:            1,
		OnSubmit: func(name string) {
			if name == "" {
				name = time.Now().Format("20060102_150405")
			}
			// Step 2: Prompt selection
			prompts, err := loadPrompts() // []models.Prompt
			if err != nil || len(prompts) == 0 {
				// Show error modal
				notice := &DynamicNoticeModal{
					Notices:  []string{"No prompts available."},
					Current:  0,
					Interval: 1,
				}
				c.PushModal(notice)
				return
			}
			promptNames := make([]string, len(prompts))
			for i, p := range prompts {
				promptNames[i] = p.Name
			}
			promptModal := &dialogs.ListModal{
				Title:           "Select Prompt",
				Options:         promptNames,
				InstructionText: "Select prompt:",
				ControlText:     "Up/Down: Navigate, Enter: Select, Esc: Cancel",
				OnSelect: func(promptIdx int) {
					// Step 3: Model selection
					modelsList, _, err := loadModelsWithMostRecent()
					if err != nil || len(modelsList) == 0 {
						notice := &DynamicNoticeModal{
							Notices:  []string{"No models available."},
							Current:  0,
							Interval: 1,
						}
						c.PushModal(notice)
						return
					}
					modelModal := &dialogs.ListModal{
						Title:           "Select Model",
						Options:         modelsList,
						InstructionText: "Select model:",
						ControlText:     "Up/Down: Navigate, Enter: Select, Esc: Cancel",
						OnSelect: func(modelIdx int) {
							// Step 4: API key selection
							keyRepo := repositories.NewAPIKeyRepository()
							apiKeys, err := keyRepo.GetAll()
							if err != nil || len(apiKeys) == 0 {
								notice := &DynamicNoticeModal{
									Notices:  []string{"No API keys available."},
									Current:  0,
									Interval: 1,
								}
								c.PushModal(notice)
								return
							}
							keyTitles := make([]string, len(apiKeys))
							for i, k := range apiKeys {
								keyTitles[i] = k.Title
							}
							keyModal := &dialogs.ListModal{
								Title:           "Select API Key",
								Options:         keyTitles,
								InstructionText: "Select API key:",
								ControlText:     "Up/Down: Navigate, Enter: Select, Esc: Cancel",
								OnSelect: func(keyIdx int) {
									// Create and persist chat
									chatFile := models.ChatFile{
										Metadata: models.ChatMetadata{
											Title:     name,
											Model:     modelsList[modelIdx],
											CreatedAt: time.Now(),
										},
										Messages: []models.Message{},
									}
									repo := repositories.NewChatRepository()
									_ = repo.Add(chatFile) // TODO: handle error
									// Open chat view (implement as needed)
									c.testAPIKeyFlow(prompts[promptIdx].Content, modelsList[modelIdx], apiKeys[keyIdx], func() {
										// API key test successful, proceed to chat view
										c.PopModal() // Close the API key modal
										// Create and push new composite chat view
										newChat := &ChatViewState{ChatTitle: name}
										c.Chats[name] = newChat
										c.ActiveChatID = name
										// TODO: Actually push composite to app navigation stack if needed
									}, func() {
										// API key test failed, allow user to retry
										c.PopModal()            // Close the API key modal
										c.startCustomChatFlow() // Restart the flow from the beginning
									})
								},
								CloseSelf: func() { c.PopModal() },
							}
							c.PushModal(keyModal)
						},
						CloseSelf: func() { c.PopModal() },
					}
					c.PushModal(modelModal)
				},
				CloseSelf: func() { c.PopModal() },
			}
			c.PushModal(promptModal)
		},
		OnCancel: func() { c.PopModal() },
	}
	c.PushModal(inputModal)
}

func (c *CompositeChatViewState) testAPIKeyFlow(prompt, model string, apiKey models.APIKey, onSuccess func(), onFailure func()) {
	notices := []string{"Testing.", "Testing. .", "Testing. . ."}
	dynModal := &DynamicNoticeModal{
		Notices:       notices,
		Current:       0,
		Interval:      1,
		Testing:       true,
		Message:       notices[0],
		ResultMessage: "",
		ResultEmoji:   "",
	}
	c.PushModal(dynModal)

	// Start API key test in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resultCh := make(chan bool, 1)
		go func() {
			// Prepare request body
			reqBody := struct {
				Model       string           `json:"model"`
				Messages    []models.Message `json:"messages"`
				Stream      bool             `json:"stream"`
				MaxTokens   int              `json:"max_tokens,omitempty"`
				Temperature float64          `json:"temperature,omitempty"`
			}{
				Model:       model,
				Messages:    []models.Message{{Role: "system", Content: prompt}},
				Stream:      true,
				MaxTokens:   16,
				Temperature: 0.7,
			}
			bodyBytes, err := json.Marshal(reqBody)
			if err != nil {
				resultCh <- false
				return
			}
			req, err := http.NewRequestWithContext(ctx, "POST", apiKey.URL, bytes.NewBuffer(bodyBytes))
			if err != nil {
				resultCh <- false
				return
			}
			req.Header.Set("Authorization", "Bearer "+apiKey.Key)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("HTTP-Referer", "https://github.com/go-ai-cli")
			req.Header.Set("X-Title", "Go AI CLI")
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				resultCh <- false
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				resultCh <- false
				return
			}
			// Wait for first response chunk
			buf := make([]byte, 256)
			_, err = resp.Body.Read(buf)
			if err != nil && err != io.EOF {
				resultCh <- false
				return
			}
			resultCh <- true
		}()
		var success bool
		select {
		case success = <-resultCh:
		case <-ctx.Done():
			success = false
		}
		// Update modal on main thread
		if success {
			dynModal.Testing = false
			dynModal.Done = true
			dynModal.Success = true
			dynModal.ResultMessage = "API key test successful!"
			dynModal.ResultEmoji = "✅"
			// Proceed after 1s
			time.Sleep(1 * time.Second)
			if onSuccess != nil {
				onSuccess()
			}
		} else {
			dynModal.Testing = false
			dynModal.Done = true
			dynModal.Success = false
			dynModal.ResultMessage = "unsuccessful, please try another key"
			dynModal.ResultEmoji = "❌"
			// Wait for Enter/Esc to confirm, then call onFailure
			dynModal.OnComplete = func(_ bool) {
				if onFailure != nil {
					onFailure()
				}
			}
		}
	}()
}

// Add this helper at the top-level of the file if not already present
func loadModelsWithMostRecent() ([]string, string, error) {
	path := filepath.Join("src", ".config", "models.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	type Model struct {
		Name      string `json:"name"`
		IsDefault bool   `json:"is_default"`
	}
	type ModelsConfig struct {
		Models []Model `json:"models"`
	}
	var config ModelsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, "", err
	}
	var models []string
	var defaultModel string
	for _, m := range config.Models {
		models = append(models, m.Name)
		if m.IsDefault {
			defaultModel = m.Name
		}
	}
	if len(models) == 0 {
		return nil, "", nil
	}
	if defaultModel == "" {
		defaultModel = models[0]
	}
	return models, defaultModel, nil
}

// getGlobalLoadModelsWithMostRecent returns the global function if available
func getGlobalLoadModelsWithMostRecent() func() ([]string, string, error) {
	return nil // Replace with actual lookup if needed
}

// Replace prompts.LoadPrompts with loadPrompts (local helper)
func loadPrompts() ([]models.Prompt, error) {
	path := filepath.Join("src", ".config", "prompts.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var prompts []models.Prompt
	if err := json.Unmarshal(data, &prompts); err == nil {
		return prompts, nil
	}
	// Try PromptsConfig struct if direct array fails
	type PromptsConfig struct {
		Prompts []models.Prompt `json:"prompts"`
	}
	var config PromptsConfig
	if err := json.Unmarshal(data, &config); err == nil {
		return config.Prompts, nil
	}
	return nil, err
}
