package clauderrs

// NetworkError represents network-related errors.
type NetworkError struct {
	*BaseError
}

// NewNetworkError creates a new network error.
func NewNetworkError(
	code ErrorCode,
	message string,
	cause error,
) *NetworkError {
	return &NetworkError{
		BaseError: NewBaseError(CategoryNetwork, code, message, cause),
	}
}

// WithHost adds host metadata to the error.
func (e *NetworkError) WithHost(host string) *NetworkError {
	_ = e.WithMetadata("host", host)

	return e
}

// ProtocolError represents protocol-related errors.
type ProtocolError struct {
	*BaseError
	messageType string
}

// NewProtocolError creates a new protocol error.
func NewProtocolError(
	code ErrorCode,
	message string,
	cause error,
) *ProtocolError {
	return &ProtocolError{
		BaseError:   NewBaseError(CategoryProtocol, code, message, cause),
		messageType: "",
	}
}

// WithMessageType adds message type metadata to the error.
func (e *ProtocolError) WithMessageType(messageType string) *ProtocolError {
	e.messageType = messageType
	_ = e.WithMetadata("message_type", messageType)

	return e
}

// MessageType returns the message type.
func (e *ProtocolError) MessageType() string {
	return e.messageType
}

// WithMessageID adds message ID metadata to the error.
func (e *ProtocolError) WithMessageID(messageID string) *ProtocolError {
	_ = e.WithMetadata("message_id", messageID)

	return e
}

// WithSessionID adds session ID metadata to the error.
func (e *ProtocolError) WithSessionID(sessionID string) *ProtocolError {
	_ = e.WithMetadata(MetadataKeySessionID, sessionID)

	return e
}

// WithRequestID adds request ID metadata to the error.
func (e *ProtocolError) WithRequestID(requestID string) *ProtocolError {
	_ = e.WithMetadata("request_id", requestID)

	return e
}

// TransportError represents transport-related errors.
type TransportError struct {
	*BaseError
}

// NewTransportError creates a new transport error.
func NewTransportError(
	code ErrorCode,
	message string,
	cause error,
) *TransportError {
	return &TransportError{
		BaseError: NewBaseError(CategoryTransport, code, message, cause),
	}
}
