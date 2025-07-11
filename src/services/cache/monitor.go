// Package cache provides monitoring and statistics for the caching system.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheHealth represents the health status of the cache system
type CacheHealth struct {
	Status      string                `json:"status"`
	LastCheck   time.Time             `json:"last_check"`
	Statistics  map[string]CacheStats `json:"statistics"`
	Performance CachePerformance      `json:"performance"`
	Errors      []string              `json:"errors,omitempty"`
}

// CachePerformance represents performance metrics for the cache
type CachePerformance struct {
	HitRate         float64       `json:"hit_rate"`
	MissRate        float64       `json:"miss_rate"`
	EvictionRate    float64       `json:"eviction_rate"`
	AverageLoadTime time.Duration `json:"average_load_time"`
}

// CacheMonitor provides monitoring and statistics for the cache system
type CacheMonitor struct {
	cacheManager *CacheManager
	statsFile    string
	lastStats    map[string]CacheStats
}

// NewCacheMonitor creates a new cache monitor
func NewCacheMonitor(cacheManager *CacheManager) *CacheMonitor {
	return &CacheMonitor{
		cacheManager: cacheManager,
		statsFile:    "src/.config/cache_stats.json",
		lastStats:    make(map[string]CacheStats),
	}
}

// GetHealth returns the current health status of the cache system
func (cm *CacheMonitor) GetHealth() CacheHealth {
	stats := cm.cacheManager.GetStats()

	health := CacheHealth{
		Status:      "healthy",
		LastCheck:   time.Now(),
		Statistics:  stats,
		Performance: cm.calculatePerformance(stats),
	}

	// Check for potential issues
	if cm.hasHighEvictionRate(stats) {
		health.Status = "warning"
		health.Errors = append(health.Errors, "High eviction rate detected")
	}

	if cm.hasLowHitRate(stats) {
		health.Status = "warning"
		health.Errors = append(health.Errors, "Low hit rate detected")
	}

	return health
}

// SaveStats saves cache statistics to a file
func (cm *CacheMonitor) SaveStats() error {
	health := cm.GetHealth()

	data, err := json.MarshalIndent(health, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache stats: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cm.statsFile), 0755); err != nil {
		return fmt.Errorf("failed to create stats directory: %w", err)
	}

	if err := os.WriteFile(cm.statsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache stats: %w", err)
	}

	return nil
}

// LoadStats loads cache statistics from file
func (cm *CacheMonitor) LoadStats() (*CacheHealth, error) {
	data, err := os.ReadFile(cm.statsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache stats: %w", err)
	}

	var health CacheHealth
	if err := json.Unmarshal(data, &health); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache stats: %w", err)
	}

	return &health, nil
}

// GetPerformanceReport returns a detailed performance report
func (cm *CacheMonitor) GetPerformanceReport() string {
	health := cm.GetHealth()
	stats := health.Statistics

	report := "Cache Performance Report\n"
	report += "========================\n\n"

	report += fmt.Sprintf("Overall Status: %s\n", health.Status)
	report += fmt.Sprintf("Last Check: %s\n\n", health.LastCheck.Format(time.RFC3339))

	for cacheType, stat := range stats {
		report += fmt.Sprintf("%s Cache:\n", cacheType)
		report += fmt.Sprintf("  Size: %d/%d entries\n", stat.Size, stat.MaxSize)
		report += fmt.Sprintf("  Hits: %d\n", stat.Hits)
		report += fmt.Sprintf("  Misses: %d\n", stat.Misses)
		report += fmt.Sprintf("  Evictions: %d\n", stat.Evictions)

		total := stat.Hits + stat.Misses
		if total > 0 {
			hitRate := float64(stat.Hits) / float64(total) * 100
			report += fmt.Sprintf("  Hit Rate: %.2f%%\n", hitRate)
		}

		report += fmt.Sprintf("  Last Updated: %s\n\n", stat.LastUpdated.Format(time.RFC3339))
	}

	if len(health.Errors) > 0 {
		report += "Warnings:\n"
		for _, err := range health.Errors {
			report += fmt.Sprintf("  - %s\n", err)
		}
	}

	return report
}

// calculatePerformance calculates performance metrics from cache statistics
func (cm *CacheMonitor) calculatePerformance(stats map[string]CacheStats) CachePerformance {
	var totalHits, totalMisses, totalEvictions int64
	var totalLoadTime time.Duration

	for _, stat := range stats {
		totalHits += stat.Hits
		totalMisses += stat.Misses
		totalEvictions += stat.Evictions
	}

	totalRequests := totalHits + totalMisses
	var hitRate, missRate, evictionRate float64

	if totalRequests > 0 {
		hitRate = float64(totalHits) / float64(totalRequests) * 100
		missRate = float64(totalMisses) / float64(totalRequests) * 100
	}

	if totalHits > 0 {
		evictionRate = float64(totalEvictions) / float64(totalHits) * 100
	}

	return CachePerformance{
		HitRate:         hitRate,
		MissRate:        missRate,
		EvictionRate:    evictionRate,
		AverageLoadTime: totalLoadTime, // Would be calculated from actual load times
	}
}

// hasHighEvictionRate checks if the eviction rate is concerning
func (cm *CacheMonitor) hasHighEvictionRate(stats map[string]CacheStats) bool {
	for _, stat := range stats {
		if stat.Hits > 0 {
			evictionRate := float64(stat.Evictions) / float64(stat.Hits) * 100
			if evictionRate > 10 { // More than 10% eviction rate
				return true
			}
		}
	}
	return false
}

// hasLowHitRate checks if the hit rate is concerning
func (cm *CacheMonitor) hasLowHitRate(stats map[string]CacheStats) bool {
	for _, stat := range stats {
		total := stat.Hits + stat.Misses
		if total > 0 {
			hitRate := float64(stat.Hits) / float64(total) * 100
			if hitRate < 50 { // Less than 50% hit rate
				return true
			}
		}
	}
	return false
}

// ResetStats resets all cache statistics
func (cm *CacheMonitor) ResetStats() {
	cm.cacheManager.ClearAll()
	cm.lastStats = make(map[string]CacheStats)
}

// GetCacheSize returns the total size of all caches
func (cm *CacheMonitor) GetCacheSize() int {
	stats := cm.cacheManager.GetStats()
	totalSize := 0

	for _, stat := range stats {
		totalSize += stat.Size
	}

	return totalSize
}

// GetCacheEfficiency returns the overall cache efficiency as a percentage
func (cm *CacheMonitor) GetCacheEfficiency() float64 {
	stats := cm.cacheManager.GetStats()
	var totalHits, totalMisses int64

	for _, stat := range stats {
		totalHits += stat.Hits
		totalMisses += stat.Misses
	}

	total := totalHits + totalMisses
	if total == 0 {
		return 0
	}

	return float64(totalHits) / float64(total) * 100
}
