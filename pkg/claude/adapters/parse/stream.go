package parse

import (
	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseStreamEvent parses a stream event message from raw data.
// Stream events represent real-time updates during message streaming.
// The raw event data is preserved for access to event-specific fields.
func (*Adapter) parseStreamEvent(
	data map[string]any,
) (messages.Message, error) {
	// Extract event type (e.g., "message_start", "content_block_delta")
	eventType, _ := getStringField(data, "event_type", false)

	// Wrap in StreamEvent to preserve all event data
	return &messages.StreamEvent{
		EventType: eventType,
		Event:     data,
	}, nil
}
