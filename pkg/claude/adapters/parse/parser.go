// Package parse provides message parsing adapters for the Claude SDK.
// It handles conversion of raw map[string]any data into strongly-typed
// message structures.
package parse

import (
	"errors"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

const (
	blockTypeText     = "text"
	blockTypeThinking = "thinking"
	blockTypeContent  = "content"
)

// Adapter implements ports.MessageParser
// This is an INFRASTRUCTURE adapter - handles low-level message
// parsing
type Adapter struct {
	parsers map[string]parserFunc
}

// parserFunc is a function that parses message data
type parserFunc func(map[string]any) (messages.Message, error)

// Verify interface compliance at compile time
var _ ports.MessageParser = (*Adapter)(nil)

// NewAdapter creates a new message parser adapter
func NewAdapter() *Adapter {
	a := &Adapter{}
	a.parsers = map[string]parserFunc{
		"user":         parseUserMessage,
		"assistant":    parseAssistantMessage,
		"system":       parseSystemMessage,
		"result":       parseResultMessage,
		"stream_event": parseStreamEvent,
	}

	return a
}

// Parse implements ports.MessageParser
// It acts as a dispatcher that routes raw message data to specialized
// parsers based on the message type. This strategy allows each message
// type to have its own parsing logic while maintaining a clean
// separation of concerns.
func (a *Adapter) Parse(
	data map[string]any,
) (messages.Message, error) {
	msgType, ok := data["type"].(string)
	if !ok {
		return nil, errors.New("message missing type field")
	}

	parser, ok := a.parsers[msgType]
	if !ok {
		return nil, fmt.Errorf("unknown message type: %s", msgType)
	}

	return parser(data)
}
