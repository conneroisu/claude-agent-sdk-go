package helpers

import "strings"

// BuildSystemPrompt combines multiple prompt parts into a single prompt.
// Parts are joined with double newlines for clear separation.
//
// Example:
//
//	prompt := helpers.BuildSystemPrompt(
//		"You are a code review assistant.",
//		"Focus on security and performance.",
//		"Provide actionable feedback.",
//	)
//	// Returns the parts joined with "\n\n"
func BuildSystemPrompt(parts ...string) string {
	return strings.Join(parts, "\n\n")
}

// AppendSystemPrompt creates a system prompt by appending to a base.
// Returns the suffix part if base is empty, base if suffix is empty,
// or both joined with double newlines.
//
// Example:
//
//	prompt := helpers.AppendSystemPrompt(
//		"You are a helpful assistant.",
//		"Focus on Go best practices.",
//	)
//	// Returns: "You are a helpful assistant.\n\nFocus on Go best practices."
func AppendSystemPrompt(base, suffix string) string {
	if base == "" {
		return suffix
	}
	if suffix == "" {
		return base
	}

	return base + "\n\n" + suffix
}
