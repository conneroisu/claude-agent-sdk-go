package jsonrpc

import "fmt"

// ProtocolError represents a JSON-RPC protocol error.
type ProtocolError struct {
	Category  string
	RequestID string
	Cause     error
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf(
		"protocol %s error [%s]: %v",
		e.Category,
		e.RequestID,
		e.Cause,
	)
}

func (e *ProtocolError) Unwrap() error {
	return e.Cause
}
