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
