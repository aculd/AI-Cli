// components/sidebar/model.go - SidebarModel for persistent sidebar menu navigation
// MIGRATION TARGET: legacy/gui.go (sidebar/menu navigation logic)

package sidebar

import (
	"aichat/src/types"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// TODO: Replace these stubs with actual imports or implementations
func listChats() ([]string, error)                              { return nil, nil } // stub
func loadChatWithMetadata(name string) (*types.ChatFile, error) { return nil, nil } // stub
func chatsPath() string                                         { return "" }       // stub

// SidebarFocusType indicates which sidebar view is focused
// (for Ctrl+T, Ctrl+N, etc.)
type SidebarFocusType int

const (
	SidebarFocusTabs     SidebarFocusType = iota // Opened tabs area
	SidebarFocusChatList                         // Chat list area
	// Added for migration compatibility:
	SidebarFocusActiveChats
	SidebarFocusAllChats
	SidebarFocusNewChat
)

// SidebarModel manages the persistent sidebar menu
// The sidebar is always visible next to the chat window
// It holds the current menu and selection state

type SidebarModel struct {
	Tabs           []string         // Currently opened chat tabs (max 10)
	ActiveTab      int              // Index of selected tab in Tabs
	ChatList       []string         // 10 most recent chats
	ChatListIndex  int              // Index of selected chat in ChatList
	Favorites      []string         // Up to 5 favorite chats
	Focus          SidebarFocusType // Which sidebar view is focused
	Width          int              // Sidebar width (for layout)
	Height         int              // Sidebar height (for layout)
	ShowNewChatOpt bool             // Always show new chat option at bottom
	// MIGRATION: Add fields for legacy compatibility
	ActiveChats   []string // List of active chats (for legacy View)
	AllChats      []string // List of all chats (for legacy View)
	SelectedIndex int      // Selected index for legacy View
}

// NewSidebarModel creates a new sidebar with placeholder data
func NewSidebarModel(mainMenu *types.MenuViewState) *SidebarModel {
	s := &SidebarModel{
		Tabs:           []string{"Tab 1", "Tab 2", "Tab 3"},
		ActiveTab:      0,
		ChatList:       []string{"Chat A", "Chat B", "Chat C", "Chat D", "Chat E", "Chat F", "Chat G", "Chat H", "Chat I", "Chat J"},
		ChatListIndex:  0,
		Favorites:      []string{"Fav 1", "Fav 2", "Fav 3"},
		Focus:          SidebarFocusTabs,
		Width:          24,
		Height:         40,
		ShowNewChatOpt: true,
	}
	// Select 'New Chat' if no chats/favorites, else select first chat
	if len(s.ChatList) == 0 && len(s.Favorites) == 0 {
		s.ChatListIndex = -1 // -1 means 'New Chat' selected
	} else {
		s.ChatListIndex = 0
	}
	return s
}

// TODO: Define FocusInputAction or import from correct location
const FocusInputAction = 1000 // Placeholder value, replace with actual if available

// Update handles key navigation for the sidebar views
// Returns (SidebarModel, tea.Cmd, navigationMsg)
func (s *SidebarModel) Update(msg tea.Msg) (*SidebarModel, tea.Cmd, *NavigationMsg) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "ctrl+t":
			s.Focus = SidebarFocusTabs
			s.ActiveTab = 0
		case "ctrl+n":
			s.Focus = SidebarFocusChatList
			s.ChatListIndex = 0
		case "up":
			if s.Focus == SidebarFocusTabs {
				if s.ActiveTab == 0 {
					s.ActiveTab = len(s.Tabs) - 1
				} else {
					s.ActiveTab--
				}
			} else if s.Focus == SidebarFocusChatList {
				if s.ChatListIndex == 0 {
					s.ChatListIndex = len(s.ChatList) + len(s.Favorites)
				} else {
					s.ChatListIndex--
				}
			}
		case "down":
			if s.Focus == SidebarFocusTabs {
				if s.ActiveTab == len(s.Tabs)-1 {
					s.ActiveTab = 0
				} else {
					s.ActiveTab++
				}
			} else if s.Focus == SidebarFocusChatList {
				if s.ChatListIndex == len(s.ChatList)+len(s.Favorites) {
					s.ChatListIndex = 0
				} else {
					s.ChatListIndex++
				}
			}
		case "enter":
			if s.Focus == SidebarFocusTabs {
				// Enter on tab: select chat, return focus to input (handled by app)
				return s, nil, &NavigationMsg{Action: FocusInputAction, Target: nil}
			} else if s.Focus == SidebarFocusChatList {
				// Enter on chat: open chat (add to tabs)
				// If on new chat option, trigger new chat modal
				if s.ChatListIndex == len(s.ChatList)+len(s.Favorites) {
					// TODO: Trigger new chat creation modal
					return s, nil, &NavigationMsg{Action: PushAction, Target: nil} // Placeholder
				}
				// TODO: Handle chat selection (open chat)
			}
		case "esc":
			// TODO: Optionally handle sidebar unfocus
		}
	}
	return s, nil, nil
}

