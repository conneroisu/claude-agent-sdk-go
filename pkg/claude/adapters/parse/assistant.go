package parse

import (
	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseAssistantMessage parses an assistant message from raw data.
func (a *Adapter) parseAssistantMessage(data map[string]any) (messages.Message, error) {
	msg, _ := data["message"].(map[string]any)
	contentArray, _ := msg["content"].([]any)

	var blocks []messages.ContentBlock
	for _, item := range contentArray {
		block, _ := item.(map[string]any)
		blockType, _ := block["type"].(string)

		switch blockType {
		case "text":
			text, _ := block["text"].(string)
			blocks = append(blocks, messages.TextBlock{Text: text})
		case "thinking":
			thinking, _ := block["thinking"].(string)
			signature, _ := block["signature"].(string)
			blocks = append(blocks, messages.ThinkingBlock{
				Thinking:  thinking,
				Signature: signature,
			})
		case "tool_use":
			id, _ := block["id"].(string)
			name, _ := block["name"].(string)
			input, _ := block["input"].(map[string]any)
			blocks = append(blocks, messages.ToolUseBlock{
				ID:    id,
				Name:  name,
				Input: input,
			})
		case "tool_result":
			// Tool result blocks are parsed separately via parseToolResultBlock
			toolResultBlock, err := parseToolResultBlock(block)
			if err != nil {
				continue
			}
			blocks = append(blocks, toolResultBlock)
		}
	}

	model, _ := msg["model"].(string)
	parentToolUseID := getStringPtr(data, "parent_tool_use_id")

	return &messages.AssistantMessage{
		Content:         blocks,
		Model:           model,
		ParentToolUseID: parentToolUseID,
	}, nil
}
