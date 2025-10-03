// Package options provides configuration types for Claude Agent.
//
// This package defines domain and infrastructure configuration options
// used throughout the SDK. Options are separated into:
//   - Pure domain configuration (PermissionMode, AgentDefinition, etc.)
//   - Infrastructure configuration (transport, MCP servers)
//   - Built-in tool type definitions (BuiltinTool constants)
package options

// PermissionMode defines how permissions are handled.
//
// This is a domain concept that affects business logic and Claude's
// behavior when requesting tool use.
type PermissionMode string

const (
	// PermissionModeDefault uses default permission handling.
	PermissionModeDefault PermissionMode = "default"
	// PermissionModeAcceptEdits auto-accepts file edit operations.
	PermissionModeAcceptEdits PermissionMode = "acceptEdits"
	// PermissionModePlan enables planning mode with review step.
	PermissionModePlan PermissionMode = "plan"
	// PermissionModeBypassPermissions bypasses all permission checks.
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
	// PermissionModeAsk prompts for permission on every tool use.
	PermissionModeAsk PermissionMode = "ask"
)

// SettingSource specifies where settings come from.
//
// Used to determine configuration precedence and override behavior.
type SettingSource string

const (
	// SettingSourceUser indicates user-level settings (~/.claude).
	SettingSourceUser SettingSource = "user"
	// SettingSourceProject indicates project-level settings (.claude/).
	SettingSourceProject SettingSource = "project"
	// SettingSourceLocal indicates local overrides.
	SettingSourceLocal SettingSource = "local"
)

// AgentDefinition defines a subagent configuration.
//
// Subagents are specialized Claude instances with specific tools,
// prompts, and models. Used via the Task tool to delegate work.
type AgentDefinition struct {
	Name         string
	Description  string
	SystemPrompt *string
	AllowedTools []BuiltinTool
	Model        *string
}

// SystemPromptConfig is configuration for system prompts.
//
// Can be either a simple string or a preset with optional append.
type SystemPromptConfig interface {
	systemPromptConfig()
}

// StringSystemPrompt is a simple string system prompt.
type StringSystemPrompt string

// PresetSystemPrompt uses a named preset with optional append.
type PresetSystemPrompt struct {
	Type   string
	Preset string
	Append *string
}

func (StringSystemPrompt) systemPromptConfig()  {}
func (PresetSystemPrompt) systemPromptConfig() {}
