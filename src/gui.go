// gui.go - Comprehensive GUI Implementation with Modal Integration
// This GUI integrates with the existing modal system, view states, and unified app model
// to provide a complete TUI experience with navigation, modals, and responsive design

package main

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"aichat/src/components/chat"
	"aichat/src/components/common"
	"aichat/src/components/modals"
	"aichat/src/services/storage"
	"aichat/src/types"

	"aichat/src/components/modals/dialogs"
	"aichat/src/navigation"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =====================================================================================
// 🎯 GUI Application Model
// =====================================================================================

// GUIAppModel represents the complete GUI application with modal integration
type GUIAppModel struct {
	// Core application
	app *UnifiedAppModel

	// Modal system
	modalManager *modals.ModalManager
	modalActive  bool

	// Navigation state
	navStack *navigation.NavigationStack // Use NavigationStack for navigation

	// UI state
	focus     string // "main", "modal", "sidebar", "chat"
	showHelp  bool
	showStats bool

	// Styling
	styles *GUIStyles

	// Performance tracking
	renderCount int64
	lastRender  time.Time

	// Store pending cleanup functions for exit
	pendingExitCleanups []func()
}

// GUIStyles contains all styling for the GUI
type GUIStyles struct {
	// Container styles
	mainContainer lipgloss.Style
	headerStyle   lipgloss.Style
	footerStyle   lipgloss.Style
	sidebarStyle  lipgloss.Style
	contentStyle  lipgloss.Style

	// Text styles
	titleStyle    lipgloss.Style
	subtitleStyle lipgloss.Style
	textStyle     lipgloss.Style
	helpStyle     lipgloss.Style

	// Interactive styles
	selectedStyle lipgloss.Style
	focusStyle    lipgloss.Style
	disabledStyle lipgloss.Style

	// Status styles
	successStyle lipgloss.Style
	errorStyle   lipgloss.Style
	warningStyle lipgloss.Style
	infoStyle    lipgloss.Style
}

// NewGUIStyles creates a new set of GUI styles
func NewGUIStyles() *GUIStyles {
	return &GUIStyles{
		// Container styles
		mainContainer: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2),

		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1),

		footerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("235")).
			Padding(0, 1),

		sidebarStyle: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1),

		contentStyle: lipgloss.NewStyle().
			Padding(0, 1),

		// Text styles
		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Bold(true),

		subtitleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")),

		textStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true),

		// Interactive styles
		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Bold(true),

		focusStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")),

		disabledStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Strikethrough(true),

		// Status styles
		successStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")),

		warningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")),

		infoStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")),
	}
}

// =====================================================================================
// 🚀 GUI Application Factory
// =====================================================================================

// NewGUIAppModel creates a new GUI application model
func NewGUIAppModel(config *AppConfig, storage storage.NavigationStorage, logger *slog.Logger) *GUIAppModel {
	// Create underlying app
	app := NewUnifiedAppModel(config, storage, logger)

	// Create modal manager
	modalManager := &modals.ModalManager{}

	// Create navigation stack with main menu as root
	mainMenu := types.NewMenuViewState(types.MainMenu, nil, nil, nil)
	navStack := navigation.NewNavigationStack(mainMenu)

	// Create GUI model
	gui := &GUIAppModel{
		app:          app,
		modalManager: modalManager,
		modalActive:  false,
		navStack:     navStack,
		focus:        "main",
		showHelp:     false,
		showStats:    false,
		styles:       NewGUIStyles(),
		renderCount:  0,
		lastRender:   time.Now(),
	}

	// Load sample data
	gui.LoadSampleData()

	return gui
}

// =====================================================================================
// 🎮 Bubble Tea Interface Implementation
// =====================================================================================

// Init initializes the GUI application
func (m *GUIAppModel) Init() tea.Cmd {
	// Initialize underlying app
	m.app.Init()

	// Load initial view state
	if m.navStack != nil {
		if current := m.navStack.Top(); current != nil {
			// The actual current view is now m.navStack.Top()
		}
	}

	return nil
}

