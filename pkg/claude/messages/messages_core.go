// Package messages defines message types for Claude agent communication.
package messages

// Message types - discriminated union.
type Message interface {
	message()
}

// UserMessage represents a user message in the conversation.
type UserMessage struct {
	Content         MessageContent `json:"content"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
	IsSynthetic     bool           `json:"isSynthetic,omitempty"`
}

func (UserMessage) message() {}

// AssistantMessage represents an assistant message in the conversation.
type AssistantMessage struct {
	Content         []ContentBlock `json:"content"`
	Model           string         `json:"model"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

func (AssistantMessage) message() {}

// StreamEvent represents a streaming event.
type StreamEvent struct {
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// Intentionally flexible - raw Anthropic API stream event
	Event           map[string]any `json:"event"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

func (StreamEvent) message() {}

// MessageContent can be string or []ContentBlock.
type MessageContent interface {
	messageContent()
}

// StringContent is a string message content.
type StringContent string

func (StringContent) messageContent() {}

// BlockListContent is a list of content blocks.
type BlockListContent []ContentBlock

func (BlockListContent) messageContent() {}
