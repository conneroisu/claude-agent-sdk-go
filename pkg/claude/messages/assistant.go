package messages

// AssistantMessage represents Claude's response message.
//
// This message type contains Claude's response content (text, thinking,
// tool uses, etc.), the model used, and optionally links to a parent
// tool use for nested conversations.
//
// Example:
//
//	msg := AssistantMessage{
//	    Content: []ContentBlock{
//	        TextBlock{Type: "text", Text: "Here are the files..."},
//	    },
//	    Model: "claude-sonnet-4-20250514",
//	}
type AssistantMessage struct {
	// Content is the assistant's response content.
	// Contains one or more content blocks (text, thinking, tool uses, etc.).
	Content []ContentBlock `json:"content"`

	// Model identifies which Claude model generated this response.
	// Example: "claude-sonnet-4-20250514"
	Model string `json:"model"`

	// ParentToolUseID links this message to a tool use in nested conversations.
	// Used when Claude spawns sub-agents via the Task tool.
	ParentToolUseID *string `json:"parent_tool_use_id,omitempty"`
}

// message implements the Message interface.
func (AssistantMessage) message() {}
