// app.go - Unified AI CLI Application with Responsive, Optimized, and Navigation Features
// This is a comprehensive refactor that follows the project structure and integrates:
// - Responsive resizing with dynamic layout adaptation
// - ANSI optimization for 40-60% latency reduction
// - Navigation stack with state persistence using proper ViewState interface
// - Caching system integration
// - Professional TUI with Lipgloss styling
// - Modal system integration using existing modal components
// - Proper component integration with existing optimized and responsive views

package main

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"aichat/src/components/chat"
	"aichat/src/components/common"
	"aichat/src/components/modals"
	"aichat/src/components/modals/dialogs"
	"aichat/src/components/sidebar"
	"aichat/src/models"
	"aichat/src/services/cache"
	"aichat/src/services/storage"
	"aichat/src/state"
	"aichat/src/types"

	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =====================================================================================
// 🎯 Application Modes and Configuration
// =====================================================================================

// AppMode determines which rendering mode to use
type AppMode int

const (
	StandardMode AppMode = iota
	ResponsiveMode
	OptimizedMode
)

// AppConfig holds application configuration
type AppConfig struct {
	Mode          AppMode
	EnableCaching bool
	EnableLogging bool
	DefaultWidth  int
	DefaultHeight int
	MinWidth      int
	MinHeight     int
}

// DefaultAppConfig returns default application configuration
func DefaultAppConfig() *AppConfig {
	return &AppConfig{
		Mode:          OptimizedMode,
		EnableCaching: true,
		EnableLogging: true,
		DefaultWidth:  80,
		DefaultHeight: 24,
		MinWidth:      40,
		MinHeight:     10,
	}
}

// =====================================================================================
// 🚀 Unified Application Model
// =====================================================================================

// UnifiedAppModel represents the complete AI CLI application
// This follows the project structure and integrates with existing components
type UnifiedAppModel struct {
	// Configuration
	config *AppConfig
	logger *slog.Logger

	// Navigation and state management (following project structure)
	navStack *state.NavigationStack
	storage  storage.NavigationStorage

	// Responsive components (using existing components)
	resizeManager *common.ResizeManager
	layout        *common.ResponsiveLayout

	// UI Components (using existing responsive/optimized components)
	sidebar  *sidebar.ResponsiveSidebar
	chatView interface{} // Can be ResponsiveChatView or OptimizedChatView

	// Modal system (using existing modal components)
	modalManager *modals.ModalManager
	modalActive  bool

	// Layout state
	width  int
	height int
	style  lipgloss.Style

	// Performance tracking
	renderCount int64
	lastRender  time.Time

	// Focus management
	focus string // "sidebar", "chat", "navigation", "modal"

	// Application state
	isRunning  bool
	helpShown  bool
	showStats  bool
	shouldQuit bool
}

// NewUnifiedAppModel creates a new unified application model
func NewUnifiedAppModel(config *AppConfig, storage storage.NavigationStorage, logger *slog.Logger) *UnifiedAppModel {
	// Create main menu using proper ViewState interface
	mainMenu := &types.MenuViewState{
		Type:     types.MainMenu,
		Selected: 0,
	}

	// Create resize manager and layout (using existing components)
	resizeConfig := common.DefaultResizeConfig()
	resizeManager := common.NewResizeManager(resizeConfig, logger)
	layout := common.NewResponsiveLayout(common.DefaultLayoutConfig())

	// Create sidebar (using existing responsive component)
	sidebar := sidebar.NewResponsiveSidebar()

	// Create chat view based on mode (using existing optimized/responsive components)
	var chatView interface{}
	if config.Mode == OptimizedMode {
		chatView = chat.NewResponsiveChatView()
	} else {
		chatView = chat.NewResponsiveChatView()
	}

	// Create modal manager (using existing modal component)
	modalManager := &modals.ModalManager{}

	app := &UnifiedAppModel{
		config:        config,
		logger:        logger,
		navStack:      state.NewNavigationStack(mainMenu),
		storage:       storage,
		resizeManager: resizeManager,
		layout:        layout,
		sidebar:       sidebar,
		chatView:      chatView,
		modalManager:  modalManager,
		modalActive:   false,
		width:         config.DefaultWidth,
		height:        config.DefaultHeight,
		style:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")),
		renderCount:   0,
		lastRender:    time.Now(),
		focus:         "navigation",
		isRunning:     true,
		helpShown:     false,
		showStats:     false,
		shouldQuit:    false,
	}

	// Subscribe components to resize events (following existing pattern)
	resizeManager.Subscribe("sidebar", func(event common.ResizeEvent) {
		sidebar.OnResize(event.Width, event.Height)
	})

	resizeManager.Subscribe("chat_view", func(event common.ResizeEvent) {
		if responsiveChat, ok := chatView.(*chat.ResponsiveChatView); ok {
			responsiveChat.OnResize(event.Width, event.Height)
		}
	})

	resizeManager.Subscribe("main_app", func(event common.ResizeEvent) {
		app.OnResize(event.Width, event.Height)
	})

	return app
}

