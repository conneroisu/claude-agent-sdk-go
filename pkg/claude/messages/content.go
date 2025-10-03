// Content block types for Claude Agent messages.
package messages

// ContentBlock is the base interface for all content block types.
//
// Content blocks can be text, thinking, tool use, or tool results.
// They are used in AssistantMessage and UserMessage content.
type ContentBlock interface {
	contentBlock()
}

// TextBlock represents plain text content.
type TextBlock struct {
	Type string `json:"type"` // Always "text"
	Text string `json:"text"`
}

func (TextBlock) contentBlock() {}

// ThinkingBlock represents Claude's extended thinking process.
//
// Available when using extended thinking mode. Contains the reasoning
// process Claude used before generating the final response.
type ThinkingBlock struct {
	Type      string `json:"type"` // Always "thinking"
	Thinking  string `json:"thinking"`
	Signature string `json:"signature,omitempty"`
}

func (ThinkingBlock) contentBlock() {}

// ToolUseBlock represents Claude's request to use a tool.
//
// Contains the tool name, unique ID, and tool-specific input parameters.
// Input is intentionally map[string]any because it varies by tool.
type ToolUseBlock struct {
	Type  string         `json:"type"` // Always "tool_use"
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"` // Tool inputs vary by tool
}

func (ToolUseBlock) contentBlock() {}

// ToolResultContent represents the content of a tool result.
//
// Can be either a simple string or a list of content blocks
// (represented as maps for flexibility).
type ToolResultContent interface {
	toolResultContent()
}

// ToolResultStringContent is a simple string tool result.
type ToolResultStringContent string

// ToolResultBlockListContent is a list of content blocks.
type ToolResultBlockListContent []map[string]any

func (ToolResultStringContent) toolResultContent()     {}
func (ToolResultBlockListContent) toolResultContent() {}

// ToolResultBlock represents the result of a tool execution.
//
// Sent by the user to provide the result of a tool use request.
// Can indicate success (IsError=false) or failure (IsError=true).
type ToolResultBlock struct {
	Type      string            `json:"type"` // Always "tool_result"
	ToolUseID string            `json:"tool_use_id"`
	Content   ToolResultContent `json:"content"`
	IsError   *bool             `json:"is_error,omitempty"`
}

func (ToolResultBlock) contentBlock() {}
