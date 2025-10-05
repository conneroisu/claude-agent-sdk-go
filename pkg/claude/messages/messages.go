// Package messages provides domain models for Claude Agent SDK messages.
// This package contains all message types used in communication between the
// SDK and Claude CLI, including user messages, assistant responses, system
// messages, and streaming events.
package messages

// Message is the root interface for all SDK messages.
// All message types in the SDK implement this interface to provide
// type safety and enable polymorphic message handling.
type Message interface {
	// message is a marker method for type safety
	message()
}

// SystemMessageData is a discriminated union for system message variants.
// System messages can represent initialization, compact boundaries, or
// other control messages from the CLI.
type SystemMessageData interface {
	systemMessageData()
}

// ResultMessage is a discriminated union for result messages.
// Results can be either successful completions or error responses.
type ResultMessage interface {
	Message
	resultMessage()
}

// ContentBlock is an interface for content block variants.
// Content blocks can be text, thinking blocks, tool use, or tool results.
type ContentBlock interface {
	contentBlock()
}

// MessageContent is a union of string or content block list.
// User and assistant messages can have content as either a simple string
// or a structured list of content blocks.
type MessageContent interface {
	messageContent()
}

// StringContent represents simple text content.
type StringContent string

func (StringContent) messageContent() {}

// BlocksContent represents structured content blocks.
type BlocksContent []ContentBlock

func (BlocksContent) messageContent() {}

// ToolResultContent is a union of string or block list for tool results.
// Tool results can return either simple text or structured blocks.
type ToolResultContent interface {
	toolResultContent()
}

// ToolResultString represents simple text tool result.
type ToolResultString string

func (ToolResultString) toolResultContent() {}

// ToolResultBlocks represents structured tool result blocks.
type ToolResultBlocks []ContentBlock

func (ToolResultBlocks) toolResultContent() {}

// UnknownMessage represents an unrecognized message type.
// This enables forward compatibility when the API adds new message types
// that the SDK hasn't been updated to handle yet.
type UnknownMessage struct {
	// Type contains the message type identifier
	Type string

	// RawData contains the unparsed message data
	RawData map[string]any
}

func (*UnknownMessage) message() {}
