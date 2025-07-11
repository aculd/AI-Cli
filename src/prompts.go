package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Prompt represents a single prompt with its content and default status
type Prompt struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Default bool   `json:"default"`
}

// PromptsConfig represents the prompts configuration stored in JSON
type PromptsConfig struct {
	Prompts []Prompt `json:"prompts"`
}

// Path helpers
func promptsConfigPath() string {
	return filepath.Join(utilPath(), "prompts.json")
}

// Load or create prompts configuration
func ensurePromptsConfig() error {
	if _, err := os.Stat(promptsConfigPath()); os.IsNotExist(err) {
		// Create initial config with default prompts
		defaultPrompts := []Prompt{
			{
				Name:    "General Assistant",
				Content: "You are a helpful assistant. Focus on providing clear, accurate information in a professional tone.",
				Default: true,
			},
			{
				Name:    "Code Helper",
				Content: "You are a coding assistant. Provide code examples and technical explanations with a focus on best practices.",
				Default: false,
			},
		}

		return savePrompts(defaultPrompts)
	}
	return nil
}

// Load prompts from JSON
func loadPrompts() ([]Prompt, error) {
	data, err := os.ReadFile(promptsConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return initializeDefaultPrompts()
		}
		return nil, fmt.Errorf("failed to read prompts.json: %w", err)
	}

	var prompts []Prompt
	if err := json.Unmarshal(data, &prompts); err != nil {
		return nil, fmt.Errorf("failed to parse prompts.json: %w", err)
	}

	return prompts, nil
}

// initializeDefaultPrompts creates default prompts if none exist
func initializeDefaultPrompts() ([]Prompt, error) {
	defaultPrompts := []Prompt{
		{
			Name:    "General Assistant",
			Content: "You are a helpful assistant. Focus on providing clear, accurate information in a professional tone.",
			Default: true,
		},
		{
			Name:    "Code Helper",
			Content: "You are a coding assistant. Provide code examples and technical explanations with a focus on best practices.",
			Default: false,
		},
	}

	if err := savePrompts(defaultPrompts); err != nil {
		return nil, err
	}

	fmt.Println("Initialized prompts file with defaults.")
	return defaultPrompts, nil
}

// Save prompts to JSON
func savePrompts(prompts []Prompt) error {
	data, err := json.MarshalIndent(prompts, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(promptsConfigPath(), data, 0644); err != nil {
		return err
	}

	return nil
}

// Get the current default prompt
func getDefaultPrompt() (Prompt, error) {
	prompts, err := loadPrompts()
	if err != nil {
		return Prompt{}, err
	}

	// Find default prompt
	for _, p := range prompts {
		if p.Default {
			return p, nil
		}
	}

	// If no default, use first prompt
	if len(prompts) > 0 {
		prompts[0].Default = true
		if err := savePrompts(prompts); err != nil {
			return Prompt{}, err
		}
		return prompts[0], nil
	}

	// If no prompts at all, create default
	prompts, err = initializeDefaultPrompts()
	if err != nil {
		return Prompt{}, err
	}
	return prompts[0], nil
}

// promptPromptSelection allows selecting a prompt for chat
func promptPromptSelection(reader *bufio.Reader) (string, string, error) {
	prompts, err := loadPrompts()
	if err != nil {
		return "", "", err
	}

	if len(prompts) == 0 {
		return "", "", fmt.Errorf("no prompts available")
	}

	fmt.Println("\nAvailable prompts:")
	for i, p := range prompts {
		mark := " "
		if p.Default {
			mark = "*"
		}
		fmt.Printf("%d) %s %s\n", i+1, p.Name, mark)
	}

	fmt.Print("Select prompt number: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(prompts) {
		return "", "", fmt.Errorf("invalid prompt number")
	}

	selected := prompts[idx-1]
	return selected.Name, selected.Content, nil
}
