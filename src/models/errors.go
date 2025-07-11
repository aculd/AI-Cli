package models

import "fmt"

// NotFoundError represents an error when a requested resource is not found
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

// ValidationError represents an error when data validation fails
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// StorageError represents an error when storage operations fail
type StorageError struct {
	Message string
	Err     error
}

func (e *StorageError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *StorageError) Unwrap() error {
	return e.Err
}
