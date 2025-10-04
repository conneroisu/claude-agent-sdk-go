package messages

// UserMessage represents a message from the user to Claude.
// User messages can contain text, tool results, or other content types.
type UserMessage struct {
	// Content can be a string or a list of ContentBlocks
	Content MessageContent `json:"content"`

	// ParentToolUseID links this message to a tool use if applicable
	ParentToolUseID *string `json:"parent_tool_use_id,omitempty"`
}

func (UserMessage) message() {}
