package messages

// ContentBlock is the base interface for all content block types.
//
// Content blocks represent structured message components:
//   - TextBlock: Plain text content
//   - ThinkingBlock: Extended thinking content (with signature verification)
//   - ToolUseBlock: Tool invocation request
//   - ToolResultBlock: Tool execution result
type ContentBlock interface {
	contentBlock()
}

// TextBlock represents plain text content.
//
// Used for simple text responses from Claude or user messages.
//
// Example:
//
//	block := TextBlock{
//	    Type: "text",
//	    Text: "The current directory contains 5 files.",
//	}
type TextBlock struct {
	// Type is always "text" for text blocks.
	Type string `json:"type"`

	// Text contains the plain text content.
	Text string `json:"text"`
}

// contentBlock implements the ContentBlock interface.
func (TextBlock) contentBlock() {}

// ThinkingBlock represents extended thinking content.
//
// Contains Claude's internal reasoning process with optional signature
// verification. Used when extended thinking mode is enabled.
//
// Example:
//
//	block := ThinkingBlock{
//	    Type: "thinking",
//	    Thinking: "Let me analyze the user's request...",
//	    Signature: "sha256:abc123",
//	}
type ThinkingBlock struct {
	// Type is always "thinking" for thinking blocks.
	Type string `json:"type"`

	// Thinking contains Claude's reasoning text.
	Thinking string `json:"thinking"`

	// Signature is an optional cryptographic signature for verification.
	Signature string `json:"signature,omitempty"`
}

// contentBlock implements the ContentBlock interface.
func (ThinkingBlock) contentBlock() {}

// ToolUseBlock represents a tool invocation request.
//
// Contains the tool name, unique ID, and input parameters.
// Claude sends these blocks when it wants to use a tool.
//
// Example:
//
//	block := ToolUseBlock{
//	    Type: "tool_use",
//	    ID: "toolu_123",
//	    Name: "Read",
//	    Input: map[string]any{"file_path": "/home/user/file.txt"},
//	}
type ToolUseBlock struct {
	// Type is always "tool_use" for tool use blocks.
	Type string `json:"type"`

	// ID uniquely identifies this tool use.
	ID string `json:"id"`

	// Name is the name of the tool to invoke.
	Name string `json:"name"`

	// Input contains tool-specific parameters.
	// Structure varies by tool (e.g., Bash needs "command",
	// Read needs "file_path").
	Input map[string]any `json:"input"`
}

// contentBlock implements the ContentBlock interface.
func (ToolUseBlock) contentBlock() {}

// ToolResultContent represents the content of a tool result.
//
// Can be either a simple string or a list of structured content blocks.
// This interface allows for flexible tool result representation.
type ToolResultContent interface {
	toolResultContent()
}

// ToolResultStringContent represents simple text tool results.
//
// Used when a tool returns plain text output.
//
// Example:
//
//	content := ToolResultStringContent("File contents: Hello, World!")
type ToolResultStringContent string

// toolResultContent implements the ToolResultContent interface.
func (ToolResultStringContent) toolResultContent() {}

// ToolResultBlockListContent represents structured tool results.
//
// Used when a tool returns complex output with multiple content types.
// Each block is a flexible map to accommodate various content structures.
//
// Example:
//
//	content := ToolResultBlockListContent{
//	    map[string]any{"type": "text", "text": "Output line 1"},
//	    map[string]any{"type": "text", "text": "Output line 2"},
//	}
type ToolResultBlockListContent []map[string]any

// toolResultContent implements the ToolResultContent interface.
func (ToolResultBlockListContent) toolResultContent() {}

// ToolResultBlock represents the result of a tool execution.
//
// Contains the tool use ID, result content, and optional error flag.
// Sent by the SDK to Claude after executing a tool.
//
// Example:
//
//	block := ToolResultBlock{
//	    Type: "tool_result",
//	    ToolUseID: "toolu_123",
//	    Content: ToolResultStringContent("Command output"),
//	}
type ToolResultBlock struct {
	// Type is always "tool_result" for tool result blocks.
	Type string `json:"type"`

	// ToolUseID matches the ID from the corresponding ToolUseBlock.
	ToolUseID string `json:"tool_use_id"`

	// Content contains the tool execution result.
	// Can be simple text or structured content blocks.
	Content ToolResultContent `json:"content"`

	// IsError indicates if the tool execution failed.
	// When true, Content typically contains error details.
	IsError *bool `json:"is_error,omitempty"`
}

// contentBlock implements the ContentBlock interface.
func (ToolResultBlock) contentBlock() {}
