package messages

// AssistantMessage represents a response from Claude.
//
// Contains structured content blocks (text, thinking,
// tool_use, tool_result) along with metadata about which
// model generated the response.
type AssistantMessage struct {
	// Content is a list of structured content blocks
	Content []ContentBlock `json:"content"`

	// Model identifies which Claude model generated
	// this response
	Model string `json:"model"`

	// ParentToolUseID links to a tool use in
	// agent workflows
	ParentToolUseID *string `json:"parent_tool_use_id,omitempty"`
}

// message implements the Message interface.
func (AssistantMessage) message() {}
