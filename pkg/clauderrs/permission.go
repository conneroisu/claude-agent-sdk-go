package clauderrs

// PermissionError represents permission-related errors.
type PermissionError struct {
	*BaseError
	resource string
	action   string
}

// NewPermissionError creates a new permission error.
func NewPermissionError(
	code ErrorCode,
	message string,
	cause error,
	resource, action string,
) *PermissionError {
	err := &PermissionError{
		BaseError: NewBaseError(CategoryPermission, code, message, cause),
		resource:  resource,
		action:    action,
	}

	// Add permission-specific metadata
	_ = err.WithMetadata("resource", resource)
	_ = err.WithMetadata("action", action)

	return err
}

// Resource returns the permission resource.
func (e *PermissionError) Resource() string {
	return e.resource
}

// Action returns the permission action.
func (e *PermissionError) Action() string {
	return e.action
}

// CallbackError represents callback-related errors.
type CallbackError struct {
	*BaseError
	callback string
	timeout  bool
}

// NewCallbackError creates a new callback error.
func NewCallbackError(
	code ErrorCode,
	message string,
	cause error,
	callback string,
	timeout bool,
) *CallbackError {
	err := &CallbackError{
		BaseError: NewBaseError(CategoryCallback, code, message, cause),
		callback:  callback,
		timeout:   timeout,
	}

	// Add callback-specific metadata
	_ = err.WithMetadata("callback", callback)
	_ = err.WithMetadata("timeout", timeout)

	return err
}

// Callback returns the callback name.
func (e *CallbackError) Callback() string {
	return e.callback
}

// Timeout returns whether the callback timed out.
func (e *CallbackError) Timeout() bool {
	return e.timeout
}

// WithSessionID adds session ID metadata to the error.
func (e *CallbackError) WithSessionID(sessionID string) *CallbackError {
	_ = e.WithMetadata(MetadataKeySessionID, sessionID)

	return e
}

// WithTimeout adds timeout metadata to the error.
func (e *CallbackError) WithTimeout(timeout bool) *CallbackError {
	e.timeout = timeout
	_ = e.WithMetadata("timeout", timeout)

	return e
}
