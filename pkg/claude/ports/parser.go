package ports

import "github.com/conneroisu/claude/pkg/claude/messages"

// MessageParser converts raw transport messages to domain types.
// This port defines what the domain needs for message parsing,
// without coupling to specific JSON unmarshaling implementations.
//
// Error Handling: Parse returns typed errors for different failure modes:
// - Unknown message types return an error
// - Malformed JSON structure returns an error
// - Missing required fields returns an error
//
// Type Discrimination: The parser must correctly identify message variants
// and return the appropriate concrete type implementing messages.Message.
type MessageParser interface {
	// Parse converts a raw message map to a typed Message.
	// Returns an error if the message type is unknown or the structure
	// is invalid.
	Parse(raw map[string]any) (messages.Message, error)
}
