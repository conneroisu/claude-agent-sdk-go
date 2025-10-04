package parse

import (
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseUserMessage parses a user message from raw data.
func (a *Adapter) parseUserMessage(data map[string]any) (messages.Message, error) {
	msg, _ := data["message"].(map[string]any)

	// Parse content (can be string or array of blocks)
	var content messages.MessageContent
	if contentStr, ok := msg["content"].(string); ok {
		content = messages.StringContent(contentStr)
	} else if contentArr, ok := msg["content"].([]any); ok {
		blocks, err := parseContentBlocks(contentArr)
		if err != nil {
			return nil, fmt.Errorf("parse user message content blocks: %w", err)
		}
		content = messages.BlockListContent(blocks)
	} else {
		return nil, fmt.Errorf("user message content must be string or array")
	}

	parentToolUseID := getStringPtr(data, "parent_tool_use_id")

	return &messages.UserMessage{
		Content:         content,
		ParentToolUseID: parentToolUseID,
	}, nil
}
