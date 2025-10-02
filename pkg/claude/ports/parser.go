package ports

import (
	"github.com/conneroisu/claude/pkg/claude/messages"
)

// MessageParser defines what the domain needs from message parsing
// This is a port because the domain needs to convert raw transport messages
// into typed domain messages, but doesn't care HOW that conversion happens
type MessageParser interface {
	// Parse converts a raw message map into a typed domain message
	Parse(raw map[string]any) (messages.Message, error)
}
