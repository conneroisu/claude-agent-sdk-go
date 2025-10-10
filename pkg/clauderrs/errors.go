// Package clauderrs provides a comprehensive error handling framework for the Claude Agent SDK.
// This package defines error types, categories, and utilities to support consistent
// error handling across the SDK while maintaining backward compatibility.
package clauderrs

import (
	"errors"
	"fmt"
	"maps"
	"time"
)

// ErrorCategory represents different categories of errors that can occur
// in the Claude Agent SDK.
type ErrorCategory string

const (
	// CategoryClient represents client-side errors.
	CategoryClient ErrorCategory = "client"
	// CategoryAPI represents API-related errors.
	CategoryAPI ErrorCategory = "api"
	// CategoryNetwork represents network-related errors.
	CategoryNetwork ErrorCategory = "network"
	// CategoryProtocol represents protocol-level errors.
	CategoryProtocol ErrorCategory = "protocol"
	// CategoryTransport represents transport-level errors.
	CategoryTransport ErrorCategory = "transport"
	// CategoryProcess represents process-related errors.
	CategoryProcess ErrorCategory = "process"
	// CategoryValidation represents validation errors.
	CategoryValidation ErrorCategory = "validation"
	// CategoryPermission represents permission-related errors.
	CategoryPermission ErrorCategory = "permission"
	// CategoryCallback represents callback-related errors.
	CategoryCallback ErrorCategory = "callback"
)

// ErrorCode represents specific error codes within each category.
type ErrorCode string

// Client error codes.
const (
	ErrCodeClientClosed  ErrorCode = "client_closed"
	ErrCodeNoActiveQuery ErrorCode = "no_active_query"
	ErrCodeInvalidState  ErrorCode = "invalid_state"
	ErrCodeMissingAPIKey ErrorCode = "missing_api_key"
	ErrCodeInvalidConfig ErrorCode = "invalid_config"
)

// API error codes.
const (
	ErrCodeAPIUnauthorized ErrorCode = "api_unauthorized"
	ErrCodeAPIForbidden    ErrorCode = "api_forbidden"
	ErrCodeAPIRateLimit    ErrorCode = "api_rate_limit"
	ErrCodeAPIServerError  ErrorCode = "api_server_error"
	ErrCodeAPIBadRequest   ErrorCode = "api_bad_request"
	ErrCodeAPINotFound     ErrorCode = "api_not_found"
)

// Network error codes.
const (
	ErrCodeNetworkTimeout   ErrorCode = "network_timeout"
	ErrCodeConnectionFailed ErrorCode = "connection_failed"
	ErrCodeConnectionClosed ErrorCode = "connection_closed"
	ErrCodeDNSError         ErrorCode = "dns_error"
)

// Protocol error codes.
const (
	ErrCodeInvalidMessage     ErrorCode = "invalid_message"
	ErrCodeMessageParseFailed ErrorCode = "message_parse_failed"
	ErrCodeUnknownMessageType ErrorCode = "unknown_message_type"
	ErrCodeProtocolError      ErrorCode = "protocol_error"
)

// Transport error codes.
const (
	ErrCodeIOError       ErrorCode = "io_error"
	ErrCodeReadFailed    ErrorCode = "read_failed"
	ErrCodeWriteFailed   ErrorCode = "write_failed"
	ErrCodeTransportInit ErrorCode = "transport_init"
)

// Process error codes.
const (
	ErrCodeProcessNotFound    ErrorCode = "process_not_found"
	ErrCodeProcessSpawnFailed ErrorCode = "process_spawn_failed"
	ErrCodeProcessCrashed     ErrorCode = "process_crashed"
	ErrCodeProcessExited      ErrorCode = "process_exited"
)

// Validation error codes.
const (
	ErrCodeMissingField   ErrorCode = "missing_field"
	ErrCodeInvalidType    ErrorCode = "invalid_type"
	ErrCodeRangeViolation ErrorCode = "range_violation"
	ErrCodeInvalidFormat  ErrorCode = "invalid_format"
)

// Permission error codes.
const (
	ErrCodeToolDenied      ErrorCode = "tool_denied"
	ErrCodeDirectoryDenied ErrorCode = "directory_denied"
	ErrCodeResourceDenied  ErrorCode = "resource_denied"
)

// Callback error codes.
const (
	ErrCodeCallbackFailed  ErrorCode = "callback_failed"
	ErrCodeCallbackTimeout ErrorCode = "callback_timeout"
	ErrCodeHookFailed      ErrorCode = "hook_failed"
	ErrCodeHookTimeout     ErrorCode = "hook_timeout"
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
	e.WithMetadata("session_id", sessionID)

	return e
}

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
	e.WithMetadata("request_id", requestID)

	return e
}

// WithResponse adds response metadata to the error.
func (e *APIError) WithResponse(response string) *APIError {
	e.WithMetadata("response", response)

	return e
}

// WithRetryAfter adds retry after metadata to the error.
func (e *APIError) WithRetryAfter(retryAfter time.Duration) *APIError {
	e.WithMetadata("retry_after", retryAfter.Seconds())

	return e
}

