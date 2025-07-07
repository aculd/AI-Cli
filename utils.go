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
	chatsDir = "chats"
)

var (
	utilPath  = filepath.Join(".", utilDir)
	chatsPath = filepath.Join(utilPath, chatsDir)
)

// APIKey represents a single API key with a title
// Fields: Title (string), Key (string), URL (string), Active (bool)
type APIKey struct {
	Title  string `json:"title"`
	Key    string `json:"key"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

// APIKeysConfig represents the configuration for multiple API keys
// Fields: Keys ([]APIKey)
type APIKeysConfig struct {
	Keys []APIKey `json:"keys"`
}

func prependSystemPrompt(messages []Message, systemPrompt Message) []Message {
	if len(messages) == 0 || messages[0].Role != "system" || messages[0].Content != systemPrompt.Content {
		return append([]Message{systemPrompt}, messages...)
	}
	return messages
}

// loadAPIKeys loads the API keys configuration from file, or returns an empty config if not found.
func loadAPIKeys() (*APIKeysConfig, error) {
	data, err := os.ReadFile(filepath.Join(utilPath, "api_keys.json"))
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &APIKeysConfig{Keys: []APIKey{}}, nil
		}
		return nil, &AppError{
			Op:      "read API keys file",
			Err:     err,
			Message: "failed to read API keys file",
		}
	}

	var config APIKeysConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, &AppError{
			Op:      "parse API keys file",
			Err:     err,
			Message: "failed to parse API keys file",
		}
	}

	return &config, nil
}

// saveAPIKeys saves the API keys configuration to file.
func saveAPIKeys(config *APIKeysConfig) error {
	if err := os.MkdirAll(utilPath, 0755); err != nil {
		return &AppError{
			Op:      "create util directory",
			Err:     err,
			Message: "failed to create directory for API keys",
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return &AppError{
			Op:      "marshal API keys",
			Err:     err,
			Message: "failed to marshal API keys",
		}
	}

	if err := os.WriteFile(filepath.Join(utilPath, "api_keys.json"), data, 0600); err != nil {
		return &AppError{
			Op:      "write API keys file",
			Err:     err,
			Message: "failed to save API keys file",
		}
	}
	return nil
}

// getActiveAPIKey returns the currently active API key from the multi-key system, or an error if not found.
func getActiveAPIKey() (string, error) {
	config, err := loadAPIKeys()
	if err == nil && len(config.Keys) > 0 {
		for _, key := range config.Keys {
			if key.Active {
				return key.Key, nil
			}
		}
		return "", &AppError{
			Op:      "get active API key",
			Err:     fmt.Errorf("no active API key found"),
			Message: "No active API key found. Please set one as active.",
		}
	}
	return "", &AppError{
		Op:      "get active API key",
		Err:     fmt.Errorf("no API keys found"),
		Message: "No API keys found. Please add an API key first",
	}
}

// getActiveAPIKeyAndURL returns the currently active API key and its URL from the multi-key system, or an error if not found.
func getActiveAPIKeyAndURL() (string, string, error) {
	config, err := loadAPIKeys()
	if err == nil && len(config.Keys) > 0 {
		for _, key := range config.Keys {
			if key.Active {
				return key.Key, key.URL, nil
			}
		}
		return "", "", &AppError{
			Op:      "get active API key and url",
			Err:     fmt.Errorf("no active API key found"),
			Message: "No active API key found. Please set one as active.",
		}
	}
	return "", "", &AppError{
		Op:      "get active API key and url",
		Err:     fmt.Errorf("no API keys found"),
		Message: "No API keys found. Please add an API key first",
	}
}

// readAPIKey returns the active API key from the multi-key system, or error if not found.
func readAPIKey() (string, error) {
	return getActiveAPIKey()
}

// ensureEnvironment creates required directories and config files if missing.
// Returns: error if any setup step fails.
func ensureEnvironment() error {
	// Create .util directory
	if err := os.MkdirAll(utilPath, 0755); err != nil {
		return &AppError{
			Op:      "create util directory",
			Err:     err,
			Message: "failed to create utility directory",
		}
	}

	// Create chats directory
	if err := os.MkdirAll(chatsPath, 0755); err != nil {
		return &AppError{
			Op:      "create chats directory",
			Err:     err,
			Message: "failed to create chats directory",
		}
	}

	// Ensure API keys file exists (will be created empty if it doesn't exist)
	if _, err := os.Stat(filepath.Join(utilPath, "api_keys.json")); os.IsNotExist(err) {
		emptyConfig := &APIKeysConfig{Keys: []APIKey{}}
		if err := saveAPIKeys(emptyConfig); err != nil {
			return &AppError{
				Op:      "create API keys file",
				Err:     err,
				Message: "failed to create API keys file",
			}
		}
	}

	// Ensure models file exists
	if _, err := os.Stat(modelsFilePath()); os.IsNotExist(err) {
		if err := initializeModelsFile(); err != nil {
			return &AppError{
				Op:      "create models file",
				Err:     err,
				Message: "failed to create models file",
			}
		}
	}

	// Ensure prompts file exists
	if _, err := os.Stat(promptsConfigPath()); os.IsNotExist(err) {
		if err := ensurePromptsConfig(); err != nil {
			return &AppError{
				Op:      "create prompts file",
				Err:     err,
				Message: "failed to create prompts file",
			}
		}
	}

	return nil
}

// promptAndSaveAPIKey interactively prompts the user for an API key and saves it.
// Params: reader (*bufio.Reader) for user input.
// Returns: error if input or saving fails.
func promptAndSaveAPIKey(reader *bufio.Reader) error {
	fmt.Print("Enter a title for this API key: ")
	title, err := reader.ReadString('\n')
	if err != nil {
		return &AppError{
			Op:      "read API key title",
			Err:     err,
			Message: "failed to read API key title from input",
		}
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Default"
	}

	fmt.Print("Enter your OpenRouter API key: ")
	key, err := reader.ReadString('\n')
	if err != nil {
		return &AppError{
			Op:      "read API key input",
			Err:     err,
			Message: "failed to read API key from input",
		}
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return &AppError{
			Op:      "validate API key",
			Err:     fmt.Errorf("empty key"),
			Message: "API key cannot be empty",
		}
	}

	fmt.Print("Enter the URL for this API key: ")
	url, err := reader.ReadString('\n')
	if err != nil {
		return &AppError{
			Op:      "read API key URL",
			Err:     err,
			Message: "failed to read API key URL from input",
		}
	}

	url = strings.TrimSpace(url)
	if url == "" {
		url = "https://openrouter.ai/api/v1/chat/completions"
	}

	if err := addAPIKey(title, key, url); err != nil {
		return err
	}

	fmt.Printf("API key '%s' saved successfully.\n", title)
	return nil
}

// addAPIKey adds a new API key with the given title, key, and URL, and sets as active if first key.
// Params: title (string), key (string), url (string)
// Returns: error if the title exists or saving fails.
func addAPIKey(title, key, url string) error {
	config, err := loadAPIKeys()
	if err != nil {
		return err
	}
	for _, existingKey := range config.Keys {
		if existingKey.Title == title {
			return &AppError{
				Op:      "add API key",
				Err:     fmt.Errorf("title already exists"),
				Message: "An API key with this title already exists",
			}
		}
	}
	active := false
	if len(config.Keys) == 0 {
		active = true
	}
	config.Keys = append(config.Keys, APIKey{Title: title, Key: key, URL: url, Active: active})
	return saveAPIKeys(config)
}

// setKeyActiveByTitle sets the given API key title as active and all others as inactive, and saves the config
func setKeyActiveByTitle(title string) error {
	config, err := loadAPIKeys()
	if err != nil {
		return err
	}
	found := false
	for i := range config.Keys {
		if config.Keys[i].Title == title {
			config.Keys[i].Active = true
			found = true
		} else {
			config.Keys[i].Active = false
		}
	}
	if !found {
		return fmt.Errorf("API key with title '%s' not found", title)
	}
	return saveAPIKeys(config)
}

// setActiveAPIKey sets the given API key title as the active key in api_keys.json
func setActiveAPIKey(title string) error {
	return setKeyActiveByTitle(title)
}
