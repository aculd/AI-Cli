// Package chat provides chat-related components for the AI CLI application.
package chat

import (
	"fmt"
	"strings"

	"aichat/src/components/common"
	"aichat/src/models"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =====================================================================================
// 🎨 Responsive Chat View – Dynamic Layout Adaptation
// =====================================================================================
// This component provides a responsive chat interface that:
//  - Adapts to terminal size changes in real-time
//  - Maintains optimal layout proportions
//  - Provides smooth transitions during resize
//  - Handles content overflow gracefully

// ResponsiveChatView represents a responsive chat view
type ResponsiveChatView struct {
	layout        *common.ResponsiveLayout
	messages      []models.Message
	width         int
	height        int
	scrollOffset  int
	selectedIndex int
	style         lipgloss.Style
	headerStyle   lipgloss.Style
	messageStyle  lipgloss.Style
	inputStyle    lipgloss.Style
	statusStyle   lipgloss.Style
}

// NewResponsiveChatView creates a new responsive chat view
func NewResponsiveChatView() *ResponsiveChatView {
	layout := common.NewResponsiveLayout(common.DefaultLayoutConfig())

	return &ResponsiveChatView{
		layout:        layout,
		messages:      []models.Message{},
		width:         80,
		height:        24,
		scrollOffset:  0,
		selectedIndex: 0,
		style:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")),
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Padding(0, 1),
		messageStyle:  lipgloss.NewStyle().Padding(0, 1).Margin(0, 0, 1, 0),
		inputStyle:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1),
		statusStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 1),
	}
}

// Init initializes the chat view
func (cv *ResponsiveChatView) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the view
func (cv *ResponsiveChatView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.ResizeMsg:
		cv.OnResize(msg.Width, msg.Height)
		return cv, nil
	case tea.KeyMsg:
		return cv.handleKeyPress(msg)
	}
	return cv, nil
}

// OnResize handles terminal resize events
func (cv *ResponsiveChatView) OnResize(width, height int) {
	cv.width = width
	cv.height = height
	cv.layout.UpdateSize(width, height)

	// Recalculate styles based on new dimensions
	cv.updateStyles()

	// Adjust scroll offset if needed
	cv.adjustScrollOffset()
}

// handleKeyPress handles keyboard input
func (cv *ResponsiveChatView) handleKeyPress(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "up":
		if cv.selectedIndex > 0 {
			cv.selectedIndex--
			cv.adjustScrollOffset()
		}
	case "down":
		if cv.selectedIndex < len(cv.messages)-1 {
			cv.selectedIndex++
			cv.adjustScrollOffset()
		}
	case "pgup":
		cv.scrollUp(5)
	case "pgdown":
		cv.scrollDown(5)
	case "home":
		cv.selectedIndex = 0
		cv.scrollOffset = 0
	case "end":
		cv.selectedIndex = len(cv.messages) - 1
		cv.adjustScrollOffset()
	}
	return cv, nil
}

// View renders the chat view
func (cv *ResponsiveChatView) View() string {
	if cv.width < 40 || cv.height < 10 {
		return cv.renderMinimalView()
	}

	contentWidth, contentHeight := cv.layout.GetContentDimensions()

	// Calculate available space for messages
	headerHeight := 2
	inputHeight := 3
	statusHeight := 1
	messageHeight := contentHeight - headerHeight - inputHeight - statusHeight

	if messageHeight < 3 {
		messageHeight = 3
	}

	// Render header
	header := cv.renderHeader(contentWidth)

	// Render messages
	messages := cv.renderMessages(contentWidth, messageHeight)

	// Render input area
	input := cv.renderInput(contentWidth)

	// Render status
	status := cv.renderStatus(contentWidth)

	// Combine all sections
	view := fmt.Sprintf("%s\n%s\n%s\n%s", header, messages, input, status)

	// Apply container style
	return cv.style.Width(contentWidth).Height(contentHeight).Render(view)
}

// renderMinimalView renders a minimal view for very small terminals
func (cv *ResponsiveChatView) renderMinimalView() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("Terminal too small for chat view")
}

// renderHeader renders the chat header
func (cv *ResponsiveChatView) renderHeader(width int) string {
	title := "AI Chat"
	if len(cv.messages) > 0 {
		title = fmt.Sprintf("AI Chat (%d messages)", len(cv.messages))
	}

	return cv.headerStyle.Width(width).Render(title)
}

// renderMessages renders the message list
func (cv *ResponsiveChatView) renderMessages(width, height int) string {
	if len(cv.messages) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Render("No messages yet")
	}

	// Calculate visible messages
	visibleMessages := cv.getVisibleMessages(height)

	var lines []string
	for i, msg := range visibleMessages {
		messageLine := cv.renderMessage(msg, width-2, i == cv.selectedIndex-cv.scrollOffset)
		lines = append(lines, messageLine)
	}

	// Pad to fill available height
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return strings.Join(lines, "\n")
}

// renderMessage renders a single message
func (cv *ResponsiveChatView) renderMessage(msg models.Message, width int, selected bool) string {
	// Create message style
	style := cv.messageStyle.Copy()
	if selected {
		style = style.Background(lipgloss.Color("240")).Foreground(lipgloss.Color("15"))
	}

	// Format message content
	content := cv.formatMessageContent(msg, width-10)

	// Add role indicator
	role := "AI"
	if msg.Role == "user" {
		role = "You"
	}

	// Format message number
	timestamp := fmt.Sprintf("#%d", msg.MessageNumber)

	// Combine elements
	line := fmt.Sprintf("[%s] %s: %s", timestamp, role, content)

	// Truncate if too long
	if len(line) > width {
		line = line[:width-3] + "..."
	}

	return style.Width(width).Render(line)
}

