package app

import (
	"aichat/src/components/chatwindow"
	"aichat/src/navigation"
	"aichat/src/services/storage"
	"aichat/src/types"

	tea "github.com/charmbracelet/bubbletea"
)

type CloseCurrentChatMsg struct{}
type CloseAllChatsMsg struct{}
type QuitAppMsg struct{}

// AppModel is the root Bubble Tea model for the app.
type AppModel struct {
	Stack    *navigation.NavigationStack
	ChatRepo storage.ChatRepository
	Sidebar  types.ViewState // placeholder for sidebar state
}

func NewAppModel(chatRepo storage.ChatRepository, sidebar types.ViewState, initialChatID string) *AppModel {
	var initialView types.ViewState
	if initialChatID != "" {
		chatFile, _ := chatRepo.GetByID(initialChatID)
		if chatFile != nil {
			initialView = &chatwindow.ChatWindowViewState{
				ChatID:   initialChatID,
				Messages: chatFile.Messages,
				Metadata: chatFile.Metadata,
			}
		}
	}
	if initialView == nil {
		initialView = types.NewMenuViewState(types.MainMenu) // Always start with main menu
	}
	stack := navigation.NewNavigationStack(initialView)
	return &AppModel{
		Stack:    stack,
		ChatRepo: chatRepo,
		Sidebar:  sidebar,
	}
}

func (m *AppModel) Init() tea.Cmd {
	return nil
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch nav := msg.(type) {
	case navigation.NavigationMsg:
		switch nav.Action {
		case navigation.PushAction:
			m.Stack.Push(nav.Target)
		case navigation.PopAction:
			m.Stack.Pop()
		case navigation.ResetAction:
			m.Stack.ResetToMainMenu(types.NewMenuViewState(types.MainMenu)) // Always reset to main menu
		}
		return m, nil
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+w":
			modal := &chatwindow.ModalViewState{
				ModalContent: "Are you sure you want to close this chat? (Yes/No)",
				OnYes:        func() tea.Msg { return CloseCurrentChatMsg{} },
				OnNo:         func() tea.Msg { return navigation.PopAction },
			}
			m.Stack.Push(modal)
			return m, nil
		case "ctrl+q":
			modal := &chatwindow.ModalViewState{
				ModalContent: "Are you sure you want to quit? (Yes/No)",
				OnYes:        func() tea.Msg { return QuitAppMsg{} },
				OnNo:         func() tea.Msg { return navigation.PopAction },
			}
			m.Stack.Push(modal)
			return m, nil
		case "f1":
			modal := &chatwindow.ModalViewState{
				ModalContent: "[Help]\n\nCtrl+W: Close chat\nCtrl+Q: Close all chats\nCtrl+N: New chat\nF1: Help\nUp/Down: Navigate\nEnter: Select\nEsc: Cancel/Back\nCtrl+I: Focus input area",
			}
			m.Stack.Push(modal)
			return m, nil
		}
	}
	// Handle custom modal results
	switch msg.(type) {
	case CloseCurrentChatMsg:
		if m.Stack.Len() > 1 {
			m.Stack.Pop()
			// Switch to most recently modified chat (if any left)
			// Find the most recently modified chat in the stack
			var mostRecentIdx int
			var mostRecentTime int64
			for i := 0; i < m.Stack.Len(); i++ {
				if chatState, ok := m.Stack.At(i).(*chatwindow.ChatWindowViewState); ok {
					if chatState.Metadata.ModifiedAt > mostRecentTime {
						mostRecentTime = chatState.Metadata.ModifiedAt
						mostRecentIdx = i
					}
				}
			}
			// Move the most recent chat to the top
			if mostRecentIdx != m.Stack.Len()-1 {
				m.Stack.MoveToTop(mostRecentIdx)
			}
		} else {
			// Last chat, reset to main menu
			m.Stack.ResetToMainMenu(types.NewMenuViewState(types.MainMenu))
		}
		return m, nil
	case CloseAllChatsMsg:
		m.Stack.ResetToMainMenu(types.NewMenuViewState(types.MainMenu))
		return m, nil
	case QuitAppMsg:
		return m, tea.Quit
	}
	// Example: handle "Create New Chat" modal
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "ctrl+n" {
		// Push a modal view state
		current := m.Stack.Top()
		if chatState, ok := current.(*chatwindow.ChatWindowViewState); ok {
			modal := &chatwindow.ModalViewState{
				PrevChatState: chatState,
				ModalContent:  "Create New Chat Flow",
			}
			m.Stack.Push(modal)
			return m, nil
		}
	}
	// Delegate to top view
	top := m.Stack.Top()
	newState, cmd := top.Update(msg)
	m.Stack.ReplaceTop(newState)
	return m, cmd
}

func (m *AppModel) View() string {
	// If the top of the stack is the main menu, render only the main menu
	top := m.Stack.Top()
	if menu, ok := top.(*types.MenuViewState); ok && menu.IsMainMenu() {
		return menu.View()
	}
	// Otherwise, render sidebar, then chat/modal area (Sidebar rendering omitted for brevity)
	return top.View()
}
