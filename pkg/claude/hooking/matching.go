package hooking

// matches checks if the hook pattern matches the input.
// Empty pattern or "*" matches all inputs (wildcard).
// Non-empty patterns match against the tool_name field for exact equality.
func (*Service) matches(pattern string, input map[string]any) bool {
	// Wildcard patterns match everything
	if pattern == "" || pattern == "*" {
		return true
	}

	// Extract tool name from input
	toolName, ok := input["tool_name"].(string)
	if !ok {
		return false
	}

	// Exact match on tool name
	return toolName == pattern
}
