// Message parser port definition.
package ports

import (
	"github.com/conneroisu/claude/pkg/claude/messages"
)

// MessageParser defines what the domain needs from message parsing.
//
// This is a port because the domain needs to convert raw transport
// messages into typed domain messages, but doesn't care HOW that
// conversion happens (implementation detail).
type MessageParser interface {
	// Parse converts a raw JSON message to a typed domain message.
	//
	// Returns one of:
	//   - messages.UserMessage
	//   - messages.AssistantMessage
	//   - messages.SystemMessage
	//   - messages.ResultMessageSuccess
	//   - messages.ResultMessageError
	//   - messages.StreamEvent
	//
	// Returns error if the message format is invalid or unknown.
	Parse(raw map[string]any) (messages.Message, error)
}
