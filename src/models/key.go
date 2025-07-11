package models

import (
	"encoding/json"
	"os"
)

// APIKey represents a single API key with a title, key, URL, and active status.
type APIKey struct {
	Title  string `json:"title"`
	Key    string `json:"key"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

// APIKeysConfig represents the configuration for multiple API keys.
type APIKeysConfig struct {
	Keys []APIKey `json:"keys"`
}

// SaveAPIKeysToFile saves API keys configuration to a JSON file
func SaveAPIKeysToFile(config APIKeysConfig, filePath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return &StorageError{
			Message: "failed to marshal API keys to JSON",
			Err:     err,
		}
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return &StorageError{
			Message: "failed to write API keys to file",
			Err:     err,
		}
	}

	return nil
}
