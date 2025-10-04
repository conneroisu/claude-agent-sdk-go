package messages

// StreamEvent represents a streaming event from the Anthropic API.
// Stream events are passed through as raw API events without parsing,
// allowing users to handle them with maximum flexibility.
type StreamEvent struct {
	// UUID uniquely identifies this stream event
	UUID string `json:"uuid"`

	// SessionID identifies the conversation session
	SessionID string `json:"session_id"`

	// Event contains the raw Anthropic API stream event
	Event map[string]any `json:"event"`

	// ParentToolUseID links this event to a tool use if applicable
	ParentToolUseID *string `json:"parent_tool_use_id,omitempty"`
}

func (StreamEvent) message() {}
