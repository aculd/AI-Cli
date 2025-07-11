// Package cache provides configuration management for the caching system.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheConfigManager manages cache configuration
type CacheConfigManager struct {
	configFile string
	config     CacheConfig
}

// NewCacheConfigManager creates a new cache configuration manager
func NewCacheConfigManager() *CacheConfigManager {
	return &CacheConfigManager{
		configFile: "src/.config/cache_config.json",
		config:     DefaultCacheConfig(),
	}
}

// Load loads the cache configuration from file
func (ccm *CacheConfigManager) Load() error {
	data, err := os.ReadFile(ccm.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config file
			return ccm.Save()
		}
		return fmt.Errorf("failed to read cache config: %w", err)
	}

	if err := json.Unmarshal(data, &ccm.config); err != nil {
		return fmt.Errorf("failed to unmarshal cache config: %w", err)
	}

	return nil
}

// Save saves the cache configuration to file
func (ccm *CacheConfigManager) Save() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(ccm.configFile), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(ccm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache config: %w", err)
	}

	if err := os.WriteFile(ccm.configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache config: %w", err)
	}

	return nil
}

// GetConfig returns the current cache configuration
func (ccm *CacheConfigManager) GetConfig() CacheConfig {
	return ccm.config
}

// UpdateConfig updates the cache configuration
func (ccm *CacheConfigManager) UpdateConfig(config CacheConfig) error {
	ccm.config = config
	return ccm.Save()
}

// SetMaxSize sets the maximum cache size
func (ccm *CacheConfigManager) SetMaxSize(maxSize int) error {
	if maxSize <= 0 {
		return fmt.Errorf("max size must be positive")
	}
	ccm.config.MaxSize = maxSize
	return ccm.Save()
}

// SetTTL sets the time-to-live for cache entries
func (ccm *CacheConfigManager) SetTTL(ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("TTL must be positive")
	}
	ccm.config.TTL = ttl
	return ccm.Save()
}

// EnableStats enables or disables cache statistics
func (ccm *CacheConfigManager) EnableStats(enable bool) error {
	ccm.config.EnableStats = enable
	return ccm.Save()
}

// EnableAutoSave enables or disables automatic cache saving
func (ccm *CacheConfigManager) EnableAutoSave(enable bool) error {
	ccm.config.AutoSave = enable
	return ccm.Save()
}

// SetSaveInterval sets the interval for automatic cache saving
func (ccm *CacheConfigManager) SetSaveInterval(interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("save interval must be positive")
	}
	ccm.config.SaveInterval = interval
	return ccm.Save()
}

// EnableFileWatch enables or disables file modification watching
func (ccm *CacheConfigManager) EnableFileWatch(enable bool) error {
	ccm.config.FileWatch = enable
	return ccm.Save()
}

// EnableCompression enables or disables cache compression
func (ccm *CacheConfigManager) EnableCompression(enable bool) error {
	ccm.config.Compression = enable
	return ccm.Save()
}

// ResetToDefaults resets the configuration to default values
func (ccm *CacheConfigManager) ResetToDefaults() error {
	ccm.config = DefaultCacheConfig()
	return ccm.Save()
}

// ValidateConfig validates the current configuration
func (ccm *CacheConfigManager) ValidateConfig() error {
	if ccm.config.MaxSize <= 0 {
		return fmt.Errorf("max size must be positive")
	}
	if ccm.config.TTL <= 0 {
		return fmt.Errorf("TTL must be positive")
	}
	if ccm.config.SaveInterval <= 0 {
		return fmt.Errorf("save interval must be positive")
	}
	return nil
}

// GetConfigSummary returns a summary of the current configuration
func (ccm *CacheConfigManager) GetConfigSummary() string {
	summary := "Cache Configuration Summary\n"
	summary += "============================\n\n"

	summary += fmt.Sprintf("Max Size: %d entries\n", ccm.config.MaxSize)
	summary += fmt.Sprintf("TTL: %s\n", ccm.config.TTL)
	summary += fmt.Sprintf("Enable Stats: %t\n", ccm.config.EnableStats)
	summary += fmt.Sprintf("Auto Save: %t\n", ccm.config.AutoSave)
	summary += fmt.Sprintf("Save Interval: %s\n", ccm.config.SaveInterval)
	summary += fmt.Sprintf("File Watch: %t\n", ccm.config.FileWatch)
	summary += fmt.Sprintf("Compression: %t\n", ccm.config.Compression)

	return summary
}

// GetRecommendedConfig returns recommended configuration based on system resources
func (ccm *CacheConfigManager) GetRecommendedConfig() CacheConfig {
	// This would typically analyze system resources
	// For now, return a conservative configuration
	return CacheConfig{
		MaxSize:      200,
		TTL:          10 * time.Minute,
		EnableStats:  true,
		AutoSave:     true,
		SaveInterval: 2 * time.Minute,
		FileWatch:    true,
		Compression:  false,
	}
}

// ApplyRecommendedConfig applies the recommended configuration
func (ccm *CacheConfigManager) ApplyRecommendedConfig() error {
	recommended := ccm.GetRecommendedConfig()
	return ccm.UpdateConfig(recommended)
}
