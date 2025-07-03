package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

// AppError represents application-level errors
type AppError struct {
	Op      string
	Err     error
	Message string
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s (%v)", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// ErrorLog represents a file-based error log
type ErrorLog struct {
	LogFile string
}

// NewErrorLog creates a new error logger
func NewErrorLog() *ErrorLog {
	return &ErrorLog{
		LogFile: filepath.Join(utilPath, "error.log"),
	}
}

// LogError logs an error with context to file and optionally prints to console
func (el *ErrorLog) LogError(err error, context string, printToConsole bool) {
	if err == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s: %v\n", timestamp, context, err)

	f, ferr := os.OpenFile(el.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if ferr == nil {
		defer f.Close()
		f.WriteString(logEntry)
	}

	if printToConsole {
		switch e := err.(type) {
		case *AppError:
			fmt.Printf("\033[31mError: %s\033[0m\n", e.Error())
		case *ModelError:
			fmt.Printf("\033[31mModel error: %s\033[0m\n", e.Error())
		case *PromptError:
			fmt.Printf("\033[31mPrompt error: %s\033[0m\n", e.Error())
		default:
			fmt.Printf("\033[31mError during %s: %v\033[0m\n", context, err)
		}
	}
}

var errorLog = NewErrorLog()

func handleError(err error, context string) {
	if err != nil {
		errorLog.LogError(err, context, true)
	}
}

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
	if _, err := os.Stat(getAPIKeysPath()); os.IsNotExist(err) {
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

// APIKey represents a single API key with a title
type APIKey struct {
	Title string `json:"title"`
	Key   string `json:"key"`
}

// APIKeysConfig represents the configuration for multiple API keys
type APIKeysConfig struct {
	Keys      []APIKey `json:"keys"`
	ActiveKey string   `json:"active_key,omitempty"` // Title of the active key
}

func getAPIKeysPath() string {
	return filepath.Join(utilPath, "api_keys.json")
}

func loadAPIKeys() (*APIKeysConfig, error) {
	data, err := os.ReadFile(getAPIKeysPath())
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

	if err := os.WriteFile(getAPIKeysPath(), data, 0600); err != nil {
		return &AppError{
			Op:      "write API keys file",
			Err:     err,
			Message: "failed to save API keys file",
		}
	}
	return nil
}

func getActiveAPIKey() (string, error) {
	config, err := loadAPIKeys()
	if err == nil && len(config.Keys) > 0 {
		// If no active key is set, use the first one
		if config.ActiveKey == "" {
			config.ActiveKey = config.Keys[0].Title
			if err := saveAPIKeys(config); err != nil {
				return "", err
			}
			return config.Keys[0].Key, nil
		}
		// Find the active key
		for _, key := range config.Keys {
			if key.Title == config.ActiveKey {
				return key.Key, nil
			}
		}
		// If active key not found, use the first one
		config.ActiveKey = config.Keys[0].Title
		if err := saveAPIKeys(config); err != nil {
			return "", err
		}
		return config.Keys[0].Key, nil
	}

	// If no API keys in JSON, fallback to legacy .api_key file in root dir
	legacyPath := ".api_key"
	data, err := os.ReadFile(legacyPath)
	if err == nil {
		key := strings.TrimSpace(string(data))
		if key != "" {
			return key, nil
		}
	}

	return "", &AppError{
		Op:      "get active API key",
		Err:     fmt.Errorf("no API keys found"),
		Message: "No API keys found. Please add an API key first",
	}
}

func addAPIKey(title, key string) error {
	config, err := loadAPIKeys()
	if err != nil {
		return err
	}

	// Check if title already exists
	for _, existingKey := range config.Keys {
		if existingKey.Title == title {
			return &AppError{
				Op:      "add API key",
				Err:     fmt.Errorf("title already exists"),
				Message: "An API key with this title already exists",
			}
		}
	}

	// Add new key
	config.Keys = append(config.Keys, APIKey{Title: title, Key: key})

	// Set as active if it's the first key
	if len(config.Keys) == 1 {
		config.ActiveKey = title
	}

	return saveAPIKeys(config)
}

func removeAPIKey(title string) error {
	config, err := loadAPIKeys()
	if err != nil {
		return err
	}

	var newKeys []APIKey
	found := false
	for _, key := range config.Keys {
		if key.Title != title {
			newKeys = append(newKeys, key)
		} else {
			found = true
		}
	}

	if !found {
		return &AppError{
			Op:      "remove API key",
			Err:     fmt.Errorf("key not found"),
			Message: "API key with this title not found",
		}
	}

	config.Keys = newKeys

	// If we removed the active key, set a new active key
	if config.ActiveKey == title {
		if len(newKeys) > 0 {
			config.ActiveKey = newKeys[0].Title
		} else {
			config.ActiveKey = ""
		}
	}

	return saveAPIKeys(config)
}

func setActiveAPIKey(title string) error {
	config, err := loadAPIKeys()
	if err != nil {
		return err
	}

	// Check if the key exists
	found := false
	for _, key := range config.Keys {
		if key.Title == title {
			found = true
			break
		}
	}

	if !found {
		return &AppError{
			Op:      "set active API key",
			Err:     fmt.Errorf("key not found"),
			Message: "API key with this title not found",
		}
	}

	config.ActiveKey = title
	return saveAPIKeys(config)
}

func listAPIKeys() ([]APIKey, string, error) {
	config, err := loadAPIKeys()
	if err != nil {
		return nil, "", err
	}

	return config.Keys, config.ActiveKey, nil
}

// Legacy support functions for backward compatibility
func getAPIKeyPath() string {
	return filepath.Join(utilPath, ".api_key")
}

func readAPIKey() (string, error) {
	// Try to read from new multi-key system first
	if key, err := getActiveAPIKey(); err == nil {
		return key, nil
	}

	// Fallback to legacy single key file
	keyPath := getAPIKeyPath()
	data, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &AppError{
				Op:      "read API key",
				Err:     err,
				Message: "API key file not found. Please add an API key",
			}
		}
		return "", &AppError{
			Op:      "read API key",
			Err:     err,
			Message: "failed to read API key file",
		}
	}
	return strings.TrimSpace(string(data)), nil
}

func writeAPIKey(key string) error {
	// Migrate to new system by creating a default key
	return addAPIKey("Default", key)
}

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

	if err := addAPIKey(title, key); err != nil {
		return err
	}

	fmt.Printf("API key '%s' saved successfully.\n", title)
	return nil
}

