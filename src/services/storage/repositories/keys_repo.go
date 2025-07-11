package repositories

import (
	"encoding/json"
	"os"
	"path/filepath"

	"aichat/src/models"
)

const apiKeysConfigPath = "src/.config/api_keys.json"

type APIKeyRepository struct {
	file string
}

func NewAPIKeyRepository() *APIKeyRepository {
	return &APIKeyRepository{file: apiKeysConfigPath}
}

func (r *APIKeyRepository) GetAll() ([]models.APIKey, error) {
	data, err := os.ReadFile(r.file)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.APIKey{}, nil
		}
		return nil, err
	}
	var config models.APIKeysConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config.Keys, nil
}

func (r *APIKeyRepository) SaveAll(keys []models.APIKey) error {
	if err := os.MkdirAll(filepath.Dir(r.file), 0755); err != nil {
		return err
	}
	config := models.APIKeysConfig{Keys: keys}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.file, data, 0600)
}

func (r *APIKeyRepository) Add(key models.APIKey) error {
	keys, err := r.GetAll()
	if err != nil {
		return err
	}
	keys = append(keys, key)
	return r.SaveAll(keys)
}

func (r *APIKeyRepository) Remove(title string) error {
	keys, err := r.GetAll()
	if err != nil {
		return err
	}
	newKeys := make([]models.APIKey, 0, len(keys))
	for _, k := range keys {
		if k.Title != title {
			newKeys = append(newKeys, k)
		}
	}
	return r.SaveAll(newKeys)
}

func (r *APIKeyRepository) SetActive(title string) error {
	keys, err := r.GetAll()
	if err != nil {
		return err
	}
	for i := range keys {
		keys[i].Active = (keys[i].Title == title)
	}
	return r.SaveAll(keys)
}