// Update handles messages and updates the GUI application
func (m *GUIAppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.renderCount++

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.app.OnResize(msg.Width, msg.Height)
		return m, nil
	case common.ResizeMsg:
		m.app.OnResize(msg.Width, msg.Height)
		return m, nil
	}

	// Update underlying app
	m.app.Update(msg)

	// Update current view
	if m.navStack != nil {
		if current := m.navStack.Top(); current != nil {
			current.Update(msg)
		}
	}

	// Update modal if active
	if m.modalActive {
		if current := m.modalManager.Current(); current != nil {
			current.Update(msg)
		}
	}

	m.lastRender = time.Now()
	return m, nil
}

// View renders the GUI application
func (m *GUIAppModel) View() string {
	if m.app.width < 40 || m.app.height < 10 {
		return m.renderMinimalView()
	}

	// Get layout dimensions
	headerWidth, _ := m.app.layout.GetHeaderDimensions()
	sidebarWidth, sidebarHeight := m.app.layout.GetSidebarDimensions()
	contentWidth, contentHeight := m.app.layout.GetContentDimensions()
	footerWidth, _ := m.app.layout.GetFooterDimensions()

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

	// Apply modal overlay if active
	if m.modalActive {
		mainContent = m.renderModalOverlay(mainContent, contentWidth, contentHeight)
	}

	// Combine all sections
	view := fmt.Sprintf("%s\n%s\n%s", header, mainContent, footer)

	// Apply container style
	return m.styles.mainContainer.Width(m.app.width).Height(m.app.height).Render(view)
}

// =====================================================================================
// 🎯 Event Handlers
// =====================================================================================

// handleKeyPress handles keyboard input
func (m *GUIAppModel) handleKeyPress(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyStr := key.String()

	// Handle modal keys first
	if m.modalActive {
		return m.handleModalKeyPress(key)
	}

	// State focus logic: query current view, get control sets, execute matching key function
	if m.navStack != nil {
		if current := m.navStack.Top(); current != nil {
			for _, ctrlSet := range current.GetControlSets() {
				for _, ctrl := range ctrlSet.Controls {
					if ctrl.Key == keyStr && ctrl.Action != nil {
						if ctrl.Action() {
							return m, nil
						}
					}
				}
			}
		}
	}

	switch keyStr {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "tab":
		m.cycleFocus()
		return m, nil
	case "h", "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "s":
		m.showStats = !m.showStats
		return m, nil
	case "m":
		return m, m.showMainMenu()
	case "esc":
		return m, m.goBack()
	case "enter":
		return m, m.handleEnter()
	case "up", "k":
		return m, m.handleUp()
	case "down", "j":
		return m, m.handleDown()
	case "left":
		return m, m.handleLeft()
	case "right", "l":
		return m, m.handleRight()
	case "ctrl+q":
		return m, m.startExitFlow() // or pass cleanup functions as needed
	}

	return m, nil
}

// handleModalKeyPress handles keyboard input when a modal is active
func (m *GUIAppModel) handleModalKeyPress(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyStr := key.String()

	if current := m.modalManager.Current(); current != nil {
		switch keyStr {
		case "enter":
			// If it's a confirmation modal, trigger OnSelect for the selected option
			if conf, ok := current.(*dialogs.ConfirmationModal); ok {
				if conf.Selected >= 0 && conf.Selected < len(conf.Options) {
					conf.Options[conf.Selected].OnSelect()
					if conf.CloseSelf != nil {
						conf.CloseSelf()
					}
				}
				return m, m.closeModal()
			}
		}
	}

	switch keyStr {
	case "esc":
		return m, m.closeModal()
	}

	return m, nil
}

// =====================================================================================
// 🎯 Navigation Methods
// =====================================================================================

// pushView adds a new view to the navigation stack
func (m *GUIAppModel) pushView(view types.ViewState) {
	m.navStack.Push(view)
}

// popView removes the current view and returns to the previous one
func (m *GUIAppModel) popView() {
	m.navStack.Pop()
}

// replaceView replaces the current view with a new one
func (m *GUIAppModel) replaceView(view types.ViewState) {
	m.navStack.ReplaceTop(view)
}

// =====================================================================================
// 🎯 Modal Methods
// =====================================================================================

// showModal displays a modal dialog
func (m *GUIAppModel) showModal(modalType string, data interface{}) tea.Cmd {
	return func() tea.Msg {
		// Simple modal implementation
		m.modalActive = true
		return nil
	}
}

