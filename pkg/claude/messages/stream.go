package messages

// StreamEvent represents a real-time event from the streaming API.
// Stream events are emitted during active conversations to provide
// incremental updates, token usage, and state changes.
type StreamEvent struct {
	// Event contains the raw event data from the API
	// The structure varies by event type and is not parsed into typed structs
	Event map[string]any

	// EventType indicates the type of streaming event
	// Common types: "message_start", "content_block_delta", "message_stop"
	EventType string
}

func (*StreamEvent) message() {}
