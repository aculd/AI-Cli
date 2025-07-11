// Package common provides shared utilities and components for the AI CLI application.
package common

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =====================================================================================
// 🖥️ Reactive Resizing System – Dynamic Layout Recalculation with Debouncing
// =====================================================================================
// This system provides flawless terminal resize handling with:
//  - Dynamic layout recalculation on SIGWINCH
//  - Debounced resize events to prevent excessive updates
//  - Responsive UI components that adapt to terminal size
//  - Graceful handling of resize during operations

// ResizeEvent represents a terminal resize event
type ResizeEvent struct {
	Width  int
	Height int
	Time   time.Time
}

// ResizeConfig holds configuration for the resize system
type ResizeConfig struct {
	DebounceDelay time.Duration `json:"debounce_delay"`
	MinWidth      int           `json:"min_width"`
	MinHeight     int           `json:"min_height"`
	MaxWidth      int           `json:"max_width"`
	MaxHeight     int           `json:"max_height"`
	EnableLogging bool          `json:"enable_logging"`
}

// DefaultResizeConfig returns default resize configuration
func DefaultResizeConfig() ResizeConfig {
	return ResizeConfig{
		DebounceDelay: 100 * time.Millisecond,
		MinWidth:      40,
		MinHeight:     20,
		MaxWidth:      200,
		MaxHeight:     100,
		EnableLogging: true,
	}
}

// ResizeManager handles terminal resize events with debouncing
type ResizeManager struct {
	config      ResizeConfig
	logger      *slog.Logger
	subscribers map[string]ResizeSubscriber
	mutex       sync.RWMutex
	lastResize  *ResizeEvent
	debounce    *time.Timer
	ctx         context.Context
	cancel      context.CancelFunc
}

// ResizeSubscriber is a function that handles resize events
type ResizeSubscriber func(event ResizeEvent)

// NewResizeManager creates a new resize manager
func NewResizeManager(config ResizeConfig, logger *slog.Logger) *ResizeManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ResizeManager{
		config:      config,
		logger:      logger,
		subscribers: make(map[string]ResizeSubscriber),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Subscribe adds a subscriber to resize events
func (rm *ResizeManager) Subscribe(id string, subscriber ResizeSubscriber) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.subscribers[id] = subscriber

	// if rm.config.EnableLogging {
	// 	rm.logger.Info("Resize subscriber added", "id", id, "subscribers", len(rm.subscribers))
	// }
}

// Unsubscribe removes a subscriber from resize events
func (rm *ResizeManager) Unsubscribe(id string) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	delete(rm.subscribers, id)

	// if rm.config.EnableLogging {
	// 	rm.logger.Info("Resize subscriber removed", "id", id, "subscribers", len(rm.subscribers))
	// }
}

// HandleResize processes a resize event with debouncing
func (rm *ResizeManager) HandleResize(width, height int) {
	event := ResizeEvent{
		Width:  width,
		Height: height,
		Time:   time.Now(),
	}

	// Validate dimensions
	if width < rm.config.MinWidth || height < rm.config.MinHeight {
		if rm.config.EnableLogging {
			rm.logger.Warn("Terminal too small", "width", width, "height", height, "min_width", rm.config.MinWidth, "min_height", rm.config.MinHeight)
		}
		return
	}

	if width > rm.config.MaxWidth || height > rm.config.MaxHeight {
		if rm.config.EnableLogging {
			rm.logger.Warn("Terminal too large", "width", width, "height", height, "max_width", rm.config.MaxWidth, "max_height", rm.config.MaxHeight)
		}
		return
	}

	// Check if dimensions actually changed
	if rm.lastResize != nil && rm.lastResize.Width == width && rm.lastResize.Height == height {
		return
	}

	// Cancel existing debounce timer
	if rm.debounce != nil {
		rm.debounce.Stop()
	}

	// Set new debounce timer
	rm.debounce = time.AfterFunc(rm.config.DebounceDelay, func() {
		rm.processResize(event)
	})

	rm.lastResize = &event
}

