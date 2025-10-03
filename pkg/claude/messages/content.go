package messages

// TextBlock represents plain text content.
type TextBlock struct {
	// Type is always "text"
	Type string `json:"type"`

	// Text contains the content
	Text string `json:"text"`
}

// contentBlock implements the ContentBlock interface.
func (TextBlock) contentBlock() {}

// ThinkingBlock represents Claude's internal reasoning.
// Available when extended thinking is enabled.
type ThinkingBlock struct {
	// Type is always "thinking"
	Type string `json:"type"`

	// Thinking contains the reasoning content
	Thinking string `json:"thinking"`

	// Signature provides verification for thinking content
	Signature string `json:"signature,omitempty"`
}

// contentBlock implements the ContentBlock interface.
func (ThinkingBlock) contentBlock() {}

// ToolUseBlock represents a tool invocation by Claude.
type ToolUseBlock struct {
	// Type is always "tool_use"
	Type string `json:"type"`

	// ID uniquely identifies this tool use
	ID string `json:"id"`

	// Name is the tool being invoked
	Name string `json:"name"`

	// Input contains tool parameters.
	// Intentionally flexible as inputs vary by tool.
	Input map[string]any `json:"input"`
}

// contentBlock implements the ContentBlock interface.
func (ToolUseBlock) contentBlock() {}

// ToolResultBlock represents the result of a tool execution.
type ToolResultBlock struct {
	// Type is always "tool_result"
	Type string `json:"type"`

	// ToolUseID links to the corresponding tool use
	ToolUseID string `json:"tool_use_id"`

	// Content is the tool's output, either string or blocks.
	// Can be ToolResultStringContent or
	// ToolResultBlockListContent.
	Content ToolResultContent `json:"content"`

	// IsError indicates if the tool execution failed
	IsError *bool `json:"is_error,omitempty"`
}

// contentBlock implements the ContentBlock interface.
func (ToolResultBlock) contentBlock() {}
