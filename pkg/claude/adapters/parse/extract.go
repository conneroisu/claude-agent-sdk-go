// Package parse provides type-safe extraction helpers for parsing.
// These helpers validate map access and provide clear error messages
// that reference the expected TypeScript SDK types.
package parse

import (
	"errors"
	"fmt"
)

// ErrMissingField indicates a required field is missing from the data.
var ErrMissingField = errors.New("missing required field")

// ErrInvalidType indicates a field has the wrong type.
var ErrInvalidType = errors.New("invalid field type")

// extractRequiredString extracts a required string field.
// Returns error if field is missing or not a string.
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
// Returns nil if field is missing or null.
func extractOptionalString(
	data map[string]any,
	key string,
) *string {
	val, ok := data[key]
	if !ok || val == nil {
		return nil
	}

	str, ok := val.(string)
	if !ok {
		return nil
	}

	return &str
}

// extractRequiredMap extracts a required map field.
// Returns error if field is missing or not a map.
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

// extractOptionalMap extracts an optional map field.
// Returns empty map if field is missing or null.
func extractOptionalMap(
	data map[string]any,
	key string,
) map[string]any {
	val, ok := data[key]
	if !ok || val == nil {
		return make(map[string]any)
	}

	mapVal, ok := val.(map[string]any)
	if !ok {
		return make(map[string]any)
	}

	return mapVal
}

// extractRequiredArray extracts a required array field.
// Returns error if field is missing or not an array.
func extractRequiredArray(
	data map[string]any,
	key string,
) ([]any, error) {
	val, ok := data[key]
	if !ok {
		return nil, fmt.Errorf(
			"%w: %s",
			ErrMissingField,
			key,
		)
	}

	arr, ok := val.([]any)
	if !ok {
		return nil, fmt.Errorf(
			"%w: %s must be array, got %T",
			ErrInvalidType,
			key,
			val,
		)
	}

	return arr, nil
}

// extractOptionalBool extracts an optional boolean field.
// Returns false if field is missing or not a bool.
func extractOptionalBool(
	data map[string]any,
	key string,
) bool {
	val, ok := data[key]
	if !ok {
		return false
	}

	boolVal, ok := val.(bool)
	if !ok {
		return false
	}

	return boolVal
}

// extractOptionalBoolPtr extracts an optional bool pointer.
// Returns nil if field is missing.
func extractOptionalBoolPtr(
	data map[string]any,
	key string,
) *bool {
	val, ok := data[key]
	if !ok || val == nil {
		return nil
	}

	boolVal, ok := val.(bool)
	if !ok {
		return nil
	}

	return &boolVal
}
