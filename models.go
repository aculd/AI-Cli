package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	apiURL = "https://openrouter.ai/api/v1/chat/completions"
	apiKey string
)

// StreamRequestBody represents the request body for chat completions
type StreamRequestBody struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Code     int                    `json:"code"`
	Message  string                 `json:"message"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// StreamResponse represents the streaming response from the API
type StreamResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		FinishReason       string `json:"finish_reason"`
		NativeFinishReason string `json:"native_finish_reason"`
		Delta              struct {
			Content string `json:"content"`
			Role    string `json:"role,omitempty"`
		} `json:"delta"`
		Error *ErrorResponse `json:"error,omitempty"`
	} `json:"choices"`
	Model string `json:"model"`
}

// Model represents a single model with its name and default status
type Model struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

// ModelsConfig represents the models configuration stored in JSON
type ModelsConfig struct {
	Models []Model `json:"models"`
}

func init() {
	key, err := readAPIKey()
	if err != nil {
		fmt.Println("Error reading API key:", err)
		os.Exit(1)
	}
	apiKey = key
}

func modelsFilePath() string {
	return filepath.Join(utilPath, "models.json")
}

// ModelError wraps model-related errors
type ModelError struct {
	Op  string
	Err error
}

func (e *ModelError) Error() string {
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// handleModelError handles model-specific errors with appropriate user feedback
func handleModelError(err error, operation string) {
	if err != nil {
		fmt.Printf("\033[31mModel error during %s: %v\033[0m\n", operation, err)
	}
}

// DefaultModel returns fallback default model string
func DefaultModel() string {
	return "deepseek/deepseek-chat-v3-0324:free"
}

// initializeModelsFile creates the models file with defaults if missing
func initializeModelsFile() error {
	defaultModel := DefaultModel()
	config := ModelsConfig{
		Models: []Model{
			{Name: defaultModel, IsDefault: true},
			{Name: "openai/gpt-4", IsDefault: false},
			{Name: "meta-llama/llama-3-8b-instruct", IsDefault: false},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return &ModelError{"marshal default models", err}
	}

	if err := os.WriteFile(modelsFilePath(), data, 0644); err != nil {
		return &ModelError{"write models file", err}
	}

	fmt.Println("Initialized models file with defaults.")
	return nil
}

// loadModelsWithMostRecent reads models from JSON and returns list plus default model
func loadModelsWithMostRecent() ([]string, string, error) {
	data, err := os.ReadFile(modelsFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			if err := initializeModelsFile(); err != nil {
				return nil, "", &ModelError{"initialize models file", err}
			}
			defaultModel := DefaultModel()
			return []string{defaultModel}, defaultModel, nil
		}
		return nil, "", &ModelError{"read models file", err}
	}

	var config ModelsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, "", &ModelError{"parse models file", err}
	}

	var models []string
	var defaultModel string
	for _, model := range config.Models {
		models = append(models, model.Name)
		if model.IsDefault {
			defaultModel = model.Name
		}
	}

	if len(models) == 0 {
		defaultModel = DefaultModel()
		models = []string{defaultModel}
	} else if defaultModel == "" {
		defaultModel = models[0]
	}

	return models, defaultModel, nil
}

// saveModelsWithMostRecent saves models list with updated default model
func saveModelsWithMostRecent(defaultModel string, modelNames []string) error {
	var config ModelsConfig
	for _, name := range modelNames {
		config.Models = append(config.Models, Model{
			Name:      name,
			IsDefault: name == defaultModel,
		})
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return &ModelError{"marshal models", err}
	}

	if err := os.WriteFile(modelsFilePath(), data, 0644); err != nil {
		return &ModelError{"save models file", err}
	}

	return nil
}

// selectModel interactively lets user pick or add models
func selectModel(reader *bufio.Reader) (string, error) {
	models, mostRecent, err := loadModelsWithMostRecent()
	if err != nil {
		return "", err
	}

	fmt.Println("\nAvailable models:")
	for i, m := range models {
		mark := " "
		if m == mostRecent {
			mark = "*"
		}
		fmt.Printf("%d) %s %s\n", i+1, m, mark)
	}
	fmt.Println("a) Add new model")
	fmt.Printf("Choose model number or 'a' to add (default *): ")

	_, _ = reader.ReadString('\n') // Ignore the input since we're using the mostRecent value
	return mostRecent, nil
}

func promptModelAtChatStart(reader *bufio.Reader) (string, error) {
	models, mostRecent, err := loadModelsWithMostRecent()
	if err != nil {
		return "", err
	}

	fmt.Println("\nSelect model for this chat:")
	for i, model := range models {
		mark := " "
		if model == mostRecent {
			mark = "*"
		}
		fmt.Printf("%d) %s %s\n", i+1, model, mark)
	}

	fmt.Printf("Enter model number (or press Enter for default '%s'): ", mostRecent)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return mostRecent, nil
	}

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(models) {
		fmt.Println("Invalid input; using default model.")
		return mostRecent, nil
	}

	return models[choice-1], nil
}

// streamChatResponse handles the chat API response streaming
func streamChatResponse(messages []Message, model string) (string, error) {
	reqBody := StreamRequestBody{
		Model:       model,
		Messages:    messages,
		Stream:      true,
		MaxTokens:   2048,
		Temperature: 0.7,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		handleError(err, "marshaling request body")
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		handleError(err, "creating API request")
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/go-ai-cli")
	req.Header.Set("X-Title", "Go AI CLI")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		handleError(err, "making API request")
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var errorResp struct {
			Error ErrorResponse `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return "", fmt.Errorf("API error %d: %s", errorResp.Error.Code, errorResp.Error.Message)
		}
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	var fullReply strings.Builder
	var buffer string

	// Only print to stdout if it's not nil
	printToStdout := os.Stdout != nil
	if printToStdout {
		fmt.Print("\033[34mAssistant:\033[0m ")
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			handleError(err, "reading stream response")
			return fullReply.String(), err
		}

		line = strings.TrimSpace(line)

		// Handle server-sent events comments
		if strings.HasPrefix(line, ":") {
			// Skip SSE comments (e.g., ": OPENROUTER PROCESSING")
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := line[len("data: "):]
		if data == "[DONE]" {
			break
		}

		// Append new chunk to buffer
		buffer += data

		// Process complete JSON objects from buffer
		for {
			openBrace := strings.Index(buffer, "{")
			if openBrace == -1 {
				break
			}

			// Find matching closing brace
			depth := 1
			closeBrace := -1
			for i := openBrace + 1; i < len(buffer); i++ {
				if buffer[i] == '{' {
					depth++
				} else if buffer[i] == '}' {
					depth--
					if depth == 0 {
						closeBrace = i
						break
					}
				}
			}

			if closeBrace == -1 {
				break
			}

			jsonStr := buffer[openBrace : closeBrace+1]
			buffer = buffer[closeBrace+1:]

			var streamResp StreamResponse
			if err := json.Unmarshal([]byte(jsonStr), &streamResp); err != nil {
				handleError(err, "parsing stream response")
				continue
			}

			if len(streamResp.Choices) > 0 {
				content := streamResp.Choices[0].Delta.Content
				if content != "" {
					if printToStdout {
						fmt.Print(content)
					}
					fullReply.WriteString(content)
					os.Stdout.Sync()
				}
			}
		}
	}

	if printToStdout {
		fmt.Println()
	}
	return fullReply.String(), nil
}

