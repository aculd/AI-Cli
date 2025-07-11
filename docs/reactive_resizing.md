# Reactive Resizing System

## Overview

The AI CLI application features a comprehensive reactive resizing system that provides flawless terminal resize handling with dynamic layout recalculation, debouncing, and responsive UI components.

## Key Features

### 🖥️ **Dynamic Layout Recalculation**
- Real-time layout updates on terminal size changes
- Proportional component sizing based on available space
- Graceful degradation for small terminals
- Optimal layout proportions maintained across all sizes

### ⏱️ **Debounced Resize Events**
- Prevents excessive updates during rapid resize operations
- Configurable debounce delay (default: 100ms)
- Smooth performance even with frequent resize events
- Intelligent event filtering and processing

### 🎨 **Responsive UI Components**
- Components that adapt to terminal size changes
- Collapsible sidebar for small terminals
- Dynamic style adjustments based on available space
- Smooth transitions during resize operations

## Architecture

### 1. Resize Manager (`src/components/common/resize.go`)

The core resize management system provides:

```go
type ResizeManager struct {
    config      ResizeConfig
    subscribers map[string]ResizeSubscriber
    debounce    *time.Timer
    // ... other fields
}
```

**Key Features:**
- **Event Debouncing**: Prevents excessive updates during rapid resizing
- **Subscriber System**: Components can subscribe to resize events
- **Validation**: Ensures terminal dimensions are within acceptable ranges
- **Thread Safety**: Concurrent-safe event handling

### 2. Responsive Layout (`src/components/common/resize.go`)

Dynamic layout calculation system:

```go
type ResponsiveLayout struct {
    width  int
    height int
    config LayoutConfig
}
```

**Layout Components:**
- **Header**: Fixed height with dynamic width
- **Sidebar**: Proportional width (default: 25% of total width)
- **Content**: Remaining space after sidebar allocation
- **Footer**: Fixed height with dynamic width

### 3. Responsive Components

#### Chat View (`src/components/chat/responsive_view.go`)
- Adapts message display to available space
- Dynamic scrolling and selection
- Responsive message formatting
- Graceful handling of overflow

#### Sidebar (`src/components/sidebar/responsive_sidebar.go`)
- Collapsible design for small terminals
- Dynamic chat list rendering
- Responsive navigation controls
- Minimal view for very small terminals

## Configuration

### Resize Configuration

```go
type ResizeConfig struct {
    DebounceDelay time.Duration `json:"debounce_delay"`
    MinWidth      int           `json:"min_width"`
    MinHeight     int           `json:"min_height"`
    MaxWidth      int           `json:"max_width"`
    MaxHeight     int           `json:"max_height"`
    EnableLogging bool          `json:"enable_logging"`
}
```

**Default Values:**
- `DebounceDelay`: 100ms
- `MinWidth`: 40 characters
- `MinHeight`: 20 lines
- `MaxWidth`: 200 characters
- `MaxHeight`: 100 lines
- `EnableLogging`: true

### Layout Configuration

```go
type LayoutConfig struct {
    SidebarRatio     float64 `json:"sidebar_ratio"`
    HeaderHeight     int     `json:"header_height"`
    FooterHeight     int     `json:"footer_height"`
    MinContentWidth  int     `json:"min_content_width"`
    MinContentHeight int     `json:"min_content_height"`
}
```

**Default Values:**
- `SidebarRatio`: 0.25 (25% of total width)
- `HeaderHeight`: 3 lines
- `FooterHeight`: 2 lines
- `MinContentWidth`: 40 characters
- `MinContentHeight`: 10 lines

## Usage Examples

### 1. Basic Resize Manager Setup

```go
// Create resize manager
config := common.DefaultResizeConfig()
resizeManager := common.NewResizeManager(config, logger)

// Subscribe to resize events
resizeManager.Subscribe("my_component", func(event common.ResizeEvent) {
    // Handle resize event
    fmt.Printf("Resized to %dx%d\n", event.Width, event.Height)
})
```

### 2. Responsive Component Integration

```go
// Create responsive chat view
chatView := chat.NewResponsiveChatView()

// Handle resize events
func (cv *ResponsiveChatView) OnResize(width, height int) {
    cv.width = width
    cv.height = height
    cv.layout.UpdateSize(width, height)
    cv.updateStyles()
    cv.adjustScrollOffset()
}
```

