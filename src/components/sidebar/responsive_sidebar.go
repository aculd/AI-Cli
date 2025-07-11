// Package sidebar provides sidebar components for the AI CLI application.
package sidebar

import (
	"fmt"
	"strings"

	"aichat/src/components/common"
	"aichat/src/models"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =====================================================================================
// 🎨 Responsive Sidebar – Dynamic Layout Adaptation
// =====================================================================================
// This component provides a responsive sidebar that:
//  - Adapts to terminal size changes in real-time
//  - Collapses gracefully on small terminals
//  - Maintains optimal proportions
//  - Provides smooth transitions during resize

// ResponsiveSidebar represents a responsive sidebar
type ResponsiveSidebar struct {
	layout        *common.ResponsiveLayout
	chats         []models.Chat
	width         int
	height        int
	selectedIndex int
	scrollOffset  int
	style         lipgloss.Style
	headerStyle   lipgloss.Style
	itemStyle     lipgloss.Style
	selectedStyle lipgloss.Style
	statusStyle   lipgloss.Style
	collapsed     bool
}

// NewResponsiveSidebar creates a new responsive sidebar
func NewResponsiveSidebar() *ResponsiveSidebar {
	layout := common.NewResponsiveLayout(common.DefaultLayoutConfig())

	return &ResponsiveSidebar{
		layout:        layout,
		chats:         []models.Chat{},
		width:         80,
		height:        24,
		selectedIndex: 0,
		scrollOffset:  0,
		style:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")),
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Padding(0, 1),
		itemStyle:     lipgloss.NewStyle().Padding(0, 1).Margin(0, 0, 1, 0),
		selectedStyle: lipgloss.NewStyle().Background(lipgloss.Color("240")).Foreground(lipgloss.Color("15")).Padding(0, 1).Margin(0, 0, 1, 0),
		statusStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 1),
		collapsed:     false,
	}
}

// Init initializes the sidebar
func (sb *ResponsiveSidebar) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the sidebar
func (sb *ResponsiveSidebar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.ResizeMsg:
		sb.OnResize(msg.Width, msg.Height)
		return sb, nil
	case tea.KeyMsg:
		return sb.handleKeyPress(msg)
	}
	return sb, nil
}

// OnResize handles terminal resize events
func (sb *ResponsiveSidebar) OnResize(width, height int) {
	sb.width = width
	sb.height = height
	sb.layout.UpdateSize(width, height)

	// Determine if sidebar should be collapsed
	sb.updateCollapsedState()

	// Recalculate styles based on new dimensions
	sb.updateStyles()

	// Adjust scroll offset if needed
	sb.adjustScrollOffset()
}

// updateCollapsedState determines if sidebar should be collapsed
func (sb *ResponsiveSidebar) updateCollapsedState() {
	sidebarWidth, _ := sb.layout.GetSidebarDimensions()
	sb.collapsed = sidebarWidth < 25
}

// handleKeyPress handles keyboard input
func (sb *ResponsiveSidebar) handleKeyPress(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if sb.collapsed {
		return sb, nil
	}

	switch key.String() {
	case "up":
		if sb.selectedIndex > 0 {
			sb.selectedIndex--
			sb.adjustScrollOffset()
		}
	case "down":
		if sb.selectedIndex < len(sb.chats)-1 {
			sb.selectedIndex++
			sb.adjustScrollOffset()
		}
	case "pgup":
		sb.scrollUp(5)
	case "pgdown":
		sb.scrollDown(5)
	case "home":
		sb.selectedIndex = 0
		sb.scrollOffset = 0
	case "end":
		sb.selectedIndex = len(sb.chats) - 1
		sb.adjustScrollOffset()
	}
	return sb, nil
}

