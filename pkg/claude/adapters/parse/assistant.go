package parse

import (
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseAssistantMessage parses an assistant message.
// Validates against TypeScript SDK SDKAssistantMessage type.
// Required fields: message.content (array), message.model (string)
//nolint:revive // receiver-naming: Interface implementation requires receiver
func (*Adapter) parseAssistantMessage(
	data map[string]any,
) (messages.Message, error) {
	// Extract message envelope (required field).
	msg, err := extractRequiredMap(data, "message")
	if err != nil {
		return nil, fmt.Errorf(
			"assistant message: %w",
			err,
		)
	}

	// Extract content array (required field in TS SDK).
	contentArray, err := extractRequiredArray(msg, "content")
	if err != nil {
		return nil, fmt.Errorf(
			"assistant message: %w",
			err,
		)
	}

	// Parse content blocks using shared helper.
	blocks, err := parseContentBlocks(contentArray)
	if err != nil {
		return nil, fmt.Errorf(
			"assistant message content: %w",
			err,
		)
	}

	// Extract model (required field in TS SDK).
	model, err := extractRequiredString(msg, "model")
	if err != nil {
		return nil, fmt.Errorf(
			"assistant message: %w",
			err,
		)
	}

	// Extract optional parent_tool_use_id.
	parentToolUseID := extractOptionalString(data, "parent_tool_use_id")

	return &messages.AssistantMessage{
		Content:         blocks,
		Model:           model,
		ParentToolUseID: parentToolUseID,
	}, nil
}
