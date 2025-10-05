package parse

import "fmt"

// getStringField extracts a string value from a map by field name.
// Returns error if field is required but missing or not a string.
// For optional fields, returns empty string if missing.
//
//nolint:revive // required flag is appropriate for internal helper
func getStringField(
	data map[string]any,
	field string,
	required bool,
) (string, error) {
	if required {
		return getRequiredStringField(data, field)
	}

	return getOptionalStringField(data, field)
}

// getRequiredStringField extracts a required string field.
func getRequiredStringField(
	data map[string]any,
	field string,
) (string, error) {
	val, ok := data[field]
	if !ok {
		return "", fmt.Errorf("missing required field: %s", field)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("field %s must be string, got %T", field, val)
	}

	return str, nil
}

// getOptionalStringField extracts an optional string field.
func getOptionalStringField(
	data map[string]any,
	field string,
) (string, error) {
	val, ok := data[field]
	if !ok {
		return "", nil
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("field %s must be string, got %T", field, val)
	}

	return str, nil
}
