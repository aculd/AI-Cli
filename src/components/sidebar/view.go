package sidebar

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderTabsSidebar renders the currently opened tabs area of the sidebar.
// Highlights the active tab and the focused entry, with wrap-around navigation.
func (s *SidebarModel) RenderTabsSidebar() string {
	var b strings.Builder
	pad := "  "
	sectionTitle := lipgloss.NewStyle().Bold(true).Render("Active Chats")
	b.WriteString(pad + sectionTitle + "\n")
	b.WriteString(pad + strings.Repeat("-", s.Width-4) + "\n")
	for i, tab := range s.Tabs {
		style := lipgloss.NewStyle()
		marker := "  "
		if i == s.ActiveTab {
			style = style.Bold(true)
			marker = "> "
		}
		if s.Focus == SidebarFocusTabs && i == s.ActiveTab {
			style = style.Foreground(lipgloss.Color("33")).Background(lipgloss.Color("236")).Bold(true)
		}
		b.WriteString(pad + marker + style.Render(tab) + "\n")
	}
	return b.String()
}

// RenderChatListSidebar renders the chat list area: 5 favorites, 10 recent, and create new chat option.
// Highlights the focused entry and sections, with wrap-around navigation.
func (s *SidebarModel) RenderChatListSidebar() string {
	var b strings.Builder
	pad := "  "
	focusIdx := s.ChatListIndex
	favCount := len(s.Favorites)
	recentCount := len(s.ChatList)
	// Favorites section
	if favCount > 0 {
		favTitle := lipgloss.NewStyle().Bold(true).Render("Favorites")
		b.WriteString(pad + favTitle + "\n")
		b.WriteString(pad + strings.Repeat("-", s.Width-4) + "\n")
		for i, fav := range s.Favorites {
			style := lipgloss.NewStyle()
			marker := "  "
			if s.Focus == SidebarFocusChatList && focusIdx == i {
				style = style.Foreground(lipgloss.Color("33")).Background(lipgloss.Color("236")).Bold(true)
				marker = "> "
			}
			b.WriteString(pad + marker + style.Render("★ "+fav) + "\n")
		}
		b.WriteString("\n")
	}
	// Recent chats section
	recentTitle := lipgloss.NewStyle().Bold(true).Render("Recent Chats")
	b.WriteString(pad + recentTitle + "\n")
	b.WriteString(pad + strings.Repeat("-", s.Width-4) + "\n")
	for i, chat := range s.ChatList {
		style := lipgloss.NewStyle()
		marker := "  "
		idx := i + favCount
		if s.Focus == SidebarFocusChatList && focusIdx == idx {
			style = style.Foreground(lipgloss.Color("33")).Background(lipgloss.Color("236")).Bold(true)
			marker = "> "
		}
		b.WriteString(pad + marker + style.Render(chat) + "\n")
	}
	b.WriteString("\n")
	// Create New Chat option
	newChatIdx := favCount + recentCount
	style := lipgloss.NewStyle()
	marker := "  "
	if s.Focus == SidebarFocusChatList && focusIdx == newChatIdx {
		style = style.Foreground(lipgloss.Color("33")).Background(lipgloss.Color("236")).Bold(true)
		marker = "> "
	}
	b.WriteString(pad + marker + style.Render("[+] Create New Chat") + "\n")
	return b.String()
}

// SidebarView renders the full sidebar, showing both areas, with only one focused at a time.
// Vertically centers the content in the sidebar pane.
func (s *SidebarModel) SidebarView() string {
	var content string
	if s.Focus == SidebarFocusTabs {
		content = s.RenderTabsSidebar()
	} else {
		content = s.RenderChatListSidebar()
	}
	// Vertically center the sidebar content
	height := s.Height
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	padLines := (height - len(lines)) / 2
	if padLines < 0 {
		padLines = 0
	}
	padding := strings.Repeat("\n", padLines)
	return padding + content
}