// NetworkError represents network-related errors.
type NetworkError struct {
	*BaseError
}

// NewNetworkError creates a new network error.
func NewNetworkError(code ErrorCode, message string, cause error) *NetworkError {
	return &NetworkError{
		BaseError: NewBaseError(CategoryNetwork, code, message, cause),
	}
}

// WithHost adds host metadata to the error.
func (e *NetworkError) WithHost(host string) *NetworkError {
	e.WithMetadata("host", host)
	return e
}

// ProtocolError represents protocol-related errors.
type ProtocolError struct {
	*BaseError
	messageType string
}

// NewProtocolError creates a new protocol error.
func NewProtocolError(code ErrorCode, message string, cause error) *ProtocolError {
	return &ProtocolError{
		BaseError:   NewBaseError(CategoryProtocol, code, message, cause),
		messageType: "",
	}
}

// WithMessageType adds message type metadata to the error.
func (e *ProtocolError) WithMessageType(messageType string) *ProtocolError {
	e.messageType = messageType
	e.WithMetadata("message_type", messageType)
	return e
}

// MessageType returns the message type.
func (e *ProtocolError) MessageType() string {
	return e.messageType
}

// WithMessageID adds message ID metadata to the error.
func (e *ProtocolError) WithMessageID(messageID string) *ProtocolError {
	e.WithMetadata("message_id", messageID)
	return e
}

// WithSessionID adds session ID metadata to the error.
func (e *ProtocolError) WithSessionID(sessionID string) *ProtocolError {
	e.WithMetadata("session_id", sessionID)
	return e
}

// WithRequestID adds request ID metadata to the error.
func (e *ProtocolError) WithRequestID(requestID string) *ProtocolError {
	e.WithMetadata("request_id", requestID)
	return e
}

// TransportError represents transport-related errors.
type TransportError struct {
	*BaseError
}

// NewTransportError creates a new transport error.
func NewTransportError(code ErrorCode, message string, cause error) *TransportError {
	return &TransportError{
		BaseError: NewBaseError(CategoryTransport, code, message, cause),
	}
}

// ProcessError represents process-related errors.
type ProcessError struct {
	*BaseError
	exitCode int
	stderr   string
}

// NewProcessError creates a new process error.
func NewProcessError(code ErrorCode, message string, cause error, exitCode int, stderr string) *ProcessError {
	err := &ProcessError{
		BaseError: NewBaseError(CategoryProcess, code, message, cause),
		exitCode:  exitCode,
		stderr:    stderr,
	}

	// Add process-specific metadata
	err.WithMetadata("exit_code", exitCode)
	err.WithMetadata("stderr", stderr)

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
	e.WithMetadata("command", command)
	return e
}

// WithSessionID adds session ID metadata to the error.
func (e *ProcessError) WithSessionID(sessionID string) *ProcessError {
	e.WithMetadata("session_id", sessionID)
	return e
}

// ValidationError represents validation-related errors.
type ValidationError struct {
	*BaseError
	field string
	value any
}

