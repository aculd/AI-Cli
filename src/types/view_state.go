// types/view_state.go - Unified ViewState implementations for navigation stack
// MIGRATION TARGET: legacy/gui.go (all menu, chat, modal state logic)

package types

import (
	"encoding/json"
	"strings"

	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ViewState now supports multiple control sets for modular/global controls
// ControlSets: first is local, others can be global or context-specific

type ViewState interface {
	ViewType() ViewType
	View() string
	Update(msg tea.Msg) (ViewState, tea.Cmd)
	GetControlSets() []ControlSet // new method for all control sets
	IsMainMenu() bool
	MarshalState() ([]byte, error)
	UnmarshalState([]byte) error
}

// MenuViewState supports multiple control sets
// MIGRATION TARGET: legacy/gui.go (menu state logic)
type MenuViewState struct {
	Type     MenuType
	Selected int
	Parent   ViewState    // Parent can be any ViewState (menu, chat, modal, etc.)
	Controls []ControlSet // now a slice
}

// GetControlSets returns all control sets for this view
func (m *MenuViewState) GetControlSets() []ControlSet {
	return m.Controls
}

// NewMenuViewState creates a menu view state with optional extra control sets
// pushView: function to push any ViewState (menu, modal, etc.) onto the stack
// parent: parent ViewState for back navigation
// exitFunc: called when at root and Esc is pressed (e.g., showQuitConfirmation)
func NewMenuViewState(menuType MenuType, pushView func(ViewState), parent ViewState, exitFunc func(), extra ...ControlSet) *MenuViewState {
	m := &MenuViewState{
		Type:     menuType,
		Selected: 0,
		Parent:   parent,
		Controls: []ControlSet{},
	}
	local := ControlSet{
		Controls: []ControlType{
			{Name: "Up", Key: "up", Action: MenuUp(m)},
			{Name: "Down", Key: "down", Action: MenuDown(m)},
			{Name: "Enter", Key: "enter", Action: MenuEnter(m, pushView)},
			{Name: "Esc", Key: "esc", Action: MenuEsc(m, exitFunc)},
		},
	}
	m.Controls = append(m.Controls, local)
	m.Controls = append(m.Controls, extra...)
	return m
}
func (m *MenuViewState) ViewType() ViewType { return MenuStateType }
func (m *MenuViewState) IsMainMenu() bool   { return m.Type == MainMenu }
func (m *MenuViewState) MarshalState() ([]byte, error) {
	return json.Marshal(struct {
		Type     MenuType `json:"type"`
		Selected int      `json:"selected"`
	}{m.Type, m.Selected})
}
func (m *MenuViewState) UnmarshalState(data []byte) error {
	var s struct {
		Type     MenuType `json:"type"`
		Selected int      `json:"selected"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	m.Type = s.Type
	m.Selected = s.Selected
	return nil
}
func (m *MenuViewState) Update(msg tea.Msg) (ViewState, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		for _, ctrl := range m.Controls {
			for _, ctrlType := range ctrl.Controls {
				if keyMsg.String() == ctrlType.Key && ctrlType.Action != nil {
					if ctrlType.Action() {
						return m, nil
					}
				}
			}
		}
		// Example: Up/Down/Enter navigation
		switch keyMsg.String() {
		case "up":
			if m.Selected > 0 {
				m.Selected--
			}
			return m, nil
		case "down":
			if m.Selected < len(Menus[m.Type].Entries)-1 {
				m.Selected++
			}
			return m, nil
		case "enter":
			// TODO: handle menu selection
			return m, nil
		case "esc":
			// TODO: handle back/cancel
			return m, nil
		}
	}
	return m, nil
}
func (m *MenuViewState) View() string {
	asciiArt := `
  █████╗ ██╗ ██████╗██╗  ██╗ █████╗ ████████╗
 ██╔══██╗██║██╔════╝██║  ██║██╔══██╗╚══██╔══╝
 ███████║██║██║     ███████║███████║   ██║   
 ██╔══██║██║██║     ██╔══██║██╔══██║   ██║   
 ██║  ██║██║╚██████╗██║  ██║██║  ██║   ██║   
 ╚═╝  ╚═╝╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝
`
	entries := Menus[m.Type].Entries
	var menuItems strings.Builder
	menuTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(menuTypeToString(m.Type))
	menuItems.WriteString(menuTitle + "\n\n")
	for i, entry := range entries {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		if entry.Disabled {
			style = style.Foreground(lipgloss.Color("240")).Italic(true)
		}
		if i == m.Selected {
			style = style.Bold(true).Foreground(lipgloss.Color("203")).Background(lipgloss.Color("236"))
			menuItems.WriteString(" > ")
		} else {
			menuItems.WriteString("   ")
		}
		menuItems.WriteString(style.Render(entry.Text))
		if entry.Description != "" {
			menuItems.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("  " + entry.Description))
		}
		menuItems.WriteString("\n")
	}
	// Menu box
	menuBox := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("245")).Padding(1, 4).Align(lipgloss.Left).Render(menuItems.String())
	// Control hints
	meta := MenuMetas[m.Type]
	controls := ControlInfoMap[meta.ControlInfoType]
	var controlInfo strings.Builder
	for _, line := range controls.Lines {
		controlInfo.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(line) + "\n")
	}
	width := getTerminalWidth()
	// Compose layout
	layout := lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.PlaceHorizontal(width, lipgloss.Center, lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(asciiArt)),
		lipgloss.PlaceHorizontal(width, lipgloss.Center, menuBox),
		lipgloss.PlaceHorizontal(width, lipgloss.Center, controlInfo.String()),
	)
	return layout
}

// ChatViewState implements ViewState for chat window
// MIGRATION TARGET: legacy/gui.go (chat state logic)
type ChatViewState struct {
	ChatTitle string
	Messages  []string // TODO: Replace with message structs
	Streaming bool
}

func (c *ChatViewState) ViewType() ViewType { return ChatStateType }
func (c *ChatViewState) IsMainMenu() bool   { return false }

// GetControlSets returns the chat view's control sets
func (c *ChatViewState) GetControlSets() []ControlSet {
	// Basic chat controls
	controls := []ControlSet{
		{
			Controls: []ControlType{
				{
					Name: "Esc", Key: "esc",
					Action: func() bool { /* TODO: handle back/cancel */ return true },
				},
			},
		},
	}
	return controls
}
func (c *ChatViewState) MarshalState() ([]byte, error) {
	return json.Marshal(struct {
		ChatTitle string   `json:"chat_title"`
		Messages  []string `json:"messages"`
		Streaming bool     `json:"streaming"`
	}{c.ChatTitle, c.Messages, c.Streaming})
}
func (c *ChatViewState) UnmarshalState(data []byte) error {
	var s struct {
		ChatTitle string   `json:"chat_title"`
		Messages  []string `json:"messages"`
		Streaming bool     `json:"streaming"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	c.ChatTitle = s.ChatTitle
	c.Messages = s.Messages
	c.Streaming = s.Streaming
	return nil
}
func (c *ChatViewState) Update(msg tea.Msg) (ViewState, tea.Cmd) {
	// TODO: Implement chat update logic
	return c, nil
}
func (c *ChatViewState) View() string {
	return "[Chat: " + c.ChatTitle + "]"
}

// ModalViewState implements ViewState for modals
// MIGRATION TARGET: legacy/gui.go (modal state logic)
type ModalViewState struct {
	ModalType string
	Content   string
}

func (m *ModalViewState) ViewType() ViewType { return ModalStateType }
func (m *ModalViewState) IsMainMenu() bool   { return false }

// GetControlSets returns the modal view's control sets
func (m *ModalViewState) GetControlSets() []ControlSet {
	// Basic modal controls
	controls := []ControlSet{
		{
			Controls: []ControlType{
				{
					Name: "Enter", Key: "enter",
					Action: func() bool { /* TODO: handle modal confirmation */ return true },
				},
				{
					Name: "Esc", Key: "esc",
					Action: func() bool { /* TODO: handle modal cancel */ return true },
				},
			},
		},
	}
	return controls
}
func (m *ModalViewState) MarshalState() ([]byte, error) {
	return json.Marshal(struct {
		ModalType string `json:"modal_type"`
		Content   string `json:"content"`
	}{m.ModalType, m.Content})
}
func (m *ModalViewState) UnmarshalState(data []byte) error {
	var s struct {
		ModalType string `json:"modal_type"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	m.ModalType = s.ModalType
	m.Content = s.Content
	return nil
}
func (m *ModalViewState) Update(msg tea.Msg) (ViewState, tea.Cmd) {
	// TODO: Implement modal update logic
	return m, nil
}
func (m *ModalViewState) View() string {
	return "[Modal: " + m.ModalType + "] " + m.Content
}

// QuitAppMsg is sent when the user confirms quitting the app
// Used by quit confirmation modal

type QuitAppMsg struct{}

// menuTypeToString returns a human-readable menu name (local helper)
func menuTypeToString(mt MenuType) string {
	switch mt {
	case MainMenu:
		return "Main Menu"
	case ChatsMenu:
		return "Chats"
	case PromptsMenu:
		return "Prompts"
	case ModelsMenu:
		return "Models"
	case APIKeyMenu:
		return "API Keys"
	case HelpMenu:
		return "Help"
	case ExitMenu:
		return "Exit"
	default:
		return "Menu"
	}
}

// Helper functions for bounds
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getTerminalWidth() int {
	if w := os.Getenv("COLUMNS"); w != "" {
		return 80 // TODO: parse w as int
	}
	return 80
}

// Generic navigation functions for menu up/down
func MenuUp(m *MenuViewState) func() bool {
	return func() bool {
		entries := Menus[m.Type].Entries
		if m.Selected > 0 {
			m.Selected--
		} else {
			m.Selected = len(entries) - 1 // wrap around
		}
		return true
	}
}

func MenuDown(m *MenuViewState) func() bool {
	return func() bool {
		entries := Menus[m.Type].Entries
		if m.Selected < len(entries)-1 {
			m.Selected++
		} else {
			m.Selected = 0 // wrap around
		}
		return true
	}
}

// MenuEnter pushes a new menu or executes an action
func MenuEnter(m *MenuViewState, pushView func(ViewState)) func() bool {
	return func() bool {
		entries := Menus[m.Type].Entries
		if m.Selected < 0 || m.Selected >= len(entries) {
			return false
		}
		entry := entries[m.Selected]
		if entry.Next != 0 && pushView != nil {
			newMenu := NewMenuViewState(entry.Next, pushView, m, nil)
			pushView(newMenu)
			return true
		}
		if entry.Action != nil {
			return entry.Action()
		}
		return false
	}
}

// MenuEsc returns a function that navigates to the parent view if present, or calls exitFunc (showQuitConfirmation) if at the root.
func MenuEsc(m *MenuViewState, exitFunc func()) func() bool {
	return func() bool {
		if m.Parent != nil {
			// Navigation logic: set current view to parent (handled by stack/dispatcher)
			return true
		}
		if exitFunc != nil {
			exitFunc() // Should call showQuitConfirmation
			return true
		}
		return false
	}
}
