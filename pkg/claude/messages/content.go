package messages

// TextBlock represents plain text content in a message.
type TextBlock struct {
	// Type is always "text"
	Type string `json:"type"`

	// Text contains the actual text content
	Text string `json:"text"`
}

func (TextBlock) contentBlock() {}

// ThinkingBlock represents Claude's internal reasoning process.
// This is part of extended thinking mode where Claude shows its work.
type ThinkingBlock struct {
	// Type is always "thinking"
	Type string `json:"type"`

	// Thinking contains Claude's internal reasoning text
	Thinking string `json:"thinking"`

	// Signature may contain cryptographic verification
	Signature string `json:"signature,omitempty"`
}

func (ThinkingBlock) contentBlock() {}

// ToolUseBlock represents a request to use a tool.
// Claude sends these when it wants to invoke a tool to complete a task.
type ToolUseBlock struct {
	// Type is always "tool_use"
	Type string `json:"type"`

	// ID uniquely identifies this tool use
	ID string `json:"id"`

	// Name is the tool being requested
	Name string `json:"name"`

	// Input contains tool-specific parameters (flexible, varies by tool)
	Input map[string]any `json:"input"`
}

func (ToolUseBlock) contentBlock() {}

// ToolResultBlock represents the result of a tool execution.
// Users send these in response to ToolUseBlocks.
type ToolResultBlock struct {
	// Type is always "tool_result"
	Type string `json:"type"`

	// ToolUseID links this result to the corresponding tool use
	ToolUseID string `json:"tool_use_id"`

	// Content can be a string or list of content blocks
	Content ToolResultContent `json:"content"`

	// IsError indicates if the tool execution failed
	IsError *bool `json:"is_error,omitempty"`
}

func (ToolResultBlock) contentBlock() {}