// setDefaultModelFlow allows selecting a model to set as default
func setDefaultModelFlow(reader *bufio.Reader) error {
	models, mostRecent, err := loadModelsWithMostRecent()
	if err != nil {
		return err
	}

	if len(models) == 0 {
		return fmt.Errorf("no models available")
	}

	fmt.Println("\nSelect model to set as default:")
	for i, m := range models {
		mark := " "
		if m == mostRecent {
			mark = "*"
		}
		fmt.Printf("%d) %s %s\n", i+1, m, mark)
	}

	fmt.Print("Enter model number to set as default: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(models) {
		return fmt.Errorf("invalid model number")
	}

	newDefault := models[idx-1]
	if err := saveModelsWithMostRecent(newDefault, models); err != nil {
		return fmt.Errorf("failed to save models: %w", err)
	}

	fmt.Printf("Set '%s' as default model\n", newDefault)
	return nil
}

func modelsMenu(reader *bufio.Reader) error {
	menu := Menu{
		Title: "Models Menu",
		Items: []MenuItem{
			{Label: "List models", Handler: func(r *bufio.Reader) error {
				listModels()
				return nil
			}},
			{Label: "Add model", Handler: addModelFlow},
			{Label: "Remove model", Handler: removeModelFlow},
			{Label: "Set default model", Handler: setDefaultModelFlow},
			{Label: "Back to main menu", ExitItem: true},
		},
	}
	RunMenu(menu, reader)
	return nil
}

func listModels() {
	models, mostRecent, err := loadModelsWithMostRecent()
	if err != nil {
		fmt.Println("Error loading models:", err)
		return
	}

	fmt.Println("\nAvailable models:")
	for i, m := range models {
		mark := " "
		if m == mostRecent {
			mark = "*"
		}
		fmt.Printf("%d) %s %s\n", i+1, m, mark)
	}
}

func addModelFlow(reader *bufio.Reader) error {
	fmt.Print("Enter new model name: ")
	newModel, _ := reader.ReadString('\n')
	newModel = strings.TrimSpace(newModel)

	if newModel == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	models, mostRecent, err := loadModelsWithMostRecent()
	if err != nil {
		return err
	}

	// Check for duplicates
	for _, m := range models {
		if m == newModel {
			return fmt.Errorf("model already exists")
		}
	}

	models = append(models, newModel)
	if err := saveModelsWithMostRecent(mostRecent, models); err != nil {
		return fmt.Errorf("failed to save models: %w", err)
	}

	fmt.Printf("Added new model: %s\n", newModel)
	return nil
}

func removeModelFlow(reader *bufio.Reader) error {
	models, mostRecent, err := loadModelsWithMostRecent()
	if err != nil {
		return err
	}

	fmt.Println("\nSelect model to remove:")
	for i, m := range models {
		mark := " "
		if m == mostRecent {
			mark = "*"
		}
		fmt.Printf("%d) %s %s\n", i+1, m, mark)
	}

	fmt.Print("Enter model number to remove: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(models) {
		return fmt.Errorf("invalid model number")
	}

	removedModel := models[idx-1]

	// Update most recent if we're removing it
	if removedModel == mostRecent {
		if len(models) > 1 {
			// Set most recent to another model
			if idx == 1 && len(models) > 1 {
				mostRecent = models[1]
			} else {
				mostRecent = models[0]
			}
		} else {
			mostRecent = DefaultModel()
		}
	}

	// Remove the model
	models = append(models[:idx-1], models[idx:]...)

	if len(models) == 0 {
		models = []string{DefaultModel()}
	}

	if err := saveModelsWithMostRecent(mostRecent, models); err != nil {
		return fmt.Errorf("failed to save models: %w", err)
	}

	fmt.Printf("Removed model: %s\n", removedModel)
	return nil
}
