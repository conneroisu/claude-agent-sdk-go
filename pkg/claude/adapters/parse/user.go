package parse

import (
	"errors"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseUserMessage parses a user message.
// User messages represent input from the user or synthetic tool results.
func (a *Adapter) parseUserMessage(
	data map[string]any,
) (messages.Message, error) {
	// Extract message envelope with proper type assertion.
	msg, ok := data["message"].(map[string]any)
	if !ok {
		return nil, errors.New(
			"user message missing or invalid message field",
		)
	}

	// Parse content (string or blocks).
	content, err := a.parseMessageContent(msg["content"])
	if err != nil {
		return nil, fmt.Errorf(
			"parse user message content: %w",
			err,
		)
	}

	// Extract optional fields with proper type assertions.
	parentToolUseID := getStringPtr(data, "parent_tool_use_id")
	isSynthetic := false
	if synth, ok := data["isSynthetic"].(bool); ok {
		isSynthetic = synth
	}

	return &messages.UserMessage{
		Content:         content,
		ParentToolUseID: parentToolUseID,
		IsSynthetic:     isSynthetic,
	}, nil
}

// parseMessageContent parses message content (string or blocks).
// Content can be a simple string or an array of content blocks.
//nolint:revive,staticcheck // receiver-naming: method interface requirement
//nolint:revive // receiver-naming: underscore receiver for method interface
func (_ *Adapter) parseMessageContent(
	content any,
) (messages.MessageContent, error) {
	// Handle string content.
	if contentStr, ok := content.(string); ok {
		return messages.StringContent(contentStr), nil
	}

	// Handle block list content.
	if contentArr, ok := content.([]any); ok {
		blocks, err := parseContentBlocks(contentArr)
		if err != nil {
			return nil, fmt.Errorf(
				"parse content blocks: %w",
				err,
			)
		}

		return messages.BlockListContent(blocks), nil
	}
//nolint:revive // use-errors-new: formatted message provides context

	return nil, fmt.Errorf(
		"content must be string or array",
	)
}
