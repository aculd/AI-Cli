// help.go - Contains HelpModal for displaying help or info content in a modal dialog in the Bubble Tea UI.

package dialogs

import "github.com/charmbracelet/lipgloss"

// HelpModal is a reusable modal for displaying help or info content.
type HelpModal struct {
	Content      string // The help/info text to display
	CloseSelf    func() // Callback to restore previous state
	RegionWidth  int    // Last-known or intended region width for rendering
	RegionHeight int    // Last-known or intended region height for rendering
}

// Init initializes the modal (Bubble Tea compatibility).
func (m *HelpModal) Init() {}

// Update handles Bubble Tea messages for the modal.
func (m *HelpModal) Update(msg interface{}) {}

// View renders the help/info modal UI as a string, centered in the stored region (RegionWidth, RegionHeight).
func (m *HelpModal) View() string {
	return m.ViewRegion(m.RegionWidth, m.RegionHeight)
}

// ViewRegion renders the help/info modal UI as a string, centered in the given region (width, height).
func (m *HelpModal) ViewRegion(regionWidth, regionHeight int) string {
	content := lipgloss.NewStyle().Padding(1, 2).Render(m.Content)
	return lipgloss.Place(regionWidth, regionHeight, lipgloss.Center, lipgloss.Center, content)
}
