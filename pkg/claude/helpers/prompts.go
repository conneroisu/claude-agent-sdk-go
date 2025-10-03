// Package helpers provides convenience utilities for common
// operations with Claude.
package helpers

import "strings"

// BuildSystemPrompt combines multiple prompt parts.
func BuildSystemPrompt(parts ...string) string {
	return strings.Join(parts, "\n\n")
}

// AppendSystemPrompt creates an append-style system prompt config.
func AppendSystemPrompt(base, appendStr string) string {
	if base == "" {
		return appendStr
	}
	if appendStr == "" {
		return base
	}

	return base + "\n\n" + appendStr
}
