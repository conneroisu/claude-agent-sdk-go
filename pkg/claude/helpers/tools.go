package helpers

import "github.com/conneroisu/claude/pkg/claude/options"

// AllowTools creates an options modifier for allowed tools.
func AllowTools(
	tools ...options.BuiltinTool,
) func(*options.AgentOptions) {
	return func(opts *options.AgentOptions) {
		opts.AllowedTools = tools
	}
}

// DisallowTools creates an options modifier for disallowed tools.
func DisallowTools(
	tools ...options.BuiltinTool,
) func(*options.AgentOptions) {
	return func(opts *options.AgentOptions) {
		opts.DisallowedTools = tools
	}
}

// WithModel sets the model to use.
func WithModel(model string) func(*options.AgentOptions) {
	return func(opts *options.AgentOptions) {
		opts.Model = &model
	}
}

// WithMaxTurns sets the maximum conversation turns.
func WithMaxTurns(turns int) func(*options.AgentOptions) {
	return func(opts *options.AgentOptions) {
		opts.MaxTurns = &turns
	}
}

// WithPermissionMode sets the permission mode.
func WithPermissionMode(
	mode options.PermissionMode,
) func(*options.AgentOptions) {
	return func(opts *options.AgentOptions) {
		opts.PermissionMode = &mode
	}
}
