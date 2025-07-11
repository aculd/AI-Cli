// Package repositories provides cached repository implementations for persistent data storage.
package repositories

import (
	"aichat/src/errors"
	"aichat/src/models"
	"aichat/src/services/cache"
)

// CachedPromptRepository provides cached access to prompt templates
type CachedPromptRepository struct {
	cacheManager *cache.CacheManager
	filePath     string
}

// NewCachedPromptRepository creates a new cached prompt repository
func NewCachedPromptRepository() *CachedPromptRepository {
	return &CachedPromptRepository{
		cacheManager: cache.NewCacheManager(),
		filePath:     "src/.config/prompts.json",
	}
}

// GetAll retrieves all prompts from cache or loads from file
func (r *CachedPromptRepository) GetAll() ([]models.Prompt, error) {
	prompts, err := r.cacheManager.GetPrompts(r.filePath)
	if err != nil {
		return nil, errors.NewCacheError("get_prompts", err)
	}
	return prompts, nil
}

// GetByID retrieves a specific prompt by name
func (r *CachedPromptRepository) GetByID(name string) (*models.Prompt, error) {
	prompts, err := r.GetAll()
	if err != nil {
		return nil, err
	}

	for _, prompt := range prompts {
		if prompt.Name == name {
			return &prompt, nil
		}
	}

	return nil, errors.NewNotFoundError("prompt", name)
}

// Save saves a prompt and invalidates the cache
func (r *CachedPromptRepository) Save(prompt *models.Prompt) error {
	if prompt == nil || prompt.Name == "" {
		return errors.NewValidationError("prompt", "invalid prompt data")
	}

	// Load existing prompts
	prompts, err := r.GetAll()
	if err != nil {
		return err
	}

	// Update or add the prompt
	found := false
	for i, p := range prompts {
		if p.Name == prompt.Name {
			prompts[i] = *prompt
			found = true
			break
		}
	}

	if !found {
		prompts = append(prompts, *prompt)
	}

	// Save to file
	if err := r.saveToFile(prompts); err != nil {
		return errors.NewStorageError("save_prompts", r.filePath, err)
	}

	// Invalidate cache
	r.cacheManager.InvalidatePrompts(r.filePath)
	return nil
}

// Delete removes a prompt and invalidates the cache
func (r *CachedPromptRepository) Delete(name string) error {
	prompts, err := r.GetAll()
	if err != nil {
		return err
	}

	// Filter out the prompt to delete
	var newPrompts []models.Prompt
	found := false
	for _, p := range prompts {
		if p.Name != name {
			newPrompts = append(newPrompts, p)
		} else {
			found = true
		}
	}

	if !found {
		return errors.NewNotFoundError("prompt", name)
	}

	// Save to file
	if err := r.saveToFile(newPrompts); err != nil {
		return errors.NewStorageError("save_prompts", r.filePath, err)
	}

	// Invalidate cache
	r.cacheManager.InvalidatePrompts(r.filePath)
	return nil
}

// GetDefault retrieves the default prompt
func (r *CachedPromptRepository) GetDefault() (*models.Prompt, error) {
	prompts, err := r.GetAll()
	if err != nil {
		return nil, err
	}

	// Find default prompt
	for _, p := range prompts {
		if p.Default {
			return &p, nil
		}
	}

	// If no default, use first prompt
	if len(prompts) > 0 {
		return &prompts[0], nil
	}

	return nil, errors.NewNotFoundError("prompt", "default")
}

// SetDefault sets a prompt as the default and invalidates cache
func (r *CachedPromptRepository) SetDefault(name string) error {
	prompts, err := r.GetAll()
	if err != nil {
		return err
	}

	// Update default flags
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
		return errors.NewNotFoundError("prompt", name)
	}

	// Save to file
	if err := r.saveToFile(prompts); err != nil {
		return errors.NewStorageError("save_prompts", r.filePath, err)
	}

	// Invalidate cache
	r.cacheManager.InvalidatePrompts(r.filePath)
	return nil
}

// GetStats returns cache statistics
func (r *CachedPromptRepository) GetStats() map[string]cache.CacheStats {
	return r.cacheManager.GetStats()
}

// saveToFile saves prompts to the JSON file
func (r *CachedPromptRepository) saveToFile(prompts []models.Prompt) error {
	return models.SavePromptsToFile(prompts, r.filePath)
}
