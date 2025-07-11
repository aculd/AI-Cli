
// === FILENAME: optimized_app.go ===
// Package app provides optimized application components
package app

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"aichat/src/components/common"
	"aichat/src/components/chat"
	"aichat/src/components/sidebar"
	"aichat/src/models"
	"aichat/src/types"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppMode determines rendering mode
type AppMode int

const (
	StandardMode AppMode = iota
	ResponsiveMode
	OptimizedMode
)

// AppConfig holds app configuration
type AppConfig struct {
	Mode          AppMode
	EnableCaching bool
	EnableLogging bool
	DefaultWidth  int
	DefaultHeight int
	MinWidth      int
	MinHeight     int
}

// DefaultAppConfig returns default config
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

// OptimizedApp represents optimized application
type OptimizedApp struct {
	resizeMgr *common.ResizeManager
	layout    *common.ResponsiveLayout
	optimizer *common.ANSIOptimizer
	monitor   *common.ANSIPerformanceMonitor
	sidebar   *sidebar.ResponsiveSidebar
	chatView  interface{}
	width     int
	height    int
	style     lipgloss.Style
	header    lipgloss.Style
	footer    lipgloss.Style
	logger    *slog.Logger
	lastRender time.Time
	renderCnt int64
	viewState types.ViewState
}

// NewOptimizedApp creates new optimized app
func NewOptimizedApp(logger *slog.Logger) *OptimizedApp {
	resizeCfg := common.DefaultResizeConfig()
	resizeMgr := common.NewResizeManager(resizeCfg, logger)
	layout := common.NewResponsiveLayout(common.DefaultLayoutConfig())

	ansiCfg := common.DefaultANSIConfig()
	optimizer := common.NewANSIOptimizer(ansiCfg)
	monitor := common.NewANSIPerformanceMonitor()

	sidebar := sidebar.NewResponsiveSidebar()
	var chatView interface{}
	if chatView = chat.NewOptimizedChatView(); chatView == nil {
		chatView = chat.NewResponsiveChatView()
	}

	mainMenu := types.NewMenuViewState(types.MainMenu)
	app := &OptimizedApp{
		resizeMgr: resizeMgr,
		layout:    layout,
		optimizer: optimizer,
		monitor:   monitor,
		sidebar:   sidebar,
		chatView:  chatView,
		width:     80,
		height:    24,
		style:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")),
		header:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Padding(0, 1),
		footer:    lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 1),
		logger:    logger,
		lastRender: time.Now(),
		viewState: mainMenu,
	}

	resizeMgr.Subscribe("sidebar", func(e common.ResizeEvent) {
		sidebar.OnResize(e.Width, e.Height)
	})

	resizeMgr.Subscribe("chat_view", func(e common.ResizeEvent) {
		if cv, ok := chatView.(*chat.OptimizedChatView); ok {
			cv.OnResize(e.Width, e.Height)
		} else if cv, ok := chatView.(*chat.ResponsiveChatView); ok {
			cv.OnResize(e.Width, e.Height)
		}
	})

	resizeMgr.Subscribe("main_app", func(e common.ResizeEvent) {
		app.OnResize(e.Width, e.Height)
	})

	return app
}

func (a *OptimizedApp) Init() tea.Cmd {
	if w, h, err := getTermSize(); err == nil {
		a.OnResize(w, h)
	}
	return nil
}

func (a *OptimizedApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	a.renderCnt++
	start := time.Now()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKey(msg)
	case tea.WindowSizeMsg:
		a.OnResize(msg.Width, msg.Height)
		return a, nil
	case common.ResizeMsg:
		a.OnResize(msg.Width, msg.Height)
		return a, nil
	}

	if top := a.viewState; top != nil {
		newState, cmd := top.Update(msg)
		if newState != top {
			a.viewState = newState
		}
		if cmd != nil {
			return a, cmd
		}
	}

	a.sidebar.Update(msg)
	if cv, ok := a.chatView.(*chat.ResponsiveChatView); ok {
		cv.Update(msg)
	} else if cv, ok := a.chatView.(*chat.OptimizedChatView); ok {
		cv.Update(msg)
	}

	a.lastRender = time.Now()
	if a.monitor != nil {
		latency := time.Since(start).Milliseconds()
		a.monitor.RecordRender(false, float64(latency))
	}

	return a, nil
}

