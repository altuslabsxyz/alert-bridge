package errors

import (
	"errors"
	"fmt"
)

// Error categories for classification and handling
type ErrorCategory string

const (
	CategoryValidation ErrorCategory = "validation"
	CategoryNotFound   ErrorCategory = "not_found"
	CategoryConflict   ErrorCategory = "conflict"
	CategoryTransient  ErrorCategory = "transient"
	CategoryPermanent  ErrorCategory = "permanent"
	CategoryInternal   ErrorCategory = "internal"
)

// DomainError represents a typed error with context
type DomainError struct {
	Category ErrorCategory
	Message  string
	Cause    error
	Fields   map[string]interface{} // Additional context
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Category, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Category, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Cause
}

func (e *DomainError) Is(target error) bool {
	t, ok := target.(*DomainError)
	if !ok {
		return false
	}
	return e.Category == t.Category && e.Message == t.Message
}

func (e *DomainError) IsCategory(cat ErrorCategory) bool {
	return e.Category == cat
}

func (e *DomainError) IsRetryable() bool {
	return e.Category == CategoryTransient
}

func (e *DomainError) WithField(key string, value interface{}) *DomainError {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	e.Fields[key] = value
	return e
}

// Constructor functions

// NewValidationError creates a validation error for invalid input
func NewValidationError(message string) *DomainError {
	return &DomainError{
		Category: CategoryValidation,
		Message:  message,
	}
}

// NewNotFoundError creates a not found error for missing resources
func NewNotFoundError(resource string) *DomainError {
	return &DomainError{
		Category: CategoryNotFound,
		Message:  fmt.Sprintf("%s not found", resource),
	}
}

// NewConflictError creates a conflict error for duplicate or concurrent modifications
func NewConflictError(message string) *DomainError {
	return &DomainError{
		Category: CategoryConflict,
		Message:  message,
	}
}

// NewTransientError creates a transient error that can be retried
func NewTransientError(message string, cause error) *DomainError {
	return &DomainError{
		Category: CategoryTransient,
		Message:  message,
		Cause:    cause,
	}
}

// NewPermanentError creates a permanent error that should not be retried
func NewPermanentError(message string, cause error) *DomainError {
	return &DomainError{
		Category: CategoryPermanent,
		Message:  message,
		Cause:    cause,
	}
}

// NewInternalError creates an internal error for unexpected failures
func NewInternalError(message string, cause error) *DomainError {
	return &DomainError{
		Category: CategoryInternal,
		Message:  message,
		Cause:    cause,
	}
}

// Wrap wraps an existing error with a category and message
func Wrap(err error, category ErrorCategory, message string) *DomainError {
	if err == nil {
		return nil
	}
	return &DomainError{
		Category: category,
		Message:  message,
		Cause:    err,
	}
}

// Helper functions for error type checking

// IsValidationError checks if the error is a validation error
func IsValidationError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Category == CategoryValidation
	}
	return false
}

// IsNotFoundError checks if the error is a not found error
func IsNotFoundError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Category == CategoryNotFound
	}
	return false
}

// IsConflictError checks if the error is a conflict error
func IsConflictError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Category == CategoryConflict
	}
	return false
}

// IsTransientError checks if the error is transient and can be retried
func IsTransientError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Category == CategoryTransient
	}
	return false
}

// IsInternalError checks if the error is an internal error
func IsInternalError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Category == CategoryInternal
	}
	return false
}
