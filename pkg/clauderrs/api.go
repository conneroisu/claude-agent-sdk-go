package clauderrs

import "time"

// APIError represents API-related errors.
type APIError struct {
	*BaseError
}

// NewAPIError creates a new API error.
func NewAPIError(code ErrorCode, message string, cause error) *APIError {
	return &APIError{
		BaseError: NewBaseError(CategoryAPI, code, message, cause),
	}
}

// WithRequestID adds request ID metadata to the error.
func (e *APIError) WithRequestID(requestID string) *APIError {
	_ = e.WithMetadata("request_id", requestID)

	return e
}

// WithResponse adds response metadata to the error.
func (e *APIError) WithResponse(response string) *APIError {
	_ = e.WithMetadata("response", response)

	return e
}

// WithRetryAfter adds retry after metadata to the error.
func (e *APIError) WithRetryAfter(retryAfter time.Duration) *APIError {
	_ = e.WithMetadata("retry_after", retryAfter.Seconds())

	return e
}
