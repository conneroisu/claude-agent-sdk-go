//nolint:revive // comments-density: code is self-documenting
package parse

import (
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// getStringPtr extracts an optional string pointer from a map.
func getStringPtr(data map[string]any, key string) *string {
	if val, ok := data[key].(string); ok {
		return &val
	}

	return nil
}

// parseContentBlocks parses an array of content blocks.
// Validates against TypeScript SDK ContentBlock types:
// TextBlock | ThinkingBlock | ToolUseBlock | ToolResultBlock
func parseContentBlocks(
	contentArr []any,
) ([]messages.ContentBlock, error) {
	blocks := make([]messages.ContentBlock, 0, len(contentArr))

	for _, item := range contentArr {
		block, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf(
				"content block must be an object",
			)
		}

		blockType, err := extractRequiredString(block, "type")
		if err != nil {
			return nil, fmt.Errorf(
				"content block: %w",
				err,
			)
		}

		switch blockType {
		case "text":
			textBlock, err := parseTextBlock(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, textBlock)

		case "thinking":
			thinkingBlock, err := parseThinkingBlock(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, thinkingBlock)

		case "tool_use":
			toolUseBlock, err := parseToolUseBlock(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, toolUseBlock)

		case "tool_result":
			toolResultBlock, err := parseToolResultBlock(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, toolResultBlock)

		default:
			return nil, fmt.Errorf(
				"unknown content block type: %s",
				blockType,
			)
		}
	}

	return blocks, nil
}

// parseTextBlock parses a TextBlock content block.
func parseTextBlock(block map[string]any) (messages.TextBlock, error) {
	text, err := extractRequiredString(block, "text")
	if err != nil {
		return messages.TextBlock{}, fmt.Errorf(
			"text block: %w",
			err,
		)
	}

	return messages.TextBlock{
		Type: "text",
		Text: text,
	}, nil
}

// parseThinkingBlock parses a ThinkingBlock content block.
func parseThinkingBlock(
	block map[string]any,
) (messages.ThinkingBlock, error) {
	thinking, err := extractRequiredString(block, "thinking")
	if err != nil {
		return messages.ThinkingBlock{}, fmt.Errorf(
			"thinking block: %w",
			err,
		)
	}

	signature := extractOptionalString(block, "signature")
	sigStr := ""
	if signature != nil {
		sigStr = *signature
	}

	return messages.ThinkingBlock{
		Type:      "thinking",
		Thinking:  thinking,
		Signature: sigStr,
	}, nil
}

// parseToolUseBlock parses a ToolUseBlock content block.
func parseToolUseBlock(
	block map[string]any,
) (messages.ToolUseBlock, error) {
	id, err := extractRequiredString(block, "id")
	if err != nil {
		return messages.ToolUseBlock{}, fmt.Errorf(
			"tool_use block: %w",
			err,
		)
	}

	name, err := extractRequiredString(block, "name")
	if err != nil {
		return messages.ToolUseBlock{}, fmt.Errorf(
			"tool_use block: %w",
			err,
		)
	}

	input := extractOptionalMap(block, "input")

	return messages.ToolUseBlock{
		Type:  "tool_use",
		ID:    id,
		Name:  name,
		Input: input,
	}, nil
}

// parseToolResultBlock parses a ToolResultBlock content block.
func parseToolResultBlock(
	block map[string]any,
) (messages.ToolResultBlock, error) {
	toolUseID, err := extractRequiredString(block, "tool_use_id")
	if err != nil {
		return messages.ToolResultBlock{}, fmt.Errorf(
			"tool_result block: %w",
			err,
		)
	}

	var content messages.ToolResultContent
	if contentStr, ok := block["content"].(string); ok {
		content = messages.ToolResultStringContent(contentStr)
	} else if contentArr, ok := block["content"].([]any); ok {
		blockMaps := make([]map[string]any, 0, len(contentArr))
		for _, item := range contentArr {
			if blockMap, ok := item.(map[string]any); ok {
				blockMaps = append(blockMaps, blockMap)
			}
		}
		content = messages.ToolResultBlockListContent(blockMaps)
	}

	isError := extractOptionalBoolPtr(block, "is_error")

	return messages.ToolResultBlock{
		Type:      "tool_result",
		ToolUseID: toolUseID,
		Content:   content,
		IsError:   isError,
	}, nil
}