// addAPIKeyFromClipboard reads the clipboard and prompts for a key name, then adds the key
func addAPIKeyFromClipboard(reader *bufio.Reader) error {
	// Windows clipboard command
	clipCmd := "powershell Get-Clipboard"
	clipOut, err := execCommand(clipCmd)
	if err != nil {
		return &AppError{
			Op:      "read clipboard",
			Err:     err,
			Message: "Failed to read clipboard. Please copy your API key first.",
		}
	}
	key := strings.TrimSpace(clipOut)
	if key == "" {
		return &AppError{
			Op:      "read clipboard",
			Err:     fmt.Errorf("clipboard empty"),
			Message: "Clipboard is empty. Please copy your API key first.",
		}
	}
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
	if err := addAPIKey(title, key); err != nil {
		return err
	}
	fmt.Printf("API key '%s' saved successfully.\n", title)
	return nil
}

// execCommand runs a shell command and returns its output as a string
func execCommand(cmd string) (string, error) {
	out, err := exec.Command("cmd", "/C", cmd).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

type Menu struct {
	Title    string
	Items    []MenuItem
	ExitText string
}

type MenuItem struct {
	Label    string
	Handler  func(*bufio.Reader) error
	ExitItem bool
}

func RunMenu(menu Menu, reader *bufio.Reader) {
	for {
		fmt.Printf("\n%s:\n", menu.Title)
		for i, item := range menu.Items {
			fmt.Printf("%d) %s\n", i+1, item.Label)
		}
		fmt.Print("Choose option: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" && menu.ExitText != "" {
			fmt.Println(menu.ExitText)
			return
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(menu.Items) {
			fmt.Println("Invalid option.")
			continue
		}

		item := menu.Items[choice-1]
		if item.Handler != nil {
			if err := item.Handler(reader); err != nil {
				handleError(err, fmt.Sprintf("executing %s", item.Label))
			}
		}

		if item.ExitItem {
			if menu.ExitText != "" {
				fmt.Println(menu.ExitText)
			}
			return
		}
	}
}

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
