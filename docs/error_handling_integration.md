# Error Handling Integration with Caching System

## Overview

The AI CLI application now features a comprehensive domain error handling system that seamlessly integrates with the caching infrastructure, providing robust error management, user-friendly messaging, and animated UI feedback.

## Architecture Integration

### 1. Domain Error System (`src/errors/domain.go`)

The domain error system provides:

- **Structured Error Types**: Categorized errors (Validation, NotFound, Network, Cache, Storage, etc.)
- **Error Builder Pattern**: Fluent API for creating rich error objects
- **User-Friendly Messages**: Separate technical and user-facing error messages
- **Retry Logic**: Built-in retry strategies with exponential backoff
- **Bubble Tea Integration**: Animated error modals and UI components

### 2. Cache System Integration

The caching system has been enhanced with domain error handling:

- **Cache-Specific Errors**: `CacheError` type for cache-related issues
- **Storage Errors**: `StorageError` type for file I/O operations
- **Configuration Errors**: `ConfigurationError` type for settings issues
- **Graceful Degradation**: Fallback to direct file access when cache fails

## Error Types and Usage

### Cache-Related Errors

```go
// Cache operation failures
err := errors.NewCacheError("load_prompts", originalError)

// Cache misses (not really errors, but tracked)
err := errors.NewCacheMissError("prompts:config.json")

// Storage operation failures
err := errors.NewStorageError("read", "config.json", originalError)
```

### Validation Errors

```go
// Input validation
err := errors.NewValidationError("prompt_name", "cannot be empty")

// Resource not found
err := errors.NewNotFoundError("prompt", "my_prompt")
```

### Network and External Service Errors

```go
// Network failures
err := errors.NewNetworkError("api_request", originalError)

// AI service failures
err := errors.NewAIServiceError("OpenAI", "chat_completion", originalError)
```

## Integration Points

### 1. Cache Manager Integration

The cache manager now uses domain errors for all operations:

```go
// Before: Generic error handling
func (cm *CacheManager) GetPrompts(filePath string) ([]models.Prompt, error) {
    prompts, err := cm.loadPromptsFromFile(filePath)
    if err != nil {
        return nil, err // Generic error
    }
    return prompts, nil
}

// After: Domain-specific error handling
func (cm *CacheManager) GetPrompts(filePath string) ([]models.Prompt, error) {
    prompts, err := cm.loadPromptsFromFile(filePath)
    if err != nil {
        return nil, errors.NewCacheError("load_prompts", err)
    }
    return prompts, nil
}
```

### 2. Repository Integration

Cached repositories provide domain-specific error handling:

```go
// CachedPromptRepository example
func (r *CachedPromptRepository) GetByID(name string) (*models.Prompt, error) {
    prompts, err := r.GetAll()
    if err != nil {
        return nil, err // Propagate cache error
    }

    for _, prompt := range prompts {
        if prompt.Name == name {
            return &prompt, nil
        }
    }

    return nil, errors.NewNotFoundError("prompt", name)
}
```

### 3. UI Integration

The error system integrates with Bubble Tea for animated error display:

```go
// Create animated error modal
errorModal := errors.NewAnimatedErrorModal(domainError)
p := tea.NewProgram(errorModal, tea.WithAltScreen())
_, err := p.Run()

// Or use simple error modal
simpleModal := errors.NewErrorModal(domainError)
p := tea.NewProgram(simpleModal, tea.WithAltScreen())
_, err := p.Run()
```

## Error Flow Examples

### 1. Cache Miss Flow

```
User Request → Cache Check → Cache Miss → Load from File → Cache Result → Return Data
                                    ↓
                              NewCacheMissError (informational)
```

### 2. Cache Failure Flow

```
User Request → Cache Check → Cache Error → Fallback to File → Return Data
                                    ↓
                              NewCacheError (retryable)
```

### 3. Storage Failure Flow

