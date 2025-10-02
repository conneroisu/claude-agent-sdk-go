// Package messages defines message types for Claude agent communication.
package messages

// Message types - discriminated union
type Message interface {
	message()
}

// UserMessage represents a user message in the conversation
type UserMessage struct {
	Content         MessageContent
	ParentToolUseID *string
	IsSynthetic     bool
}

func (UserMessage) message() {}

// AssistantMessage represents an assistant message in the conversation
type AssistantMessage struct {
	Content         []ContentBlock
	Model           string
	ParentToolUseID *string
}

func (AssistantMessage) message() {}

// StreamEvent represents a streaming event
type StreamEvent struct {
	UUID      string
	SessionID string
	// Intentionally flexible - raw Anthropic API stream event
	Event           map[string]any
	ParentToolUseID *string
}

func (StreamEvent) message() {}

// MessageContent can be string or []ContentBlock
type MessageContent interface {
	messageContent()
}

// StringContent is a string message content
type StringContent string

func (StringContent) messageContent() {}

// BlockListContent is a list of content blocks
type BlockListContent []ContentBlock

func (BlockListContent) messageContent() {}