// NewValidationError creates a new validation error.
func NewValidationError(code ErrorCode, message string, cause error, field string, value any) *ValidationError {
	err := &ValidationError{
		BaseError: NewBaseError(CategoryValidation, code, message, cause),
		field:     field,
		value:     value,
	}

	// Add validation-specific metadata
	err.WithMetadata("field", field)
	err.WithMetadata("value", value)

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

// PermissionError represents permission-related errors.
type PermissionError struct {
	*BaseError
	resource string
	action   string
}

// NewPermissionError creates a new permission error.
func NewPermissionError(code ErrorCode, message string, cause error, resource, action string) *PermissionError {
	err := &PermissionError{
		BaseError: NewBaseError(CategoryPermission, code, message, cause),
		resource:  resource,
		action:    action,
	}

	// Add permission-specific metadata
	err.WithMetadata("resource", resource)
	err.WithMetadata("action", action)

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
func NewCallbackError(code ErrorCode, message string, cause error, callback string, timeout bool) *CallbackError {
	err := &CallbackError{
		BaseError: NewBaseError(CategoryCallback, code, message, cause),
		callback:  callback,
		timeout:   timeout,
	}

	// Add callback-specific metadata
	err.WithMetadata("callback", callback)
	err.WithMetadata("timeout", timeout)

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
	e.WithMetadata("session_id", sessionID)
	return e
}

// WithTimeout adds timeout metadata to the error.
func (e *CallbackError) WithTimeout(timeout bool) *CallbackError {
	e.timeout = timeout
	e.WithMetadata("timeout", timeout)
	return e
}

// CreateProcessError is a convenience function to create process errors.
func CreateProcessError(code ErrorCode, message string, cause error, exitCode int, stderr string) *ProcessError {
	return NewProcessError(code, message, cause, exitCode, stderr)
}

// CreateTransportError is a convenience function to create transport errors.
func CreateTransportError(code ErrorCode, message string, cause error) *TransportError {
	return NewTransportError(code, message, cause)
}

// CreateValidationError is a convenience function to create validation errors.
func CreateValidationError(code ErrorCode, message string, cause error, field string, value any) *ValidationError {
	return NewValidationError(code, message, cause, field, value)
}

// CreatePermissionError is a convenience function to create permission errors.
func CreatePermissionError(code ErrorCode, message string, cause error, resource, action string) *PermissionError {
	return NewPermissionError(code, message, cause, resource, action)
}

// WrapError wraps an error with additional context.
func WrapError(category ErrorCategory, code ErrorCode, message string, err error) SDKError {
	switch category {
	case CategoryClient:
		return NewClientError(code, message, err)
	case CategoryAPI:
		return NewAPIError(code, message, err)
	case CategoryNetwork:
		return NewNetworkError(code, message, err)
	case CategoryProtocol:
		return NewProtocolError(code, message, err)
	case CategoryTransport:
		return NewTransportError(code, message, err)
	case CategoryProcess:
		if procErr, ok := err.(*ProcessError); ok {
			return NewProcessError(code, message, err, procErr.ExitCode(), procErr.Stderr())
		}
		return NewProcessError(code, message, err, 0, "")
	case CategoryValidation:
		if valErr, ok := err.(*ValidationError); ok {
			return NewValidationError(code, message, err, valErr.Field(), valErr.Value())
		}
		return NewValidationError(code, message, err, "", nil)
	case CategoryPermission:
		if permErr, ok := err.(*PermissionError); ok {
			return NewPermissionError(code, message, err, permErr.Resource(), permErr.Action())
		}
		return NewPermissionError(code, message, err, "", "")
	case CategoryCallback:
		if cbErr, ok := err.(*CallbackError); ok {
			return NewCallbackError(code, message, err, cbErr.Callback(), cbErr.Timeout())
		}
		return NewCallbackError(code, message, err, "", false)
	default:
		return NewBaseError(category, code, message, err)
	}
}

// AsSDKError extracts an SDKError from the error chain.
func AsSDKError(err error) (SDKError, bool) {
	var sdkErr SDKError
	if errors.As(err, &sdkErr) {
		return sdkErr, true
	}
	return nil, false
}

// IsClientError checks if the error is a client error.
func IsClientError(err error) bool {
	if sdkErr, ok := AsSDKError(err); ok {
		return sdkErr.Category() == CategoryClient
	}
	return false
}

// IsAPIError checks if the error is an API error.
func IsAPIError(err error) bool {
	if sdkErr, ok := AsSDKError(err); ok {
		return sdkErr.Category() == CategoryAPI
	}
	return false
}

// IsNetworkError checks if the error is a network error.
func IsNetworkError(err error) bool {
	if sdkErr, ok := AsSDKError(err); ok {
		return sdkErr.Category() == CategoryNetwork
	}
	return false
}

// IsNetworkTimeout checks if the error is a network timeout error.
func IsNetworkTimeout(err error) bool {
	if sdkErr, ok := AsSDKError(err); ok {
		return sdkErr.Category() == CategoryNetwork && sdkErr.Code() == ErrCodeNetworkTimeout
	}
	return false
}

// IsProtocolError checks if the error is a protocol error.
func IsProtocolError(err error) bool {
	if sdkErr, ok := AsSDKError(err); ok {
		return sdkErr.Category() == CategoryProtocol
	}
	return false
}

// IsTransportError checks if the error is a transport error.
func IsTransportError(err error) bool {
	if sdkErr, ok := AsSDKError(err); ok {
		return sdkErr.Category() == CategoryTransport
	}
	return false
}

// IsProcessError checks if the error is a process error.
func IsProcessError(err error) bool {
	if sdkErr, ok := AsSDKError(err); ok {
		return sdkErr.Category() == CategoryProcess
	}
	return false
}

// IsValidationError checks if the error is a validation error.
func IsValidationError(err error) bool {
	if sdkErr, ok := AsSDKError(err); ok {
		return sdkErr.Category() == CategoryValidation
	}
	return false
}

// IsPermissionError checks if the error is a permission error.
func IsPermissionError(err error) bool {
	if sdkErr, ok := AsSDKError(err); ok {
		return sdkErr.Category() == CategoryPermission
	}
	return false
}

// IsCallbackError checks if the error is a callback error.
func IsCallbackError(err error) bool {
	if sdkErr, ok := AsSDKError(err); ok {
		return sdkErr.Category() == CategoryCallback
	}
	return false
}

// Deprecated types for backward compatibility.

// AbortError represents an aborted operation.
// Deprecated: Use NewCallbackError or appropriate error type instead.
type AbortError = CallbackError

// NewAbortError creates a new abort error.
// Deprecated: Use NewCallbackError instead.
func NewAbortError(message string, cause error) *AbortError {
	return NewCallbackError(ErrCodeCallbackFailed, message, cause, "", false)
}

// IsAbortError checks if error is an abort error.
// Deprecated: Use IsCallbackError instead.
func IsAbortError(err error) bool {
	return IsCallbackError(err)
}
