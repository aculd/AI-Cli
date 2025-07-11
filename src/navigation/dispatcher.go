package navigation

import (
	tea "github.com/charmbracelet/bubbletea"
)

// AppModel is the root Bubble Tea model with navigation stack.
type AppModel struct {
	Stack *NavigationStack
	// ... other fields ...
}

// Init implements tea.Model's Init method.
func (m *AppModel) Init() tea.Cmd {
	return nil
}

// Update handles navigation messages and delegates to the top view.
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch nav := msg.(type) {
	case NavigationMsg:
		switch nav.Action {
		case PushAction:
			m.Stack.Push(nav.Target)
		case PopAction:
			m.Stack.Pop()
		case ResetAction:
			// TODO: Reset to main menu
		}
		return m, nil
	}
	// Delegate to top view
	top := m.Stack.Top()
	newState, cmd := top.Update(msg)
	m.Stack.ReplaceTop(newState)
	return m, cmd
}

// View renders the current top view.
func (m *AppModel) View() string {
	return m.Stack.Top().View()
}
