package parse

import (
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseContentBlocks parses an array of content blocks.
func parseContentBlocks(contentArr []any) ([]messages.ContentBlock, error) {
	blocks := make([]messages.ContentBlock, 0, len(contentArr))

	for _, item := range contentArr {
		block, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("content block must be an object")
		}

		blockType, ok := block["type"].(string)
		if !ok {
			return nil, fmt.Errorf("content block missing type field")
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
			return nil, fmt.Errorf("unknown content block type: %s", blockType)
		}
	}

	return blocks, nil
}

func parseTextBlock(block map[string]any) (messages.TextBlock, error) {
	text, ok := block["text"].(string)
	if !ok {
		return messages.TextBlock{}, fmt.Errorf("text block missing text field")
	}

	return messages.TextBlock{Text: text}, nil
}

func parseThinkingBlock(block map[string]any) (messages.ThinkingBlock, error) {
	thinking, ok := block["thinking"].(string)
	if !ok {
		return messages.ThinkingBlock{}, fmt.Errorf("thinking block missing thinking field")
	}
	signature, _ := block["signature"].(string)

	return messages.ThinkingBlock{Thinking: thinking, Signature: signature}, nil
}

func parseToolUseBlock(block map[string]any) (messages.ToolUseBlock, error) {
	id, ok := block["id"].(string)
	if !ok {
		return messages.ToolUseBlock{}, fmt.Errorf("tool_use block missing id field")
	}
	name, ok := block["name"].(string)
	if !ok {
		return messages.ToolUseBlock{}, fmt.Errorf("tool_use block missing name field")
	}
	input, ok := block["input"].(map[string]any)
	if !ok {
		input = make(map[string]any)
	}

	return messages.ToolUseBlock{ID: id, Name: name, Input: input}, nil
}

func parseToolResultBlock(block map[string]any) (messages.ToolResultBlock, error) {
	toolUseID, ok := block["tool_use_id"].(string)
	if !ok {
		return messages.ToolResultBlock{}, fmt.Errorf("tool_result block missing tool_use_id field")
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
	} else {
		return messages.ToolResultBlock{}, fmt.Errorf("tool_result content must be string or array")
	}

	var isError *bool
	if isErrorVal, ok := block["is_error"].(bool); ok {
		isError = &isErrorVal
	}

	return messages.ToolResultBlock{
		ToolUseID: toolUseID,
		Content:   content,
		IsError:   isError,
	}, nil
}
