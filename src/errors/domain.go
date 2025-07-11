// =====================================================================================
// 🧠 Domain Error Handling Flow – Unified, Animated, and User-Centric
// =====================================================================================
// This error handling flow draws inspiration from the API key validation modal:
//  - Live testing with feedback
//  - Clear animated UI states
//  - Robust categorization
//  - Human-readable user messaging

package errors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// =====================================================================================
// 1. 🔖 Error Type System
// =====================================================================================

type ErrorType string

const (
	ValidationError      ErrorType = "validation"
	NotFoundError        ErrorType = "not_found"
	ConflictError        ErrorType = "conflict"
	NetworkError         ErrorType = "network"
	AuthenticationError  ErrorType = "authentication"
	InternalError        ErrorType = "internal"
	ExternalServiceError ErrorType = "external_service"
	CacheError           ErrorType = "cache"
	StorageError         ErrorType = "storage"
	ConfigurationError   ErrorType = "configuration"
)

// Structured error
// Inspired by dynamic testing flows: carries all the data needed for diagnosis & UI rendering

type DomainError struct {
	Type      ErrorType              `json:"type"`
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	UserMsg   string                 `json:"user_message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Cause     error                  `json:"-"`
	Timestamp time.Time              `json:"timestamp"`
	Retryable bool                   `json:"retryable"`
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Type, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Type, e.Code, e.Message)
}

func (e *DomainError) Unwrap() error { return e.Cause }

func (e *DomainError) Is(target error) bool {
	t, ok := target.(*DomainError)
	return ok && e.Type == t.Type && e.Code == t.Code
}

// =====================================================================================
// 2. 🏗️ Error Builder (Factory + Defaults)
// =====================================================================================

type ErrorBuilder struct {
	err *DomainError
}

func NewError(t ErrorType, code string) *ErrorBuilder {
	return &ErrorBuilder{err: &DomainError{
		Type: t, Code: code, Timestamp: time.Now(), Details: map[string]interface{}{},
	}}
}

func (b *ErrorBuilder) Message(msg string) *ErrorBuilder     { b.err.Message = msg; return b }
func (b *ErrorBuilder) UserMessage(msg string) *ErrorBuilder { b.err.UserMsg = msg; return b }
func (b *ErrorBuilder) Cause(cause error) *ErrorBuilder      { b.err.Cause = cause; return b }
func (b *ErrorBuilder) Detail(k string, v any) *ErrorBuilder { b.err.Details[k] = v; return b }
func (b *ErrorBuilder) Retryable(r bool) *ErrorBuilder       { b.err.Retryable = r; return b }

func (b *ErrorBuilder) Build() *DomainError {
	if b.err.UserMsg == "" {
		b.err.UserMsg = defaultUserMsg(b.err.Type)
	}
	return b.err
}

func defaultUserMsg(t ErrorType) string {
	switch t {
	case ValidationError:
		return "Please check your input and try again."
	case NotFoundError:
		return "The requested item could not be found."
	case NetworkError:
		return "Network issue detected. Try again."
	case AuthenticationError:
		return "Login failed. Check your credentials."
	case CacheError:
		return "Cache operation failed. Data will be reloaded."
	case StorageError:
		return "Storage operation failed. Check your configuration."
	case ConfigurationError:
		return "Configuration error. Check your settings."
	default:
		return "Something went wrong."
	}
}

// =====================================================================================
// 3. 🧰 Predefined Common Errors
// =====================================================================================

func NewValidationError(field, msg string) *DomainError {
	return NewError(ValidationError, "VALIDATION_FAILED").
		Message(fmt.Sprintf("Invalid field '%s': %s", field, msg)).
		UserMessage(fmt.Sprintf("Please check your %s.", field)).
		Detail("field", field).
		Detail("reason", msg).
		Build()
}

func NewNotFoundError(resource, id string) *DomainError {
	return NewError(NotFoundError, "RESOURCE_NOT_FOUND").
		Message(fmt.Sprintf("%s with ID '%s' not found", resource, id)).
		UserMessage(fmt.Sprintf("Can't find that %s.", resource)).
		Detail("resource", resource).
		Detail("id", id).
		Build()
}

func NewNetworkError(operation string, cause error) *DomainError {
	return NewError(NetworkError, "NETWORK_FAIL").
		Message(fmt.Sprintf("Operation '%s' failed", operation)).
		UserMessage("Network failure. Retry or check your connection.").
		Cause(cause).
		Detail("operation", operation).
		Retryable(true).
		Build()
}

func NewAIServiceError(provider, op string, cause error) *DomainError {
	return NewError(ExternalServiceError, "AI_SERVICE_FAIL").
		Message(fmt.Sprintf("%s failed during %s", provider, op)).
		UserMessage("AI service unavailable. Retry shortly.").
		Cause(cause).
		Detail("provider", provider).
		Detail("operation", op).
		Retryable(true).
		Build()
}

// Cache-specific errors
func NewCacheError(operation string, cause error) *DomainError {
	return NewError(CacheError, "CACHE_FAIL").
		Message(fmt.Sprintf("Cache operation '%s' failed", operation)).
		UserMessage("Cache error. Data will be reloaded from source.").
		Cause(cause).
		Detail("operation", operation).
		Retryable(true).
		Build()
}

func NewCacheMissError(key string) *DomainError {
	return NewError(CacheError, "CACHE_MISS").
		Message(fmt.Sprintf("Cache miss for key '%s'", key)).
		UserMessage("Loading data from source...").
		Detail("key", key).
		Retryable(false).
		Build()
}

// Storage-specific errors
func NewStorageError(operation, path string, cause error) *DomainError {
	return NewError(StorageError, "STORAGE_FAIL").
		Message(fmt.Sprintf("Storage operation '%s' failed for '%s'", operation, path)).
		UserMessage("Storage error. Check file permissions and disk space.").
		Cause(cause).
		Detail("operation", operation).
		Detail("path", path).
		Retryable(true).
		Build()
}

// Configuration errors
func NewConfigurationError(setting, reason string) *DomainError {
	return NewError(ConfigurationError, "CONFIG_FAIL").
		Message(fmt.Sprintf("Configuration error for '%s': %s", setting, reason)).
		UserMessage("Configuration error. Check your settings.").
		Detail("setting", setting).
		Detail("reason", reason).
		Retryable(false).
		Build()
}

// =====================================================================================
// 4. 🔁 Retry Strategy (Linear / Exponential Backoff)
// =====================================================================================

type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Backoff      BackoffStrategy
}

type BackoffStrategy string

const (
	LinearBackoff      BackoffStrategy = "linear"
	ExponentialBackoff BackoffStrategy = "exponential"
)

type RetryableOperation func() error

func Retry(ctx context.Context, cfg RetryConfig, op RetryableOperation) error {
	var last error
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := op(); err != nil {
			last = err
			var dErr *DomainError
			if errors.As(err, &dErr) && !dErr.Retryable {
				return err
			}
			if attempt == cfg.MaxAttempts {
				break
			}
			delay := cfg.calcDelay(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				continue
			}
		} else {
			return nil
		}
	}
	return last
}

func (cfg RetryConfig) calcDelay(n int) time.Duration {
	switch cfg.Backoff {
	case LinearBackoff:
		return min(cfg.InitialDelay*time.Duration(n), cfg.MaxDelay)
	case ExponentialBackoff:
		return min(cfg.InitialDelay*time.Duration(1<<uint(n-1)), cfg.MaxDelay)
	default:
		return cfg.InitialDelay
	}
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// =====================================================================================
// 5. 🎯 Error Handler & Logger Integration
// =====================================================================================

type ErrorHandler struct {
	logger *slog.Logger
}

func NewErrorHandler(logger *slog.Logger) *ErrorHandler {
	return &ErrorHandler{logger: logger}
}

func (h *ErrorHandler) Handle(err error) {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		h.logger.Error("Domain error occurred",
			"type", domainErr.Type,
			"code", domainErr.Code,
			"message", domainErr.Message,
			"user_message", domainErr.UserMsg,
			"details", domainErr.Details,
			"retryable", domainErr.Retryable,
			"timestamp", domainErr.Timestamp,
			"cause", domainErr.Cause,
		)
	} else {
		h.logger.Error("Unexpected error occurred",
			"error", err,
		)
	}
}

// =====================================================================================
// 6. 🔔 Bubble Tea Integration
// =====================================================================================

// ErrorMsg represents an error message in the Bubble Tea model
type ErrorMsg struct {
	Error *DomainError
}

// ErrorModal represents a modal for displaying errors
type ErrorModal struct {
	Error       *DomainError
	Title       string
	Content     string
	Width       int
	Height      int
	ShowDetails bool
	Quitting    bool
}

func NewErrorModal(err *DomainError) *ErrorModal {
	return &ErrorModal{
		Error:   err,
		Title:   "Error",
		Content: err.UserMsg,
		Width:   60,
		Height:  10,
	}
}

func (m *ErrorModal) Init() tea.Cmd {
	return nil
}

func (m *ErrorModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			m.Quitting = true
			return m, tea.Quit
		case "tab":
			m.ShowDetails = !m.ShowDetails
			return m, nil
		}
	}
	return m, nil
}

func (m *ErrorModal) View() string {
	if m.Quitting {
		return ""
	}

	title := m.Title
	if m.Error != nil {
		title = fmt.Sprintf("%s: %s", m.Title, m.Error.Type)
	}

	content := m.Content
	if m.ShowDetails && m.Error != nil {
		content = fmt.Sprintf("%s\n\nDetails:\nType: %s\nCode: %s\nMessage: %s",
			m.Content, m.Error.Type, m.Error.Code, m.Error.Message)
	}

	// Simple modal rendering - can be enhanced with lipgloss styling
	return fmt.Sprintf(`
┌─ %s ─┐
│      │
│ %s │
│      │
│ Press Enter to continue │
│ Press Tab for details   │
└──────┘`, title, content)
}

// =====================================================================================
// 7. 🎨 Animated Error Display (Inspired by DynamicNoticeModal)
// =====================================================================================

type AnimatedErrorModal struct {
	Error       *DomainError
	Title       string
	Messages    []string
	Current     int
	Interval    time.Duration
	Testing     bool
	Done        bool
	Success     bool
	ResultMsg   string
	ResultEmoji string
	OnComplete  func(bool)
}

func NewAnimatedErrorModal(err *DomainError) *AnimatedErrorModal {
	messages := []string{
		"Analyzing error...",
		"Checking system status...",
		"Preparing recovery options...",
	}

	return &AnimatedErrorModal{
		Error:    err,
		Title:    "Error Analysis",
		Messages: messages,
		Current:  0,
		Interval: 1 * time.Second,
		Testing:  true,
	}
}

func (m *AnimatedErrorModal) Init() tea.Cmd {
	return m.tick()
}

func (m *AnimatedErrorModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.Done {
			switch msg.String() {
			case "enter", "esc":
				if m.OnComplete != nil {
					m.OnComplete(m.Success)
				}
				return m, tea.Quit
			}
		}
	case tickMsg:
		if m.Testing {
			m.Current = (m.Current + 1) % len(m.Messages)
			return m, m.tick()
		}
	}

	return m, nil
}

func (m *AnimatedErrorModal) View() string {
	if m.Testing {
		return fmt.Sprintf(`
┌─ %s ─┐
│      │
│ %s %s │
│      │
│ Analyzing... │
└──────┘`, m.Title, m.Messages[m.Current], getSpinner(m.Current))
	}

	return fmt.Sprintf(`
┌─ %s ─┐
│      │
│ %s %s │
│      │
│ %s │
│      │
│ Press Enter to continue │
└──────┘`, m.Title, m.ResultEmoji, m.ResultMsg, m.Error.UserMsg)
}

func (m *AnimatedErrorModal) tick() tea.Cmd {
	return tea.Tick(m.Interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *AnimatedErrorModal) Complete(success bool, message string) {
	m.Testing = false
	m.Done = true
	m.Success = success
	m.ResultMsg = message
	if success {
		m.ResultEmoji = "✅"
	} else {
		m.ResultEmoji = "❌"
	}
}

type tickMsg time.Time

func getSpinner(index int) string {
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return spinners[index%len(spinners)]
}
