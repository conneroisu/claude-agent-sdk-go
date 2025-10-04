package parse

// getStringPtr extracts an optional string pointer from a map.
func getStringPtr(data map[string]any, key string) *string {
	if val, ok := data[key].(string); ok {
		return &val
	}

	return nil
}
