// Package parse provides message parsing adapters for the Claude SDK.
package parse

import (
	"github.com/conneroisu/claude/pkg/claude/messages"
)

// getStringPtr extracts optional string pointers from data
func getStringPtr(data map[string]any, key string) *string {
	if val, ok := data[key].(string); ok {
		return &val
	}

	return nil
}

// parseIsErrorField parses the optional is_error field
func parseIsErrorField(block map[string]any) *bool {
	// is_error is optional - use pointer to distinguish between
	// false and absent
	if isErrorVal, ok := block["is_error"].(bool); ok {
		return &isErrorVal
	}

	return nil
}

// assistantBlockParser is a function that parses assistant blocks
type assistantBlockParser func(map[string]any) messages.ContentBlock

// assistantBlockParsers maps block types to parsers for assistant
// messages
var assistantBlockParsers = map[string]assistantBlockParser{
	blockTypeText:     parseTextContentBlock,
	blockTypeThinking: parseThinkingContentBlock,
	"tool_use":        parseToolUseContentBlock,
	"tool_result":     parseToolResultContentBlock,
}

// parseAssistantContentBlocks parses content blocks from assistant
// messages
func parseAssistantContentBlocks(
	contentArray []any,
) []messages.ContentBlock {
	var blocks []messages.ContentBlock

	// Process each block inline for efficiency since we need
	// type-specific handling
	for _, item := range contentArray {
		block := parseAssistantContentBlock(item)
		if block != nil {
			blocks = append(blocks, block)
		}
	}

	return blocks
}

// parseAssistantContentBlock parses a single assistant content block
func parseAssistantContentBlock(item any) messages.ContentBlock {
	block, _ := item.(map[string]any)
	blockType, _ := block["type"].(string)

	parser, ok := assistantBlockParsers[blockType]
	if !ok {
		return nil
	}

	return parser(block)
}

// parseTextContentBlock parses a text block for assistant messages
func parseTextContentBlock(
	block map[string]any,
) messages.ContentBlock {
	text, _ := block[blockTypeText].(string)

	return messages.TextBlock{Text: text}
}

// parseThinkingContentBlock parses a thinking block for assistant
// messages
func parseThinkingContentBlock(
	block map[string]any,
) messages.ContentBlock {
	thinking, _ := block[blockTypeThinking].(string)
	signature, _ := block["signature"].(string)

	return messages.ThinkingBlock{
		Thinking:  thinking,
		Signature: signature,
	}
}

// parseToolUseContentBlock parses a tool use block for assistant
// messages
func parseToolUseContentBlock(
	block map[string]any,
) messages.ContentBlock {
	id, _ := block["id"].(string)
	name, _ := block["name"].(string)
	input, _ := block["input"].(map[string]any)

	return messages.ToolUseBlock{
		ID:    id,
		Name:  name,
		Input: input,
	}
}

// parseToolResultContentBlock parses a tool result block for
// assistant messages
func parseToolResultContentBlock(
	block map[string]any,
) messages.ContentBlock {
	toolResultBlock, err := parseToolResultBlock(block)
	if err != nil {
		// Skip malformed tool result blocks rather than
		// failing the entire message. This provides
		// resilience against partial data corruption.
		return nil
	}

	return toolResultBlock
}