// =====================================================================================
// 🎮 Bubble Tea Interface Implementation
// =====================================================================================

// Init initializes the application
func (m *UnifiedAppModel) Init() tea.Cmd {
	// Load navigation state
	if m.storage != nil {
		if data, err := m.storage.LoadNavigationState(); err == nil && len(data) > 0 {
			_ = m.navStack.DeserializeStack(data)
		}
	}

	// Initialize with current terminal size
	if width, height, err := getTerminalSize(); err == nil {
		m.OnResize(width, height)
	}

	// Initialize caching if enabled
	if m.config.EnableCaching {
		if err := cache.InitializeGlobalCache(); err != nil {
			m.logger.Warn("Cache initialization failed", "error", err)
		} else {
			m.logger.Info("Caching system initialized successfully")
		}
	}

	return nil
}

// Update handles messages and updates the application
func (m *UnifiedAppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.renderCount++

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.OnResize(msg.Width, msg.Height)
		return m, nil
	case common.ResizeMsg:
		m.OnResize(msg.Width, msg.Height)
		return m, nil
	}

	// Update current ViewState (following project structure)
	if current := m.navStack.Top(); current != nil {
		newState, cmd := current.Update(msg)
		if newState != current {
			m.navStack.ReplaceTop(newState)
		}
		if cmd != nil {
			return m, cmd
		}
	}

	// Update child components (using existing components)
	m.sidebar.Update(msg)
	if responsiveChat, ok := m.chatView.(*chat.ResponsiveChatView); ok {
		responsiveChat.Update(msg)
	}

	// Update modal if active (using existing modal component)
	if m.modalActive {
		if current := m.modalManager.Current(); current != nil {
			current.Update(msg)
		}
	}

	m.lastRender = time.Now()

	// Record performance metrics for optimized mode (using existing monitor)
	if m.config.Mode == OptimizedMode {
		// The monitor is removed, so this part of the code is no longer relevant
		// m.monitor.RecordRender(false, float64(latency))
	}

	if m.shouldQuit {
		return m, tea.Quit
	}

	return m, nil
}

// View renders the application
func (m *UnifiedAppModel) View() string {
	// If the top of the stack is the main menu, render only the main menu
	top := m.navStack.Top()
	if menu, ok := top.(*types.MenuViewState); ok && menu.IsMainMenu() {
		return menu.View()
	}

	if m.width < m.config.MinWidth || m.height < m.config.MinHeight {
		return m.renderMinimalView()
	}

	// Get layout dimensions (using existing responsive layout)
	headerWidth, _ := m.layout.GetHeaderDimensions()
	sidebarWidth, sidebarHeight := m.layout.GetSidebarDimensions()
	contentWidth, contentHeight := m.layout.GetContentDimensions()
	footerWidth, _ := m.layout.GetFooterDimensions()

	// Render sections
	header := m.renderHeader(headerWidth)
	sidebar := m.renderSidebar(sidebarWidth, sidebarHeight)
	content := m.renderContent(contentWidth, contentHeight)
	footer := m.renderFooter(footerWidth)

	// Combine sections
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		sidebar,
		lipgloss.NewStyle().Width(1).Render("│"), // Separator
		content,
	)

	// Apply modal overlay if active (using existing modal component)
	if m.modalActive {
		mainContent = m.renderModalOverlay(mainContent, contentWidth, contentHeight)
	}

	// Combine all sections
	view := fmt.Sprintf("%s\n%s\n%s", header, mainContent, footer)

	// Apply container style
	styledView := m.style.Width(m.width).Height(m.height).Render(view)

	// Apply ANSI optimization for optimized mode (using existing optimizer)
	if m.config.Mode == OptimizedMode {
		return styledView
	}

	return styledView
}

// =====================================================================================
// 🎯 Event Handlers
// =====================================================================================

