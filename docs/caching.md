# Caching System Documentation

## Overview

The AI CLI application now includes a comprehensive caching system that provides in-memory caching for frequently accessed data, significantly improving performance by reducing disk I/O operations.

## Architecture

The caching system is built with a modular architecture consisting of several key components:

### Core Components

1. **Cache Manager** (`src/services/cache/cache.go`)
   - Thread-safe in-memory cache with LRU eviction
   - File modification detection for cache invalidation
   - Configurable cache size and TTL
   - Statistics tracking (hits, misses, evictions)

2. **Cache Configuration** (`src/services/cache/config.go`)
   - Configurable cache settings
   - Default and recommended configurations
   - Persistent configuration storage

3. **Cache Monitoring** (`src/services/cache/monitor.go`)
   - Health monitoring and performance metrics
   - Cache statistics and reporting
   - Performance analysis and recommendations

4. **Cache Integration** (`src/services/cache/integration.go`)
   - Unified interface for the caching system
   - Global cache management
   - Easy integration with existing code

### Cached Repositories

1. **Cached Prompt Repository** (`src/services/storage/repositories/cached_prompts_repo.go`)
   - Caches prompt templates loaded from JSON files
   - Automatic cache invalidation when files are modified
   - Thread-safe access to prompt data

2. **Cached Model Repository** (`src/services/storage/repositories/cached_models_repo.go`)
   - Caches AI model configurations
   - Supports default model selection
   - Efficient model lookup and management

3. **Cached API Key Repository** (`src/services/storage/repositories/cached_keys_repo.go`)
   - Caches API key configurations
   - Active key management
   - Secure key storage with caching

## Features

### Performance Improvements

- **Reduced Disk I/O**: Frequently accessed data is cached in memory
- **Faster Response Times**: Eliminates repeated file reads for the same data
- **Scalable Architecture**: Configurable cache sizes based on system resources

### Cache Invalidation

- **File Modification Detection**: Automatically invalidates cache when source files change
- **Manual Invalidation**: Support for manual cache clearing when needed
- **LRU Eviction**: Removes least recently used entries when cache is full

### Monitoring and Statistics

- **Real-time Statistics**: Track cache hits, misses, and evictions
- **Performance Metrics**: Monitor cache efficiency and health
- **Health Monitoring**: Detect performance issues and provide recommendations

### Configuration Management

- **Flexible Configuration**: Customizable cache settings
- **Persistent Settings**: Configuration saved to disk
- **Default Values**: Sensible defaults for most use cases

## Usage

### Basic Usage

The caching system is automatically initialized when the application starts:

```go
// Initialize the global cache
if err := cache.InitializeGlobalCache(); err != nil {
    log.Printf("Cache initialization failed: %v", err)
}

// Get cached data
prompts, err := cache.GetCachedPrompts()
models, err := cache.GetCachedModels()
keys, err := cache.GetCachedAPIKeys()
```

### Advanced Usage

```go
// Get cache statistics
stats := cache.GetGlobalCacheStats()
health := cache.GetGlobalCacheHealth()

// Clear cache
cache.ClearGlobalCache()

// Get performance report
report := cache.GetGlobalCacheIntegration().GetPerformanceReport()
```

### Configuration

```go
// Load configuration
configManager := cache.NewCacheConfigManager()
config := configManager.GetConfig()

// Update settings
configManager.SetMaxSize(200)
configManager.SetTTL(10 * time.Minute)
configManager.EnableStats(true)
```

## Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `max_size` | 100 | Maximum number of cache entries |
| `ttl` | 5m | Time-to-live for cache entries |
| `enable_stats` | true | Enable cache statistics tracking |
| `auto_save` | true | Automatically save cache statistics |
| `save_interval` | 1m | Interval for saving cache statistics |
| `file_watch` | true | Enable file modification detection |
| `compression` | false | Enable cache compression |

## Performance Benefits

### Before Caching
- Every data access required a disk read
- Repeated access to the same data was inefficient
- No performance optimization for frequently used data

### After Caching
- First access loads data into cache
- Subsequent accesses are served from memory
- Significant reduction in disk I/O operations
- Improved application responsiveness

## Monitoring and Maintenance

### Cache Statistics

The system provides comprehensive statistics:

- **Hit Rate**: Percentage of cache hits vs total requests
- **Miss Rate**: Percentage of cache misses
- **Eviction Rate**: Rate of cache entries being evicted
- **Cache Size**: Current number of cached entries
- **Performance Metrics**: Load times and efficiency

### Health Monitoring

The cache monitor provides health status:

- **Healthy**: Cache is performing well
- **Warning**: Performance issues detected
- **Error**: Critical problems requiring attention

### Maintenance Tasks

1. **Regular Statistics Review**: Monitor cache performance
2. **Configuration Tuning**: Adjust settings based on usage patterns
3. **Cache Clearing**: Clear cache when needed for troubleshooting

## Integration with Existing Code

The caching system is designed to be minimally invasive to existing code:

### Repository Pattern

Existing repository interfaces are extended with cached implementations:

```go
// Before: Direct file access
prompts, err := loadPromptsFromFile()

// After: Cached access
prompts, err := cache.GetCachedPrompts()
```

### Backward Compatibility

- Existing code continues to work unchanged
- Caching is transparent to the application logic
- Gradual migration to cached repositories

## Error Handling

The caching system includes robust error handling:

- **Graceful Degradation**: Falls back to direct file access if cache fails
- **Error Logging**: Comprehensive error reporting
- **Recovery Mechanisms**: Automatic cache recovery from errors

## Security Considerations

- **API Key Security**: Cached API keys are stored securely
- **File Permissions**: Proper file permissions for configuration files
- **Access Control**: Thread-safe access to cached data

## Future Enhancements

### Planned Features

1. **Distributed Caching**: Support for shared cache across multiple instances
2. **Advanced Eviction Policies**: More sophisticated cache eviction strategies
3. **Cache Persistence**: Persistent cache across application restarts
4. **Compression**: Optional cache compression for memory efficiency

### Performance Optimizations

1. **Background Warming**: Preload frequently accessed data
2. **Predictive Caching**: Cache data based on usage patterns
3. **Memory Optimization**: More efficient memory usage patterns

## Troubleshooting

### Common Issues

1. **High Miss Rate**: Consider increasing cache size or TTL
2. **High Eviction Rate**: Cache size may be too small
3. **Memory Usage**: Monitor cache memory consumption

### Debugging

```go
// Enable debug logging
configManager.EnableStats(true)

// Get detailed cache information
info := cache.GetGlobalCacheIntegration().GetCacheInfo()
fmt.Printf("Cache Info: %+v\n", info)
```

## Conclusion

The caching system provides significant performance improvements while maintaining the existing application architecture. It's designed to be transparent, configurable, and maintainable, making it an essential component for the AI CLI application's performance optimization. 