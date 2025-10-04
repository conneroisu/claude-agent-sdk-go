// Package options provides configuration types for the Claude Agent SDK.
// This package defines domain options (affecting business logic) and
// infrastructure options (affecting connection/execution).
package options

// PermissionMode defines how permissions are handled.
// This is a domain concept that affects business logic.
type PermissionMode string

const (
	// PermissionModeDefault uses standard permission handling.
	PermissionModeDefault PermissionMode = "default"

	// PermissionModeAcceptEdits auto-accepts file edits.
	PermissionModeAcceptEdits PermissionMode = "acceptEdits"

	// PermissionModePlan enables plan mode.
	PermissionModePlan PermissionMode = "plan"

	// PermissionModeBypassPermissions skips all permission checks.
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"

	// PermissionModeAsk always prompts for permission.
	PermissionModeAsk PermissionMode = "ask"
)

// SettingSource specifies where settings come from.
type SettingSource string

const (
	// SettingSourceUser indicates user-level settings.
	SettingSourceUser SettingSource = "user"

	// SettingSourceProject indicates project-level settings.
	SettingSourceProject SettingSource = "project"

	// SettingSourceLocal indicates local-level settings.
	SettingSourceLocal SettingSource = "local"
)

// AgentDefinition defines a subagent configuration.
// Subagents are specialized agents with specific tools and prompts.
type AgentDefinition struct {
	// Name identifies the subagent
	Name string

	// Description explains what the subagent does
	Description string

	// SystemPrompt is the agent's system prompt (optional)
	SystemPrompt *string

	// AllowedTools lists which tools this agent can use
	AllowedTools []BuiltinTool

	// Model specifies which AI model to use (optional)
	Model *string
}

// SystemPromptConfig is configuration for system prompts.
type SystemPromptConfig interface {
	systemPromptConfig()
}

// StringSystemPrompt is a plain string system prompt.
type StringSystemPrompt string

func (StringSystemPrompt) systemPromptConfig() {}

// PresetSystemPrompt uses a predefined prompt with optional append.
type PresetSystemPrompt struct {
	// Type indicates this is a preset
	Type string

	// Preset is the name of the preset to use
	Preset string

	// Append is text to append to the preset (optional)
	Append *string
}

func (PresetSystemPrompt) systemPromptConfig() {}
