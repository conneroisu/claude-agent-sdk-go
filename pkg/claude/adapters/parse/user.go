package parse

import (
	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseUser parses a user message from raw data.
// User messages can contain either a simple string or complex content blocks.
// Supports both string content and structured block arrays.
func (a *Adapter) parseUser(data map[string]any) (messages.Message, error) {
	// Parse content - supports multiple formats
	var content messages.MessageContent

	if promptStr, ok := data["content"].(string); ok {
		// Simple string content
		content = messages.StringContent(promptStr)
	} else if contentArr, ok := data["content"].([]any); ok {
		// Structured block array content
		blocks := make([]messages.ContentBlock, 0, len(contentArr))
		for _, item := range contentArr {
			blockMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			block, _ := a.parseContentBlock(blockMap)
			if block != nil {
				blocks = append(blocks, block)
			}
		}
		content = messages.BlocksContent(blocks)
	} else {
		// Fallback to empty string if content missing
		content = messages.StringContent("")
	}

	return &messages.UserMessage{
		Content: content,
	}, nil
}
