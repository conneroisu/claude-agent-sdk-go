package parse

import (
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseStreamEvent parses a stream event from raw data.
func (a *Adapter) parseStreamEvent(data map[string]any) (messages.Message, error) {
	uuid, ok := data["uuid"].(string)
	if !ok {
		return nil, fmt.Errorf("stream event missing uuid field")
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("stream event missing session_id field")
	}

	event, ok := data["event"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("stream event missing event field")
	}

	parentToolUseID := getStringPtr(data, "parent_tool_use_id")

	return &messages.StreamEvent{
		UUID:            uuid,
		SessionID:       sessionID,
		Event:           event,
		ParentToolUseID: parentToolUseID,
	}, nil
}
