package parse

import (
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseResultMessage parses a result message from raw data.
func (a *Adapter) parseResultMessage(data map[string]any) (messages.Message, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, fmt.Errorf("result message missing subtype field")
	}

	// Type-safe approach: marshal map to JSON, then unmarshal into typed struct
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal result message: %w", err)
	}

	switch subtype {
	case "success":
		var result messages.ResultMessageSuccess
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			return nil, fmt.Errorf("unmarshal success result: %w", err)
		}
		return &result, nil

	case "error_max_turns", "error_during_execution":
		var result messages.ResultMessageError
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			return nil, fmt.Errorf("unmarshal error result: %w", err)
		}
		return &result, nil

	default:
		return nil, fmt.Errorf("unknown result subtype: %s", subtype)
	}
}
