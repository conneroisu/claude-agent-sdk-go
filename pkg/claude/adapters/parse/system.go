package parse

import (
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseSystemMessage parses a system message from raw data.
func (a *Adapter) parseSystemMessage(data map[string]any) (messages.Message, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, fmt.Errorf("system message missing subtype field")
	}

	// Data field is intentionally kept as map[string]any
	// Users can parse it into specific SystemMessageData types if needed
	systemData, _ := data["data"].(map[string]any)
	if systemData == nil {
		systemData = make(map[string]any)
	}

	return &messages.SystemMessage{
		Subtype: subtype,
		Data:    systemData,
	}, nil
}
