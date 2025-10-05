package messages

// TextBlock represents plain text content.
// Text blocks contain simple string content from the user or assistant.
type TextBlock struct {
	// Text contains the plain text content
	Text string
}

func (*TextBlock) contentBlock() {}

// ThinkingBlock represents extended thinking content.
// Thinking blocks capture the model's reasoning process and may include
// a signature for verification.
type ThinkingBlock struct {
	// Thinking contains the model's reasoning text
	Thinking string

	// Signature optionally contains a verification signature
	Signature *string
}

func (*ThinkingBlock) contentBlock() {}

// ToolUseBlock represents a tool invocation request.
// Tool use blocks indicate that the model wants to invoke a specific tool
// with the provided input parameters.
type ToolUseBlock struct {
	// ID uniquely identifies this tool use
	ID string

	// Name is the name of the tool to invoke
	Name string

	// Input contains the tool's input parameters as a flexible map
	// The structure varies by tool and is not parsed into typed structs
	Input map[string]any
}

func (*ToolUseBlock) contentBlock() {}

// ToolResultBlock represents a tool execution result.
// Tool result blocks contain the output from a tool invocation, either as
// a success with content or a failure with an error.
type ToolResultBlock struct {
	// ToolUseID references the tool use this result corresponds to
	ToolUseID string

	// Content contains the result content if successful
	Content ToolResultContent

	// IsError indicates whether the tool execution failed
	IsError bool

	// ErrorMessage contains the error description if IsError is true
	ErrorMessage *string
}

func (*ToolResultBlock) contentBlock() {}

// UnknownContentBlock represents an unrecognized content block type.
// This enables forward compatibility when the API adds new block types
// that the SDK hasn't been updated to handle yet.
type UnknownContentBlock struct {
	// Type contains the block type identifier
	Type string

	// RawData contains the unparsed block data
	RawData map[string]any
}

func (*UnknownContentBlock) contentBlock() {}
