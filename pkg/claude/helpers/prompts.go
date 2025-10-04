// Package helpers provides utility functions for SDK users.
package helpers

import "github.com/conneroisu/claude/pkg/claude/options"

// StringPrompt creates a simple string system prompt.
func StringPrompt(text string) options.SystemPromptConfig {
	return options.StringSystemPrompt(text)
}

// PresetPrompt creates a preset system prompt with optional append.
//
//nolint:revive // Parameter name matches Claude API naming convention
func PresetPrompt(append *string) options.SystemPromptConfig {
	return options.PresetSystemPrompt{Append: append}
}