func (a *OptimizedApp) View() string {
	if a.width < a.MinWidth || a.height < a.MinHeight {
		return a.minView()
	}

	w, h := a.layout.GetHeaderDimensions()
	header := a.renderHeader(w)
	sidebar := a.sidebar.View()
	content := a.viewState.View()
	footer := a.renderFooter(w)

	main := lipgloss.JoinHorizontal(lipgloss.Left, sidebar, lipgloss.NewStyle().Width(1).Render("│"), content)
	view := fmt.Sprintf("%s\n%s\n%s", header, main, footer)
	styled := a.style.Width(a.width).Height(a.height).Render(view)

	if a.optimizer != nil {
		return a.optimizer.OptimizeRender(styled, a.width, a.height)
	}
	return styled
}

func (a *OptimizedApp) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "ctrl+c", "q":
		return a, tea.Quit
	case "tab":
		return a, nil
	case "h", "?":
		return a, nil
	case "s":
		return a, nil
	case "m":
		return a, nil
	case "esc":
		return a, nil
	case "r":
		a.OnResize(a.width, a.height)
		return a, nil
	case "p":
		a.printStats()
		return a, nil
	}
	return a, nil
}

func (a *OptimizedApp) OnResize(w, h int) {
	a.width, a.height = w, h
	a.layout.UpdateSize(w, h)
	a.updateStyles()
	a.resizeMgr.HandleResize(w, h)
	a.logger.Info("Resized", "width", w, "height", h)
}

func (a *OptimizedApp) minView() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Align(lipgloss.Center, lipgloss.Center).
		Width(a.width).
		Height(a.height).
		Render("Terminal too small")
}

func (a *OptimizedApp) renderHeader(w int) string {
	title := fmt.Sprintf("AI CLI - %s", a.getMode())
	status := fmt.Sprintf("Size: %dx%d", a.width, a.height)
	return a.header.Width(w).Render(fmt.Sprintf("%s | %s", title, status))
}

func (a *OptimizedApp) renderFooter(w int) string {
	help := "Tab: Switch | Esc: Back | q: Quit"
	return a.footer.Width(w).Render(help)
}

func (a *OptimizedApp) updateStyles() {
	if a.width < 60 {
		a.style = a.style.BorderStyle(lipgloss.NormalBorder())
	} else if a.width < 100 {
		a.style = a.style.BorderStyle(lipgloss.RoundedBorder())
	} else {
		a.style = a.style.BorderStyle(lipgloss.RoundedBorder())
	}
}

func (a *OptimizedApp) getMode() string {
	switch a.Mode {
	case StandardMode: return "Standard"
	case ResponsiveMode: return "Responsive"
	case OptimizedMode: return "Optimized"
	default: return "Unknown"
	}
}

func (a *OptimizedApp) printStats() {
	if a.monitor != nil {
		stats := a.monitor.GetStats()
		a.logger.Info("Stats",
			"renders", stats.TotalRenders,
			"partial", stats.PartialUpdates,
			"full", stats.FullUpdates,
			"opt_rate", fmt.Sprintf("%.1f%%", stats.OptimizationRate),
			"latency", fmt.Sprintf("%.1fms", stats.AverageLatency),
			"count", a.renderCnt,
			"last", time.Since(a.lastRender).Milliseconds(),
		)
	}
}

func getTermSize() (int, int, error) {
	if w := os.Getenv("COLUMNS"); w != "" {
		if h := os.Getenv("LINES"); h != "" {
			return 80, 24, nil
		}
	}
	return 80, 24, nil
}

// === FILENAME: responsive_app.go ===
// Package app provides responsive application components
package app

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"aichat/src/components/common"
	"aichat/src/components/chat"
	"aichat/src/components/sidebar"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ResponsiveApp struct {
	resizeMgr *common.ResizeManager
	layout    *common.ResponsiveLayout
	sidebar   *sidebar.ResponsiveSidebar
	chatView  *chat.ResponsiveChatView
	width     int
	height    int
	style     lipgloss.Style
	header    lipgloss.Style
	footer    lipgloss.Style
	logger    *slog.Logger
}

