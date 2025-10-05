package cli

import "fmt"

// CLIError represents an error from the CLI adapter.
// It captures the stage where the error occurred, a descriptive message,
// and optionally wraps an underlying error cause.
type CLIError struct {
	Stage   string // Operation stage where error occurred (connect, read)
	Message string // Human-readable error description
	Cause   error  // Underlying error that caused this error, if any
}

// Error implements the error interface for CLIError.
// It formats the error with stage information and optional cause details.
func (e *CLIError) Error() string {
	// Include cause details if available
	if e.Cause != nil {
		return fmt.Sprintf("CLI %s: %s: %v", e.Stage, e.Message, e.Cause)
	}

	return fmt.Sprintf("CLI %s: %s", e.Stage, e.Message)
}

// Unwrap returns the underlying cause error for error chain inspection.
// This enables using errors.Is and errors.As with CLIError.
func (e *CLIError) Unwrap() error {
	return e.Cause
}
