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
	"strings"
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

func modelsFilePath() string {
	return filepath.Join(utilPath(), "models.json")
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
		return err
	}

	if err := os.WriteFile(modelsFilePath(), data, 0644); err != nil {
		return err
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
				return nil, "", err
			}
			defaultModel := DefaultModel()
			return []string{defaultModel}, defaultModel, nil
		}
		return nil, "", err
	}

	var config ModelsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, "", err
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

// streamChatResponse handles the chat API response streaming
func streamChatResponse(messages []Message, model string) (string, error) {
	key, url, err := getActiveAPIKeyAndURL()
	if err != nil {
		handleError(err, "getting active API key and url")
		return "", err
	}
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

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		handleError(err, "creating API request")
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+key)
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