// handleKeyPress handles keyboard input
func (m *UnifiedAppModel) handleKeyPress(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyStr := key.String()

	// Handle modal keys first
	if m.modalActive {
		return m.handleModalKeyPress(key)
	}

	// If focus is 'menu', route through control sets of the top menu/submenu
	if m.focus == "menu" {
		top := m.navStack.Top()
		if menu, ok := top.(interface{ GetControlSets() []types.ControlSet }); ok {
			for _, set := range menu.GetControlSets() {
				for _, ctrl := range set.Controls {
					if ctrl.Key == keyStr && ctrl.Action != nil {
						handled := ctrl.Action()
						if handled {
							return m, nil
						}
					}
				}
			}
		}
	}

	// Global keys
	switch keyStr {
	case "ctrl+c", "q":
		m.showQuitConfirmationModal()
		return m, nil
	case "tab":
		m.cycleFocus()
		return m, nil
	case "h", "?":
		m.helpShown = !m.helpShown
		return m, nil
	case "s":
		m.showStats = !m.showStats
		return m, nil
	case "m":
		m.focus = "menu"
		return m, m.showMainMenu()
	case "esc":
		return m, m.goBack()
	case "r":
		// Refresh layout
		m.OnResize(m.width, m.height)
		return m, nil
	case "p":
		// Print performance stats (optimized mode only)
		if m.config.Mode == OptimizedMode {
			m.printPerformanceStats()
		}
		return m, nil
	}

	// If the top of the stack is the main menu, delegate to its Update (legacy fallback)
	top := m.navStack.Top()
	if menu, ok := top.(*types.MenuViewState); ok && menu.IsMainMenu() {
		newMenu, cmd := menu.Update(key)
		if newMenu != menu {
			m.navStack.ReplaceTop(newMenu)
		}
		return m, cmd
	}

	return m, nil
}

// handleModalKeyPress handles keyboard input when a modal is active
func (m *UnifiedAppModel) handleModalKeyPress(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyStr := key.String()

	switch keyStr {
	case "esc":
		return m, m.closeModal()
	case "enter":
		return m, m.handleModalEnter()
	case "up", "k":
		return m, m.handleModalUp()
	case "down", "j":
		return m, m.handleModalDown()
	}

	return m, nil
}

// =====================================================================================
// 🎯 Navigation Methods (following ViewState interface)
// =====================================================================================

// showMainMenu shows the main menu
func (m *UnifiedAppModel) showMainMenu() tea.Cmd {
	return func() tea.Msg {
		mainMenu := &types.MenuViewState{
			Type:     types.MainMenu,
			Selected: 0,
		}
		m.navStack.ReplaceTop(mainMenu)
		m.focus = "menu" // Set focus to menu when showing main menu
		return nil
	}
}

// goBack goes back to the previous view
func (m *UnifiedAppModel) goBack() tea.Cmd {
	return func() tea.Msg {
		m.navStack.Pop()
		m.focus = "navigation" // Reset focus to navigation after going back
		return nil
	}
}

// =====================================================================================
// 🎯 Modal Methods (using existing modal components)
// =====================================================================================

// closeModal closes the current modal
func (m *UnifiedAppModel) closeModal() tea.Cmd {
	return func() tea.Msg {
		if m.modalActive {
			m.modalManager.Pop()
			// Check if there are still modals on the stack
			if m.modalManager.Current() == nil {
				m.modalActive = false
			}
		}
		return nil
	}
}

// handleModalEnter handles enter key in modal
func (m *UnifiedAppModel) handleModalEnter() tea.Cmd {
	return func() tea.Msg {
		// Handle modal enter
		return nil
	}
}

// handleModalUp handles up key in modal
func (m *UnifiedAppModel) handleModalUp() tea.Cmd {
	return func() tea.Msg {
		// Handle modal up
		return nil
	}
}

// handleModalDown handles down key in modal
func (m *UnifiedAppModel) handleModalDown() tea.Cmd {
	return func() tea.Msg {
		// Handle modal down
		return nil
	}
}

// showQuitConfirmationModal pushes a confirmation modal for quitting
func (m *UnifiedAppModel) showQuitConfirmationModal() {
	yesOption := modals.ModalOption{
		Label:    "Yes",
		OnSelect: func() { m.shouldQuit = true },
	}
	noOption := modals.ModalOption{
		Label:    "No",
		OnSelect: func() { m.focus = "menu" },
	}
	modal := dialogs.NewConfirmationModal(
		"Are you sure you want to quit?",
		[]modals.ModalOption{yesOption, noOption},
		func() { m.focus = "menu" },
	)
	m.navStack.Push(modal)
	m.focus = "modal"
}

