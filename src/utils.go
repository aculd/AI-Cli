package main

import (
	"aichat/src/models"
	"aichat/src/services/storage/repositories"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Directory constants
const (
	utilDir  = ".util"
	chatsDir = "chats"
)

// utilPath returns the .util directory path relative to the executable location
func utilPath() string {
	exePath, err := os.Executable()
	if err != nil {
		// fallback to current directory
		return filepath.Join(".", utilDir)
	}
	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, utilDir)
}

// chatsPath returns the chats directory path relative to the executable location
func chatsPath() string {
	return filepath.Join(utilPath(), chatsDir)
}

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

var apiKeyRepo = repositories.NewAPIKeyRepository()

func prependSystemPrompt(messages []Message, systemPrompt Message) []Message {
	if len(messages) == 0 || messages[0].Role != "system" || messages[0].Content != systemPrompt.Content {
		return append([]Message{systemPrompt}, messages...)
	}
	return messages
}

// loadAPIKeys loads the API keys configuration from the repository.
func loadAPIKeys() (*models.APIKeysConfig, error) {
	keys, err := apiKeyRepo.GetAll()
	if err != nil {
		return nil, err
	}
	return &models.APIKeysConfig{Keys: keys}, nil
}

// saveAPIKeys saves the API keys configuration to the repository.
func saveAPIKeys(config *models.APIKeysConfig) error {
	return apiKeyRepo.SaveAll(config.Keys)
}

// getActiveAPIKey returns the currently active API key from the repository, or an error if not found.
func getActiveAPIKey() (string, error) {
	keys, err := apiKeyRepo.GetAll()
	if err != nil {
		return "", err
	}
	for _, key := range keys {
		if key.Active {
			return key.Key, nil
		}
	}
	return "", fmt.Errorf("no active API key found")
}

// getActiveAPIKeyAndURL returns the currently active API key and its URL from the repository, or an error if not found.
func getActiveAPIKeyAndURL() (string, string, error) {
	keys, err := apiKeyRepo.GetAll()
	if err != nil {
		return "", "", err
	}
	for _, key := range keys {
		if key.Active {
			return key.Key, key.URL, nil
		}
	}
	return "", "", fmt.Errorf("no active API key found")
}

// readAPIKey returns the active API key from the multi-key system, or error if not found.
func readAPIKey() (string, error) {
	return getActiveAPIKey()
}

// TestAPIKeyWithModel sends a test message to the selected model using the active API key and returns the result.
// If a normal response is received within 10s, returns "Key is working". Otherwise returns error or timeout message.
func TestAPIKeyWithModel(model string) string {
	key, url, err := getActiveAPIKeyAndURL()
	if err != nil {
		return "Error: " + err.Error()
	}
	reqBody := StreamRequestBody{
		Model:       model,
		Messages:    []Message{{Role: "user", Content: "Hello"}},
		Stream:      true,
		MaxTokens:   16,
		Temperature: 0.0,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "Error: failed to marshal request body: " + err.Error()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "Error: failed to create request: " + err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/go-ai-cli")
	req.Header.Set("X-Title", "Go AI CLI")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "No response (timeout)"
		}
		return "Error: " + err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var errorResp struct {
			Error ErrorResponse `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return "Error: " + errorResp.Error.Message
		}
		return "Error: API returned status " + fmt.Sprint(resp.StatusCode) + ": " + string(body)
	}

	reader := bufio.NewReader(resp.Body)
	var buffer string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "Error: reading response: " + err.Error()
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[len("data: "):]
		if data == "[DONE]" {
			break
		}
		buffer += data
		// If we get any content, consider it working
		if len(buffer) > 0 {
			return "Key is working"
		}
	}
	// If we get here, no content was received
	return "No response (empty)"
}

// ensureEnvironment creates required directories and config files if missing.
// Returns: error if any setup step fails.
func ensureEnvironment() error {
	// Create .util directory
	if err := os.MkdirAll(utilPath(), 0755); err != nil {
		return &AppError{
			Op:      "create util directory",
			Err:     err,
			Message: "failed to create utility directory",
		}
	}

	// Create chats directory
	if err := os.MkdirAll(chatsPath(), 0755); err != nil {
		return &AppError{
			Op:      "create chats directory",
			Err:     err,
			Message: "failed to create chats directory",
		}
	}

	// DO NOT create api_keys.json here; onboarding flow will handle it

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
func addAPIKey(title, key, url string) error {
	keys, err := apiKeyRepo.GetAll()
	if err != nil {
		return err
	}
	active := len(keys) == 0
	newKey := models.APIKey{Title: title, Key: key, URL: url, Active: active}
	return apiKeyRepo.Add(newKey)
}

// setKeyActiveByTitle sets the given API key title as active and all others as inactive, and saves the config
func setKeyActiveByTitle(title string) error {
	keys, err := apiKeyRepo.GetAll()
	if err != nil {
		return err
	}
	found := false
	for i := range keys {
		if keys[i].Title == title {
			keys[i].Active = true
			found = true
		} else {
			keys[i].Active = false
		}
	}
	if !found {
		return fmt.Errorf("API key with title '%s' not found", title)
	}
	return apiKeyRepo.SaveAll(keys)
}

// setActiveAPIKey sets the given API key title as the active key in the repository.
func setActiveAPIKey(title string) error {
	return apiKeyRepo.SetActive(title)
}
