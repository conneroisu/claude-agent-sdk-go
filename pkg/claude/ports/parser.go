package ports

import (
	"github.com/conneroisu/claude/pkg/claude/messages"
)

// MessageParser defines what the domain needs from message parsing.
// The domain needs to convert raw transport messages into typed
// domain messages, but doesn't care HOW that conversion happens.
type MessageParser interface {
	// Parse converts a raw JSON map into a typed Message
	Parse(raw map[string]any) (messages.Message, error)
}