// View renders the sidebar
func (sb *ResponsiveSidebar) View() string {
	if sb.collapsed {
		return sb.renderCollapsedView()
	}

	sidebarWidth, sidebarHeight := sb.layout.GetSidebarDimensions()

	// Calculate available space for chat list
	headerHeight := 2
	statusHeight := 1
	listHeight := sidebarHeight - headerHeight - statusHeight

	if listHeight < 3 {
		listHeight = 3
	}

	// Render header
	header := sb.renderHeader(sidebarWidth)

	// Render chat list
	chatList := sb.renderChatList(sidebarWidth, listHeight)

	// Render status
	status := sb.renderStatus(sidebarWidth)

	// Combine all sections
	view := fmt.Sprintf("%s\n%s\n%s", header, chatList, status)

	// Apply container style
	return sb.style.Width(sidebarWidth).Height(sidebarHeight).Render(view)
}

// renderCollapsedView renders a collapsed sidebar
func (sb *ResponsiveSidebar) renderCollapsedView() string {
	sidebarWidth, sidebarHeight := sb.layout.GetSidebarDimensions()

	// Create a minimal collapsed view
	content := "📁"
	if len(sb.chats) > 0 {
		content = fmt.Sprintf("📁%d", len(sb.chats))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Foreground(lipgloss.Color("15")).
		Width(sidebarWidth).
		Height(sidebarHeight).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

// renderHeader renders the sidebar header
func (sb *ResponsiveSidebar) renderHeader(width int) string {
	title := "Chats"
	if len(sb.chats) > 0 {
		title = fmt.Sprintf("Chats (%d)", len(sb.chats))
	}

	return sb.headerStyle.Width(width).Render(title)
}

// renderChatList renders the list of chats
func (sb *ResponsiveSidebar) renderChatList(width, height int) string {
	if len(sb.chats) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Render("No chats")
	}

	// Calculate visible chats
	visibleChats := sb.getVisibleChats(height)

	var lines []string
	for i, chat := range visibleChats {
		chatLine := sb.renderChatItem(chat, width-2, i == sb.selectedIndex-sb.scrollOffset)
		lines = append(lines, chatLine)
	}

	// Pad to fill available height
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return strings.Join(lines, "\n")
}

// renderChatItem renders a single chat item
func (sb *ResponsiveSidebar) renderChatItem(chat models.Chat, width int, selected bool) string {
	// Choose style based on selection
	style := sb.itemStyle
	if selected {
		style = sb.selectedStyle
	}

	// Format chat name
	name := chat.Name
	if name == "" {
		name = "Untitled Chat"
	}

	// Add favorite indicator
	if chat.Favorite {
		name = "⭐ " + name
	}

	// Add unread indicator
	if chat.UnreadCount > 0 {
		name = fmt.Sprintf("%s (%d)", name, chat.UnreadCount)
	}

	// Truncate if too long
	if len(name) > width {
		name = name[:width-3] + "..."
	}

	return style.Width(width).Render(name)
}

// renderStatus renders the status bar
func (sb *ResponsiveSidebar) renderStatus(width int) string {
	status := fmt.Sprintf("Selected: %d/%d", sb.selectedIndex+1, len(sb.chats))

	return sb.statusStyle.Width(width).Render(status)
}

// getVisibleChats returns chats visible in the current view
func (sb *ResponsiveSidebar) getVisibleChats(height int) []models.Chat {
	if len(sb.chats) == 0 {
		return []models.Chat{}
	}

	start := sb.scrollOffset
	end := start + height

	if end > len(sb.chats) {
		end = len(sb.chats)
	}

	if start >= len(sb.chats) {
		start = len(sb.chats) - 1
		if start < 0 {
			start = 0
		}
	}

	return sb.chats[start:end]
}

// adjustScrollOffset adjusts scroll offset to keep selected chat visible
func (sb *ResponsiveSidebar) adjustScrollOffset() {
	if len(sb.chats) == 0 {
		return
	}

	_, sidebarHeight := sb.layout.GetSidebarDimensions()
	listHeight := sidebarHeight - 3 // Account for header and status

	if listHeight < 3 {
		listHeight = 3
	}

	// Ensure selected index is within visible range
	if sb.selectedIndex < sb.scrollOffset {
		sb.scrollOffset = sb.selectedIndex
	} else if sb.selectedIndex >= sb.scrollOffset+listHeight {
		sb.scrollOffset = sb.selectedIndex - listHeight + 1
	}

	// Ensure scroll offset is valid
	if sb.scrollOffset < 0 {
		sb.scrollOffset = 0
	}
	if sb.scrollOffset >= len(sb.chats) {
		sb.scrollOffset = len(sb.chats) - 1
		if sb.scrollOffset < 0 {
			sb.scrollOffset = 0
		}
	}
}

// scrollUp scrolls the view up
func (sb *ResponsiveSidebar) scrollUp(lines int) {
	sb.scrollOffset -= lines
	if sb.scrollOffset < 0 {
		sb.scrollOffset = 0
	}
}

// scrollDown scrolls the view down
func (sb *ResponsiveSidebar) scrollDown(lines int) {
	sb.scrollOffset += lines
	if sb.scrollOffset >= len(sb.chats) {
		sb.scrollOffset = len(sb.chats) - 1
		if sb.scrollOffset < 0 {
			sb.scrollOffset = 0
		}
	}
}

// updateStyles updates styles based on current dimensions
func (sb *ResponsiveSidebar) updateStyles() {
	sidebarWidth, _ := sb.layout.GetSidebarDimensions()

	// Adjust styles based on sidebar width
	if sidebarWidth < 30 {
		sb.style = sb.style.BorderStyle(lipgloss.NormalBorder())
		sb.headerStyle = sb.headerStyle.Padding(0, 0)
		sb.itemStyle = sb.itemStyle.Padding(0, 0)
		sb.selectedStyle = sb.selectedStyle.Padding(0, 0)
		sb.statusStyle = sb.statusStyle.Padding(0, 0)
	} else if sidebarWidth < 50 {
		sb.style = sb.style.BorderStyle(lipgloss.RoundedBorder())
		sb.headerStyle = sb.headerStyle.Padding(0, 1)
		sb.itemStyle = sb.itemStyle.Padding(0, 1)
		sb.selectedStyle = sb.selectedStyle.Padding(0, 1)
		sb.statusStyle = sb.statusStyle.Padding(0, 1)
	} else {
		sb.style = sb.style.BorderStyle(lipgloss.RoundedBorder())
		sb.headerStyle = sb.headerStyle.Padding(0, 2)
		sb.itemStyle = sb.itemStyle.Padding(0, 2)
		sb.selectedStyle = sb.selectedStyle.Padding(0, 2)
		sb.statusStyle = sb.statusStyle.Padding(0, 2)
	}
}

// SetChats sets all chats in the sidebar
func (sb *ResponsiveSidebar) SetChats(chats []models.Chat) {
	sb.chats = chats
	if len(chats) > 0 {
		sb.selectedIndex = len(chats) - 1
	} else {
		sb.selectedIndex = 0
	}
	sb.scrollOffset = 0
	sb.adjustScrollOffset()
}

// AddChat adds a chat to the sidebar
func (sb *ResponsiveSidebar) AddChat(chat models.Chat) {
	sb.chats = append(sb.chats, chat)
	sb.selectedIndex = len(sb.chats) - 1
	sb.adjustScrollOffset()
}

// GetSelectedChat returns the currently selected chat
func (sb *ResponsiveSidebar) GetSelectedChat() *models.Chat {
	if sb.selectedIndex >= 0 && sb.selectedIndex < len(sb.chats) {
		return &sb.chats[sb.selectedIndex]
	}
	return nil
}

// IsCollapsed returns whether the sidebar is collapsed
func (sb *ResponsiveSidebar) IsCollapsed() bool {
	return sb.collapsed
}

// GetSidebarWidth returns the current sidebar width
func (sb *ResponsiveSidebar) GetSidebarWidth() int {
	if sb.collapsed {
		return 5 // Minimal width for collapsed state
	}
	width, _ := sb.layout.GetSidebarDimensions()
	return width
}

// Clear clears all chats
func (sb *ResponsiveSidebar) Clear() {
	sb.chats = []models.Chat{}
	sb.selectedIndex = 0
	sb.scrollOffset = 0
}
