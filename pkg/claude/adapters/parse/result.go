package parse

import (
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseResultMessage parses a result message.
// Result messages indicate completion (success or error).
// Uses JSON marshal/unmarshal for type-safe conversion.
//nolint:revive,staticcheck // receiver-naming: method interface requirement
//nolint:revive // receiver-naming: underscore receiver for method interface
func (_ *Adapter) parseResultMessage(
	data map[string]any,
) (messages.Message, error) {
	// Extract result subtype (success or error variant).
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, fmt.Errorf(
			"result message missing subtype field",
		)
	}

	// Marshal to JSON for structured parsing.
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal result message: %w", err)
	}

	// Parse based on subtype.
	switch subtype {
	case "success":
		// Successful completion with final message.
		var result messages.ResultMessageSuccess
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			return nil, fmt.Errorf(
				"unmarshal success result: %w",
				err,
			)
		}

		return &result, nil

	case "error_max_turns", "error_during_execution":
		// Error completion with error details.
		var result messages.ResultMessageError
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			return nil, fmt.Errorf(
				"unmarshal error result: %w",
				err,
			)
		}

		return &result, nil

	default:
		return nil, fmt.Errorf(
			"unknown result subtype: %s",
			subtype,
		)
	}
}
