// Package messages provides domain models for Claude Agent SDK.
// This package defines message types used in bidirectional communication
// with Claude CLI, following hexagonal architecture principles where
// domain models are infrastructure-independent.
package messages

// Message represents any message type in the Claude protocol.
// This is the top-level discriminated union for all message types.
type Message interface {
	message()
}

// MessageContent can be either a simple string or a list of content blocks.
// This allows messages to have structured or unstructured content.
type MessageContent interface {
	messageContent()
}

// StringContent represents simple text content in a message.
type StringContent string

func (StringContent) messageContent() {}

// BlockListContent represents structured content as a list of blocks.
type BlockListContent []ContentBlock

func (BlockListContent) messageContent() {}

// ContentBlock represents a structured content element.
// Content blocks can be text, thinking, tool use, or tool results.
type ContentBlock interface {
	contentBlock()
}

// SystemMessageData is a discriminated union for SystemMessage.Data.
// Parse this from map[string]any based on the Subtype field.
type SystemMessageData interface {
	systemMessageData()
}

// ResultMessage is a discriminated union based on Subtype field.
// Results can be successful completions or errors during execution.
type ResultMessage interface {
	resultMessage()
	Message
}

// ToolResultContent can be either a string or a list of content blocks.
// This matches the flexibility of the Anthropic API tool result format.
type ToolResultContent interface {
	toolResultContent()
}

// ToolResultStringContent represents simple string tool results.
type ToolResultStringContent string

func (ToolResultStringContent) toolResultContent() {}

// ToolResultBlockListContent represents structured tool result content.
type ToolResultBlockListContent []map[string]any

func (ToolResultBlockListContent) toolResultContent() {}