// closeModal closes the current modal
func (m *GUIAppModel) closeModal() tea.Cmd {
	return func() tea.Msg {
		if m.modalActive {
			m.modalActive = false
		}
		return nil
	}
}

// =====================================================================================
// 🎯 Input Handlers
// =====================================================================================

// handleEnter handles the enter key press
func (m *GUIAppModel) handleEnter() tea.Cmd {
	return func() tea.Msg {
		if m.navStack != nil {
			if current := m.navStack.Top(); current != nil {
				switch current.(type) {
				case *types.MenuViewState:
					if menuView, ok := current.(*types.MenuViewState); ok {
						if entries := types.Menus[menuView.Type].Entries; len(entries) > 0 {
							if menuView.Selected < len(entries) {
								option := entries[menuView.Selected]

								// Execute action if present
								if option.Action != nil {
									if option.Action() {
										return nil // Action handled
									}
								}

								// Navigate to next menu if specified
								if option.Next != types.MainMenu {
									nextView := &types.MenuViewState{
										Type:     option.Next,
										Selected: 0,
									}
									m.pushView(nextView)
								}
							}
						}
					}
				case *types.ChatViewState:
					// Handle chat input
				case *types.ModalViewState:
					// Handle modal input
				}
			}
		}
		return nil
	}
}

// handleUp handles the up arrow key press
func (m *GUIAppModel) handleUp() tea.Cmd {
	return func() tea.Msg {
		if m.navStack != nil {
			if current := m.navStack.Top(); current != nil {
				switch current.(type) {
				case *types.MenuViewState:
					if menuView, ok := current.(*types.MenuViewState); ok {
						if menuView.Selected > 0 {
							menuView.Selected--
						}
					}
				case *types.ChatViewState:
					// Handle chat up
				case *types.ModalViewState:
					// Handle modal up
				}
			}
		}
		return nil
	}
}

// handleDown handles the down arrow key press
func (m *GUIAppModel) handleDown() tea.Cmd {
	return func() tea.Msg {
		if m.navStack != nil {
			if current := m.navStack.Top(); current != nil {
				switch current.(type) {
				case *types.MenuViewState:
					if menuView, ok := current.(*types.MenuViewState); ok {
						if entries := types.Menus[menuView.Type].Entries; len(entries) > 0 {
							if menuView.Selected < len(entries)-1 {
								menuView.Selected++
							}
						}
					}
				case *types.ChatViewState:
					// Handle chat down
				case *types.ModalViewState:
					// Handle modal down
				}
			}
		}
		return nil
	}
}

// handleLeft handles the left arrow key press
func (m *GUIAppModel) handleLeft() tea.Cmd {
	return func() tea.Msg {
		// Handle left navigation
		return nil
	}
}

// handleRight handles the right arrow key press
func (m *GUIAppModel) handleRight() tea.Cmd {
	return func() tea.Msg {
		// Handle right navigation
		return nil
	}
}

// =====================================================================================
// 🎯 Menu Handlers
// =====================================================================================

// handleMenuSelection handles menu item selection
func (m *GUIAppModel) handleMenuSelection(menuView *types.MenuViewState) tea.Msg {
	entries := types.Menus[menuView.Type].Entries
	if menuView.Selected < len(entries) {
		option := entries[menuView.Selected]

		// Execute action if present
		if option.Action != nil {
			if option.Action() {
				return nil // Action handled
			}
		}

		// Navigate to next menu if specified
		if option.Next != types.MainMenu {
			nextView := &types.MenuViewState{
				Type:     option.Next,
				Selected: 0,
			}
			m.pushView(nextView)
		}
	}
	return nil
}

// showMainMenu shows the main menu
func (m *GUIAppModel) showMainMenu() tea.Cmd {
	return func() tea.Msg {
		mainMenu := &types.MenuViewState{
			Type:     types.MainMenu,
			Selected: 0,
		}
		m.replaceView(mainMenu)
		return nil
	}
}

// goBack goes back to the previous view
func (m *GUIAppModel) goBack() tea.Cmd {
	return func() tea.Msg {
		m.popView()
		return nil
	}
}

// =====================================================================================
// 🎯 Focus Management
// =====================================================================================

