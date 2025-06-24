package main

import (
	"bytes"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type StreamRequestBody struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type StreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
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

var (
	apiURL = "https://openrouter.ai/api/v1/chat/completions"
	apiKey string
)

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
		return fmt.Errorf("failed to marshal default models: %w", err)
	}

	return os.WriteFile(modelsFilePath(), data, 0644)
}

// loadModelsWithMostRecent reads models from JSON and returns list plus default model
func loadModelsWithMostRecent() ([]string, string, error) {
	data, err := os.ReadFile(modelsFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			defaultModel := DefaultModel()
			return []string{defaultModel}, defaultModel, nil
		}
		return nil, "", err
	}

	var config ModelsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, "", fmt.Errorf("failed to parse models file: %w", err)
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
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	return os.WriteFile(modelsFilePath(), data, 0644)
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

	_, _ = reader.ReadString('\n')  // Ignore the input since we're using the mostRecent value
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

func runChat(chatName string, messages []Message, reader *bufio.Reader, model string) {
	// Ensure system prompt is first message
	messages = prependSystemPrompt(messages, systemPrompt)

	// If only system prompt exists, send it once to AI and print response
	if len(messages) == 1 {
		fmt.Println("Sending initial system prompt to AI...")
		resp, err := streamChatResponse(messages, model)
		if err != nil {
			fmt.Println("API error:", err)
		} else {
			fmt.Println("\nAssistant:", resp)
			messages = append(messages, Message{Role: "assistant", Content: resp})
		}
	}

	for {
		fmt.Print("\033[31mYou:\033[0m ") // Red colored "You:"
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)
		if userInput == "" {
			continue
		}

		switch userInput {
		case "!q", "!quit", "!exit", "!e":
			if len(messages) > 1 {
				if err := saveChat(chatName, messages); err != nil {
					fmt.Println("Error saving chat:", err)
				} else {
					fmt.Println("Chat saved as:", chatName)
				}
			}
			fmt.Println("Exiting chat.")
			return
		case "!save":
			if len(messages) > 1 {
				if err := saveChat(chatName, messages); err != nil {
					fmt.Println("Error saving chat:", err)
				} else {
					fmt.Println("Chat saved as:", chatName)
				}
			} else {
				fmt.Println("No messages to save.")
			}
			continue
		}

		// Append user message
		messages = append(messages, Message{Role: "user", Content: userInput})

		// Get AI response
		reply, err := streamChatResponse(messages, model)
		if err != nil {
			fmt.Println("API error:", err)
			messages = messages[:len(messages)-1] // Remove last user message on error
			continue
		}

		// Append assistant message
		messages = append(messages, Message{Role: "assistant", Content: reply})

		// Save after each response
		if err := saveChat(chatName, messages); err != nil {
			fmt.Println("Error auto-saving chat:", err)
		}
	}
}

func streamChatResponse(messages []Message, model string) (string, error) {
	reqBody := StreamRequestBody{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	var fullReply strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fullReply.String(), err
		}

		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := line[len("data: "):]
		if data == "[DONE]" {
			break
		}

		var streamResp StreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil || len(streamResp.Choices) == 0 {
			continue
		}

		content := streamResp.Choices[0].Delta.Content
		if content != "" {
			fmt.Print(content)  // Real-time print
			fullReply.WriteString(content)
			os.Stdout.Sync()
		}
	}

	fmt.Println() // Final newline after streaming completes
	return fullReply.String(), nil
}

func modelsMenu(reader *bufio.Reader) {
    for {
        fmt.Println("\nModels Menu:")
        fmt.Println("1) List models")
        fmt.Println("2) Add model")
        fmt.Println("3) Remove model")
        fmt.Println("4) Back to main menu")
        fmt.Print("Choose option: ")
        
        input, _ := reader.ReadString('\n')
        input = strings.TrimSpace(input)

        switch input {
        case "1":
            listModels()
        case "2":
            if err := addModelFlow(reader); err != nil {
                fmt.Println("Error:", err)
            }
        case "3":
            if err := removeModelFlow(reader); err != nil {
                fmt.Println("Error:", err)
            }
        case "4", "":
            return
        default:
            fmt.Println("Invalid models menu option.")
        }
    }
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
