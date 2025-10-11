package clauderrs

import "errors"

// CreateProcessError is a convenience function to create process errors.
func CreateProcessError(
	code ErrorCode,
	message string,
	cause error,
	exitCode int,
	stderr string,
) *ProcessError {
	return NewProcessError(code, message, cause, exitCode, stderr)
}

// CreateTransportError is a convenience function to create transport errors.
func CreateTransportError(
	code ErrorCode,
	message string,
	cause error,
) *TransportError {
	return NewTransportError(code, message, cause)
}

// CreateValidationError is a convenience function to create validation errors.
func CreateValidationError(
	code ErrorCode,
	message string,
	cause error,
	field string,
	value any,
) *ValidationError {
	return NewValidationError(code, message, cause, field, value)
}

// CreatePermissionError is a convenience function to create permission errors.
func CreatePermissionError(
	code ErrorCode,
	message string,
	cause error,
	resource, action string,
) *PermissionError {
	return NewPermissionError(code, message, cause, resource, action)
}

// WrapError wraps an error with additional context.
func WrapError(
	category ErrorCategory,
	code ErrorCode,
	message string,
	err error,
) SDKError {
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
			return NewProcessError(
				code,
				message,
				err,
				procErr.ExitCode(),
				procErr.Stderr(),
			)
		}

		return NewProcessError(code, message, err, 0, "")
	case CategoryValidation:
		if valErr, ok := err.(*ValidationError); ok {
			return NewValidationError(
				code,
				message,
				err,
				valErr.Field(),
				valErr.Value(),
			)
		}

		return NewValidationError(code, message, err, "", nil)
	case CategoryPermission:
		if permErr, ok := err.(*PermissionError); ok {
			return NewPermissionError(
				code,
				message,
				err,
				permErr.Resource(),
				permErr.Action(),
			)
		}

		return NewPermissionError(code, message, err, "", "")
	case CategoryCallback:
		if cbErr, ok := err.(*CallbackError); ok {
			return NewCallbackError(
				code,
				message,
				err,
				cbErr.Callback(),
				cbErr.Timeout(),
			)
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
		return sdkErr.Category() == CategoryNetwork &&
			sdkErr.Code() == ErrCodeNetworkTimeout
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
