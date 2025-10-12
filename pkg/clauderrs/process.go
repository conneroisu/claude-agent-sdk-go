package clauderrs

// ProcessError represents process-related errors.
type ProcessError struct {
	*BaseError
	exitCode int
	stderr   string
}

// NewProcessError creates a new process error.
func NewProcessError(
	code ErrorCode,
	message string,
	cause error,
	exitCode int,
	stderr string,
) *ProcessError {
	err := &ProcessError{
		BaseError: NewBaseError(CategoryProcess, code, message, cause),
		exitCode:  exitCode,
		stderr:    stderr,
	}

	// Add process-specific metadata
	_ = err.WithMetadata("exit_code", exitCode)
	_ = err.WithMetadata("stderr", stderr)

	return err
}

// ExitCode returns the process exit code.
func (e *ProcessError) ExitCode() int {
	return e.exitCode
}

// Stderr returns the process stderr output.
func (e *ProcessError) Stderr() string {
	return e.stderr
}

// WithCommand adds command metadata to the error.
func (e *ProcessError) WithCommand(command string) *ProcessError {
	_ = e.WithMetadata("command", command)

	return e
}

// WithSessionID adds session ID metadata to the error.
func (e *ProcessError) WithSessionID(sessionID string) *ProcessError {
	_ = e.WithMetadata(MetadataKeySessionID, sessionID)

	return e
}

// ValidationError represents validation-related errors.
type ValidationError struct {
	*BaseError
	field string
	value any
}

// NewValidationError creates a new validation error.
func NewValidationError(
	code ErrorCode,
	message string,
	cause error,
	field string,
	value any,
) *ValidationError {
	err := &ValidationError{
		BaseError: NewBaseError(CategoryValidation, code, message, cause),
		field:     field,
		value:     value,
	}

	// Add validation-specific metadata
	_ = err.WithMetadata("field", field)
	_ = err.WithMetadata("value", value)

	return err
}

// Field returns the validation field name.
func (e *ValidationError) Field() string {
	return e.field
}

// Value returns the validation value.
func (e *ValidationError) Value() any {
	return e.value
}
