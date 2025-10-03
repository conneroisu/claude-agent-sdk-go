// Package jsonrpc provides type-safe extraction helpers.
// These helpers validate map access for control protocol messages.
package jsonrpc

import (
	"errors"
	"fmt"
)

// ErrMissingField indicates a required field is missing.
var ErrMissingField = errors.New("missing required field")

// ErrInvalidType indicates a field has the wrong type.
var ErrInvalidType = errors.New("invalid field type")

// extractRequiredString extracts a required string field.
func extractRequiredString(
	data map[string]any,
	key string,
) (string, error) {
	val, ok := data[key]
	if !ok {
		return "", fmt.Errorf(
			"%w: %s",
			ErrMissingField,
			key,
		)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf(
			"%w: %s must be string, got %T",
			ErrInvalidType,
			key,
			val,
		)
	}

	return str, nil
}

// extractOptionalString extracts an optional string field.
func extractOptionalString(
	data map[string]any,
	key string,
) string {
	val, ok := data[key]
	if !ok || val == nil {
		return ""
	}

	str, ok := val.(string)
	if !ok {
		return ""
	}

	return str
}

// extractRequiredMap extracts a required map field.
func extractRequiredMap(
	data map[string]any,
	key string,
) (map[string]any, error) {
	val, ok := data[key]
	if !ok {
		return nil, fmt.Errorf(
			"%w: %s",
			ErrMissingField,
			key,
		)
	}

	mapVal, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf(
			"%w: %s must be object, got %T",
			ErrInvalidType,
			key,
			val,
		)
	}

	return mapVal, nil
}
