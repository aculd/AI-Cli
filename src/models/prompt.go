package models

import (
	"encoding/json"
	"os"
)

// Prompt represents a prompt template for the assistant.
type Prompt struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Default bool   `json:"default,omitempty"`
}

// SavePromptsToFile saves a slice of prompts to a JSON file
func SavePromptsToFile(prompts []Prompt, filePath string) error {
	data, err := json.MarshalIndent(prompts, "", "  ")
	if err != nil {
		return &StorageError{
			Message: "failed to marshal prompts to JSON",
			Err:     err,
		}
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return &StorageError{
			Message: "failed to write prompts to file",
			Err:     err,
		}
	}

	return nil
}
