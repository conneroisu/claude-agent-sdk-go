package cli

import (
	"strconv"
)

// buildCommand constructs CLI arguments from options.
// It converts adapter options into command-line flags.
// Returns args with --output-format set to stream-json by default.
func (a *Adapter) buildCommand() ([]string, error) {
	// Start with stream-json output format for real-time message streaming
	args := []string{"--output-format", "stream-json"}

	// Add model if specified
	if a.opts.Model != "" {
		args = append(args, "--model", a.opts.Model)
	}

	// Add max turns limit if configured
	if a.opts.MaxTurns != nil && *a.opts.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(*a.opts.MaxTurns))
	}

	// Add system prompt for conversation context
	if a.opts.SystemPrompt != "" {
		args = append(args, "--system-prompt", a.opts.SystemPrompt)
	}

	// Add allowed tools whitelist
	if len(a.opts.AllowedTools) > 0 {
		toolStr := buildToolString(a.opts.AllowedTools)
		args = append(args, "--allowed-tools", toolStr)
	}

	// Add denied tools blacklist
	if len(a.opts.DeniedTools) > 0 {
		toolStr := buildToolString(a.opts.DeniedTools)
		args = append(args, "--denied-tools", toolStr)
	}

	// Add working directory if specified
	if a.opts.Cwd != "" {
		args = append(args, "--cwd", a.opts.Cwd)
	}

	return args, nil
}

// buildToolString converts a tool name slice to a comma-separated string.
// Returns empty string if the slice is empty.
func buildToolString(tools []string) string {
	if len(tools) == 0 {
		return ""
	}

	// Build comma-separated tool list
	result := tools[0]
	for i := 1; i < len(tools); i++ {
		result += "," + tools[i]
	}

	return result
}
