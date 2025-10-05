// Package options provides domain configuration types for the Claude Agent SDK.
// This package contains option types for configuring agents, permissions,
// tools, and MCP servers.
package options

// PermissionMode defines the tool permission behavior.
// Permission modes control how the agent handles tool execution requests.
type PermissionMode string

const (
	// PermissionModeDefault uses standard permission checking.
	PermissionModeDefault PermissionMode = "default"

	// PermissionModeAcceptEdits automatically accepts file edits.
	PermissionModeAcceptEdits PermissionMode = "accept_edits"

	// PermissionModePlan enables planning mode with restricted tools.
	PermissionModePlan PermissionMode = "plan"

	// PermissionModeAllow allows all tool uses without prompting.
	PermissionModeAllow PermissionMode = "allow"

	// PermissionModeDeny denies all tool uses.
	PermissionModeDeny PermissionMode = "deny"
)

// SettingSource indicates where a setting value originated.
// Setting sources help track configuration precedence (user, project, local).
type SettingSource string

const (
	// SettingSourceUser indicates a user-level setting.
	SettingSourceUser SettingSource = "user"

	// SettingSourceProject indicates a project-level setting.
	SettingSourceProject SettingSource = "project"

	// SettingSourceLocal indicates a local override setting.
	SettingSourceLocal SettingSource = "local"
)

// AgentDefinition defines a subagent configuration.
// Subagents enable hierarchical agent structures with delegated tasks.
type AgentDefinition struct {
	// Name identifies the subagent
	Name string

	// SystemPrompt configures the subagent's behavior
	SystemPrompt SystemPromptConfig

	// AllowedTools restricts which tools the subagent can use
	AllowedTools []BuiltinTool

	// PermissionMode sets the default permission behavior
	PermissionMode PermissionMode
}

// SystemPromptConfig represents system prompt configuration.
// System prompts can be either direct strings or preset names.
type SystemPromptConfig interface {
	systemPromptConfig()
}

// StringSystemPrompt uses a direct prompt string.
type StringSystemPrompt string

func (StringSystemPrompt) systemPromptConfig() {}

// PresetSystemPrompt references a named preset.
type PresetSystemPrompt string

func (PresetSystemPrompt) systemPromptConfig() {}
