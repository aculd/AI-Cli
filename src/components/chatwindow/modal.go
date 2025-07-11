package chatwindow

import (
	"aichat/src/types"
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
)

// ModalViewState represents a modal/dialog that covers the chat window.
type ModalViewState struct {
	PrevChatState *ChatWindowViewState
	ModalContent  string // or a struct for more complex modals
	OnYes         func() tea.Msg
	OnNo          func() tea.Msg
}

func (m *ModalViewState) Update(msg tea.Msg) (types.ViewState, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "y", "Y", "enter":
			if m.OnYes != nil {
				return m, func() tea.Msg { return m.OnYes() }
			}
		case "n", "N", "esc":
			if m.OnNo != nil {
				return m, func() tea.Msg { return m.OnNo() }
			}
		}
	}
	return m, nil
}

func (m *ModalViewState) View() string {
	// Render the modal content, hiding chat messages
	return "[MODAL: " + m.ModalContent + "]"
}

func (m *ModalViewState) MarshalState() ([]byte, error) {
	return json.Marshal(m)
}

func (m *ModalViewState) UnmarshalState(data []byte) error {
	return json.Unmarshal(data, m)
}

func (m *ModalViewState) ViewType() types.ViewType { return types.ModalStateType }
func (m *ModalViewState) IsMainMenu() bool         { return false }

// GetControlSets returns the modal view's control sets
func (m *ModalViewState) GetControlSets() []types.ControlSet {
	controls := []types.ControlSet{
		{
			Controls: []types.ControlType{
				{
					Name: "Yes", Key: "y",
					Action: func() bool {
						if m.OnYes != nil {
							// TODO: trigger OnYes
						}
						return true
					},
				},
				{
					Name: "No", Key: "n",
					Action: func() bool {
						if m.OnNo != nil {
							// TODO: trigger OnNo
						}
						return true
					},
				},
				{
					Name: "Esc", Key: "esc",
					Action: func() bool {
						if m.OnNo != nil {
							// TODO: trigger OnNo
						}
						return true
					},
				},
			},
		},
	}
	return controls
}
