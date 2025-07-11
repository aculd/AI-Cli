package models

import (
	"encoding/json"
	"os"
)

// Model represents an AI model configuration.
type Model struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

// ModelsConfig represents the models configuration stored in JSON.
type ModelsConfig struct {
	Models []Model `json:"models"`
}

// SaveModelsToFile saves a slice of models to a JSON file
func SaveModelsToFile(models []Model, filePath string) error {
	config := ModelsConfig{Models: models}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return &StorageError{
			Message: "failed to marshal models to JSON",
			Err:     err,
		}
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return &StorageError{
			Message: "failed to write models to file",
			Err:     err,
		}
	}

	return nil
}