### 3. Resize-Aware Program

```go
// Create resize-aware program
program := common.NewResizeAwareProgram(model, config, logger)

// Run with resize handling
program.Run()
```

## Event Flow

### 1. Terminal Resize Detection

```
SIGWINCH Signal → Resize Manager → Debounce Timer → Event Processing
```

### 2. Component Update Flow

```
Resize Event → Layout Recalculation → Component Updates → UI Re-render
```

### 3. Debouncing Process

```
Rapid Resize Events → Timer Reset → Single Event → Component Updates
```

## Responsive Behavior

### Small Terminals (< 60 characters wide)
- Simplified borders and padding
- Collapsed sidebar (minimal view)
- Condensed message display
- Reduced header/footer content

### Medium Terminals (60-100 characters wide)
- Standard borders and padding
- Partial sidebar display
- Normal message formatting
- Standard header/footer

### Large Terminals (> 100 characters wide)
- Enhanced borders and padding
- Full sidebar display
- Rich message formatting
- Extended header/footer information

## Performance Optimizations

### 1. Debouncing
- Prevents excessive updates during rapid resizing
- Configurable delay to balance responsiveness and performance
- Intelligent event filtering

### 2. Layout Caching
- Cached layout calculations for repeated sizes
- Efficient dimension calculations
- Minimal recalculation overhead

### 3. Component Updates
- Selective component updates based on size changes
- Efficient style recalculation
- Minimal re-rendering

## Error Handling

### 1. Invalid Dimensions
- Validation of minimum and maximum dimensions
- Graceful handling of out-of-range sizes
- Fallback to safe default dimensions

### 2. Component Failures
- Panic recovery in resize subscribers
- Graceful degradation when components fail
- Logging of resize-related errors

### 3. Resource Management
- Proper cleanup of timers and subscribers
- Memory leak prevention
- Graceful shutdown procedures

## Monitoring and Statistics

### Resize Statistics

```go
type ResizeStats struct {
    TotalEvents     int64     `json:"total_events"`
    DebouncedEvents int64     `json:"debounced_events"`
    LastResize      time.Time `json:"last_resize"`
    AverageWidth    float64   `json:"average_width"`
    AverageHeight   float64   `json:"average_height"`
    MinWidth        int       `json:"min_width"`
    MaxWidth        int       `json:"max_width"`
    MinHeight       int       `json:"min_height"`
    MaxHeight       int       `json:"max_height"`
}
```

### Monitoring Features
- Event frequency tracking
- Dimension range monitoring
- Performance metrics
- Error rate tracking

## Integration with Existing Systems

### 1. Error Handling Integration
- Domain error system integration
- Graceful error handling during resize
- Error recovery mechanisms

### 2. Caching System Integration
- Cache-aware layout adjustments
- Performance optimization during resize
- Cache statistics integration

### 3. Logging Integration
- Structured logging of resize events
- Performance monitoring
- Debug information for development

## Best Practices

### 1. Component Design
- Always implement `OnResize` method
- Use responsive layout calculations
- Handle edge cases (very small/large terminals)
- Provide graceful degradation

### 2. Performance
- Minimize calculations during resize
- Use efficient style updates
- Implement proper cleanup
- Monitor memory usage

### 3. User Experience
- Provide smooth transitions
- Maintain context during resize
- Clear visual feedback
- Intuitive navigation

## Future Enhancements

### 1. Advanced Layouts
- Multi-column layouts
- Dynamic component positioning
- Adaptive content flow
- Custom layout algorithms

### 2. Enhanced Responsiveness
- Touch-friendly interfaces
- Gesture support
- Accessibility improvements
- Cross-platform optimization

### 3. Performance Improvements
- GPU acceleration
- Advanced caching strategies
- Predictive resizing
- Real-time optimization

## Conclusion

The reactive resizing system provides a robust, performant, and user-friendly solution for terminal resize handling. With its debounced events, dynamic layout recalculation, and responsive components, it ensures a smooth and consistent user experience across all terminal sizes.

The system is designed to be extensible, maintainable, and well-integrated with the existing codebase, providing a solid foundation for future enhancements and improvements. 