// cycleFocus cycles through focus areas
func (m *GUIAppModel) cycleFocus() {
	switch m.focus {
	case "main":
		m.focus = "sidebar"
	case "sidebar":
		m.focus = "chat"
	case "chat":
		m.focus = "main"
	}
}

// =====================================================================================
// 🎯 Rendering Methods
// =====================================================================================

// renderMinimalView renders a minimal view for very small terminals
func (m *GUIAppModel) renderMinimalView() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Align(lipgloss.Center, lipgloss.Center).
		Width(m.app.width).
		Height(m.app.height).
		Render("Terminal too small for GUI application")
}

// renderHeader renders the application header
func (m *GUIAppModel) renderHeader(width int) string {
	title := "AI CLI - GUI Interface"
	mode := m.app.getModeString()
	status := fmt.Sprintf("Size: %dx%d | Focus: %s", m.app.width, m.app.height, m.focus)

	content := fmt.Sprintf("%s | %s | %s", title, mode, status)

	return m.styles.headerStyle.Width(width).Render(content)
}

// renderSidebar renders the sidebar
func (m *GUIAppModel) renderSidebar(width, height int) string {
	// Get sidebar content from underlying app
	sidebarContent := m.app.sidebar.View()

	// Apply focus styling
	if m.focus == "sidebar" {
		sidebarContent = m.styles.focusStyle.Render(sidebarContent)
	}

	return m.styles.sidebarStyle.Width(width).Height(height).Render(sidebarContent)
}

// renderContent renders the main content area
func (m *GUIAppModel) renderContent(width, height int) string {
	var content string

	// Render based on current view type
	if m.navStack != nil {
		if current := m.navStack.Top(); current != nil {
			switch current.(type) {
			case *types.MenuViewState:
				content = m.renderMenuView(current.(*types.MenuViewState), width, height)
			case *types.ChatViewState:
				content = m.renderChatView(current.(*types.ChatViewState), width, height)
			case *types.ModalViewState:
				content = m.renderModalView(current.(*types.ModalViewState), width, height)
			default:
				content = m.renderDefaultView(width, height)
			}
		}
	}

	// Apply focus styling
	if m.focus == "main" {
		content = m.styles.focusStyle.Render(content)
	}

	return m.styles.contentStyle.Width(width).Height(height).Render(content)
}

// AsciiArtView: renders fixed-width ASCII art
func asciiArtView() string {
	return `  
  █████╗ ██╗ ██████╗██╗  ██╗ █████╗ ████████╗
 ██╔══██╗██║██╔════╝██║  ██║██╔══██╗╚══██╔══╝
 ███████║██║██║     ███████║███████║   ██║   
 ██╔══██║██║██║     ██╔══██║██╔══██║   ██║   
 ██║  ██║██║╚██████╗██║  ██║██║  ██║   ██║   
 ╚═╝  ╚═╝╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   `
}

// MenuBoxView: renders menu heading and items
func menuBoxView(m *GUIAppModel, menuView *types.MenuViewState, width, height int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("245")).
		Padding(1, 4).
		Width(max(300, width)).
		Height(max(400, height)).
		Align(lipgloss.Center)

	heading := lipgloss.NewStyle().Width(max(300, width)).Align(lipgloss.Center).Render(m.getMenuTitle(menuView.Type))

	var menuLines []string
	for i, option := range types.Menus[menuView.Type].Entries {
		line := option.Text
		if option.Description != "" {
			line += " - " + option.Description
		}
		var itemStyle lipgloss.Style
		if option.Description != "" {
			itemStyle = lipgloss.NewStyle().Width(max(300, width)).Align(lipgloss.Left)
		} else {
			itemStyle = lipgloss.NewStyle().Width(max(300, width)).Align(lipgloss.Center)
		}
		if i == menuView.Selected {
			line = m.styles.selectedStyle.Render(line)
		} else {
			line = m.styles.textStyle.Render(line)
		}
		if option.Disabled {
			line = m.styles.disabledStyle.Render(line)
		}
		menuLines = append(menuLines, itemStyle.Render(line))
	}
	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, heading, "", lipgloss.JoinVertical(lipgloss.Left, menuLines...)))
}