// View renders the sidebar with padding/margin for good UI/UX
func (s *SidebarModel) View() string {
	var b strings.Builder
	pad := "  "
	b.WriteString("\n")
	b.WriteString(pad + "Active Chats:\n")
	b.WriteString(pad + strings.Repeat("-", s.Width-4) + "\n")
	for i, chat := range s.ActiveChats {
		line := pad + chat
		if s.Focus == SidebarFocusActiveChats && i == s.SelectedIndex {
			line = pad + "> " + chat
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
	b.WriteString(pad + "All Chats:\n")
	b.WriteString(pad + strings.Repeat("-", s.Width-4) + "\n")
	for i, chat := range s.AllChats {
		line := pad + chat
		if s.Focus == SidebarFocusAllChats && i == s.SelectedIndex {
			line = pad + "> " + chat
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
	b.WriteString(pad + "[+] New Chat\n")
	if s.Focus == SidebarFocusNewChat {
		b.WriteString(pad + "> [+] New Chat\n")
	}
	return b.String()
}

// Refresh repopulates the sidebar data from app state and storage.
// openTabs: list of currently opened chat names (tab order)
// activeTabIdx: index of the active tab
func (s *SidebarModel) Refresh(openTabs []string, activeTabIdx int) {
	// Update tabs and active tab
	s.Tabs = openTabs
	s.ActiveTab = activeTabIdx

	// Get 10 most recent chats
	recent, err := listChats()
	if err != nil {
		s.ChatList = nil
	} else {
		s.ChatList = recent
	}

	// Find up to 5 most recently modified favorites
	var favInfos []struct {
		Name       string
		ModifiedAt int64
	}
	for _, chat := range s.ChatList {
		chatFile, err := loadChatWithMetadata(chat)
		if err == nil && chatFile.Metadata.Favorite {
			// Get file mod time
			fileInfo, ferr := os.Stat(chatsPath() + "/" + chat + ".json")
			modTime := int64(0)
			if ferr == nil {
				modTime = fileInfo.ModTime().Unix()
			}
			favInfos = append(favInfos, struct {
				Name       string
				ModifiedAt int64
			}{chat, modTime})
		}
	}
	// Sort favorites by ModifiedAt desc
	sort.Slice(favInfos, func(i, j int) bool {
		return favInfos[i].ModifiedAt > favInfos[j].ModifiedAt
	})
	// Limit to 5
	maxFavs := 5
	if len(favInfos) > maxFavs {
		favInfos = favInfos[:maxFavs]
	}
	// Populate Favorites
	var favs []string
	for _, fi := range favInfos {
		favs = append(favs, fi.Name)
	}
	s.Favorites = favs

	s.ShowNewChatOpt = true
}

// Sidebar control set for active chats area
var sidebarActiveChatsControlSet = []struct {
	Name   string
	Key    string
	Action func(s *SidebarModel) bool
}{
	{"Up", "up", func(s *SidebarModel) bool {
		if s.ActiveTab == 0 {
			s.ActiveTab = len(s.Tabs) - 1
		} else {
			s.ActiveTab--
		}
		return true
	}},
	{"Down", "down", func(s *SidebarModel) bool {
		if s.ActiveTab == len(s.Tabs)-1 {
			s.ActiveTab = 0
		} else {
			s.ActiveTab++
		}
		return true
	}},
	{"Enter", "enter", func(s *SidebarModel) bool {
		// TODO: select active chat
		return true
	}},
	{"Tab", "tab", func(s *SidebarModel) bool {
		s.Focus = SidebarFocusChatList
		return true
	}},
	{"Ctrl+N", "ctrl+n", func(s *SidebarModel) bool {
		s.Focus = SidebarFocusChatList
		return true
	}},
	{"Esc", "esc", func(s *SidebarModel) bool {
		// Return focus to input
		return true
	}},
	{"Ctrl+I", "ctrl+i", func(s *SidebarModel) bool {
		// Return focus to input
		return true
	}},
}

// Sidebar control set for list chats/favorites/add new chat area
var sidebarListChatsControlSet = []struct {
	Name   string
	Key    string
	Action func(s *SidebarModel) bool
}{
	{"Up", "up", func(s *SidebarModel) bool {
		if s.ChatListIndex <= 0 {
			s.ChatListIndex = len(s.ChatList) + len(s.Favorites) - 1
		} else {
			s.ChatListIndex--
		}
		return true
	}},
	{"Down", "down", func(s *SidebarModel) bool {
		if s.ChatListIndex >= len(s.ChatList)+len(s.Favorites)-1 {
			s.ChatListIndex = 0
		} else {
			s.ChatListIndex++
		}
		return true
	}},
	{"Enter", "enter", func(s *SidebarModel) bool {
		// TODO: select chat or trigger new chat
		return true
	}},
	{"Tab", "tab", func(s *SidebarModel) bool {
		s.Focus = SidebarFocusTabs
		return true
	}},
	{"Ctrl+T", "ctrl+t", func(s *SidebarModel) bool {
		s.Focus = SidebarFocusTabs
		return true
	}},
	{"Esc", "esc", func(s *SidebarModel) bool {
		// Return focus to input
		return true
	}},
	{"Ctrl+I", "ctrl+i", func(s *SidebarModel) bool {
		// Return focus to input
		return true
	}},
}

// GetControlSet returns the sidebar's control set for the current focus area
func (s *SidebarModel) GetControlSet() interface{} {
	switch s.Focus {
	case SidebarFocusTabs:
		return sidebarActiveChatsControlSet
	case SidebarFocusChatList:
		return sidebarListChatsControlSet
	default:
		return sidebarListChatsControlSet
	}
}

// NavigationMsg and actions (local copy for now)
type NavigationAction int

const (
	PushAction NavigationAction = iota
	PopAction
	ResetAction
)

type NavigationMsg struct {
	Action NavigationAction
	Target types.ViewState // Only for PushAction
}
