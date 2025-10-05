package parse

import "fmt"

// ParseError represents a message parsing error.
type ParseError struct {
	MessageType string
	Field       string
	Cause       error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf(
		"parse %s message field %s: %v",
		e.MessageType,
		e.Field,
		e.Cause,
	)
}

func (e *ParseError) Unwrap() error {
	return e.Cause
}
