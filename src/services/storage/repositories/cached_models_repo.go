// Package repositories provides cached repository implementations for persistent data storage.
package repositories

import (
	"aichat/src/errors"
	"aichat/src/models"
	"aichat/src/services/cache"
)

// CachedModelRepository provides cached access to AI model configurations
type CachedModelRepository struct {
	cacheManager *cache.CacheManager
	filePath     string
}

// NewCachedModelRepository creates a new cached model repository
func NewCachedModelRepository() *CachedModelRepository {
	return &CachedModelRepository{
		cacheManager: cache.NewCacheManager(),
		filePath:     "src/.config/models.json",
	}
}

// GetAll retrieves all models from cache or loads from file
func (r *CachedModelRepository) GetAll() ([]models.Model, error) {
	modelList, err := r.cacheManager.GetModels(r.filePath)
	if err != nil {
		return nil, errors.NewCacheError("get_models", err)
	}
	return modelList, nil
}

// GetByID retrieves a specific model by name
func (r *CachedModelRepository) GetByID(name string) (*models.Model, error) {
	modelList, err := r.GetAll()
	if err != nil {
		return nil, err
	}

	for _, model := range modelList {
		if model.Name == name {
			return &model, nil
		}
	}

	return nil, errors.NewNotFoundError("model", name)
}

// Save saves a model and invalidates the cache
func (r *CachedModelRepository) Save(model *models.Model) error {
	if model == nil || model.Name == "" {
		return errors.NewValidationError("model", "invalid model data")
	}

	// Load existing models
	modelList, err := r.GetAll()
	if err != nil {
		return err
	}

	// Update or add the model
	found := false
	for i, m := range modelList {
		if m.Name == model.Name {
			modelList[i] = *model
			found = true
			break
		}
	}

	if !found {
		modelList = append(modelList, *model)
	}

	// Save to file
	if err := r.saveToFile(modelList); err != nil {
		return errors.NewStorageError("save_models", r.filePath, err)
	}

	// Invalidate cache
	r.cacheManager.InvalidateModels(r.filePath)
	return nil
}

// Delete removes a model and invalidates the cache
func (r *CachedModelRepository) Delete(name string) error {
	modelList, err := r.GetAll()
	if err != nil {
		return err
	}

	// Filter out the model to delete
	var newModels []models.Model
	found := false
	for _, m := range modelList {
		if m.Name != name {
			newModels = append(newModels, m)
		} else {
			found = true
		}
	}

	if !found {
		return errors.NewNotFoundError("model", name)
	}

	// Save to file
	if err := r.saveToFile(newModels); err != nil {
		return errors.NewStorageError("save_models", r.filePath, err)
	}

	// Invalidate cache
	r.cacheManager.InvalidateModels(r.filePath)
	return nil
}

// GetDefault retrieves the default model
func (r *CachedModelRepository) GetDefault() (*models.Model, error) {
	modelList, err := r.GetAll()
	if err != nil {
		return nil, err
	}

	// Find default model
	for _, m := range modelList {
		if m.IsDefault {
			return &m, nil
		}
	}

	// If no default, use first model
	if len(modelList) > 0 {
		return &modelList[0], nil
	}

	return nil, errors.NewNotFoundError("model", "default")
}

// SetDefault sets a model as the default and invalidates cache
func (r *CachedModelRepository) SetDefault(name string) error {
	modelList, err := r.GetAll()
	if err != nil {
		return err
	}

	// Update default flags
	found := false
	for i := range modelList {
		if modelList[i].Name == name {
			modelList[i].IsDefault = true
			found = true
		} else {
			modelList[i].IsDefault = false
		}
	}

	if !found {
		return errors.NewNotFoundError("model", name)
	}

	// Save to file
	if err := r.saveToFile(modelList); err != nil {
		return errors.NewStorageError("save_models", r.filePath, err)
	}

	// Invalidate cache
	r.cacheManager.InvalidateModels(r.filePath)
	return nil
}

// GetStats returns cache statistics
func (r *CachedModelRepository) GetStats() map[string]cache.CacheStats {
	return r.cacheManager.GetStats()
}

// saveToFile saves models to the JSON file
func (r *CachedModelRepository) saveToFile(modelList []models.Model) error {
	return models.SaveModelsToFile(modelList, r.filePath)
}
