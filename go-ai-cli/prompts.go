package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Prompt represents a single prompt with its content and default status
type Prompt struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	IsDefault bool   `json:"is_default"`
}

// PromptsConfig represents the prompts configuration stored in JSON
type PromptsConfig struct {
	Prompts []Prompt `json:"prompts"`
}

// Path helpers
func promptsConfigPath() string {
	return filepath.Join(utilPath, "prompts.json")
}

// Load or create prompts configuration
func ensurePromptsConfig() error {
	if _, err := os.Stat(promptsConfigPath()); os.IsNotExist(err) {
		// Create initial config with default prompt
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
			return fmt.Errorf("failed to marshal default prompts: %w", err)
		}
		return os.WriteFile(promptsConfigPath(), data, 0644)
	}
	return nil
}

// Load prompts from JSON
func loadPrompts() ([]Prompt, error) {
	data, err := os.ReadFile(promptsConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			if err := ensurePromptsConfig(); err != nil {
				return nil, err
			}
			data, err = os.ReadFile(promptsConfigPath())
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var config PromptsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse prompts file: %w", err)
	}

	if len(config.Prompts) == 0 {
		// Ensure there's always at least the default prompt
		config.Prompts = []Prompt{
			{
				Name:      "Default Prompt",
				Content:   systemPrompt.Content,
				IsDefault: true,
			},
		}
		// Save the fixed config
		if err := savePrompts(config.Prompts); err != nil {
			return nil, err
		}
	}

	return config.Prompts, nil
}

// Save prompts to JSON
func savePrompts(prompts []Prompt) error {
	config := PromptsConfig{Prompts: prompts}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal prompts: %w", err)
	}
	return os.WriteFile(promptsConfigPath(), data, 0644)
}

// Get the current default prompt
func getDefaultPrompt() (Prompt, error) {
	prompts, err := loadPrompts()
	if err != nil {
		return Prompt{}, err
	}

	for _, p := range prompts {
		if p.IsDefault {
			return p, nil
		}
	}

	// If no default found, use the first prompt or create a new default
	if len(prompts) > 0 {
		prompts[0].IsDefault = true
		if err := savePrompts(prompts); err != nil {
			return Prompt{}, err
		}
		return prompts[0], nil
	}

	// Create new default prompt
	defaultPrompt := Prompt{
		Name:      "Default Prompt",
		Content:   systemPrompt.Content,
		IsDefault: true,
	}
	if err := savePrompts([]Prompt{defaultPrompt}); err != nil {
		return Prompt{}, err
	}
	return defaultPrompt, nil
}

// Set a prompt as the default
func setPromptAsDefault(name string) error {
	prompts, err := loadPrompts()
	if err != nil {
		return err
	}

	found := false
	for i := range prompts {
		if prompts[i].Name == name {
			prompts[i].IsDefault = true
			found = true
		} else {
			prompts[i].IsDefault = false
		}
	}

	if !found {
		return fmt.Errorf("prompt '%s' not found", name)
	}

	return savePrompts(prompts)
}

// addPromptFlow adds a new prompt interactively
func addPromptFlow(reader *bufio.Reader) error {
	fmt.Print("Enter new prompt name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("prompt name cannot be empty")
	}

	fmt.Println("Enter prompt text (end with a single line containing only '~end'):")
	var lines []string
	for {
		line, _ := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line == "~end" {
			break
		}
		lines = append(lines, line)
	}
	content := strings.Join(lines, "\n")

	prompts, err := loadPrompts()
	if err != nil {
		return err
	}

	// Check for duplicates
	for _, p := range prompts {
		if p.Name == name {
			return fmt.Errorf("prompt '%s' already exists", name)
		}
	}

	// Add new prompt (non-default)
	prompts = append(prompts, Prompt{
		Name:      name,
		Content:   content,
		IsDefault: false,
	})

	if err := savePrompts(prompts); err != nil {
		return err
	}

	fmt.Printf("Added new prompt: %s\n", name)
	return nil
}

