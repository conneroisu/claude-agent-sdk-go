package clauderrs

import (
	"fmt"
	"maps"
)

// SDKError represents the base interface for all SDK errors.
type SDKError interface {
	error
	// Code returns the error code.
	Code() ErrorCode
	// Category returns the error category.
	Category() ErrorCategory
	// Unwrap returns the underlying error.
	Unwrap() error
	// Metadata returns additional error metadata.
	Metadata() map[string]any
}

// BaseError provides the base implementation for SDK errors.
type BaseError struct {
	code     ErrorCode
	category ErrorCategory
	message  string
	cause    error
	metadata map[string]any
}

// NewBaseError creates a new base error.
func NewBaseError(
	category ErrorCategory,
	code ErrorCode,
	message string,
	cause error,
) *BaseError {
	return &BaseError{
		code:     code,
		category: category,
		message:  message,
		cause:    cause,
		metadata: make(map[string]any),
	}
}

// Error implements the error interface.
func (e *BaseError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.category, e.message, e.cause)
	}

	return fmt.Sprintf("%s: %s", e.category, e.message)
}

// Code returns the error code.
func (e *BaseError) Code() ErrorCode {
	return e.code
}

// Category returns the error category.
func (e *BaseError) Category() ErrorCategory {
	return e.category
}

// Unwrap returns the underlying error.
func (e *BaseError) Unwrap() error {
	return e.cause
}

// Metadata returns the error metadata.
func (e *BaseError) Metadata() map[string]any {
	return e.metadata
}

// WithMetadata adds metadata to the error.
func (e *BaseError) WithMetadata(key string, value any) *BaseError {
	e.metadata[key] = value

	return e
}

// WithMetadataMap adds multiple metadata items to the error.
func (e *BaseError) WithMetadataMap(metadata map[string]any) *BaseError {
	maps.Copy(e.metadata, metadata)

	return e
}
