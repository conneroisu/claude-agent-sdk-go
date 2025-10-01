package ports

import "github.com/conneroisu/claude/pkg/claude/messages"

// MessageParser defines what the domain needs from message parsing
type MessageParser interface {
	Parse(raw map[string]any) (messages.Message, error)
}
