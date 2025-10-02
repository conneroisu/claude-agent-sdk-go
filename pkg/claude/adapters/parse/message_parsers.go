// Package parse provides message parsing adapters for the Claude SDK.
package parse

import (
	"errors"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseUserMessage handles user message parsing with polymorphic
// content support. User messages can contain either simple string
// content or complex block-based content, requiring dynamic type
// checking to determine the appropriate parsing strategy.
func parseUserMessage(
	data map[string]any,
) (messages.Message, error) {
	msg, _ := data["message"].(map[string]any)

	// Parse content (can be string or array of blocks)
	// The Anthropic API supports both formats for flexibility:
	// - String: simple text messages
	// - Array: rich content with images, tool results, etc.
	var content messages.MessageContent
	if contentStr, ok := msg["content"].(string); ok {
		content = messages.StringContent(contentStr)
	} else if contentArr, ok := msg["content"].([]any); ok {
		blocks, err := parseContentBlocks(contentArr)
		if err != nil {
			return nil, fmt.Errorf(
				"parse user message content blocks: %w",
				err,
			)
		}
		content = messages.BlockListContent(blocks)
	} else {
		return nil, errors.New(
			"user message content must be string or array",
		)
	}

	// Extract optional fields that provide message context
	parentToolUseID := getStringPtr(data, "parent_tool_use_id")
	isSynthetic, _ := data["isSynthetic"].(bool)

	return &messages.UserMessage{
		Content:         content,
		ParentToolUseID: parentToolUseID,
		IsSynthetic:     isSynthetic,
	}, nil
}

// parseSystemMessage handles internal system messages for SDK
// coordination. These messages manage session lifecycle and
// boundaries but don't directly interact with the Anthropic API.
func parseSystemMessage(
	data map[string]any,
) (messages.Message, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, errors.New("system message missing subtype field")
	}

	// Data field is intentionally kept as map[string]any
	// Users can parse it into specific SystemMessageData types
	// if needed (SystemMessageInit, SystemMessageCompactBoundary)
	systemData, _ := data["data"].(map[string]any)
	if systemData == nil {
		systemData = make(map[string]any)
	}

	return &messages.SystemMessage{
		Subtype: subtype,
		Data:    systemData,
	}, nil
}

// parseResultMessage handles result messages which contain session
// execution metrics. These messages provide valuable telemetry
// including token usage, costs, and errors.
func parseResultMessage(
	data map[string]any,
) (messages.Message, error) {
	// Use V2 parser which leverages JSON unmarshaling for type safety
	return parseResultMessageV2(data)
}

// parseStreamEvent handles streaming events from the Anthropic API.
// These events provide real-time updates during message generation,
// allowing clients to display partial responses and track
// progress.
func parseStreamEvent(
	data map[string]any,
) (messages.Message, error) {
	uuid, ok := data["uuid"].(string)
	if !ok {
		return nil, errors.New("stream event missing uuid field")
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, errors.New("stream event missing session_id field")
	}

	event, ok := data["event"].(map[string]any)
	if !ok {
		return nil, errors.New("stream event missing event field")
	}

	parentToolUseID := getStringPtr(data, "parent_tool_use_id")

	return &messages.StreamEvent{
		UUID:      uuid,
		SessionID: sessionID,
		// Keep as map[string]any (raw Anthropic API event)
		Event:           event,
		ParentToolUseID: parentToolUseID,
	}, nil
}

// parseAssistantMessage handles assistant messages which contain the
// model's response. Assistant messages always use block-based content
// and may include text, thinking, tool use, and tool result blocks in
// a single message.
func parseAssistantMessage(
	data map[string]any,
) (messages.Message, error) {
	// Parse content blocks - assistant messages always use
	// structured content
	msg, _ := data["message"].(map[string]any)
	contentArray, _ := msg["content"].([]any)
	blocks := parseAssistantContentBlocks(contentArray)
	model, _ := msg["model"].(string)
	parentToolUseID := getStringPtr(data, "parent_tool_use_id")

	return &messages.AssistantMessage{
		Content:         blocks,
		Model:           model,
		ParentToolUseID: parentToolUseID,
	}, nil
}
