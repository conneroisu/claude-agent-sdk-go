package helpers

import "strings"

// BuildSystemPrompt joins prompt parts with double newlines.
func BuildSystemPrompt(parts ...string) string {
	return strings.Join(parts, "\n\n")
}

// AppendSystemPrompt safely appends to a base prompt with separator.
func AppendSystemPrompt(base, suffix string) string {
	if base == "" {
		return suffix
	}
	if suffix == "" {
		return base
	}

	return base + "\n\n" + suffix
}
