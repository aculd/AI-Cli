// Package cache provides integration utilities for the caching system.
package cache

import (
	"fmt"
	"sync"
)

// CacheIntegration provides a unified interface for the caching system
type CacheIntegration struct {
	cacheManager  *CacheManager
	configManager *CacheConfigManager
	monitor       *CacheMonitor
	promptsRepo   interface{} // Will be set after initialization
	modelsRepo    interface{} // Will be set after initialization
	keysRepo      interface{} // Will be set after initialization
	mutex         sync.RWMutex
	initialized   bool
}

// NewCacheIntegration creates a new cache integration instance
func NewCacheIntegration() *CacheIntegration {
	return &CacheIntegration{
		cacheManager:  NewCacheManager(),
		configManager: NewCacheConfigManager(),
		monitor:       nil, // Will be initialized after cacheManager
		promptsRepo:   nil,
		modelsRepo:    nil,
		keysRepo:      nil,
		initialized:   false,
	}
}

// Initialize initializes the cache integration system
func (ci *CacheIntegration) Initialize() error {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()

	if ci.initialized {
		return nil
	}

	// Load configuration
	if err := ci.configManager.Load(); err != nil {
		return fmt.Errorf("failed to load cache config: %w", err)
	}

	// Create cache manager with loaded config
	config := ci.configManager.GetConfig()
	ci.cacheManager = NewCacheManagerWithConfig(config)

	// Initialize monitor
	ci.monitor = NewCacheMonitor(ci.cacheManager)

	ci.initialized = true
	return nil
}

// GetCacheManager returns the cache manager
func (ci *CacheIntegration) GetCacheManager() *CacheManager {
	ci.ensureInitialized()
	return ci.cacheManager
}

// GetConfigManager returns the configuration manager
func (ci *CacheIntegration) GetConfigManager() *CacheConfigManager {
	ci.ensureInitialized()
	return ci.configManager
}

// GetMonitor returns the cache monitor
func (ci *CacheIntegration) GetMonitor() *CacheMonitor {
	ci.ensureInitialized()
	return ci.monitor
}

// GetHealth returns the current health status
func (ci *CacheIntegration) GetHealth() CacheHealth {
	ci.ensureInitialized()
	return ci.monitor.GetHealth()
}

// GetStats returns cache statistics
func (ci *CacheIntegration) GetStats() map[string]CacheStats {
	ci.ensureInitialized()
	return ci.cacheManager.GetStats()
}

// SaveStats saves cache statistics to file
func (ci *CacheIntegration) SaveStats() error {
	ci.ensureInitialized()
	return ci.monitor.SaveStats()
}

// ClearAll clears all caches
func (ci *CacheIntegration) ClearAll() {
	ci.ensureInitialized()
	ci.cacheManager.ClearAll()
}

// ReloadConfig reloads the cache configuration
func (ci *CacheIntegration) ReloadConfig() error {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()

	if err := ci.configManager.Load(); err != nil {
		return fmt.Errorf("failed to reload cache config: %w", err)
	}

	// Recreate cache manager with new config
	config := ci.configManager.GetConfig()
	ci.cacheManager = NewCacheManagerWithConfig(config)

	// Update monitor
	ci.monitor = NewCacheMonitor(ci.cacheManager)

	return nil
}

// GetPerformanceReport returns a detailed performance report
func (ci *CacheIntegration) GetPerformanceReport() string {
	ci.ensureInitialized()
	return ci.monitor.GetPerformanceReport()
}

// GetCacheInfo returns information about the cache system
func (ci *CacheIntegration) GetCacheInfo() map[string]interface{} {
	ci.ensureInitialized()

	stats := ci.GetStats()
	health := ci.GetHealth()
	config := ci.configManager.GetConfig()

	info := map[string]interface{}{
		"initialized":   ci.initialized,
		"health_status": health.Status,
		"cache_size":    ci.monitor.GetCacheSize(),
		"efficiency":    ci.monitor.GetCacheEfficiency(),
		"statistics":    stats,
		"configuration": config,
		"last_check":    health.LastCheck,
		"performance":   health.Performance,
	}

	if len(health.Errors) > 0 {
		info["errors"] = health.Errors
	}

	return info
}

// ensureInitialized ensures the cache integration is initialized
func (ci *CacheIntegration) ensureInitialized() {
	if !ci.initialized {
		if err := ci.Initialize(); err != nil {
			// Log error but don't panic
			fmt.Printf("Cache integration initialization failed: %v\n", err)
		}
	}
}

// Global cache integration instance
var globalCacheIntegration *CacheIntegration
var globalCacheMutex sync.Mutex

// GetGlobalCacheIntegration returns the global cache integration instance
func GetGlobalCacheIntegration() *CacheIntegration {
	globalCacheMutex.Lock()
	defer globalCacheMutex.Unlock()

	if globalCacheIntegration == nil {
		globalCacheIntegration = NewCacheIntegration()
		if err := globalCacheIntegration.Initialize(); err != nil {
			fmt.Printf("Failed to initialize global cache integration: %v\n", err)
		}
	}

	return globalCacheIntegration
}

// InitializeGlobalCache initializes the global cache integration
func InitializeGlobalCache() error {
	integration := GetGlobalCacheIntegration()
	return integration.Initialize()
}

// ClearGlobalCache clears all global caches
func ClearGlobalCache() {
	integration := GetGlobalCacheIntegration()
	integration.ClearAll()
}

// GetGlobalCacheStats returns global cache statistics
func GetGlobalCacheStats() map[string]CacheStats {
	integration := GetGlobalCacheIntegration()
	return integration.GetStats()
}

// GetGlobalCacheHealth returns global cache health status
func GetGlobalCacheHealth() CacheHealth {
	integration := GetGlobalCacheIntegration()
	return integration.GetHealth()
}