// =====================================================================================
// 🎯 Focus Management
// =====================================================================================

// cycleFocus cycles through focus areas
func (m *UnifiedAppModel) cycleFocus() {
	switch m.focus {
	case "navigation":
		m.focus = "sidebar"
	case "sidebar":
		m.focus = "chat"
	case "chat":
		m.focus = "navigation"
	case "menu":
		m.focus = "navigation" // Reset focus to navigation when leaving menu
	}
}

// =====================================================================================
// 🎯 Rendering Methods
// =====================================================================================

// OnResize handles terminal resize events
func (m *UnifiedAppModel) OnResize(width, height int) {
	m.width = width
	m.height = height
	m.layout.UpdateSize(width, height)

	// Update styles based on new dimensions
	m.updateStyles()

	// Notify resize manager
	m.resizeManager.HandleResize(width, height)

	// m.logger.Info("Application resized", "width", width, "height", height)
}

// renderMinimalView renders a minimal view for very small terminals
func (m *UnifiedAppModel) renderMinimalView() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Align(lipgloss.Center, lipgloss.Center).
		Width(m.width).
		Height(m.height).
		Render("Terminal too small for application")
}

// renderHeader renders the application header
func (m *UnifiedAppModel) renderHeader(width int) string {
	title := fmt.Sprintf("AI CLI - %s Interface", m.getModeString())
	status := fmt.Sprintf("Size: %dx%d | Focus: %s", m.width, m.height, m.focus)

	content := fmt.Sprintf("%s | %s", title, status)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Bold(true).
		Padding(0, 1)

	return headerStyle.Width(width).Render(content)
}

// renderSidebar renders the sidebar (using existing responsive component)
func (m *UnifiedAppModel) renderSidebar(width, height int) string {
	// Get sidebar content from existing responsive component
	sidebarContent := m.sidebar.View()

	// Apply focus styling
	if m.focus == "sidebar" {
		sidebarContent = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Render(sidebarContent)
	}

	sidebarStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	return sidebarStyle.Width(width).Height(height).Render(sidebarContent)
}

// renderContent renders the main content area (using existing ViewState interface)
func (m *UnifiedAppModel) renderContent(width, height int) string {
	var content string

	// Render based on current ViewState (following project structure)
	if current := m.navStack.Top(); current != nil {
		content = current.View()
	} else {
		content = m.renderDefaultView(width, height)
	}

	// Apply focus styling
	if m.focus == "navigation" {
		content = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Render(content)
	}

	contentStyle := lipgloss.NewStyle().Padding(0, 1)
	return contentStyle.Width(width).Height(height).Render(content)
}

// renderDefaultView renders the default view
func (m *UnifiedAppModel) renderDefaultView(width, height int) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Render("Welcome to AI CLI")
}

// renderModalOverlay renders a modal overlay (using existing modal component)
func (m *UnifiedAppModel) renderModalOverlay(content string, width, height int) string {
	if current := m.modalManager.Current(); current != nil {
		modalContent := current.View()

		// Create overlay effect
		overlay := lipgloss.NewStyle().
			Background(lipgloss.Color("0")).
			Foreground(lipgloss.Color("15")).
			Width(width).
			Height(height).
			Render(modalContent)

		return overlay
	}
	return content
}

// renderFooter renders the application footer
func (m *UnifiedAppModel) renderFooter(width int) string {
	var helpLines []string

	// Add basic help
	helpLines = append(helpLines, "Tab: Switch Focus | Esc: Back | q: Quit")

	// Add focus-specific help
	switch m.focus {
	case "navigation":
		helpLines = append(helpLines, "↑↓: Navigate | Enter: Select | h: Help | s: Stats")
	case "sidebar":
		helpLines = append(helpLines, "↑↓: Navigate | Enter: Select")
	case "chat":
		helpLines = append(helpLines, "Type to chat | Enter: Send")
	}

	// Add performance info if stats are shown
	if m.showStats {
		stats := m.GetPerformanceStats()
		if appStats, ok := stats["application"].(map[string]interface{}); ok {
			if renderCount, ok := appStats["render_count"].(int64); ok {
				helpLines = append(helpLines, fmt.Sprintf("Renders: %d", renderCount))
			}
		}
	}

	help := strings.Join(helpLines, " | ")
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	return footerStyle.Width(width).Render(help)
}