// ControlInfoView: renders control info, left-aligned, matches menu box width
func controlInfoView(m *GUIAppModel, menuType types.MenuType, width int) string {
	var controlInfo string
	if meta, exists := types.MenuMetas[menuType]; exists {
		if ci, exists := types.ControlInfoMap[meta.ControlInfoType]; exists {
			for _, controlLine := range ci.Lines {
				controlInfo += m.styles.helpStyle.Width(max(300, width)).Align(lipgloss.Left).Render(controlLine) + "\n"
			}
		}
	}
	return controlInfo
}

// Refactored renderMenuView to use the three components
func (m *GUIAppModel) renderMenuView(menuView *types.MenuViewState, width, height int) string {
	if len(types.Menus[menuView.Type].Entries) == 0 {
		return m.styles.errorStyle.Render("Invalid menu type")
	}
	ascii := asciiArtView()
	menuBox := menuBoxView(m, menuView, width, height)
	controlInfo := controlInfoView(m, menuView.Type, width)
	return lipgloss.JoinVertical(
		lipgloss.Center,
		lipgloss.PlaceHorizontal(width, lipgloss.Center, ascii),
		lipgloss.PlaceHorizontal(width, lipgloss.Center, menuBox),
		lipgloss.PlaceHorizontal(width, lipgloss.Left, controlInfo),
	)
}

// renderChatView renders a chat view
func (m *GUIAppModel) renderChatView(chatView *types.ChatViewState, width, height int) string {
	// Get chat content from underlying app
	var chatContent string
	if responsiveChat, ok := m.app.chatView.(*chat.ResponsiveChatView); ok {
		chatContent = responsiveChat.View()
	} else {
		chatContent = "Chat view not available"
	}
	return chatContent
}

// renderModalView renders a modal view
func (m *GUIAppModel) renderModalView(modalView *types.ModalViewState, width, height int) string {
	return "Modal view content"
}

// renderDefaultView renders the default view
func (m *GUIAppModel) renderDefaultView(width, height int) string {
	return m.styles.textStyle.Render("Welcome to AI CLI GUI")
}

