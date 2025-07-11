// Package repositories provides cached repository implementations for persistent data storage.
package repositories

import (
	"aichat/src/errors"
	"aichat/src/models"
	"aichat/src/services/cache"
)

// CachedAPIKeyRepository provides cached access to API key configurations
type CachedAPIKeyRepository struct {
	cacheManager *cache.CacheManager
	filePath     string
}

// NewCachedAPIKeyRepository creates a new cached API key repository
func NewCachedAPIKeyRepository() *CachedAPIKeyRepository {
	return &CachedAPIKeyRepository{
		cacheManager: cache.NewCacheManager(),
		filePath:     "src/.config/api_keys.json",
	}
}

// GetAll retrieves all API keys from cache or loads from file
func (r *CachedAPIKeyRepository) GetAll() ([]models.APIKey, error) {
	keys, err := r.cacheManager.GetAPIKeys(r.filePath)
	if err != nil {
		return nil, errors.NewCacheError("get_apikeys", err)
	}
	return keys, nil
}

// GetByTitle retrieves a specific API key by title
func (r *CachedAPIKeyRepository) GetByTitle(title string) (*models.APIKey, error) {
	keys, err := r.GetAll()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if key.Title == title {
			return &key, nil
		}
	}

	return nil, errors.NewNotFoundError("API key", title)
}

// GetActive retrieves the currently active API key
func (r *CachedAPIKeyRepository) GetActive() (*models.APIKey, error) {
	keys, err := r.GetAll()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if key.Active {
			return &key, nil
		}
	}

	return nil, errors.NewNotFoundError("API key", "active")
}

// Add adds a new API key and invalidates the cache
func (r *CachedAPIKeyRepository) Add(key models.APIKey) error {
	if key.Title == "" || key.Key == "" {
		return errors.NewValidationError("API key", "invalid API key data")
	}

	// Load existing keys
	keys, err := r.GetAll()
	if err != nil {
		return err
	}

	// Check if title already exists
	for _, existingKey := range keys {
		if existingKey.Title == key.Title {
			return errors.NewValidationError("API key title", "already exists")
		}
	}

	// Add the new key
	keys = append(keys, key)

	// Save to file
	if err := r.saveToFile(keys); err != nil {
		return errors.NewStorageError("save_apikeys", r.filePath, err)
	}

	// Invalidate cache
	r.cacheManager.InvalidateAPIKeys(r.filePath)
	return nil
}

// Remove removes an API key by title and invalidates the cache
func (r *CachedAPIKeyRepository) Remove(title string) error {
	keys, err := r.GetAll()
	if err != nil {
		return err
	}

	// Filter out the key to remove
	var newKeys []models.APIKey
	found := false
	for _, key := range keys {
		if key.Title != title {
			newKeys = append(newKeys, key)
		} else {
			found = true
		}
	}

	if !found {
		return errors.NewNotFoundError("API key", title)
	}

	// Save to file
	if err := r.saveToFile(newKeys); err != nil {
		return errors.NewStorageError("save_apikeys", r.filePath, err)
	}

	// Invalidate cache
	r.cacheManager.InvalidateAPIKeys(r.filePath)
	return nil
}

// SetActive sets an API key as active and invalidates the cache
func (r *CachedAPIKeyRepository) SetActive(title string) error {
	keys, err := r.GetAll()
	if err != nil {
		return err
	}

	// Update active flags
	found := false
	for i := range keys {
		if keys[i].Title == title {
			keys[i].Active = true
			found = true
		} else {
			keys[i].Active = false
		}
	}

	if !found {
		return errors.NewNotFoundError("API key", title)
	}

	// Save to file
	if err := r.saveToFile(keys); err != nil {
		return errors.NewStorageError("save_apikeys", r.filePath, err)
	}

	// Invalidate cache
	r.cacheManager.InvalidateAPIKeys(r.filePath)
	return nil
}

// Update updates an existing API key and invalidates the cache
func (r *CachedAPIKeyRepository) Update(key models.APIKey) error {
	if key.Title == "" || key.Key == "" {
		return errors.NewValidationError("API key", "invalid API key data")
	}

	// Load existing keys
	keys, err := r.GetAll()
	if err != nil {
		return err
	}

	// Update the key
	found := false
	for i := range keys {
		if keys[i].Title == key.Title {
			keys[i] = key
			found = true
			break
		}
	}

	if !found {
		return errors.NewNotFoundError("API key", key.Title)
	}

	// Save to file
	if err := r.saveToFile(keys); err != nil {
		return errors.NewStorageError("save_apikeys", r.filePath, err)
	}

	// Invalidate cache
	r.cacheManager.InvalidateAPIKeys(r.filePath)
	return nil
}

// GetStats returns cache statistics
func (r *CachedAPIKeyRepository) GetStats() map[string]cache.CacheStats {
	return r.cacheManager.GetStats()
}

// saveToFile saves API keys to the JSON file
func (r *CachedAPIKeyRepository) saveToFile(keys []models.APIKey) error {
	config := models.APIKeysConfig{Keys: keys}
	return models.SaveAPIKeysToFile(config, r.filePath)
}
