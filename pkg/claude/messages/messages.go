// Package messages provides domain models for Claude Agent messages.
//
// This package defines the core message types used throughout the SDK.
// Messages represent the communication protocol between the SDK and Claude.
//
// Message Types:
//   - UserMessage: User input to Claude
//   - AssistantMessage: Claude's response
//   - SystemMessage: System events and initialization
//   - ResultMessage: Final query results (success or error)
//   - StreamEvent: Real-time streaming events
//
// Design Decision: Uses typed structs for well-defined structures and
// map[string]any for flexible/variable data (e.g., tool inputs).
package messages

// Message is the base interface for all message types.
//
// All message types implement this interface to enable type-safe
// message handling throughout the SDK.
type Message interface {
	message()
}

// MessageContent represents the content of a user message.
//
// Can be either a simple string or a list of content blocks.
// This flexibility supports both simple text input and complex
// multi-modal content (text, images, tool results, etc.).
type MessageContent interface {
	messageContent()
}

// StringContent is a simple string message content.
type StringContent string

// BlockListContent is a list of content blocks.
type BlockListContent []ContentBlock

func (StringContent) messageContent()    {}
func (BlockListContent) messageContent() {}