func NewResponsiveApp(logger *slog.Logger) *ResponsiveApp {
	resizeCfg := common.DefaultResizeConfig()
	resizeMgr := common.NewResizeManager(resizeCfg, logger)
	layout := common.NewResponsiveLayout(common.DefaultLayoutConfig())
	sidebar := sidebar.NewResponsiveSidebar()
	chatView := chat.NewResponsiveChatView()

	app := &ResponsiveApp{
		resizeMgr: resizeMgr,
		layout:    layout,
		sidebar:   sidebar,
		chatView:  chatView,
		width:     80,
		height:    24,
		style:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")),
		header:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Padding(0, 1),
		footer:    lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 1),
		logger:    logger,
	}

	resizeMgr.Subscribe("sidebar", func(e common.ResizeEvent) {
		sidebar.OnResize(e.Width, e.Height)
	})

	resizeMgr.Subscribe("chat_view", func(e common.ResizeEvent) {
		chatView.OnResize(e.Width, e.Height)
	})

	resizeMgr.Subscribe("main_app", func(e common.ResizeEvent) {
		app.OnResize(e.Width, e.Height)
	})

	return app
}

func (a *ResponsiveApp) Init() tea.Cmd {
	if w, h, err := getTermSize(); err == nil {
		a.OnResize(w, h)
	}
	return nil
}

func (a *ResponsiveApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKey(msg)
	case tea.WindowSizeMsg:
		a.OnResize(msg.Width, msg.Height)
		return a, nil
	case common.ResizeMsg:
		a.OnResize(msg.Width, msg.Height)
		return a, nil
	}

	a.sidebar.Update(msg)
	a.chatView.Update(msg)
	return a, nil
}

func (a *ResponsiveApp) View() string {
	if a.width < 40 || a.height < 10 {
		return a.minView()
	}

	w, _ := a.layout.GetHeaderDimensions()
	header := a.renderHeader(w)
	sidebar := a.sidebar.View()
	content := a.chatView.View()
	footer := a.renderFooter(w)

	main := lipgloss.JoinHorizontal(lipgloss.Left, sidebar, lipgloss.NewStyle().Width(1).Render("│"), content)
	view := fmt.Sprintf("%s\n%s\n%s", header, main, footer)
	return a.style.Width(a.width).Height(a.height).Render(view)
}

func (a *ResponsiveApp) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "ctrl+c", "q":
		return a, tea.Quit
	case "tab":
		return a, nil
	case "r":
		a.OnResize(a.width, a.height)
		return a, nil
	}
	return a, nil
}

func (a *ResponsiveApp) OnResize(w, h int) {
	a.width, a.height = w, h
	a.layout.UpdateSize(w, h)
	a.updateStyles()
	a.resizeMgr.HandleResize(w, h)
	a.logger.Info("Resized", "width", w, "height", h)
}

func (a *ResponsiveApp) minView() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Align(lipgloss.Center, lipgloss.Center).
		Width(a.width).
		Height(a.height).
		Render("Terminal too small")
}

func (a *ResponsiveApp) renderHeader(w int) string {
	title := "AI CLI - Responsive"
	status := fmt.Sprintf("Size: %dx%d", a.width, a.height)
	return a.header.Width(w).Render(fmt.Sprintf("%s | %s", title, status))
}

func (a *ResponsiveApp) renderFooter(w int) string {
	help := "Tab: Switch | q: Quit"
	return a.footer.Width(w).Render(help)
}

func (a *ResponsiveApp) updateStyles() {
	if a.width < 60 {
		a.style = a.style.BorderStyle(lipgloss.NormalBorder())
	} else if a.width < 100 {
		a.style = a.style.BorderStyle(lipgloss.RoundedBorder())
	} else {
		a.style = a.style.BorderStyle(lipgloss.RoundedBorder())
	}
}

func getTermSize() (int, int, error) {
	if w := os.Getenv("COLUMNS"); w != "" {
		if h := os.Getenv("LINES"); h != "" {
			return 80, 24, nil
		}
	}
	return 80, 24, nil
}
