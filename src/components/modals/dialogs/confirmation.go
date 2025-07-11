// confirmation.go - Contains the ConfirmationModal for displaying confirmation dialogs with 1-3 options in the Bubble Tea UI.
// Update logic supports left/right navigation, enter to select, esc to close/cancel.

package dialogs

import (
	"aichat/src/components/modals"
	"aichat/src/types"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmationModal is a reusable modal for confirmation dialogs (1-3 options).
type ConfirmationModal struct {
	modals.BaseModal
	RegionWidth  int
	RegionHeight int
	focused      bool
}

// NewConfirmationModal creates a new ConfirmationModal with the given message, options, and closeSelf callback.
func NewConfirmationModal(message string, options []modals.ModalOption, closeSelf modals.CloseSelfFunc) *ConfirmationModal {
	if len(options) < 1 || len(options) > 3 {
		panic("ConfirmationModal must have 1-3 options")
	}
	return &ConfirmationModal{
		BaseModal: modals.BaseModal{
			Message:   message,
			Options:   options,
			CloseSelf: closeSelf,
			Selected:  0,
		},
		RegionWidth:  60,
		RegionHeight: 10,
		focused:      true,
	}
}

func (m *ConfirmationModal) Init() tea.Cmd { return nil }

func (m *ConfirmationModal) Update(msg tea.Msg) (types.ViewState, tea.Cmd) {
	if !m.focused {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			if m.Selected > 0 {
				m.Selected--
			} else {
				m.Selected = len(m.Options) - 1
			}
		case "right":
			if m.Selected < len(m.Options)-1 {
				m.Selected++
			} else {
				m.Selected = 0
			}
		case "enter":
			if m.Selected >= 0 && m.Selected < len(m.Options) {
				m.Options[m.Selected].OnSelect()
				if m.CloseSelf != nil {
					m.CloseSelf()
				}
			}
		case "esc":
			if m.CloseSelf != nil {
				m.CloseSelf()
			}
		}
	}
	return m, nil
}

func (m *ConfirmationModal) View() string {
	return m.ViewRegion(m.RegionWidth, m.RegionHeight)
}

func (m *ConfirmationModal) ViewRegion(regionWidth, regionHeight int) string {
	// Style for the modal box (match menu style)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("245")).
		Padding(1, 4).
		Align(lipgloss.Center)

	msg := lipgloss.NewStyle().Bold(true).Render(m.Message)
	var opts string
	for i, opt := range m.Options {
		style := lipgloss.NewStyle().Padding(0, 2)
		if i == m.Selected {
			style = style.Bold(true).Foreground(lipgloss.Color("33")).Background(lipgloss.Color("236"))
		}
		opts += style.Render(opt.Label)
	}
	content := msg + "\n\n" + opts
	box := boxStyle.Render(content)
	return lipgloss.Place(regionWidth, regionHeight, lipgloss.Center, lipgloss.Center, box)
}

// --- ViewState interface ---
func (m *ConfirmationModal) GetControlSets() []types.ControlSet {
	return []types.ControlSet{
		{
			Controls: []types.ControlType{
				{Name: "Left", Key: "left", Action: func() bool {
					m.BaseModal.Selected = (m.BaseModal.Selected + len(m.BaseModal.Options) - 1) % len(m.BaseModal.Options)
					return true
				}},
				{Name: "Right", Key: "right", Action: func() bool { m.BaseModal.Selected = (m.BaseModal.Selected + 1) % len(m.BaseModal.Options); return true }},
				{Name: "Enter", Key: "enter", Action: func() bool {
					if m.BaseModal.Selected >= 0 && m.BaseModal.Selected < len(m.BaseModal.Options) {
						m.BaseModal.Options[m.BaseModal.Selected].OnSelect()
						if m.BaseModal.CloseSelf != nil {
							m.BaseModal.CloseSelf()
						}
						return true
					}
					return false
				}},
				{Name: "Esc", Key: "esc", Action: func() bool {
					if m.BaseModal.CloseSelf != nil {
						m.BaseModal.CloseSelf()
						return true
					}
					return false
				}},
			},
		},
	}
}
func (m *ConfirmationModal) IsMainMenu() bool                 { return false }
func (m *ConfirmationModal) MarshalState() ([]byte, error)    { return nil, nil }
func (m *ConfirmationModal) UnmarshalState(data []byte) error { return nil }
func (m *ConfirmationModal) ViewType() types.ViewType         { return types.ModalStateType }
