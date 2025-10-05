package parse

import (
	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseSystem parses a system message from raw data.
// System messages contain metadata and control information like session init.
// Different subtypes are parsed into their specific message data structures.
func (*Adapter) parseSystem(data map[string]any) (messages.Message, error) {
	// Extract message subtype to determine parsing strategy
	subtype, _ := getStringField(data, "subtype", false)

	// Parse based on subtype to create proper SystemMessageData
	var msgData messages.SystemMessageData
	switch subtype {
	case "init":
		// Parse session initialization message
		sessionID, _ := getStringField(data, "session_id", false)
		msgData = &messages.SystemInitMessage{
			SessionID: sessionID,
		}
	case "compact_boundary":
		// Parse compaction boundary marker
		compactID, _ := getStringField(data, "compact_id", false)
		msgData = &messages.SystemCompactBoundaryMessage{
			CompactID: compactID,
		}
	default:
		// Handle unknown system message types generically
		msgData = &messages.SystemGenericMessage{
			Type: subtype,
			Data: data,
		}
	}

	return &messages.SystemMessage{
		Data: msgData,
	}, nil
}
