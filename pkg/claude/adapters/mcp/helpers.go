package mcp

import (
	"encoding/json"
	"fmt"
)

// SerializeToolArgs converts tool arguments to JSON.
// This is used when sending tool calls over the wire.
func SerializeToolArgs(args map[string]any) (string, error) {
	data, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("serialize failed: %w", err)
	}

	return string(data), nil
}

// DeserializeToolArgs parses JSON tool arguments.
// This is used when receiving tool calls.
func DeserializeToolArgs(data string) (map[string]any, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(data), &args); err != nil {
		return nil, fmt.Errorf("deserialize failed: %w", err)
	}

	return args, nil
}

// ValidateToolResult checks if a tool result is valid.
// Tool results should be JSON-serializable.
func ValidateToolResult(result map[string]any) error {
	_, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("invalid tool result: %w", err)
	}

	return nil
}
