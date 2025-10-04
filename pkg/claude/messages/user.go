package messages

// UserMessage represents a user's input to Claude.
//
// This message type contains the user's content (text or content blocks)
// and optionally links to a parent tool use for nested conversations.
//
// Example:
//
//	msg := UserMessage{
//	    Content: StringContent("Hello, Claude!"),
//	}
type UserMessage struct {
	// Content is the user's message content.
	// Can be a simple string or structured content blocks.
	Content MessageContent `json:"content"`

	// ParentToolUseID links this message to a tool use in nested conversations.
	// Used when Claude spawns sub-agents via the Task tool.
	ParentToolUseID *string `json:"parent_tool_use_id,omitempty"`
}

// message implements the Message interface.
func (UserMessage) message() {}

// MessageContent represents the content of a message.
//
// Content can be either a simple string or a list of structured content blocks.
// This interface allows for flexible message representation.
type MessageContent interface {
	messageContent()
}

// StringContent represents simple text content.
//
// Used for straightforward text messages without additional structure.
//
// Example:
//
//	content := StringContent("What files are in this directory?")
type StringContent string

// messageContent implements the MessageContent interface.
func (StringContent) messageContent() {}

// BlockListContent represents structured content blocks.
//
// Used for complex messages containing multiple content types
// (text, tool uses, tool results, etc.).
//
// Example:
//
//	content := BlockListContent{
//	    TextBlock{Type: "text", Text: "Hello"},
//	    ToolUseBlock{Type: "tool_use", ID: "123", Name: "Read"},
//	}
type BlockListContent []ContentBlock

// messageContent implements the MessageContent interface.
func (BlockListContent) messageContent() {}
