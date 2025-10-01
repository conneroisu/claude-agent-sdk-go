package messages

// Message represents a message in the Claude conversation
type Message interface {
	message()
}

// UserMessage represents a message from the user
type UserMessage struct {
	Content         MessageContent
	ParentToolUseID *string
}

func (UserMessage) message() {}

// AssistantMessage represents a message from the assistant
type AssistantMessage struct {
	Content         []ContentBlock
	Model           string
	ParentToolUseID *string
}

func (AssistantMessage) message() {}

// SystemMessage represents a system message
type SystemMessage struct {
	Subtype string
	Data    map[string]any
}

func (SystemMessage) message() {}

// ResultMessage represents the final result of a query
type ResultMessage struct {
	Subtype       string
	DurationMs    int
	DurationAPIMs int
	IsError       bool
	NumTurns      int
	SessionID     string
	TotalCostUSD  *float64
	Usage         map[string]any
	Result        *string
}

func (ResultMessage) message() {}

// StreamEvent represents a streaming event
type StreamEvent struct {
	UUID            string
	SessionID       string
	Event           map[string]any
	ParentToolUseID *string
}

func (StreamEvent) message() {}

// ContentBlock represents a block of content in a message
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
	Input map[string]any
}

func (ToolUseBlock) contentBlock() {}

// ToolResultBlock represents the result of a tool use
type ToolResultBlock struct {
	ToolUseID string
	Content   any // string or []map[string]any
	IsError   *bool
}

func (ToolResultBlock) contentBlock() {}

// MessageContent can be either a string or a list of content blocks
type MessageContent interface {
	messageContent()
}

// StringContent represents simple string content
type StringContent string

func (StringContent) messageContent() {}

// BlockListContent represents content as a list of blocks
type BlockListContent []ContentBlock

func (BlockListContent) messageContent() {}