// menuListPrompts lists prompts and allows viewing content
func menuListPrompts(reader *bufio.Reader) {
	for {
		prompts, err := loadPrompts()
		if err != nil {
			fmt.Println("Error loading prompts:", err)
			return
		}

		fmt.Println("\nSaved Prompts:")
		for i, p := range prompts {
			mark := " "
			if p.IsDefault {
				mark = "*"
			}
			fmt.Printf("%d) %s %s\n", i+1, p.Name, mark)
		}
		fmt.Print("Select a prompt to view it (or press Enter to return to main menu): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return // back to main menu
		}

		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(prompts) {
			fmt.Println("Invalid selection.")
			continue
		}

		selected := prompts[choice-1]
		fmt.Printf("\n--- %s %s---\n%s\n--------------------\n\n",
			selected.Name,
			map[bool]string{true: "(Default) ", false: ""}[selected.IsDefault],
			selected.Content)
	}
}

// promptPromptSelection allows selecting a prompt for chat
func promptPromptSelection(reader *bufio.Reader) (string, string, error) {
	prompts, err := loadPrompts()
	if err != nil {
		return "", "", err
	}

	fmt.Println("\nSelect a prompt for this chat:")
	for i, p := range prompts {
		mark := " "
		if p.IsDefault {
			mark = "*"
		}
		fmt.Printf("%d) %s %s\n", i+1, p.Name, mark)
	}

	fmt.Print("Enter prompt number (or press Enter for default): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		for _, p := range prompts {
			if p.IsDefault {
				return p.Name, p.Content, nil
			}
		}
		// If no default found, use first prompt
		if len(prompts) > 0 {
			return prompts[0].Name, prompts[0].Content, nil
		}
		return "", "", fmt.Errorf("no prompts available")
	}

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(prompts) {
		// On invalid input, use default prompt
		for _, p := range prompts {
			if p.IsDefault {
				return p.Name, p.Content, nil
			}
		}
		return "", "", fmt.Errorf("invalid selection and no default prompt found")
	}

	selected := prompts[choice-1]
	return selected.Name, selected.Content, nil
}

// setDefaultPromptFlow allows selecting a prompt to set as default
func setDefaultPromptFlow(reader *bufio.Reader) error {
	prompts, err := loadPrompts()
	if err != nil {
		return err
	}

	if len(prompts) == 0 {
		return fmt.Errorf("no prompts available")
	}

	fmt.Println("\nAvailable prompts:")
	for i, p := range prompts {
		mark := " "
		if p.IsDefault {
			mark = "*"
		}
		fmt.Printf("%d) %s %s\n", i+1, p.Name, mark)
	}

	fmt.Print("Select prompt number to set as default: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(prompts) {
		return fmt.Errorf("invalid prompt selection")
	}

	selected := prompts[choice-1]
	for i := range prompts {
		prompts[i].IsDefault = (i == choice-1)
	}

	if err := savePrompts(prompts); err != nil {
		return fmt.Errorf("failed to save prompts: %w", err)
	}

	fmt.Printf("Set '%s' as the default prompt.\n", selected.Name)
	return nil
}

// promptsMenu handles the prompts submenu
func promptsMenu(reader *bufio.Reader) {
	for {
		fmt.Println("\nPrompts Menu:")
		fmt.Println("1) List prompts")
		fmt.Println("2) Add new prompt")
		fmt.Println("3) Set Default Prompt")
		fmt.Println("4) Back to main menu")
		fmt.Print("Choose option: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			menuListPrompts(reader)
		case "2":
			if err := addPromptFlow(reader); err != nil {
				fmt.Println("Error:", err)
			}
		case "3":
			if err := setDefaultPromptFlow(reader); err != nil {
				fmt.Println("Error:", err)
			}
		case "4", "":
			return
		default:
			fmt.Println("Invalid prompt menu option.")
		}
	}
}