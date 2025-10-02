// Package options provides domain types and configuration options for
// Claude agent behavior, including permission modes, setting sources,
// agent definitions, and system prompt configurations.
package options

// PermissionMode defines how permissions are handled
// This is a DOMAIN concept - it affects business logic.
type PermissionMode string

const (
	PermissionModeDefault           PermissionMode = "default"
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModePlan              PermissionMode = "plan"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
	PermissionModeAsk               PermissionMode = "ask"
)

// SettingSource specifies where settings come from.
type SettingSource string

const (
	SettingSourceUser    SettingSource = "user"
	SettingSourceProject SettingSource = "project"
	SettingSourceLocal   SettingSource = "local"
)

// AgentDefinition defines a subagent configuration
// This is domain configuration - defines behavior of agents.
type AgentDefinition struct {
	Name         string
	Description  string
	SystemPrompt *string
	AllowedTools []string
	Model        *string
}

// SystemPromptConfig is configuration for system prompts.
type SystemPromptConfig interface {
	systemPromptConfig()
}

// StringSystemPrompt is a simple string system prompt.
type StringSystemPrompt string

func (StringSystemPrompt) systemPromptConfig() {}

// PresetSystemPrompt references a preset with optional append.
type PresetSystemPrompt struct {
	Type   string
	Preset string
	Append *string
}

func (PresetSystemPrompt) systemPromptConfig() {}