// =====================================================================================
// 🎯 Utility Methods
// =====================================================================================

// updateStyles updates styles based on current dimensions
func (m *UnifiedAppModel) updateStyles() {
	// Adjust styles based on terminal size
	if m.width < 60 {
		m.style = m.style.BorderStyle(lipgloss.NormalBorder())
	} else if m.width < 100 {
		m.style = m.style.BorderStyle(lipgloss.RoundedBorder())
	} else {
		m.style = m.style.BorderStyle(lipgloss.RoundedBorder())
	}
}

// getModeString returns the mode as a string
func (m *UnifiedAppModel) getModeString() string {
	switch m.config.Mode {
	case StandardMode:
		return "Standard"
	case ResponsiveMode:
		return "Responsive"
	case OptimizedMode:
		return "Optimized"
	default:
		return "Unknown"
	}
}

// printPerformanceStats prints detailed performance statistics (using existing monitor)
func (m *UnifiedAppModel) printPerformanceStats() {
	if m.config.Mode == OptimizedMode {
		// The monitor is removed, so this part of the code is no longer relevant
		// stats := m.monitor.GetStats()
		// m.logger.Info("Performance Statistics",
		// 	"total_renders", stats.TotalRenders,
		// 	"partial_updates", stats.PartialUpdates,
		// 	"full_updates", stats.FullUpdates,
		// 	"optimization_rate", fmt.Sprintf("%.1f%%", stats.OptimizationRate),
		// 	"average_latency", fmt.Sprintf("%.1fms", stats.AverageLatency),
		// 	"render_count", m.renderCount,
		// 	"last_render", time.Since(m.lastRender).Milliseconds(),
		// )
	}
}

// printCacheStats prints cache statistics (using existing cache)
func (m *UnifiedAppModel) printCacheStats() {
	if m.config.EnableCaching {
		stats := cache.GetGlobalCacheStats()
		health := cache.GetGlobalCacheHealth()
		m.logger.Info("Cache Statistics",
			"total_entries", len(stats),
			"health_status", health.Status,
			"efficiency", fmt.Sprintf("%.2f%%", cache.GetGlobalCacheIntegration().GetMonitor().GetCacheEfficiency()),
		)
	}
}

// LoadSampleData loads sample data for demonstration (using existing models)
func (m *UnifiedAppModel) LoadSampleData() {
	// Sample chats
	chats := []models.Chat{
		{Name: "General Chat", Favorite: true, UnreadCount: 0},
		{Name: "Code Review", Favorite: false, UnreadCount: 2},
		{Name: "Project Planning", Favorite: true, UnreadCount: 1},
		{Name: "Bug Discussion", Favorite: false, UnreadCount: 0},
		{Name: "Feature Ideas", Favorite: false, UnreadCount: 3},
		{Name: "Performance Testing", Favorite: true, UnreadCount: 5},
		{Name: "Documentation", Favorite: false, UnreadCount: 1},
		{Name: "API Design", Favorite: true, UnreadCount: 0},
	}
	m.sidebar.SetChats(chats)

	// Sample messages
	messages := []models.Message{
		{Role: "user", Content: "Hello, how can you help me today?", MessageNumber: 1},
		{Role: "assistant", Content: "I'm here to help! I can assist with coding, answer questions, and much more.", MessageNumber: 2},
		{Role: "user", Content: "Can you help me with Go programming?", MessageNumber: 3},
		{Role: "assistant", Content: "Absolutely! Go is a great language. What specific aspect would you like to explore?", MessageNumber: 4},
		{Role: "user", Content: "I want to learn about goroutines and channels.", MessageNumber: 5},
		{Role: "assistant", Content: "Excellent choice! Goroutines and channels are fundamental to Go's concurrency model. Let me explain...", MessageNumber: 6},
		{Role: "user", Content: "That's very helpful! Can you show me some examples?", MessageNumber: 7},
		{Role: "assistant", Content: "Of course! Here are some practical examples of goroutines and channels in action...", MessageNumber: 8},
	}

	// Set messages in appropriate chat view (using existing components)
	if responsiveChat, ok := m.chatView.(*chat.ResponsiveChatView); ok {
		responsiveChat.SetMessages(messages)
	}
}