```
User Request → Cache Check → File Load Error → Domain Error → User Notification
                                    ↓
                              NewStorageError (retryable)
```

## Retry Strategy Integration

The error system includes built-in retry logic:

```go
// Retry configuration
config := errors.RetryConfig{
    MaxAttempts:  3,
    InitialDelay: 1 * time.Second,
    MaxDelay:     10 * time.Second,
    Backoff:      errors.ExponentialBackoff,
}

// Retry operation
err := errors.Retry(context.Background(), config, func() error {
    return cacheManager.GetPrompts(filePath)
})
```

## User Experience Enhancements

### 1. Animated Error Display

The system provides animated error modals inspired by the API key validation flow:

```go
// Animated error modal with cycling messages
modal := errors.NewAnimatedErrorModal(domainError)
modal.Complete(false, "Operation failed")

// Displays:
// ┌─ Error Analysis ─┐
// │                  │
// │ ⠋ Analyzing...   │
// │                  │
// │ Analyzing...     │
// └──────────────────┘
```

### 2. Error Details Toggle

Users can toggle between user-friendly and technical error messages:

```go
// Press Tab to show technical details
modal := errors.NewErrorModal(domainError)
// Shows user message by default
// Press Tab to show: Type, Code, Message, Details
```

### 3. Error Recovery

The system provides clear recovery instructions:

```go
// Cache errors suggest reloading
"Cache error. Data will be reloaded from source."

// Storage errors suggest checking permissions
"Storage error. Check file permissions and disk space."

// Network errors suggest retrying
"Network failure. Retry or check your connection."
```

## Error Logging and Monitoring

### 1. Structured Logging

All domain errors are logged with structured data:

```go
logger.Error("Domain error occurred",
    "type", domainErr.Type,
    "code", domainErr.Code,
    "message", domainErr.Message,
    "user_message", domainErr.UserMsg,
    "details", domainErr.Details,
    "retryable", domainErr.Retryable,
    "timestamp", domainErr.Timestamp,
    "cause", domainErr.Cause,
)
```

### 2. Error Statistics

The system tracks error patterns for monitoring:

- Error type distribution
- Retry success rates
- Cache hit/miss ratios
- Storage operation failures

## Configuration Integration

Error handling integrates with the cache configuration system:

```go
// Error handling respects cache configuration
if config.EnableStats {
    // Log detailed error statistics
}

if config.AutoSave {
    // Save error statistics to disk
}
```

## Benefits of Integration

### 1. Improved User Experience

- **Clear Error Messages**: Users understand what went wrong
- **Recovery Guidance**: Clear instructions on how to fix issues
- **Animated Feedback**: Engaging UI that shows progress and status

### 2. Better Developer Experience

- **Structured Errors**: Consistent error handling across the application
- **Rich Context**: Errors include all relevant information for debugging
- **Retry Logic**: Built-in retry mechanisms for transient failures

### 3. Enhanced Monitoring

- **Error Tracking**: Comprehensive error logging and statistics
- **Performance Insights**: Cache performance metrics with error context
- **Health Monitoring**: System health status with error indicators

### 4. Robust Error Recovery

- **Graceful Degradation**: System continues working even when cache fails
- **Automatic Retries**: Transient errors are automatically retried
- **Fallback Mechanisms**: Multiple recovery strategies for different error types

## Future Enhancements

### 1. Error Analytics

- Error pattern analysis
- Predictive error prevention
- Performance impact assessment

### 2. Advanced Recovery

- Automatic error recovery strategies
- User-guided error resolution
- Error prevention suggestions

### 3. Enhanced UI

- More sophisticated error animations
- Interactive error resolution flows
- Error history and learning

## Conclusion

The integration of domain error handling with the caching system provides a robust, user-friendly, and maintainable error management solution. The system ensures that errors are handled consistently, users receive clear feedback, and developers have the tools needed for effective debugging and monitoring. 