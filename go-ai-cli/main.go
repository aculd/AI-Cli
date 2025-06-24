package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func apiKeyMenu(reader *bufio.Reader) {
	fmt.Println("\nAPI Key Menu:")
	fmt.Println("1) Change API Key")
	fmt.Println("2) Back to main menu")
	fmt.Print("Choose option: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "1":
		if err := promptAndSaveAPIKey(reader); err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("API key updated successfully.")
		}
	case "2", "":
		return
	default:
		fmt.Println("Invalid option.")
	}
}

func main() {
	if err := ensureEnvironment(); err != nil {
		fmt.Println("Setup error:", err)
		return
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n=== Chat CLI ===")
		fmt.Println("1) Chats")
		fmt.Println("2) Prompts")
		fmt.Println("3) Models")
		fmt.Println("4) API Key")
		fmt.Println("5) Exit")
		fmt.Print("Choose option: ")

		opt, _ := reader.ReadString('\n')
		opt = strings.TrimSpace(opt)

		switch opt {
		case "1":
			chatsMenu(reader)
		case "2":
			promptsMenu(reader)
		case "3":
			modelsMenu(reader)
		case "4":
			apiKeyMenu(reader)
		case "5", "exit", "quit", "q":
			fmt.Println("Goodbye.")
			return
		default:
			fmt.Println("Invalid option.")
		}
	}
}