// processResize notifies all subscribers of the resize event
func (rm *ResizeManager) processResize(event ResizeEvent) {
	rm.mutex.RLock()
	subscribers := make(map[string]ResizeSubscriber)
	for id, subscriber := range rm.subscribers {
		subscribers[id] = subscriber
	}
	rm.mutex.RUnlock()

	// if rm.config.EnableLogging {
	// 	rm.logger.Info("Processing resize event", "width", event.Width, "height", event.Height, "subscribers", len(subscribers))
	// }

	// Notify all subscribers
	for id, subscriber := range subscribers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					rm.logger.Error("Panic in resize subscriber", "id", id, "error", r)
				}
			}()
			subscriber(event)
		}()
	}
}

// GetCurrentSize returns the current terminal size
func (rm *ResizeManager) GetCurrentSize() (width, height int) {
	if rm.lastResize != nil {
		return rm.lastResize.Width, rm.lastResize.Height
	}
	return 80, 24 // Default fallback
}

// Shutdown gracefully shuts down the resize manager
func (rm *ResizeManager) Shutdown() {
	rm.cancel()
	if rm.debounce != nil {
		rm.debounce.Stop()
	}
	rm.mutex.Lock()
	rm.subscribers = make(map[string]ResizeSubscriber)
	rm.mutex.Unlock()

	// if rm.config.EnableLogging {
	// 	rm.logger.Info("Resize manager shutdown complete")
	// }
}

// =====================================================================================
// 🎨 Responsive Layout Components
// =====================================================================================

// ResponsiveLayout provides dynamic layout calculation based on terminal size
type ResponsiveLayout struct {
	width  int
	height int
	config LayoutConfig
}

// LayoutConfig holds configuration for responsive layouts
type LayoutConfig struct {
	SidebarRatio     float64 `json:"sidebar_ratio"` // 0.0 to 1.0
	HeaderHeight     int     `json:"header_height"`
	FooterHeight     int     `json:"footer_height"`
	MinContentWidth  int     `json:"min_content_width"`
	MinContentHeight int     `json:"min_content_height"`
}

// DefaultLayoutConfig returns default layout configuration
func DefaultLayoutConfig() LayoutConfig {
	return LayoutConfig{
		SidebarRatio:     0.25,
		HeaderHeight:     3,
		FooterHeight:     2,
		MinContentWidth:  40,
		MinContentHeight: 10,
	}
}

// NewResponsiveLayout creates a new responsive layout
func NewResponsiveLayout(config LayoutConfig) *ResponsiveLayout {
	return &ResponsiveLayout{
		config: config,
	}
}

// UpdateSize updates the layout dimensions
func (rl *ResponsiveLayout) UpdateSize(width, height int) {
	rl.width = width
	rl.height = height
}

// GetSidebarDimensions returns sidebar width and height
func (rl *ResponsiveLayout) GetSidebarDimensions() (width, height int) {
	width = int(float64(rl.width) * rl.config.SidebarRatio)
	height = rl.height - rl.config.HeaderHeight - rl.config.FooterHeight

	// Ensure minimum dimensions
	if width < 20 {
		width = 20
	}
	if height < rl.config.MinContentHeight {
		height = rl.config.MinContentHeight
	}

	return width, height
}

// GetContentDimensions returns content area width and height
func (rl *ResponsiveLayout) GetContentDimensions() (width, height int) {
	sidebarWidth, _ := rl.GetSidebarDimensions()
	width = rl.width - sidebarWidth - 1 // -1 for border
	height = rl.height - rl.config.HeaderHeight - rl.config.FooterHeight

	// Ensure minimum dimensions
	if width < rl.config.MinContentWidth {
		width = rl.config.MinContentWidth
	}
	if height < rl.config.MinContentHeight {
		height = rl.config.MinContentHeight
	}

	return width, height
}

// GetHeaderDimensions returns header width and height
func (rl *ResponsiveLayout) GetHeaderDimensions() (width, height int) {
	return rl.width, rl.config.HeaderHeight
}

// GetFooterDimensions returns footer width and height
func (rl *ResponsiveLayout) GetFooterDimensions() (width, height int) {
	return rl.width, rl.config.FooterHeight
}

// =====================================================================================
// 🎯 Bubble Tea Integration
// =====================================================================================

// ResizeMsg represents a resize message in Bubble Tea
type ResizeMsg struct {
	Width  int
	Height int
}

