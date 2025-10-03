// Package parse implements the message parser adapter.
//
// This adapter implements the MessageParser port, converting raw
// JSON messages from the transport into typed domain messages.
package parse

import (
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.MessageParser.
// This is an INFRASTRUCTURE adapter - handles low-level message
// parsing.
type Adapter struct{}

// Verify interface compliance at compile time.
var _ ports.MessageParser = (*Adapter)(nil)

// NewAdapter creates a new message parser adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// Parse implements ports.MessageParser.
func (a *Adapter) Parse(
	data map[string]any,
) (messages.Message, error) {
	msgType, ok := data["type"].(string)
	if !ok {
		return nil, fmt.Errorf("message missing type field") //nolint:revive // unnecessary-format: message clarity
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
