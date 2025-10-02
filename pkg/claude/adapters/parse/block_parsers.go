// Package parse provides message parsing adapters for the Claude SDK.
package parse

import (
	"errors"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// contentBlockParser is a function that parses a specific block type.
type contentBlockParser func(
	map[string]any,
) (messages.ContentBlock, error)

// contentBlockParsers maps block types to their parsers.
var contentBlockParsers = map[string]contentBlockParser{
	blockTypeText:     parseTextBlockWithError,
	blockTypeThinking: parseThinkingBlockWithError,
	"tool_use":        parseToolUseBlockWithError,
	"tool_result":     parseToolResultBlockWithError,
}

// parseContentBlocks parses an array of content blocks.
func parseContentBlocks(
	contentArr []any,
) ([]messages.ContentBlock, error) {
	blocks := make([]messages.ContentBlock, 0, len(contentArr))

	for _, item := range contentArr {
		block, err := parseContentBlock(item)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// parseContentBlock parses a single content block.
func parseContentBlock(item any) (messages.ContentBlock, error) {
	block, ok := item.(map[string]any)
	if !ok {
		return nil, errors.New("content block must be an object")
	}

	blockType, ok := block["type"].(string)
	if !ok {
		return nil, errors.New("content block missing type field")
	}

	parser, ok := contentBlockParsers[blockType]
	if !ok {
		return nil, fmt.Errorf(
			"unknown content block type: %s",
			blockType,
		)
	}

	return parser(block)
}

// parseTextBlockWithError parses text block with wrapped error.
func parseTextBlockWithError(
	block map[string]any,
) (messages.ContentBlock, error) {
	textBlock, err := parseTextBlock(block)
	if err != nil {
		return nil, fmt.Errorf("parse text block: %w", err)
	}

	return textBlock, nil
}

// parseThinkingBlockWithError parses thinking block with wrapped error.
func parseThinkingBlockWithError(
	block map[string]any,
) (messages.ContentBlock, error) {
	thinkingBlock, err := parseThinkingBlock(block)
	if err != nil {
		return nil, fmt.Errorf("parse thinking block: %w", err)
	}

	return thinkingBlock, nil
}

// parseToolUseBlockWithError parses tool use block with wrapped error.
func parseToolUseBlockWithError(
	block map[string]any,
) (messages.ContentBlock, error) {
	toolUseBlock, err := parseToolUseBlock(block)
	if err != nil {
		return nil, fmt.Errorf("parse tool use block: %w", err)
	}

	return toolUseBlock, nil
}

// parseToolResultBlockWithError parses tool result block with wrapped
// error.
func parseToolResultBlockWithError(
	block map[string]any,
) (messages.ContentBlock, error) {
	toolResultBlock, err := parseToolResultBlock(block)
	if err != nil {
		return nil, fmt.Errorf("parse tool result block: %w", err)
	}

	return toolResultBlock, nil
}

// parseTextBlock parses a text content block.
func parseTextBlock(
	block map[string]any,
) (messages.TextBlock, error) {
	text, ok := block[blockTypeText].(string)
	if !ok {
		return messages.TextBlock{},
			errors.New("text block missing text field")
	}

	return messages.TextBlock{
		Text: text,
	}, nil
}

// parseThinkingBlock parses a thinking content block.
func parseThinkingBlock(
	block map[string]any,
) (messages.ThinkingBlock, error) {
	thinking, ok := block[blockTypeThinking].(string)
	if !ok {
		return messages.ThinkingBlock{},
			errors.New("thinking block missing thinking field")
	}

	signature, _ := block["signature"].(string)

	return messages.ThinkingBlock{
		Thinking:  thinking,
		Signature: signature,
	}, nil
}

// parseToolUseBlock parses a tool_use content block.
func parseToolUseBlock(
	block map[string]any,
) (messages.ToolUseBlock, error) {
	id, ok := block["id"].(string)
	if !ok {
		return messages.ToolUseBlock{},
			errors.New("tool_use block missing id field")
	}

	name, ok := block["name"].(string)
	if !ok {
		return messages.ToolUseBlock{},
			errors.New("tool_use block missing name field")
	}

	input, ok := block["input"].(map[string]any)
	if !ok {
		// Input can be missing or null
		input = make(map[string]any)
	}

	return messages.ToolUseBlock{
		ID:    id,
		Name:  name,
		Input: input,
	}, nil
}

// parseToolResultBlock parses a tool_result content block.
// Tool results link back to tool_use blocks via tool_use_id and can
// contain either simple string output or complex structured content
// (images, documents, etc).
func parseToolResultBlock(
	block map[string]any,
) (messages.ToolResultBlock, error) {
	toolUseID, ok := block["tool_use_id"].(string)
	if !ok {
		return messages.ToolResultBlock{},
			errors.New("tool_result block missing tool_use_id field")
	}

	content, err := parseToolResultContent(block)
	if err != nil {
		return messages.ToolResultBlock{}, err
	}

	isError := parseIsErrorField(block)

	return messages.ToolResultBlock{
		ToolUseID: toolUseID,
		Content:   content,
		IsError:   isError,
	}, nil
}

// parseToolResultContent parses the content field of a tool result.
func parseToolResultContent(
	block map[string]any,
) (messages.ToolResultContent, error) {
	// Parse content (can be string or array of content blocks)
	// This polymorphic approach matches the API's flexibility in
	// representing tool outputs
	if contentStr, ok := block[blockTypeContent].(string); ok {
		return messages.ToolResultStringContent(contentStr), nil
	}

	if contentArr, ok := block[blockTypeContent].([]any); ok {
		return parseToolResultBlockList(contentArr), nil
	}

	return nil, errors.New(
		"tool_result content must be string or array",
	)
}

// parseToolResultBlockList parses an array of content blocks for
// tool results.
func parseToolResultBlockList(
	contentArr []any,
) messages.ToolResultContent {
	// Tool result content can be an array of raw content
	// blocks (maps). We preserve them as maps to maintain
	// forward compatibility with new block types
	blockMaps := make([]map[string]any, 0, len(contentArr))
	for _, item := range contentArr {
		if blockMap, ok := item.(map[string]any); ok {
			blockMaps = append(blockMaps, blockMap)
		}
	}

	return messages.ToolResultBlockListContent(blockMaps)
}
