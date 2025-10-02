// Package parse provides message parsing adapters for the Claude SDK.
package parse

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseResultMessageV2 uses JSON marshaling for type-safe parsing.
// This approach eliminates manual type assertions and provides better
// error messages when parsing fails.
func parseResultMessageV2(data map[string]any) (messages.Message, error) {
	// First, check the subtype to know which struct to unmarshal into
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, errors.New("result message missing subtype field")
	}

	// Marshal the map back to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal result data: %w", err)
	}

	// Unmarshal into the appropriate typed struct based on subtype
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
