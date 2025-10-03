package parse

import (
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseSystemMessage parses a system message.
// System messages provide lifecycle notifications and status updates.
// Examples: session_start, mcp_server_status, compact_boundary.
//nolint:revive,staticcheck // receiver-naming: unused receiver for interface
//nolint:revive // receiver-naming: Interface implementation requires receiver
func (*Adapter) parseSystemMessage(
	data map[string]any,
) (messages.Message, error) {
	// Extract required subtype field.
//nolint:revive // use-errors-new: formatted message provides context
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, fmt.Errorf(
			"system message missing or invalid subtype field",
		)
	}

	// Extract optional data payload with proper type assertion.
	systemData := make(map[string]any)
	if dataPayload, ok := data["data"].(map[string]any); ok {
		systemData = dataPayload
	}

	return &messages.SystemMessage{
		Subtype: subtype,
		Data:    systemData,
	}, nil
}
