// Stream event types for Claude Agent.
package messages

// StreamEvent represents a real-time streaming event from Claude.
//
// Streaming events provide incremental updates as Claude generates
// responses. The Event field contains raw Anthropic API stream events
// (e.g., content_block_start, content_block_delta, message_stop).
type StreamEvent struct {
	UUID            string         `json:"uuid"`
	SessionID       string         `json:"session_id"`
	Event           map[string]any `json:"event"` // Raw API event
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

func (StreamEvent) message() {}
