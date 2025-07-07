// Go AI CLI - Program Flow Overview
//
// 1. On startup, initializes error logging and ensures the environment is set up (creates config directories/files if missing).
// 2. Checks for an API key; if missing, prompts the user to enter and save one.
// 3. Launches the Bubble Tea GUI main menu (RunGUIMainMenu), which is the entry point for all user interactions.
// 4. The GUI main menu allows navigation to Chats, Favorites, Prompts, Models, API Key management, and Help.
// 5. All chat, prompt, and model management is handled via Bubble Tea GUI flows (MenuModel, ChatModel, etc.), not legacy CLI menus.
// 6. All user data (API keys, models, prompts, chats) is stored in the .util/ directory in JSON files.
// 7. The application supports real-time AI chat, markdown rendering, scrollable chat history, and Vim-style commands.
//
// See README.md for a user-facing overview and controls cheat sheet.
//
// Main entry point below:
package main

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Initialize error logging
	errorLog = NewErrorLog()

	// Ensure environment is set up
	if err := ensureEnvironment(); err != nil {
		handleError(err, "initialization")
		return
	}

	var err error

	// Try to get active API key and URL, catch any error and enter onboarding flow
	for {
		_, _, err = getActiveAPIKeyAndURL()
		if err == nil {
			break // Valid key found, proceed
		}

		// Show information modal (Bubble Tea GUI)
		infoModal := informationModalModel{
			InformationModal: InformationModal{
				Title:   "API Key Required",
				Content: "You need to set a key before proceeding.\nPress enter to continue.",
				Width:   60,
				Height:  8,
			},
		}
		p := tea.NewProgram(infoModal, tea.WithAltScreen())
		_, _ = p.Run()

		// Prompt for API Key Title
		titleModal := InputBoxModal{
			Prompt: "Enter API Key title:",
			Value:  "",
			Cursor: 0,
			Width:  80,
			Height: 24,
		}
		p = tea.NewProgram(titleModal, tea.WithAltScreen())
		finalTitle, err := p.Run()
		if err != nil {
			handleError(err, "prompt for API key title")
			return
		}
		titleResult := finalTitle.(InputBoxModal)
		title := strings.TrimSpace(titleResult.Value)
		if title == "" {
			title = "Default"
		}

		// Prompt for API Key URL
		urlModal := InputBoxModal{
			Prompt: "Enter the URL for this API key (leave blank to read from clipboard or use OpenRouter default):",
			Value:  "",
			Cursor: 0,
			Width:  80,
			Height: 24,
		}
		p = tea.NewProgram(urlModal, tea.WithAltScreen())
		finalURL, err := p.Run()
		if err != nil {
			handleError(err, "prompt for API key url")
			return
		}
		urlResult := finalURL.(InputBoxModal)
		url := strings.TrimSpace(urlResult.Value)
		if url == "" {
			clipText, err := clipboard.ReadAll()
			if err == nil {
				url = strings.TrimSpace(clipText)
			}
			if url == "" {
				url = "https://openrouter.ai/api/v1/chat/completions"
			}
		}

		// Prompt for API Key (with clipboard fallback)
		keyModal := InputBoxModal{
			Prompt: "Enter your API key (leave blank to read from clipboard):",
			Value:  "",
			Cursor: 0,
			Width:  80,
			Height: 24,
		}
		p = tea.NewProgram(keyModal, tea.WithAltScreen())
		finalKey, err := p.Run()
		if err != nil {
			handleError(err, "prompt for API key value")
			return
		}
		keyResult := finalKey.(InputBoxModal)
		key := strings.TrimSpace(keyResult.Value)
		if key == "" {
			clipText, err := clipboard.ReadAll()
			if err == nil {
				key = strings.TrimSpace(clipText)
			}
		}
		if key == "" {
			ShowErrorModal("API key cannot be empty.")
			continue // Loop again
		}

		// Add as active key
		config, err := loadAPIKeys()
		if err != nil {
			handleError(err, "load API keys for save")
			return
		}
		newKey := APIKey{Title: title, Key: key, URL: url, Active: true}
		// Set all others inactive
		for i := range config.Keys {
			config.Keys[i].Active = false
		}
		config.Keys = append(config.Keys, newKey)
		if err := saveAPIKeys(config); err != nil {
			handleError(err, "save API key config")
			return
		}
	}

	// Always launch the GUI main menu
	if err := RunGUIMainMenu(); err != nil {
		fmt.Printf("GUI error: %v\n", err)
	}
}
