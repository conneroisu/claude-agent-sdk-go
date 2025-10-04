// Package parse implements message parsing from JSON-RPC to domain messages.
// It provides adapters that convert raw JSON messages from the CLI into
// type-safe domain message structures.
package parse

import (
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.MessageParser.
// This is an INFRASTRUCTURE adapter - handles low-level message parsing.
// It converts raw JSON messages from the transport into typed domain messages.
type Adapter struct{}

// Verify interface compliance at compile time.
var _ ports.MessageParser = (*Adapter)(nil)

// NewAdapter creates a new message parser adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// Parse implements ports.MessageParser.
// It routes messages to type-specific parsers based on the type field.
func (a *Adapter) Parse(data map[string]any) (messages.Message, error) {
	msgType, ok := data["type"].(string)
	if !ok {
		return nil, fmt.Errorf("message missing type field")
	}

	switch msgType {
	case "user":
		return a.parseUserMessage(data)
	case "assistant":
		return a.parseAssistantMessage(data)
	case "system":
		return a.parseSystemMessage(data)
	case "result":
		return a.parseResultMessage(data)
	case "stream_event":
		return a.parseStreamEvent(data)
	default:
		return nil, fmt.Errorf("unknown message type: %s", msgType)
	}
}
