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
	"bufio"
	"fmt"
	"os"
)

func main() {
	// Initialize error logging
	errorLog = NewErrorLog()

	// Ensure environment is set up
	if err := ensureEnvironment(); err != nil {
		handleError(err, "initialization")
		return
	}

	reader := bufio.NewReader(os.Stdin)

	// Check API key on startup
	if _, err := readAPIKey(); err != nil {
		fmt.Println("No API key found.")
		if err := promptAndSaveAPIKey(reader); err != nil {
			handleError(err, "initial API key setup")
			return
		}
	}

	// Always launch the GUI main menu
	if err := RunGUIMainMenu(); err != nil {
		fmt.Printf("GUI error: %v\n", err)
	}
}
