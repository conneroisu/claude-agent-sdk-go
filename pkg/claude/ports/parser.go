package ports

import (
	"github.com/conneroisu/claude/pkg/claude/messages"
)

// MessageParser defines what the domain needs from message parsing.
//
// This port interface converts raw transport messages (map[string]any)
// into typed domain messages that the SDK can work with.
//
// The domain layer depends on this interface to transform untyped JSON
// data into strongly-typed Go structs representing Claude messages.
//
// This is a port because the domain needs message parsing but doesn't
// care about the implementation details of HOW parsing happens.
//
// Example implementation: JSONParser that unmarshals and type-checks messages.
type MessageParser interface {
	// Parse converts a raw message into a typed domain message.
	//
	// The raw parameter contains the JSON message as a map.
	// Returns a typed Message interface or an error if parsing fails.
	//
	// The returned Message can be one of several types:
	//   - UserMessage
	//   - AssistantMessage
	//   - SystemMessage
	//   - ResultMessageSuccess
	//   - ResultMessageError
	//   - StreamEvent
	//
	// Example:
	//
	//	raw := map[string]any{
	//	    "subtype": "success",
	//	    "result": "Task completed",
	//	}
	//	msg, err := parser.Parse(raw)
	//	if err != nil {
	//	    return fmt.Errorf("parse failed: %w", err)
	//	}
	//	if success, ok := msg.(messages.ResultMessageSuccess); ok {
	//	    fmt.Printf("Result: %s\n", success.Result)
	//	}
	Parse(raw map[string]any) (messages.Message, error)
}