// renderModalOverlay renders a modal overlay
func (m *GUIAppModel) renderModalOverlay(content string, width, height int) string {
	if m.modalManager != nil {
		if current := m.modalManager.Current(); current != nil {
			// If it's a confirmation modal, render only the dialog centered, no background box
			if conf, ok := current.(*dialogs.ConfirmationModal); ok {
				return conf.ViewRegion(width, height)
			}
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
	}
	return content
}

// renderFooter renders the application footer
func (m *GUIAppModel) renderFooter(width int) string {
	var helpLines []string

	// Add basic help
	helpLines = append(helpLines, "Tab: Switch Focus | Esc: Back | q: Quit")

	// Add focus-specific help
	switch m.focus {
	case "main":
		helpLines = append(helpLines, "↑↓: Navigate | Enter: Select | h: Help | s: Stats")
	case "sidebar":
		helpLines = append(helpLines, "↑↓: Navigate | Enter: Select")
	case "chat":
		helpLines = append(helpLines, "Type to chat | Enter: Send")
	}

	// Add performance info if stats are shown
	if m.showStats {
		stats := m.app.GetPerformanceStats()
		if appStats, ok := stats["application"].(map[string]interface{}); ok {
			if renderCount, ok := appStats["render_count"].(int64); ok {
				helpLines = append(helpLines, fmt.Sprintf("Renders: %d", renderCount))
			}
		}
	}

	help := strings.Join(helpLines, " | ")
	return m.styles.footerStyle.Width(width).Render(help)
}

// =====================================================================================
// 🎯 Utility Methods
// =====================================================================================

// getMenuTitle gets the title for a menu type
func (m *GUIAppModel) getMenuTitle(menuType types.MenuType) string {
	switch menuType {
	case types.MainMenu:
		return "Main Menu"
	case types.ChatsMenu:
		return "Chats"
	case types.PromptsMenu:
		return "Prompts"
	case types.ModelsMenu:
		return "Models"
	case types.APIKeyMenu:
		return "API Keys"
	case types.HelpMenu:
		return "Help"
	case types.ExitMenu:
		return "Exit"
	default:
		return "Menu"
	}
}

// LoadSampleData loads sample data for demonstration
func (m *GUIAppModel) LoadSampleData() {
	// Load sample data into underlying app
	m.app.LoadSampleData()
}

// GetPerformanceStats returns performance statistics
func (m *GUIAppModel) GetPerformanceStats() map[string]interface{} {
	appStats := m.app.GetPerformanceStats()

	// Add GUI-specific stats
	guiStats := map[string]interface{}{
		"gui_renders":    m.renderCount,
		"last_render_ms": time.Since(m.lastRender).Milliseconds(),
		"modal_active":   m.modalActive,
		"current_focus":  m.focus,
		"view_history":   m.navStack.Len(), // Use navStack.Len()
	}

	appStats["gui"] = guiStats
	return appStats
}

// Shutdown gracefully shuts down the GUI application
func (m *GUIAppModel) Shutdown() {
	// Save state
	if m.app.storage != nil {
		_ = m.app.SaveState()
	}

	// Shutdown underlying app
	m.app.Shutdown()

	// Print final statistics
	m.printFinalStats()
}

// printFinalStats prints final application statistics
func (m *GUIAppModel) printFinalStats() {
	stats := m.GetPerformanceStats()

	if guiStats, ok := stats["gui"].(map[string]interface{}); ok {
		if renderCount, ok := guiStats["gui_renders"].(int64); ok {
			m.app.logger.Info("GUI Statistics",
				"total_renders", renderCount,
				"modal_sessions", guiStats["modal_active"],
				"focus_changes", guiStats["current_focus"],
			)
		}
	}
}

// startExitFlow shows the quit confirmation modal and stores cleanup functions to run if confirmed
func (m *GUIAppModel) startExitFlow(cleanupFuncs ...func()) tea.Cmd {
	return func() tea.Msg {
		m.pendingExitCleanups = cleanupFuncs
		m.showQuitConfirmation()
		return nil
	}
}

// showQuitConfirmation launches the confirmation modal
func (m *GUIAppModel) showQuitConfirmation() {
	modal := dialogs.NewConfirmationModal(
		"Are you sure you want to quit?",
		[]modals.ModalOption{
			{Label: "Yes", OnSelect: func() { m.executeExitCleanupsAndQuit() }},
			{Label: "No", OnSelect: func() { m.closeModal() }},
		},
		func() { m.closeModal() }, // Correct signature for CloseSelfFunc
	)
	m.modalManager.Push(modal)
	m.modalActive = true
}

// executeExitCleanupsAndQuit runs all pending cleanup functions and quits
func (m *GUIAppModel) executeExitCleanupsAndQuit() {
	for _, fn := range m.pendingExitCleanups {
		if fn != nil {
			fn()
		}
	}
	m.modalActive = false
	// Bubble Tea quit command or set a flag for your main loop
}

// =====================================================================================
// 🚀 Application Factory Functions
// =====================================================================================

// NewStandardGUI creates a new standard GUI application
func NewStandardGUI(storage storage.NavigationStorage, logger *slog.Logger) *GUIAppModel {
	config := DefaultAppConfig()
	config.Mode = StandardMode
	return NewGUIAppModel(config, storage, logger)
}

// NewResponsiveGUI creates a new responsive GUI application
func NewResponsiveGUI(storage storage.NavigationStorage, logger *slog.Logger) *GUIAppModel {
	config := DefaultAppConfig()
	config.Mode = ResponsiveMode
	return NewGUIAppModel(config, storage, logger)
}

// NewOptimizedGUI creates a new optimized GUI application
func NewOptimizedGUI(storage storage.NavigationStorage, logger *slog.Logger) *GUIAppModel {
	config := DefaultAppConfig()
	config.Mode = OptimizedMode
	return NewGUIAppModel(config, storage, logger)
}

// NewGUIProgram creates a new GUI program with the specified mode
func NewGUIProgram(mode AppMode, storage storage.NavigationStorage, logger *slog.Logger) *tea.Program {
	var gui *GUIAppModel

	switch mode {
	case StandardMode:
		gui = NewStandardGUI(storage, logger)
	case ResponsiveMode:
		gui = NewResponsiveGUI(storage, logger)
	case OptimizedMode:
		gui = NewOptimizedGUI(storage, logger)
	default:
		gui = NewOptimizedGUI(storage, logger)
	}

	return tea.NewProgram(gui, tea.WithAltScreen(), tea.WithMouseCellMotion())
}

// Helper for min width/height
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
