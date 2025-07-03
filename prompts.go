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
	return filepath.Join(utilPath, "prompts.json")
}

// PromptError wraps prompt-related errors
type PromptError struct {
	Op  string
	Err error
}

func (e *PromptError) Error() string {
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// handlePromptError handles prompt-specific errors with appropriate user feedback
func handlePromptError(err error, operation string) {
	if err != nil {
		fmt.Printf("\033[31mPrompt error during %s: %v\033[0m\n", operation, err)
	}
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
		return nil, &PromptError{"read prompts file", err}
	}

	var prompts []Prompt
	if err := json.Unmarshal(data, &prompts); err != nil {
		return nil, &PromptError{"parse prompts file", err}
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
		return nil, &PromptError{"initialize default prompts", err}
	}

	fmt.Println("Initialized prompts file with defaults.")
	return defaultPrompts, nil
}

// Save prompts to JSON
func savePrompts(prompts []Prompt) error {
	data, err := json.MarshalIndent(prompts, "", "  ")
	if err != nil {
		return &PromptError{"marshal prompts", err}
	}

	if err := os.WriteFile(promptsConfigPath(), data, 0644); err != nil {
		return &PromptError{"write prompts file", err}
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

// Set a prompt as the default
func setPromptAsDefault(name string) error {
	prompts, err := loadPrompts()
	if err != nil {
		return err
	}

	found := false
	for i := range prompts {
		if prompts[i].Name == name {
			prompts[i].Default = true
			found = true
		} else {
			prompts[i].Default = false
		}
	}

	if !found {
		return fmt.Errorf("prompt '%s' not found", name)
	}

	return savePrompts(prompts)
}

// addPromptFlow adds a new prompt interactively
func addPromptFlow(reader *bufio.Reader) error {
	fmt.Print("Enter prompt name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		return &PromptError{"validate input", fmt.Errorf("prompt name cannot be empty")}
	}

	fmt.Print("Enter prompt content: ")
	content, _ := reader.ReadString('\n')
	content = strings.TrimSpace(content)
	if content == "" {
		return &PromptError{"validate input", fmt.Errorf("prompt content cannot be empty")}
	}

	prompts, err := loadPrompts()
	if err != nil {
		return err
	}

	// Check for duplicate names and ask for confirmation
	for i, p := range prompts {
		if p.Name == name {
			fmt.Printf("%s exists, do you want to overwrite?\n", name)
			fmt.Println("1) yes")
			fmt.Println("2) no")

			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)

			if choice == "1" || strings.ToLower(choice) == "yes" {
				// Overwrite the existing prompt
				prompts[i].Content = content
				if err := savePrompts(prompts); err != nil {
					return err
				}
				fmt.Printf("Overwritten prompt: %s\n", name)
				return nil
			}
			return nil // User chose not to overwrite
		}
	}

	// If we get here, it's a new prompt
	prompts = append(prompts, Prompt{
		Name:    name,
		Content: content,
		Default: len(prompts) == 0, // Make default if it's the first prompt
	})

	if err := savePrompts(prompts); err != nil {
		return err
	}

	fmt.Printf("Added new prompt: %s\n", name)
	return nil
}

// menuListPrompts lists prompts and allows viewing content
func menuListPrompts(reader *bufio.Reader) error {
	for {
		prompts, err := loadPrompts()
		if err != nil {
			fmt.Println("Error loading prompts:", err)
			return err
		}

		fmt.Println("\nSaved Prompts:")
		for i, p := range prompts {
			mark := " "
			if p.Default {
				mark = "*"
			}
			fmt.Printf("%d) %s %s\n", i+1, p.Name, mark)
		}
		fmt.Print("Select a prompt to view it (or press Enter to return to main menu): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return nil // back to main menu
		}

		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(prompts) {
			fmt.Println("Invalid selection.")
			continue
		}

		selected := prompts[choice-1]
		fmt.Printf("\n--- %s %s---\n%s\n--------------------\n\n",
			selected.Name,
			map[bool]string{true: "(Default) ", false: ""}[selected.Default],
			selected.Content)
	}
}

// promptPromptSelection allows selecting a prompt for chat
func promptPromptSelection(reader *bufio.Reader) (string, string, error) {
	prompts, err := loadPrompts()
	if err != nil {
		return "", "", err
	}

	if len(prompts) == 0 {
		return "", "", &PromptError{"select prompt", fmt.Errorf("no prompts available")}
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
		return "", "", &PromptError{"validate input", fmt.Errorf("invalid prompt number")}
	}

	selected := prompts[idx-1]
	return selected.Name, selected.Content, nil
}

// setDefaultPromptFlow allows selecting a prompt to set as default
func setDefaultPromptFlow(reader *bufio.Reader) error {
	prompts, err := loadPrompts()
	if err != nil {
		return err
	}

	if len(prompts) == 0 {
		return &PromptError{"set default", fmt.Errorf("no prompts available")}
	}

	fmt.Println("\nAvailable prompts:")
	for i, p := range prompts {
		mark := " "
		if p.Default {
			mark = "*"
		}
		fmt.Printf("%d) %s %s\n", i+1, p.Name, mark)
	}

	fmt.Print("Select prompt number to set as default: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(prompts) {
		return &PromptError{"validate input", fmt.Errorf("invalid prompt number")}
	}

	// Update default status
	for i := range prompts {
		prompts[i].Default = i == idx-1
	}

	if err := savePrompts(prompts); err != nil {
		return err
	}

	fmt.Printf("Set '%s' as default prompt\n", prompts[idx-1].Name)
	return nil
}

// removePromptFlow allows removing a prompt
func removePromptFlow(reader *bufio.Reader) error {
	prompts, err := loadPrompts()
	if err != nil {
		return err
	}

	if len(prompts) == 0 {
		return &PromptError{"remove prompt", fmt.Errorf("no prompts available")}
	}

	fmt.Println("\nSelect prompt to remove:")
	for i, p := range prompts {
		mark := " "
		if p.Default {
			mark = "*"
		}
		fmt.Printf("%d) %s %s\n", i+1, p.Name, mark)
	}

	fmt.Print("Enter prompt number to remove: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(prompts) {
		return &PromptError{"validate input", fmt.Errorf("invalid prompt number")}
	}

	removedPrompt := prompts[idx-1].Name
	wasDefault := prompts[idx-1].Default

	// Remove the prompt
	prompts = append(prompts[:idx-1], prompts[idx:]...)

	// If we removed the default prompt, set a new default
	if wasDefault && len(prompts) > 0 {
		prompts[0].Default = true
	}

	if err := savePrompts(prompts); err != nil {
		return err
	}

	fmt.Printf("Removed prompt: %s\n", removedPrompt)
	return nil
}

// promptsMenu handles the prompts submenu
func promptsMenu(reader *bufio.Reader) error {
	menu := Menu{
		Title: "Prompts Menu",
		Items: []MenuItem{
			{Label: "List prompts", Handler: func(r *bufio.Reader) error {
				return menuListPrompts(r)
			}},
			{Label: "Add prompt", Handler: addPromptFlow},
			{Label: "Remove prompt", Handler: removePromptFlow},
			{Label: "Set default prompt", Handler: setDefaultPromptFlow},
			{Label: "Back to main menu", ExitItem: true},
		},
	}
	RunMenu(menu, reader)
	return nil
}
