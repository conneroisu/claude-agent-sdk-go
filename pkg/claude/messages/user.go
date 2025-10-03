// User message types for Claude Agent.
package messages

// UserMessage represents a message from the user to Claude.
//
// Contains user input in the form of text and/or content blocks.
// Can be linked to a parent tool use via ParentToolUseID for
// hierarchical tool execution (e.g., subagent results).
type UserMessage struct {
	Content         MessageContent `json:"content"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

func (UserMessage) message() {}
