// Package app provides the main application components for the AI CLI.
package app

import (
	"fmt"
	"log/slog"
	"os"

	"aichat/src/components/chat"
	"aichat/src/components/common"
	"aichat/src/components/sidebar"
	"aichat/src/models"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =====================================================================================
// 🚀 Responsive Application – Dynamic Layout Management
// =====================================================================================
// This application provides a responsive interface that:
//  - Handles terminal resize events with debouncing
//  - Maintains optimal layout proportions
//  - Provides smooth transitions during resize
//  - Integrates all responsive components seamlessly

// ResponsiveApp represents the main responsive application
type ResponsiveApp struct {
	resizeManager *common.ResizeManager
	layout        *common.ResponsiveLayout
	sidebar       *sidebar.ResponsiveSidebar
	chatView      *chat.ResponsiveChatView
	width         int
	height        int
	style         lipgloss.Style
	headerStyle   lipgloss.Style
	footerStyle   lipgloss.Style
	logger        *slog.Logger
}

// NewResponsiveApp creates a new responsive application
func NewResponsiveApp(logger *slog.Logger) *ResponsiveApp {
	// Create resize manager with default config
	resizeConfig := common.DefaultResizeConfig()
	resizeManager := common.NewResizeManager(resizeConfig, logger)

	// Create layout
	layout := common.NewResponsiveLayout(common.DefaultLayoutConfig())

	// Create components
	sidebar := sidebar.NewResponsiveSidebar()
	chatView := chat.NewResponsiveChatView()

	app := &ResponsiveApp{
		resizeManager: resizeManager,
		layout:        layout,
		sidebar:       sidebar,
		chatView:      chatView,
		width:         80,
		height:        24,
		style:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")),
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Padding(0, 1),
		footerStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 1),
		logger:        logger,
	}

	// Subscribe components to resize events
	resizeManager.Subscribe("sidebar", func(event common.ResizeEvent) {
		sidebar.OnResize(event.Width, event.Height)
	})

	resizeManager.Subscribe("chat_view", func(event common.ResizeEvent) {
		chatView.OnResize(event.Width, event.Height)
	})

	resizeManager.Subscribe("main_app", func(event common.ResizeEvent) {
		app.OnResize(event.Width, event.Height)
	})

	return app
}

// Init initializes the application
func (app *ResponsiveApp) Init() tea.Cmd {
	// Initialize with current terminal size
	if width, height, err := getResponsiveTerminalSize(); err == nil {
		app.OnResize(width, height)
	}

	return nil
}

// Update handles messages and updates the application
func (app *ResponsiveApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.ResizeMsg:
		app.OnResize(msg.Width, msg.Height)
		return app, nil
	case tea.KeyMsg:
		return app.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		// Handle initial window size
		app.OnResize(msg.Width, msg.Height)
		return app, nil
	}

	// Update child components
	app.sidebar.Update(msg)
	app.chatView.Update(msg)

	return app, nil
}

// OnResize handles terminal resize events
func (app *ResponsiveApp) OnResize(width, height int) {
	app.width = width
	app.height = height
	app.layout.UpdateSize(width, height)

	// Update styles based on new dimensions
	app.updateStyles()

	// Notify resize manager
	app.resizeManager.HandleResize(width, height)

	app.logger.Info("Application resized", "width", width, "height", height)
}

// handleKeyPress handles keyboard input
func (app *ResponsiveApp) handleKeyPress(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "ctrl+c", "q":
		return app, tea.Quit
	case "tab":
		// Switch focus between sidebar and chat view
		// This could be enhanced with a focus system
		return app, nil
	case "r":
		// Refresh layout
		app.OnResize(app.width, app.height)
		return app, nil
	}

	return app, nil
}

// View renders the application
func (app *ResponsiveApp) View() string {
	if app.width < 40 || app.height < 10 {
		return app.renderMinimalView()
	}

	// Get layout dimensions
	headerWidth, _ := app.layout.GetHeaderDimensions()
	sidebarWidth, sidebarHeight := app.layout.GetSidebarDimensions()
	contentWidth, contentHeight := app.layout.GetContentDimensions()
	footerWidth, _ := app.layout.GetFooterDimensions()

	// Render header
	header := app.renderHeader(headerWidth)

	// Render main content area
	mainContent := app.renderMainContent(sidebarWidth, sidebarHeight, contentWidth, contentHeight)

	// Render footer
	footer := app.renderFooter(footerWidth)

	// Combine all sections
	view := fmt.Sprintf("%s\n%s\n%s", header, mainContent, footer)

	// Apply container style
	return app.style.Width(app.width).Height(app.height).Render(view)
}

