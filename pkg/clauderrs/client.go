package clauderrs

// ClientError represents client-related errors.
type ClientError struct {
	*BaseError
}

// NewClientError creates a new client error.
func NewClientError(code ErrorCode, message string, cause error) *ClientError {
	return &ClientError{
		BaseError: NewBaseError(CategoryClient, code, message, cause),
	}
}

// WithSessionID adds session ID metadata to the error.
func (e *ClientError) WithSessionID(sessionID string) *ClientError {
	_ = e.WithMetadata(MetadataKeySessionID, sessionID)

	return e
}
