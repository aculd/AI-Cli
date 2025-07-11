// base.go - Contains the BaseModal struct and ModalOption/CloseSelf types for reusable modal dialogs in the Bubble Tea UI.

package modals

import "github.com/charmbracelet/lipgloss"

// ModalOption represents a selectable option in a modal dialog.
type ModalOption struct {
	Label    string
	OnSelect func()
}

// CloseSelfFunc is a callback to restore previous app state when a modal is closed.
type CloseSelfFunc func()

// BaseModal is the base struct for all modal dialogs, containing message, options, close logic, and selection state.
type BaseModal struct {
	Message   string
	Options   []ModalOption
	CloseSelf CloseSelfFunc
	Selected  int
}

// Init initializes the modal (Bubble Tea compatibility).
func (m *BaseModal) Init() {}

// Update handles Bubble Tea messages for the modal.
func (m *BaseModal) Update(msg interface{}) {}

// View renders the modal UI as a string, centered in the given region (width, height).
// All modals should use this pattern for region-aware rendering.
func (m *BaseModal) View(regionWidth, regionHeight int) string {
	content := lipgloss.NewStyle().Bold(true).Render(m.Message)
	return lipgloss.Place(regionWidth, regionHeight, lipgloss.Center, lipgloss.Center, content)
}
