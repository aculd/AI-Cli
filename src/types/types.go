package types

import "time"

// Message represents a chat message (shared across app, for JSON serialization).
type Message struct {
	Role          string `json:"role"`
	Content       string `json:"content"`
	MessageNumber int    `json:"message_number"`
}

// ChatMetadata stores additional information about a chat session.
type ChatMetadata struct {
	Summary    string    `json:"summary,omitempty"`
	Title      string    `json:"title,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	Model      string    `json:"model,omitempty"`
	Favorite   bool      `json:"favorite,omitempty"`
	ModifiedAt int64     `json:"modified_at,omitempty"` // Unix timestamp for last modification
}

// ChatFile represents the complete chat file structure for JSON storage.
type ChatFile struct {
	Metadata ChatMetadata `json:"metadata"`
	Messages []Message    `json:"messages"`
}

// Model represents an AI model configuration.
type Model struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

// ModelsConfig represents the models configuration stored in JSON.
type ModelsConfig struct {
	Models []Model `json:"models"`
}

// Control represents a generalized control input (key binding, description, and action).
type Control struct {
	Key         string      // The input key (e.g., "ctrl+c", "enter")
	Description string      // Human-readable description of the control
	Action      func() bool // The action to execute (returns true if handled)
}

// ControlAction is a function signature for control actions.
type ControlAction func() bool

// ControlInfo represents the outline of controls for a menu (displayed below the menu box)
type ControlInfo struct {
	Lines []string // Each line is a row of control info (e.g., "↑↓ navigate, Enter select, Esc back")
}

// Predefined control info for menus
var DefaultControlInfo = ControlInfo{
	Lines: []string{"↑↓ navigate", "Enter select", "Esc back"},
}

var FavoritesControlInfo = ControlInfo{
	Lines: []string{"↑↓ navigate", "Enter load chat", "F unfavorite", "Esc back"},
}

// ControlInfoType enumerates the types of control info layouts
// Used to select the correct control info for each menu or entry
type ControlInfoType int

const (
	DefaultControlInfoType ControlInfoType = iota
	FavoritesControlInfoType
	ListChatsControlInfoType
)

// ControlInfoMap maps ControlInfoType to the actual ControlInfo
var ControlInfoMap = map[ControlInfoType]ControlInfo{
	DefaultControlInfoType:   {Lines: []string{"↑↓ navigate", "Enter select", "Esc back"}},
	FavoritesControlInfoType: {Lines: []string{"↑↓ navigate", "Enter load chat", "F unfavorite", "Esc back"}},
	ListChatsControlInfoType: {Lines: []string{"↑↓ navigate", "Enter view chat", "Esc back"}},
}

// MenuMeta holds metadata for each menu, including its control info type
// This allows each menu to display the correct control outline
type MenuMeta struct {
	ControlInfoType ControlInfoType
}

// MenuMetas maps each MenuType to its metadata (including control info type)
// By default, menus use DefaultControlInfoType unless specified otherwise
var MenuMetas = map[MenuType]MenuMeta{
	MainMenu:    {ControlInfoType: DefaultControlInfoType},
	ChatsMenu:   {ControlInfoType: DefaultControlInfoType},
	PromptsMenu: {ControlInfoType: DefaultControlInfoType},
	ModelsMenu:  {ControlInfoType: DefaultControlInfoType},
	APIKeyMenu:  {ControlInfoType: DefaultControlInfoType},
	HelpMenu:    {ControlInfoType: DefaultControlInfoType},
	ExitMenu:    {ControlInfoType: DefaultControlInfoType},
}

// MenuEntry represents a single menu or submenu option.
// Each entry has display text, an action (function to execute), an optional next MenuType for navigation, and a description.
// This structure supports extensible, modular menu-driven UI design.
type MenuEntry struct {
	Text        string      // Display text for the menu entry
	Action      func() bool // Action to execute when selected (nil if navigation-only)
	Next        MenuType    // The MenuType to navigate to (if this entry leads to a submenu)
	Description string      // Optional: further description or tooltip
	Disabled    bool        // If true, entry is disabled (e.g., when API key is missing)
	// Extend with more fields as needed (e.g., icon, shortcut, enabled/disabled, etc.)
}

// MenuType represents the type of a menu in the app.
// Each menu or submenu is a MenuType, supporting modular navigation and state management.
type MenuType int

const (
	MainMenu MenuType = iota
	ChatsMenu
	PromptsMenu
	ModelsMenu
	APIKeyMenu
	HelpMenu
	ExitMenu
)

// MenuDef defines a menu: its entries and control sets
// This enables a fully data-driven, extensible menu system
// All menus and submenus are defined here

type MenuDef struct {
	Entries     []MenuEntry
	ControlSets []ControlSet
}

// Central registry of all menus
var Menus = map[MenuType]MenuDef{
	MainMenu: {
		Entries: []MenuEntry{
			{Text: "Chats", Next: ChatsMenu, Description: "View and manage chats"},
			{Text: "Prompts", Next: PromptsMenu, Description: "Manage prompt templates"},
			{Text: "Models", Next: ModelsMenu, Description: "Configure AI models"},
			{Text: "API Key", Next: APIKeyMenu, Description: "Manage API keys"},
			{Text: "Help", Next: HelpMenu, Description: "Show help and shortcuts"},
			{Text: "Exit", Action: nil, Description: "Exit the application"},
		},
		ControlSets: []ControlSet{
			DefaultControlSet,
		},
	},
	ChatsMenu: {
		Entries: []MenuEntry{
			{Text: "List Chats", Action: nil},
			{Text: "Add New Chat", Action: nil},
			{Text: "Custom Chat", Action: nil},
			{Text: "Load Chat", Action: nil},
			{Text: "Favorites", Action: nil},
			{Text: "Back", Next: MainMenu},
		},
		ControlSets: []ControlSet{
			DefaultControlSet,
		},
	},
	PromptsMenu: {
		Entries: []MenuEntry{
			{Text: "List Prompts", Action: nil},
			{Text: "Add Prompt", Action: nil},
			{Text: "Remove Prompt", Action: nil},
			{Text: "Set Default Prompt", Action: nil},
			{Text: "Back", Next: MainMenu},
		},
		ControlSets: []ControlSet{
			DefaultControlSet,
		},
	},
	ModelsMenu: {
		Entries: []MenuEntry{
			{Text: "List Models", Action: nil},
			{Text: "Add Model", Action: nil},
			{Text: "Remove Model", Action: nil},
			{Text: "Set Default Model", Action: nil},
			{Text: "Back", Next: MainMenu},
		},
		ControlSets: []ControlSet{
			DefaultControlSet,
		},
	},
	APIKeyMenu: {
		Entries: []MenuEntry{
			{Text: "List API Keys", Action: nil},
			{Text: "Add API Key", Action: nil},
			{Text: "Remove API Key", Action: nil},
			{Text: "Set Active API Key", Action: nil},
			{Text: "Back", Next: MainMenu},
		},
		ControlSets: []ControlSet{
			DefaultControlSet,
		},
	},
	HelpMenu: {
		Entries: []MenuEntry{
			{Text: "Back", Next: MainMenu},
		},
		ControlSets: []ControlSet{
			DefaultControlSet,
		},
	},
	ExitMenu: {
		Entries: []MenuEntry{},
		ControlSets: []ControlSet{
			DefaultControlSet,
		},
	},
}

// MenuState tracks the current menu, selection, and previous state for back navigation.
// This enables accurate navigation and state restoration.
type MenuState struct {
	Type     MenuType
	Selected int
	Previous *MenuState
	// Extend with more fields as needed for submenu state
}

// ShowMenu signature (to be implemented in app):
// func ShowMenu(menuType MenuType, prev *MenuState)
// This function sets the current menu state and renders the menu, supporting modular navigation.

// Prompt represents a prompt template for the assistant.
type Prompt struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Default bool   `json:"default,omitempty"`
}

// ChatInfo holds summary info for a chat (for sidebar/recent/favorites).
type ChatInfo struct {
	Name       string
	Favorite   bool
	ModifiedAt int64 // Unix timestamp
}

// ControlType represents a single control action (e.g., Up, Down, Enter, Esc) with an associated action
// Name: human-readable name, Key: key binding, Action: function to execute
// Action can be nil or a function with a standard signature (e.g., func() bool)
type ControlType struct {
	Name   string
	Key    string
	Action func() bool // or interface{} for more flexibility
}

// ControlSet represents a set of controls for a view/state
// Each ControlSet can be customized per view/state
// Example: {ControlUp, ControlDown, ControlEnter, ControlEsc}
type ControlSet struct {
	Controls []ControlType
}

// Example: DefaultControlSet for menus
var DefaultControlSet = ControlSet{
	Controls: []ControlType{
		{Name: "Up", Key: "up", Action: nil},
		{Name: "Down", Key: "down", Action: nil},
		{Name: "Enter", Key: "enter", Action: nil},
		{Name: "Esc", Key: "esc", Action: nil},
	},
}