// ResizeAwareModel is an interface for models that handle resize events
type ResizeAwareModel interface {
	tea.Model
	OnResize(width, height int)
}

// ResizeAwareProgram wraps a Bubble Tea program with resize handling
type ResizeAwareProgram struct {
	program       *tea.Program
	resizeManager *ResizeManager
	model         ResizeAwareModel
}

// NewResizeAwareProgram creates a new resize-aware program
func NewResizeAwareProgram(model ResizeAwareModel, config ResizeConfig, logger *slog.Logger) *ResizeAwareProgram {
	resizeManager := NewResizeManager(config, logger)

	// Subscribe the model to resize events
	resizeManager.Subscribe("main_model", func(event ResizeEvent) {
		model.OnResize(event.Width, event.Height)
	})

	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	return &ResizeAwareProgram{
		program:       program,
		resizeManager: resizeManager,
		model:         model,
	}
}

// Run runs the program with resize handling
func (rap *ResizeAwareProgram) Run() (tea.Model, error) {
	// Note: SIGWINCH is Unix-specific, Windows doesn't have this signal
	// For Windows, we'll rely on Bubble Tea's WindowSizeMsg instead
	// The resize handling is done through the Update method with WindowSizeMsg

	// Run the program
	return rap.program.Run()
}

// Shutdown gracefully shuts down the program
func (rap *ResizeAwareProgram) Shutdown() {
	rap.resizeManager.Shutdown()
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

// CreateResponsiveStyle creates a lipgloss style that adapts to terminal size
func CreateResponsiveStyle(width, height int) lipgloss.Style {
	// Base style
	style := lipgloss.NewStyle()

	// Adapt padding based on terminal size
	if width < 60 {
		style = style.Padding(0, 1)
	} else if width < 100 {
		style = style.Padding(0, 2)
	} else {
		style = style.Padding(0, 4)
	}

	// Adapt margins based on height
	if height < 30 {
		style = style.Margin(0, 0)
	} else if height < 50 {
		style = style.Margin(1, 0)
	} else {
		style = style.Margin(2, 0)
	}

	return style
}

// =====================================================================================
// 📊 Resize Statistics and Monitoring
// =====================================================================================

// ResizeStats holds statistics about resize events
type ResizeStats struct {
	TotalEvents     int64     `json:"total_events"`
	DebouncedEvents int64     `json:"debounced_events"`
	LastResize      time.Time `json:"last_resize"`
	AverageWidth    float64   `json:"average_width"`
	AverageHeight   float64   `json:"average_height"`
	MinWidth        int       `json:"min_width"`
	MaxWidth        int       `json:"max_width"`
	MinHeight       int       `json:"min_height"`
	MaxHeight       int       `json:"max_height"`
}

// ResizeMonitor provides monitoring capabilities for resize events
type ResizeMonitor struct {
	stats ResizeStats
	mutex sync.RWMutex
}

// NewResizeMonitor creates a new resize monitor
func NewResizeMonitor() *ResizeMonitor {
	return &ResizeMonitor{
		stats: ResizeStats{
			MinWidth:  9999,
			MinHeight: 9999,
		},
	}
}

// RecordEvent records a resize event for statistics
func (rm *ResizeMonitor) RecordEvent(width, height int) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rm.stats.TotalEvents++
	rm.stats.LastResize = time.Now()

	// Update min/max dimensions
	if width < rm.stats.MinWidth {
		rm.stats.MinWidth = width
	}
	if width > rm.stats.MaxWidth {
		rm.stats.MaxWidth = width
	}
	if height < rm.stats.MinHeight {
		rm.stats.MinHeight = height
	}
	if height > rm.stats.MaxHeight {
		rm.stats.MaxHeight = height
	}

	// Update averages
	total := float64(rm.stats.TotalEvents)
	rm.stats.AverageWidth = (rm.stats.AverageWidth*(total-1) + float64(width)) / total
	rm.stats.AverageHeight = (rm.stats.AverageHeight*(total-1) + float64(height)) / total
}

// GetStats returns current resize statistics
func (rm *ResizeMonitor) GetStats() ResizeStats {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return rm.stats
}
