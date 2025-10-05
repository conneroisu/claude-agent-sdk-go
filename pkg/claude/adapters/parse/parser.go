// Package parse provides message parsing from raw JSON to typed messages.
package parse

import (
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.MessageParser for JSON message parsing.
type Adapter struct{}

// NewAdapter creates a new message parser adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// Parse converts raw JSON maps to typed message structures.
func (a *Adapter) Parse(data map[string]any) (messages.Message, error) {
	msgType, err := getStringField(data, "type", true)
	if err != nil {
		return nil, err
	}

	switch msgType {
	case "user":
		return a.parseUser(data)
	case "assistant":
		return a.parseAssistant(data)
	case "system":
		return a.parseSystem(data)
	case "result":
		return a.parseResult(data)
	case "stream_event":
		return a.parseStreamEvent(data)
	default:
		return &messages.UnknownMessage{
			Type:    msgType,
			RawData: data,
		}, nil
	}
}

// Compile-time interface verification.
var _ ports.MessageParser = (*Adapter)(nil)