// formatMessageContent formats message content for display
func (cv *ResponsiveChatView) formatMessageContent(msg models.Message, maxWidth int) string {
	content := msg.Content

	// Truncate if too long
	if len(content) > maxWidth {
		content = content[:maxWidth-3] + "..."
	}

	return content
}

// renderInput renders the input area
func (cv *ResponsiveChatView) renderInput(width int) string {
	prompt := "Type your message..."
	return cv.inputStyle.Width(width).Render(prompt)
}

// renderStatus renders the status bar
func (cv *ResponsiveChatView) renderStatus(width int) string {
	status := fmt.Sprintf("Messages: %d | Selected: %d | Scroll: %d",
		len(cv.messages), cv.selectedIndex+1, cv.scrollOffset+1)

	return cv.statusStyle.Width(width).Render(status)
}

// getVisibleMessages returns messages visible in the current view
func (cv *ResponsiveChatView) getVisibleMessages(height int) []models.Message {
	if len(cv.messages) == 0 {
		return []models.Message{}
	}

	start := cv.scrollOffset
	end := start + height

	if end > len(cv.messages) {
		end = len(cv.messages)
	}

	if start >= len(cv.messages) {
		start = len(cv.messages) - 1
		if start < 0 {
			start = 0
		}
	}

	return cv.messages[start:end]
}

// adjustScrollOffset adjusts scroll offset to keep selected message visible
func (cv *ResponsiveChatView) adjustScrollOffset() {
	if len(cv.messages) == 0 {
		return
	}

	_, contentHeight := cv.layout.GetContentDimensions()
	messageHeight := contentHeight - 6 // Account for header, input, status

	if messageHeight < 3 {
		messageHeight = 3
	}

	// Ensure selected index is within visible range
	if cv.selectedIndex < cv.scrollOffset {
		cv.scrollOffset = cv.selectedIndex
	} else if cv.selectedIndex >= cv.scrollOffset+messageHeight {
		cv.scrollOffset = cv.selectedIndex - messageHeight + 1
	}

	// Ensure scroll offset is valid
	if cv.scrollOffset < 0 {
		cv.scrollOffset = 0
	}
	if cv.scrollOffset >= len(cv.messages) {
		cv.scrollOffset = len(cv.messages) - 1
		if cv.scrollOffset < 0 {
			cv.scrollOffset = 0
		}
	}
}

// scrollUp scrolls the view up
func (cv *ResponsiveChatView) scrollUp(lines int) {
	cv.scrollOffset -= lines
	if cv.scrollOffset < 0 {
		cv.scrollOffset = 0
	}
}

// scrollDown scrolls the view down
func (cv *ResponsiveChatView) scrollDown(lines int) {
	cv.scrollOffset += lines
	if cv.scrollOffset >= len(cv.messages) {
		cv.scrollOffset = len(cv.messages) - 1
		if cv.scrollOffset < 0 {
			cv.scrollOffset = 0
		}
	}
}

// updateStyles updates styles based on current dimensions
func (cv *ResponsiveChatView) updateStyles() {
	// Adjust styles based on terminal size
	if cv.width < 60 {
		cv.style = cv.style.BorderStyle(lipgloss.NormalBorder())
		cv.headerStyle = cv.headerStyle.Padding(0, 0)
		cv.messageStyle = cv.messageStyle.Padding(0, 0)
		cv.inputStyle = cv.inputStyle.Padding(0, 0)
		cv.statusStyle = cv.statusStyle.Padding(0, 0)
	} else if cv.width < 100 {
		cv.style = cv.style.BorderStyle(lipgloss.RoundedBorder())
		cv.headerStyle = cv.headerStyle.Padding(0, 1)
		cv.messageStyle = cv.messageStyle.Padding(0, 1)
		cv.inputStyle = cv.inputStyle.Padding(0, 1)
		cv.statusStyle = cv.statusStyle.Padding(0, 1)
	} else {
		cv.style = cv.style.BorderStyle(lipgloss.RoundedBorder())
		cv.headerStyle = cv.headerStyle.Padding(0, 2)
		cv.messageStyle = cv.messageStyle.Padding(0, 2)
		cv.inputStyle = cv.inputStyle.Padding(0, 2)
		cv.statusStyle = cv.statusStyle.Padding(0, 2)
	}
}

// AddMessage adds a message to the chat
func (cv *ResponsiveChatView) AddMessage(msg models.Message) {
	cv.messages = append(cv.messages, msg)
	cv.selectedIndex = len(cv.messages) - 1
	cv.adjustScrollOffset()
}

// SetMessages sets all messages in the chat
func (cv *ResponsiveChatView) SetMessages(messages []models.Message) {
	cv.messages = messages
	if len(messages) > 0 {
		cv.selectedIndex = len(messages) - 1
	} else {
		cv.selectedIndex = 0
	}
	cv.scrollOffset = 0
	cv.adjustScrollOffset()
}

// GetSelectedMessage returns the currently selected message
func (cv *ResponsiveChatView) GetSelectedMessage() *models.Message {
	if cv.selectedIndex >= 0 && cv.selectedIndex < len(cv.messages) {
		return &cv.messages[cv.selectedIndex]
	}
	return nil
}

// Clear clears all messages
func (cv *ResponsiveChatView) Clear() {
	cv.messages = []models.Message{}
	cv.selectedIndex = 0
	cv.scrollOffset = 0
}
