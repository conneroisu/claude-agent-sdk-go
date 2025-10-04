package messages

// AssistantMessage represents a response from Claude.
// Assistant messages contain the AI's response content,
// including text, thinking blocks, and tool use requests.
type AssistantMessage struct {
	// Content is always a list of ContentBlocks for assistant messages
	Content []ContentBlock `json:"content"`

	// Model identifies which AI model generated this response
	Model string `json:"model"`

	// ParentToolUseID links this response to a tool use if applicable
	ParentToolUseID *string `json:"parent_tool_use_id,omitempty"`
}

func (AssistantMessage) message() {}
