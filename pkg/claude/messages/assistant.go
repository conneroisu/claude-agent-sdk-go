// Assistant message types for Claude Agent.
package messages

// AssistantMessage represents a message from Claude.
//
// Contains Claude's response, which can include text, thinking blocks,
// tool use requests, and other content types. The Model field indicates
// which Claude model generated this response.
type AssistantMessage struct {
	Content         []ContentBlock `json:"content"`
	Model           string         `json:"model"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

func (AssistantMessage) message() {}
