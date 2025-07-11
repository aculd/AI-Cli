// Package cache provides in-memory caching for application data with file modification detection.
package cache

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"aichat/src/errors"
	"aichat/src/models"
)

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Data         interface{} `json:"data"`
	Hash         string      `json:"hash"`
	LastModified time.Time   `json:"last_modified"`
	CreatedAt    time.Time   `json:"created_at"`
	AccessCount  int64       `json:"access_count"`
}

// CacheStats provides statistics about cache usage
type CacheStats struct {
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	Evictions   int64     `json:"evictions"`
	Size        int       `json:"size"`
	MaxSize     int       `json:"max_size"`
	LastUpdated time.Time `json:"last_updated"`
}

// Cache provides thread-safe in-memory caching with file modification detection
type Cache struct {
	entries map[string]*CacheEntry
	mutex   sync.RWMutex
	stats   CacheStats
	maxSize int
}

// NewCache creates a new cache instance with the specified maximum size
func NewCache(maxSize int) *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSize,
		stats:   CacheStats{MaxSize: maxSize},
	}
}

// Get retrieves a cached item by key, checking file modification if applicable
func (c *Cache) Get(key string, filePath string) (interface{}, bool) {
	c.mutex.RLock()
	entry, exists := c.entries[key]
	c.mutex.RUnlock()

	if !exists {
		c.stats.Misses++
		return nil, false
	}

	// Check if file has been modified since last cache
	if filePath != "" {
		if modified, err := c.isFileModified(filePath, entry.LastModified); err == nil && modified {
			// File modified, invalidate cache
			c.mutex.Lock()
			delete(c.entries, key)
			c.stats.Evictions++
			c.stats.Size = len(c.entries)
			c.mutex.Unlock()
			c.stats.Misses++
			return nil, false
		}
	}

	// Update access statistics
	c.mutex.Lock()
	entry.AccessCount++
	c.stats.Hits++
	c.stats.LastUpdated = time.Now()
	c.mutex.Unlock()

	return entry.Data, true
}

// Set stores an item in the cache with optional file path for modification detection
func (c *Cache) Set(key string, data interface{}, filePath string) error {
	// Check if we need to evict entries due to size limit
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.entries) >= c.maxSize {
		c.evictLRU()
	}

	// Calculate file hash if path provided
	var hash string
	var lastModified time.Time
	if filePath != "" {
		if fileInfo, err := os.Stat(filePath); err == nil {
			lastModified = fileInfo.ModTime()
			if fileHash, err := c.calculateFileHash(filePath); err == nil {
				hash = fileHash
			}
		}
	}

	entry := &CacheEntry{
		Data:         data,
		Hash:         hash,
		LastModified: lastModified,
		CreatedAt:    time.Now(),
		AccessCount:  0,
	}

	c.entries[key] = entry
	c.stats.Size = len(c.entries)
	c.stats.LastUpdated = time.Now()

	return nil
}

// Invalidate removes a specific key from the cache
func (c *Cache) Invalidate(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.entries[key]; exists {
		delete(c.entries, key)
		c.stats.Evictions++
		c.stats.Size = len(c.entries)
	}
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.stats.Evictions += int64(c.stats.Size)
	c.stats.Size = 0
	c.stats.LastUpdated = time.Now()
}

// GetStats returns current cache statistics
func (c *Cache) GetStats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	stats := c.stats
	stats.Size = len(c.entries)
	return stats
}

// isFileModified checks if a file has been modified since the given time
func (c *Cache) isFileModified(filePath string, since time.Time) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return true, errors.NewStorageError("stat", filePath, err)
	}
	return fileInfo.ModTime().After(since), nil
}

// calculateFileHash calculates MD5 hash of a file
func (c *Cache) calculateFileHash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", errors.NewStorageError("read", filePath, err)
	}

	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:]), nil
}

// evictLRU removes the least recently used entry from the cache
func (c *Cache) evictLRU() {
	if len(c.entries) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time
	var lowestAccess int64 = -1

	for key, entry := range c.entries {
		if lowestAccess == -1 || entry.AccessCount < lowestAccess {
			oldestKey = key
			lowestAccess = entry.AccessCount
			oldestTime = entry.CreatedAt
		} else if entry.AccessCount == lowestAccess && entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.stats.Evictions++
	}
}

// CacheManager provides a centralized cache management system
type CacheManager struct {
	promptsCache *Cache
	modelsCache  *Cache
	keysCache    *Cache
	config       CacheConfig
}

// CacheConfig holds configuration for the cache manager
type CacheConfig struct {
	MaxSize      int           `json:"max_size"`
	TTL          time.Duration `json:"ttl"`
	EnableStats  bool          `json:"enable_stats"`
	AutoSave     bool          `json:"auto_save"`
	SaveInterval time.Duration `json:"save_interval"`
	FileWatch    bool          `json:"file_watch"`
	Compression  bool          `json:"compression"`
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		MaxSize:      100,
		TTL:          5 * time.Minute,
		EnableStats:  true,
		AutoSave:     true,
		SaveInterval: 1 * time.Minute,
		FileWatch:    true,
		Compression:  false,
	}
}

