package messages

// ContentBlock is a discriminated union for content blocks
type ContentBlock interface {
	contentBlock()
}

// TextBlock represents a text content block
type TextBlock struct {
	Text string
}

func (TextBlock) contentBlock() {}

// ThinkingBlock represents extended thinking content
type ThinkingBlock struct {
	Thinking  string
	Signature string
}

func (ThinkingBlock) contentBlock() {}

// ToolUseBlock represents a tool use request
type ToolUseBlock struct {
	ID    string
	Name  string
	Input map[string]any // Intentionally flexible - tool inputs vary by tool
}

func (ToolUseBlock) contentBlock() {}

// ToolResultContent can be string or a list of content blocks
type ToolResultContent interface {
	toolResultContent()
}

// ToolResultStringContent is a string tool result
type ToolResultStringContent string

func (ToolResultStringContent) toolResultContent() {}

// ToolResultBlockListContent is a list of content blocks as maps
type ToolResultBlockListContent []map[string]any

func (ToolResultBlockListContent) toolResultContent() {}

// ToolResultBlock represents the result of a tool execution
type ToolResultBlock struct {
	ToolUseID string
	Content   ToolResultContent
	IsError   *bool
}

func (ToolResultBlock) contentBlock() {}
