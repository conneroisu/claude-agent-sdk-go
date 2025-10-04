// Package options provides domain and infrastructure config types.
// This package defines pure domain configuration.
package options

// PermissionMode defines how permissions are handled during agent execution.
// This is a domain concept that affects business logic.
type PermissionMode string

const (
	// PermissionModeDefault uses the CLI's default permission behavior.
	PermissionModeDefault PermissionMode = "default"
	// PermissionModeAcceptEdits automatically accepts all file edits.
	PermissionModeAcceptEdits PermissionMode = "acceptEdits"
	// PermissionModePlan enables planning mode for task lists.
	PermissionModePlan PermissionMode = "plan"
	// PermissionModeBypassPermissions bypasses all permission checks.
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
	// PermissionModeAsk prompts for permission before each tool use.
	PermissionModeAsk PermissionMode = "ask"
)

// SettingSource specifies the origin of configuration settings.
// Settings can come from user, project, or local config.
type SettingSource string

const (
	// SettingSourceUser indicates settings from user-level configuration.
	SettingSourceUser SettingSource = "user"
	// SettingSourceProject indicates settings from project configuration.
	SettingSourceProject SettingSource = "project"
	// SettingSourceLocal indicates settings from local overrides.
	SettingSourceLocal SettingSource = "local"
)

// AgentDefinition defines configuration for a subagent.
// Subagents are specialized agents with restricted tool access
// and custom system prompts. This is pure domain configuration.
type AgentDefinition struct {
	// Name is the unique identifier for the subagent
	Name string
	// Description explains the subagent's purpose and capabilities
	Description string
	// SystemPrompt is an optional custom system prompt for the subagent
	SystemPrompt *string
	// AllowedTools restricts which built-in tools this subagent can use
	AllowedTools []BuiltinTool
	// Model optionally specifies a different model for this subagent
	Model *string
}

// SystemPromptConfig is a discriminated union for system prompts.
// System prompts can be simple strings or preset-based configurations.
type SystemPromptConfig interface {
	systemPromptConfig()
}

// StringSystemPrompt represents a simple string-based system prompt.
type StringSystemPrompt string

func (StringSystemPrompt) systemPromptConfig() {}

// PresetSystemPrompt represents a preset-based system prompt configuration.
// Presets are predefined prompt templates that can be optionally extended.
type PresetSystemPrompt struct {
	// Type identifies this as a preset configuration
	Type string
	// Preset is the name of the preset template to use
	Preset string
	// Append is optional additional content to append to the preset
	Append *string
}

func (PresetSystemPrompt) systemPromptConfig() {}