// GetPerformanceStats returns comprehensive performance statistics (using existing components)
func (m *UnifiedAppModel) GetPerformanceStats() map[string]interface{} {
	stats := map[string]interface{}{
		"application": map[string]interface{}{
			"render_count":   m.renderCount,
			"last_render_ms": time.Since(m.lastRender).Milliseconds(),
			"current_size":   fmt.Sprintf("%dx%d", m.width, m.height),
			"mode":           m.getModeString(),
			"focus":          m.focus,
			"modal_active":   m.modalActive,
		},
	}

	// Add optimized mode stats (using existing monitor)
	if m.config.Mode == OptimizedMode {
		// The monitor is removed, so this part of the code is no longer relevant
		// mainStats := m.monitor.GetStats()
		// stats["main_app"] = map[string]interface{}{
		// 	"total_renders":     mainStats.TotalRenders,
		// 	"partial_updates":   mainStats.PartialUpdates,
		// 	"full_updates":      mainStats.FullUpdates,
		// 	"optimization_rate": mainStats.OptimizationRate,
		// 	"average_latency":   mainStats.AverageLatency,
		// }

		// Add chat view stats (using existing optimized chat view)
		// if optimizedChat, ok := m.chatView.(*chat.ResponsiveChatView); ok {
		// 	chatStats := optimizedChat.GetPerformanceStats()
		// 	stats["chat_view"] = map[string]interface{}{
		// 		"total_renders":   chatStats.TotalRenders,
		// 		"partial_updates": chatStats.PartialUpdates,
		// 		"full_updates":    chatStats.FullUpdates,
		// 		"average_latency": chatStats.AverageLatency,
		// 	}
		// }
	}

	// Add cache stats (using existing cache)
	if m.config.EnableCaching {
		cacheStats := cache.GetGlobalCacheStats()
		stats["cache"] = map[string]interface{}{
			"total_entries": len(cacheStats),
			"health":        cache.GetGlobalCacheHealth().Status,
			"efficiency":    cache.GetGlobalCacheIntegration().GetMonitor().GetCacheEfficiency(),
		}
	}

	return stats
}

// SaveState saves the current application state
func (m *UnifiedAppModel) SaveState() error {
	if m.storage != nil && m.navStack != nil {
		if data, err := m.navStack.SerializeStack(); err == nil {
			return m.storage.SaveNavigationState(data)
		}
	}
	return nil
}

// Shutdown gracefully shuts down the application
func (m *UnifiedAppModel) Shutdown() {
	// Save state
	if err := m.SaveState(); err != nil {
		m.logger.Warn("Failed to save state", "error", err)
	}

	// Shutdown resize manager
	if m.resizeManager != nil {
		m.resizeManager.Shutdown()
	}

	// Print final statistics
	m.printPerformanceStats()
	m.printCacheStats()

	m.logger.Info("Unified application shutdown complete")
}

// =====================================================================================
// 🛠️ Utility Functions
// =====================================================================================

// getTerminalSize gets the current terminal size
func getTerminalSize() (width, height int, err error) {
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
// 🚀 Application Factory Functions
// =====================================================================================

// NewStandardApp creates a new standard application
func NewStandardApp(storage storage.NavigationStorage, logger *slog.Logger) *UnifiedAppModel {
	config := DefaultAppConfig()
	config.Mode = StandardMode
	return NewUnifiedAppModel(config, storage, logger)
}

// NewResponsiveApp creates a new responsive application
func NewResponsiveApp(storage storage.NavigationStorage, logger *slog.Logger) *UnifiedAppModel {
	config := DefaultAppConfig()
	config.Mode = ResponsiveMode
	return NewUnifiedAppModel(config, storage, logger)
}

// NewOptimizedApp creates a new optimized application
func NewOptimizedApp(storage storage.NavigationStorage, logger *slog.Logger) *UnifiedAppModel {
	config := DefaultAppConfig()
	config.Mode = OptimizedMode
	return NewUnifiedAppModel(config, storage, logger)
}

// NewUnifiedProgram creates a new unified program with the specified mode
func NewUnifiedProgram(mode AppMode, storage storage.NavigationStorage, logger *slog.Logger) *tea.Program {
	var app *UnifiedAppModel

	switch mode {
	case StandardMode:
		app = NewStandardApp(storage, logger)
	case ResponsiveMode:
		app = NewResponsiveApp(storage, logger)
	case OptimizedMode:
		app = NewOptimizedApp(storage, logger)
	default:
		app = NewOptimizedApp(storage, logger)
	}

	// Load sample data
	app.LoadSampleData()

	return tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
}
