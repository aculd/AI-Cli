package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AppError represents application-level errors
// Params: Op (operation), Err (underlying error), Message (optional user message)
type AppError struct {
	Op      string
	Err     error
	Message string
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s (%v)", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// PromptError wraps prompt-related errors
// Params: Op (operation), Err (underlying error)
type PromptError struct {
	Op  string
	Err error
}

func (e *PromptError) Error() string {
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// ModelError wraps model-related errors
// Params: Op (operation), Err (underlying error)
type ModelError struct {
	Op  string
	Err error
}

func (e *ModelError) Error() string {
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// ErrorLog represents a file-based error log
// Params: LogFile (path to log file)
type ErrorLog struct {
	LogFile string
}

// NewErrorLog creates a new error logger
// Returns: pointer to ErrorLog
func NewErrorLog() *ErrorLog {
	return &ErrorLog{
		LogFile: filepath.Join(utilPath(), "error.log"),
	}
}

// LogError logs an error with context to file and optionally prints to console
// Params: err (error), context (string), printToConsole (bool)
func (el *ErrorLog) LogError(err error, context string, printToConsole bool) {
	if err == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s: %v\n", timestamp, context, err)

	f, ferr := os.OpenFile(el.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if ferr == nil {
		defer f.Close()
		f.WriteString(logEntry)
	}

	if printToConsole {
		switch e := err.(type) {
		case *AppError:
			fmt.Printf("\033[31mError: %s\033[0m\n", e.Error())
		case *ModelError:
			fmt.Printf("\033[31mModel error: %s\033[0m\n", e.Error())
		case *PromptError:
			fmt.Printf("\033[31mPrompt error: %s\033[0m\n", e.Error())
		default:
			fmt.Printf("\033[31mError during %s: %v\033[0m\n", context, err)
		}
	}
}

// handleError logs and prints any error using ErrorLog
// Params: err (error), context (string)
func handleError(err error, context string) {
	if err != nil {
		errorLog.LogError(err, context, true)
	}
}

// errorLog is the global error logger instance
var errorLog = NewErrorLog()

var ErrMenuBack = errors.New("menu back")
