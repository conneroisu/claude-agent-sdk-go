// Package options defines configuration types for the Claude
// Agent SDK.
//
// This package separates pure domain configuration
// (permission modes, agent definitions) from infrastructure
// configuration (transport settings, MCP servers).
package options

// PermissionMode defines how tool permissions are handled.
// This is a domain concept that affects business logic.
type PermissionMode string

const (
	// PermissionModeDefault uses standard permission checking.
	PermissionModeDefault PermissionMode = "default"

	// PermissionModeAcceptEdits automatically accepts file edits.
	PermissionModeAcceptEdits PermissionMode = "acceptEdits"

	// PermissionModePlan runs in planning mode without execution.
	PermissionModePlan PermissionMode = "plan"

	// PermissionModeBypassPermissions skips all permission checks.
//nolint:revive // line-length-limit: constant name clarity
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions" //nolint:lll
)

// SettingSource specifies where settings are loaded from.
type SettingSource string

const (
	// SettingSourceUser loads from user-level settings.
	SettingSourceUser SettingSource = "user"

	// SettingSourceProject loads from project-level settings.
	SettingSourceProject SettingSource = "project"

	// SettingSourceLocal loads from local directory settings.
	SettingSourceLocal SettingSource = "local"
)

// AgentDefinition defines a subagent configuration.
// This is domain configuration defining agent behavior.
type AgentDefinition struct {
	// Name identifies the agent
	Name string

	// Description explains the agent's purpose
	Description string

	// SystemPrompt optionally overrides the system prompt
	SystemPrompt *string

	// AllowedTools restricts which tools the agent can use
	AllowedTools []string

	// Model optionally specifies a different Claude model
	Model *string
}

// SystemPromptConfig is configuration for system prompts.
// Supports both string prompts and preset-based prompts.
type SystemPromptConfig interface {
	systemPromptConfig()
}

// StringSystemPrompt is a simple string-based system prompt.
type StringSystemPrompt string

// systemPromptConfig implements the SystemPromptConfig interface.
func (StringSystemPrompt) systemPromptConfig() {}

// PresetSystemPrompt uses a named preset with optional append.
type PresetSystemPrompt struct {
	// Type identifies this as a preset
	Type string

	// Preset is the name of the preset to use
	Preset string

	// Append optionally extends the preset prompt
	Append *string
}

// systemPromptConfig implements the SystemPromptConfig interface.
func (PresetSystemPrompt) systemPromptConfig() {}