// renderMinimalView renders a minimal view for very small terminals
func (app *ResponsiveApp) renderMinimalView() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Align(lipgloss.Center, lipgloss.Center).
		Width(app.width).
		Height(app.height).
		Render("Terminal too small for application")
}

// renderHeader renders the application header
func (app *ResponsiveApp) renderHeader(width int) string {
	title := "AI CLI - Responsive Interface"
	status := fmt.Sprintf("Size: %dx%d", app.width, app.height)

	content := fmt.Sprintf("%s | %s", title, status)

	return app.headerStyle.Width(width).Render(content)
}

// renderMainContent renders the main content area with sidebar and chat
func (app *ResponsiveApp) renderMainContent(sidebarWidth, sidebarHeight, contentWidth, contentHeight int) string {
	// Render sidebar
	sidebarView := app.sidebar.View()

	// Render chat view
	chatView := app.chatView.View()

	// Create a horizontal layout
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		sidebarView,
		lipgloss.NewStyle().Width(1).Render("│"), // Separator
		chatView,
	)

	return mainContent
}

// renderFooter renders the application footer
func (app *ResponsiveApp) renderFooter(width int) string {
	help := "q: Quit | Tab: Switch Focus | r: Refresh | ↑↓: Navigate"

	return app.footerStyle.Width(width).Render(help)
}

// updateStyles updates styles based on current dimensions
func (app *ResponsiveApp) updateStyles() {
	// Adjust styles based on terminal size
	if app.width < 60 {
		app.style = app.style.BorderStyle(lipgloss.NormalBorder())
		app.headerStyle = app.headerStyle.Padding(0, 0)
		app.footerStyle = app.footerStyle.Padding(0, 0)
	} else if app.width < 100 {
		app.style = app.style.BorderStyle(lipgloss.RoundedBorder())
		app.headerStyle = app.headerStyle.Padding(0, 1)
		app.footerStyle = app.footerStyle.Padding(0, 1)
	} else {
		app.style = app.style.BorderStyle(lipgloss.RoundedBorder())
		app.headerStyle = app.headerStyle.Padding(0, 2)
		app.footerStyle = app.footerStyle.Padding(0, 2)
	}
}

// LoadSampleData loads sample data for demonstration
func (app *ResponsiveApp) LoadSampleData() {
	// Sample chats
	chats := []models.Chat{
		{Name: "General Chat", Favorite: true, UnreadCount: 0},
		{Name: "Code Review", Favorite: false, UnreadCount: 2},
		{Name: "Project Planning", Favorite: true, UnreadCount: 1},
		{Name: "Bug Discussion", Favorite: false, UnreadCount: 0},
		{Name: "Feature Ideas", Favorite: false, UnreadCount: 3},
	}
	app.sidebar.SetChats(chats)

	// Sample messages
	messages := []models.Message{
		{Role: "user", Content: "Hello, how can you help me today?", MessageNumber: 1},
		{Role: "assistant", Content: "I'm here to help! I can assist with coding, answer questions, and much more.", MessageNumber: 2},
		{Role: "user", Content: "Can you help me with Go programming?", MessageNumber: 3},
		{Role: "assistant", Content: "Absolutely! Go is a great language. What specific aspect would you like to explore?", MessageNumber: 4},
	}
	app.chatView.SetMessages(messages)
}

// GetResizeStats returns resize statistics
func (app *ResponsiveApp) GetResizeStats() map[string]interface{} {
	// This would return actual resize statistics
	// For now, return empty stats
	return map[string]interface{}{}
}

// Shutdown gracefully shuts down the application
func (app *ResponsiveApp) Shutdown() {
	app.resizeManager.Shutdown()
	app.logger.Info("Responsive application shutdown complete")
}

// =====================================================================================
// 🛠️ Utility Functions
// =====================================================================================

// getResponsiveTerminalSize gets the current terminal size for responsive app
func getResponsiveTerminalSize() (width, height int, err error) {
	// Try to get size from environment variables first
	if w := os.Getenv("COLUMNS"); w != "" {
		if h := os.Getenv("LINES"); h != "" {
			// Parse width and height from environment
			// This is a simplified implementation
			return 80, 24, nil
		}
	}

	// Fallback to default size
	return 80, 24, nil
}

// =====================================================================================
// 🚀 Application Factory
// =====================================================================================

// NewResponsiveProgram creates a new responsive program with resize handling
func NewResponsiveProgram(logger *slog.Logger) *common.ResizeAwareProgram {
	app := NewResponsiveApp(logger)

	// Load sample data
	app.LoadSampleData()

	// Create resize-aware program
	resizeConfig := common.DefaultResizeConfig()
	program := common.NewResizeAwareProgram(app, resizeConfig, logger)

	return program
}
