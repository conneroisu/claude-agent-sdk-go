package messages

// ContentBlock is a discriminated union for content blocks.
type ContentBlock interface {
	contentBlock()
}

// TextBlock represents a text content block.
type TextBlock struct {
	Text string `json:"text"`
}

func (TextBlock) contentBlock() {}

// ThinkingBlock represents extended thinking content.
type ThinkingBlock struct {
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

func (ThinkingBlock) contentBlock() {}

// ToolUseBlock represents a tool use request.
type ToolUseBlock struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"` // Keep flexible - varies by tool
}

func (ToolUseBlock) contentBlock() {}

// ToolResultContent can be string or a list of content blocks.
type ToolResultContent interface {
	toolResultContent()
}

// ToolResultStringContent is a string tool result.
type ToolResultStringContent string

func (ToolResultStringContent) toolResultContent() {}

// ToolResultBlockListContent is a list of content blocks as maps.
type ToolResultBlockListContent []map[string]any

func (ToolResultBlockListContent) toolResultContent() {}

// ToolResultBlock represents the result of a tool execution.
type ToolResultBlock struct {
	ToolUseID string            `json:"tool_use_id"`
	Content   ToolResultContent `json:"content"`
	IsError   *bool             `json:"is_error,omitempty"`
}

func (ToolResultBlock) contentBlock() {}