// NewCacheManager creates a new cache manager with default configuration
func NewCacheManager() *CacheManager {
	config := DefaultCacheConfig()
	return &CacheManager{
		promptsCache: NewCache(config.MaxSize),
		modelsCache:  NewCache(config.MaxSize),
		keysCache:    NewCache(config.MaxSize),
		config:       config,
	}
}

// NewCacheManagerWithConfig creates a new cache manager with custom configuration
func NewCacheManagerWithConfig(config CacheConfig) *CacheManager {
	return &CacheManager{
		promptsCache: NewCache(config.MaxSize),
		modelsCache:  NewCache(config.MaxSize),
		keysCache:    NewCache(config.MaxSize),
		config:       config,
	}
}

// GetPrompts retrieves prompts from cache or loads from file
func (cm *CacheManager) GetPrompts(filePath string) ([]models.Prompt, error) {
	key := fmt.Sprintf("prompts:%s", filePath)

	if cached, hit := cm.promptsCache.Get(key, filePath); hit {
		if prompts, ok := cached.([]models.Prompt); ok {
			return prompts, nil
		}
	}

	// Load from file
	prompts, err := cm.loadPromptsFromFile(filePath)
	if err != nil {
		return nil, errors.NewCacheError("load_prompts", err)
	}

	// Cache the result
	if err := cm.promptsCache.Set(key, prompts, filePath); err != nil {
		return nil, errors.NewCacheError("cache_prompts", err)
	}

	return prompts, nil
}

// GetModels retrieves models from cache or loads from file
func (cm *CacheManager) GetModels(filePath string) ([]models.Model, error) {
	key := fmt.Sprintf("models:%s", filePath)

	if cached, hit := cm.modelsCache.Get(key, filePath); hit {
		if models, ok := cached.([]models.Model); ok {
			return models, nil
		}
	}

	// Load from file
	models, err := cm.loadModelsFromFile(filePath)
	if err != nil {
		return nil, errors.NewCacheError("load_models", err)
	}

	// Cache the result
	if err := cm.modelsCache.Set(key, models, filePath); err != nil {
		return nil, errors.NewCacheError("cache_models", err)
	}

	return models, nil
}

// GetAPIKeys retrieves API keys from cache or loads from file
func (cm *CacheManager) GetAPIKeys(filePath string) ([]models.APIKey, error) {
	key := fmt.Sprintf("apikeys:%s", filePath)

	if cached, hit := cm.keysCache.Get(key, filePath); hit {
		if keys, ok := cached.([]models.APIKey); ok {
			return keys, nil
		}
	}

	// Load from file
	keys, err := cm.loadAPIKeysFromFile(filePath)
	if err != nil {
		return nil, errors.NewCacheError("load_apikeys", err)
	}

	// Cache the result
	if err := cm.keysCache.Set(key, keys, filePath); err != nil {
		return nil, errors.NewCacheError("cache_apikeys", err)
	}

	return keys, nil
}

// InvalidatePrompts invalidates the prompts cache
func (cm *CacheManager) InvalidatePrompts(filePath string) {
	key := fmt.Sprintf("prompts:%s", filePath)
	cm.promptsCache.Invalidate(key)
}

// InvalidateModels invalidates the models cache
func (cm *CacheManager) InvalidateModels(filePath string) {
	key := fmt.Sprintf("models:%s", filePath)
	cm.modelsCache.Invalidate(key)
}

// InvalidateAPIKeys invalidates the API keys cache
func (cm *CacheManager) InvalidateAPIKeys(filePath string) {
	key := fmt.Sprintf("apikeys:%s", filePath)
	cm.keysCache.Invalidate(key)
}

// ClearAll clears all caches
func (cm *CacheManager) ClearAll() {
	cm.promptsCache.Clear()
	cm.modelsCache.Clear()
	cm.keysCache.Clear()
}

// GetStats returns statistics for all caches
func (cm *CacheManager) GetStats() map[string]CacheStats {
	return map[string]CacheStats{
		"prompts": cm.promptsCache.GetStats(),
		"models":  cm.modelsCache.GetStats(),
		"keys":    cm.keysCache.GetStats(),
	}
}

// loadPromptsFromFile loads prompts from a JSON file
func (cm *CacheManager) loadPromptsFromFile(filePath string) ([]models.Prompt, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.NewStorageError("read", filePath, err)
	}

	var prompts []models.Prompt
	if err := json.Unmarshal(data, &prompts); err != nil {
		// Try alternative format with wrapper
		var config struct {
			Prompts []models.Prompt `json:"prompts"`
		}
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, errors.NewStorageError("unmarshal", filePath, err)
		}
		prompts = config.Prompts
	}

	return prompts, nil
}

// loadModelsFromFile loads models from a JSON file
func (cm *CacheManager) loadModelsFromFile(filePath string) ([]models.Model, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.NewStorageError("read", filePath, err)
	}

	var config struct {
		Models []models.Model `json:"models"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.NewStorageError("unmarshal", filePath, err)
	}

	return config.Models, nil
}

// loadAPIKeysFromFile loads API keys from a JSON file
func (cm *CacheManager) loadAPIKeysFromFile(filePath string) ([]models.APIKey, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.NewStorageError("read", filePath, err)
	}

	var config models.APIKeysConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.NewStorageError("unmarshal", filePath, err)
	}

	return config.Keys, nil
}
