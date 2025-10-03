// Package messages defines core domain message types for
// Claude Agent SDK.
//
// Messages represent the data exchanged between the SDK and
// Claude Code CLI. This package uses discriminated unions via
// Go interfaces to provide type-safe message handling while
// maintaining flexibility where needed.
package messages

// Message is the top-level interface for all message types.
// Implementations include UserMessage, AssistantMessage,
// SystemMessage, ResultMessage variants, and StreamEvent.
type Message interface {
	message()
}

// MessageContent represents content that can be either a
// simple string or a list of structured content blocks.
type MessageContent interface {
	messageContent()
}

// StringContent represents simple text content.
type StringContent string

func (StringContent) messageContent() {}

// BlockListContent represents structured content blocks.
type BlockListContent []ContentBlock

func (BlockListContent) messageContent() {}

// ContentBlock is an interface for assistant message content
// blocks (text, thinking, tool_use, tool_result).
type ContentBlock interface {
	contentBlock()
}

// ToolResultContent represents content in a tool result, which
// can be a string or a list of content block maps.
type ToolResultContent interface {
	toolResultContent()
}

// ToolResultStringContent is a string tool result.
type ToolResultStringContent string

func (ToolResultStringContent) toolResultContent() {}

// ToolResultBlockListContent is a list of raw content blocks.
type ToolResultBlockListContent []map[string]any

func (ToolResultBlockListContent) toolResultContent() {}

// SystemMessageData is a discriminated union for SystemMessage
// data payloads. Parse from map[string]any based on Subtype.
type SystemMessageData interface {
	systemMessageData()
}

// ResultMessage is a discriminated union for result messages
// based on Subtype (success vs error variants).
type ResultMessage interface {
	resultMessage()
	Message
}
