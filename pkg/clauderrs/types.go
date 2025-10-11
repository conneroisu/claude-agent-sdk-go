// Package clauderrs provides a comprehensive error handling framework for the
// Claude Agent SDK. This package defines error types, categories, and utilities
// to support consistent error handling across the SDK while maintaining
// backward compatibility.
package clauderrs

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

// Metadata keys.
const (
	MetadataKeySessionID = "session_id"
)
