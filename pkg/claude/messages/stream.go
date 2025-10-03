package messages

// StreamEvent represents a raw streaming event from the
// Anthropic API, proxied through Claude CLI.
//
// The Event field contains the raw API event as a flexible map
// since stream events vary widely in structure. Users can parse
// specific event types as needed.
type StreamEvent struct {
	// UUID uniquely identifies this stream event
	UUID string `json:"uuid"`

	// SessionID identifies the conversation session
	SessionID string `json:"session_id"`

	// Event contains the raw Anthropic API stream event.
	// Intentionally flexible as event structures vary.
	Event map[string]any `json:"event"`

	// ParentToolUseID links to a tool use in agent workflows
	ParentToolUseID *string `json:"parent_tool_use_id,omitempty"`
}

// message implements the Message interface.
func (StreamEvent) message() {}
