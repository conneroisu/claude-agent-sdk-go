package parse

import (
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseStreamEvent parses a stream event message.
// Stream events are raw Anthropic API events forwarded through the CLI.
// These events provide real-time updates during message generation.
//nolint:revive // unused-receiver: method signature required
func (a *Adapter) parseStreamEvent(
	data map[string]any,
) (messages.Message, error) {
	// Extract required UUID field.
	uuid, ok := data["uuid"].(string)
//nolint:revive // use-errors-new: formatted message provides context
	if !ok {
		return nil, fmt.Errorf(
			"stream event missing or invalid uuid field",
		)
	}

	// Extract required session ID field.
	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf(
			"stream event missing or invalid session_id field",
		)
	}

	// Extract raw event payload.
	event, ok := data["event"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf(
			"stream event missing or invalid event field",
		)
	}

	// Extract optional parent tool use ID.
	parentToolUseID := getStringPtr(data, "parent_tool_use_id")

	return &messages.StreamEvent{
		UUID:            uuid,
		SessionID:       sessionID,
		Event:           event,
		ParentToolUseID: parentToolUseID,
	}, nil
}
