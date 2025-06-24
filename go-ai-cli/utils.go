package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Directory constants
const (
	utilDir  = ".util"
	chatsDir = "chats" // This is now a subdirectory under .util
)

// Initialize with proper path separators
var (
	utilPath  = filepath.Join(".", utilDir)
	chatsPath = filepath.Join(utilPath, chatsDir)
)

func getAPIKeyPath() string {
	return filepath.Join(utilPath, ".api_key")
}

// readAPIKey reads the API key from .util/.api_key file
func readAPIKey() (string, error) {
	keyPath := getAPIKeyPath()
	data, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, return empty string which will trigger API key prompt
			return "", nil
		}
		return "", fmt.Errorf("failed to read API key: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func writeAPIKey(key string) error {
	keyPath := getAPIKeyPath()
	return os.WriteFile(keyPath, []byte(key), 0600)
}

func promptAndSaveAPIKey(reader *bufio.Reader) error {
	for {
		fmt.Print("Enter OpenRouter API Key (or !q to exit): ")
		key, _ := reader.ReadString('\n')
		key = strings.TrimSpace(key)

		if key == "!q" {
			os.Exit(0)
		}

		if key == "" {
			fmt.Println("API key cannot be empty. Please try again.")
			continue
		}

		return writeAPIKey(key)
	}
}

// ChatMetadata stores additional information about the chat
type ChatMetadata struct {
	Summary string `json:"summary,omitempty"`
}

// ChatFile represents the complete chat file structure
type ChatFile struct {
	Metadata ChatMetadata `json:"metadata"`
	Messages []Message    `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// systemPrompt default system message
var systemPrompt = Message{
	Role:    "system",
	Content: "You are here to assist me in my endeavours. Responses should fit the tone, if we are discussing code, math or other scientific/logical ideas your answer should also be in a scientific tone. avoid colorful and frivolous language. keep responses prompt and answer the question posed. Please reply with 'Understood, how can I help you today?' if you have understood this message and will remember it when replying in the future.",
}

// formatRoleWithColor returns colored role names for CLI output
func formatRoleWithColor(role string) string {
	const (
		redColor   = "\033[31m"
		blueColor  = "\033[34m"
		resetColor = "\033[0m"
	)

	switch strings.ToLower(role) {
	case "user":
		return redColor + "You" + resetColor
	case "assistant":
		return blueColor + "Assistant" + resetColor
	default:
		return strings.Title(role)
	}
}

func prependSystemPrompt(messages []Message, systemPrompt Message) []Message {
	if len(messages) == 0 || messages[0].Role != "system" || messages[0].Content != systemPrompt.Content {
		return append([]Message{systemPrompt}, messages...)
	}
	return messages
}

// ensureEnvironment makes sure the necessary folders and files exist,
// creating them with defaults if missing.
func ensureEnvironment() error {
	// Create .util directory if missing
	if err := os.MkdirAll(utilPath, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", utilPath, err)
	}

	// Create chats directory under .util if missing
	if err := os.MkdirAll(chatsPath, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", chatsPath, err)
	}

	// Check for API key and prompt if missing
	keyPath := getAPIKeyPath()
	_, err := os.Stat(keyPath)
	if os.IsNotExist(err) {
		fmt.Println("No API key found. You'll need an OpenRouter API key to use this application.")
		reader := bufio.NewReader(os.Stdin)
		if err := promptAndSaveAPIKey(reader); err != nil {
			return fmt.Errorf("failed to save API key: %w", err)
		}
		fmt.Println("API key saved successfully.")
	} else if err != nil {
		return fmt.Errorf("error checking API key file: %w", err)
	}

	// Try to read the API key to validate it
	apiKey, err := readAPIKey()
	if err != nil {
		// If we can't read the key or it's invalid, prompt for a new one
		fmt.Printf("Error reading API key (%v). Please enter a new one.\n", err)
		reader := bufio.NewReader(os.Stdin)
		if err := promptAndSaveAPIKey(reader); err != nil {
			return fmt.Errorf("failed to save API key: %w", err)
		}
		fmt.Println("API key saved successfully.")
	} else if strings.TrimSpace(apiKey) == "" {
		fmt.Println("API key file exists but is empty. Please enter a new one.")
		reader := bufio.NewReader(os.Stdin)
		if err := promptAndSaveAPIKey(reader); err != nil {
			return fmt.Errorf("failed to save API key: %w", err)
		}
		fmt.Println("API key saved successfully.")
	}

	// Initialize models in .util/models.json
	modelsPath := filepath.Join(utilPath, "models.json")
	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
		// Initial models configuration
		config := ModelsConfig{
			Models: []Model{
				{Name: "deepseek/deepseek-chat-v3-0324:free", IsDefault: true},
				{Name: "openai/gpt-4", IsDefault: false},
				{Name: "meta-llama/llama-3-8b-instruct", IsDefault: false},
			},
		}
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal initial models config: %w", err)
		}
		if err := os.WriteFile(modelsPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write initial models config: %w", err)
		}
	}

	// Initialize prompts in .util/prompts.json
	promptsPath := filepath.Join(utilPath, "prompts.json")
	if _, err := os.Stat(promptsPath); os.IsNotExist(err) {
		// Initial prompts configuration with system prompt as first entry
		config := PromptsConfig{
			Prompts: []Prompt{
				{
					Name:      "Default Prompt",
					Content:   systemPrompt.Content,
					IsDefault: true,
				},
			},
		}
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal initial prompts config: %w", err)
		}
		if err := os.WriteFile(promptsPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write initial prompts config: %w", err)
		}
	}

	return nil
}
