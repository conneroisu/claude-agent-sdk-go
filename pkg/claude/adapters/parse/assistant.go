package parse

import (
	"errors"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseAssistant parses an assistant message from raw data.
// It extracts the model and content blocks from the message field.
func (a *Adapter) parseAssistant(
	data map[string]any,
) (messages.Message, error) {
	// Extract message data from the raw data
	msgData, ok := data["message"].(map[string]any)
	if !ok {
		return nil, errors.New("assistant message missing message field")
	}

	// Extract model information
	model, _ := getStringField(msgData, "model", false)
	modelPtr := &model

	// Parse content blocks from message
	contentBlocks, err := a.parseContent(msgData)
	if err != nil {
		return nil, err
	}

	return &messages.AssistantMessage{
		Model:   modelPtr,
		Content: contentBlocks,
	}, nil
}

// parseContent extracts and parses content blocks from message data.
// Returns nil if content field is missing or empty.
func (a *Adapter) parseContent(
	data map[string]any,
) ([]messages.ContentBlock, error) {
	// Check if content exists
	contentVal, ok := data["content"]
	if !ok {
		return nil, nil
	}

	// Content must be an array
	contentArr, ok := contentVal.([]any)
	if !ok {
		return nil, errors.New("content must be array")
	}

	// Parse each content block
	blocks := make([]messages.ContentBlock, 0, len(contentArr))

	for _, item := range contentArr {
		blockMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		block, err := a.parseContentBlock(blockMap)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}

// parseContentBlock parses a single content block based on its type.
// Supports text, thinking, tool_use, and tool_result blocks.
// Unknown types are wrapped in UnknownContentBlock.
func (a *Adapter) parseContentBlock(
	data map[string]any,
) (messages.ContentBlock, error) {
	// Extract block type (required)
	blockType, err := getStringField(data, "type", true)
	if err != nil {
		return nil, err
	}

	// Parse based on block type
	switch blockType {
	case "text":
		text, _ := getStringField(data, "text", false)

		return &messages.TextBlock{Text: text}, nil

	case "thinking":
		text, _ := getStringField(data, "thinking", false)

		return &messages.ThinkingBlock{Thinking: text}, nil

	case "tool_use":
		return a.parseToolUse(data)

	case "tool_result":
		return a.parseToolResult(data)

	default:
		// Preserve unknown block types
		return &messages.UnknownContentBlock{
			Type:    blockType,
			RawData: data,
		}, nil
	}
}

// parseToolUse parses a tool use block containing tool invocation details.
// Extracts ID, name, and input parameters for the tool.
func (*Adapter) parseToolUse(
	data map[string]any,
) (messages.ContentBlock, error) {
	id, _ := getStringField(data, "id", false)
	name, _ := getStringField(data, "name", false)
	input, _ := data["input"].(map[string]any)

	return &messages.ToolUseBlock{
		ID:    id,
		Name:  name,
		Input: input,
	}, nil
}

// parseToolResult parses a tool result block containing tool execution results.
// Supports both string and block array content formats.
func (a *Adapter) parseToolResult(
	data map[string]any,
) (messages.ContentBlock, error) {
	// Extract tool use ID
	id, _ := getStringField(data, "tool_use_id", false)

	// Check for error flag
	isError := false
	if errVal, ok := data["is_error"].(bool); ok {
		isError = errVal
	}

	// Parse content - can be string or block array
	var content messages.ToolResultContent
	if contentVal, ok := data["content"]; ok {
		if str, ok := contentVal.(string); ok {
			// String content
			content = messages.ToolResultString(str)
		} else if arr, ok := contentVal.([]any); ok {
			// Block array content
			blocks := a.parseToolResultBlocks(arr)
			content = messages.ToolResultBlocks(blocks)
		}
	}

	return &messages.ToolResultBlock{
		ToolUseID: id,
		Content:   content,
		IsError:   isError,
	}, nil
}

// parseToolResultBlocks parses an array of content blocks for tool results.
func (a *Adapter) parseToolResultBlocks(arr []any) []messages.ContentBlock {
	blocks := make([]messages.ContentBlock, 0, len(arr))
	for _, item := range arr {
		blockMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		block, _ := a.parseContentBlock(blockMap)
		if block != nil {
			blocks = append(blocks, block)
		}
	}

	return blocks
}
