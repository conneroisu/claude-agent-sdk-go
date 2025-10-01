package options

// PermissionMode defines how permissions are handled
type PermissionMode string

const (
	PermissionModeDefault           PermissionMode = "default"
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModePlan              PermissionMode = "plan"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// SettingSource specifies where settings come from
type SettingSource string

const (
	SettingSourceUser    SettingSource = "user"
	SettingSourceProject SettingSource = "project"
	SettingSourceLocal   SettingSource = "local"
)

// AgentDefinition defines a subagent configuration
type AgentDefinition struct {
	Name         string
	Description  string
	SystemPrompt *string
	AllowedTools []string
	Model        *string
}

// SystemPromptConfig is configuration for system prompts
type SystemPromptConfig interface {
	systemPromptConfig()
}

// StringSystemPrompt is a simple string system prompt
type StringSystemPrompt string

func (StringSystemPrompt) systemPromptConfig() {}

// PresetSystemPrompt uses a preset with optional append
type PresetSystemPrompt struct {
	Type   string
	Preset string
	Append *string
}

func (PresetSystemPrompt) systemPromptConfig() {}
