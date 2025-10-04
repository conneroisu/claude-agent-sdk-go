package messages

// StreamEvent represents a real-time event from the Anthropic API.
//
// Stream events provide incremental updates during query execution,
// including text deltas, thinking blocks, tool uses, and API events.
// The Event field contains raw API event data in a flexible format.
//
// Example:
//
//	event := StreamEvent{
//	    UUID: "evt_123",
//	    SessionID: "sess_456",
//	    Event: map[string]any{
//	        "type": "content_block_delta",
//	        "delta": map[string]any{
//	            "type": "text_delta",
//	            "text": "Hello",
//	        },
//	    },
//	}
type StreamEvent struct {
	// UUID uniquely identifies this stream event.
	UUID string `json:"uuid"`

	// SessionID identifies the conversation session.
	SessionID string `json:"session_id"`

	// Event contains the raw Anthropic API stream event.
	// Format varies by event type (content_block_delta, message_start, etc.).
	Event map[string]any `json:"event"`

	// ParentToolUseID links this event to a tool use in nested conversations.
	// Used when Claude spawns sub-agents via the Task tool.
	ParentToolUseID *string `json:"parent_tool_use_id,omitempty"`
}

// message implements the Message interface.
func (StreamEvent) message() {}
