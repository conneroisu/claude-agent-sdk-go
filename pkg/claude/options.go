package claude

import "context"

// Options configures the Claude SDK client.
type Options struct {
	// Cancellation and control
	Context context.Context

	// Directory and tool configuration
	AdditionalDirectories []string
	AllowedTools          []string
	DisallowedTools       []string
	Cwd                   string

	// System prompt customization
	// nil for vanilla, literal, or preset+append
	SystemPrompt SystemPromptConfig

	// Permission handling
	CanUseTool     CanUseToolFunc
	PermissionMode PermissionMode
	// Customize which tool is used for permission prompts
	PermissionPromptToolName string

	// Session management
	Continue        bool
	Resume          string
	ResumeSessionAt string
	ForkSession     bool

	// Environment and execution
	Env            map[string]string
	Executable     string // "node", "bun", "deno"
	ExecutableArgs []string
	ExtraArgs      map[string]*string

	// Model configuration
	Model             string
	FallbackModel     string
	MaxThinkingTokens int
	MaxTurns          int

	// Budget and output constraints
	// MaxBudgetUsd enforces a maximum spending limit in USD for API calls during the query session.
	// Precision is maintained to two decimal places (penny precision). A value of 0 or omission
	// means no budget enforcement.
	MaxBudgetUsd float64 `json:"maxBudgetUsd,omitempty"`

	// OutputFormat specifies the desired output format for structured outputs.
	// When set, the model's responses will conform to the specified JSON schema format.
	// A nil value uses the default text output format without schema constraints.
	OutputFormat *JsonSchemaOutputFormat `json:"outputFormat,omitempty"`

	// AllowDangerouslySkipPermissions bypasses permission checks when set to true.
	// WARNING: This is a security risk. When enabled, tools execute without user approval prompts.
	// Only use this in controlled environments where the implications of disabling permission
	// checks are fully understood. In production or untrusted environments, keep this false
	// to ensure user approval is required for tool execution.
	AllowDangerouslySkipPermissions bool `json:"allowDangerouslySkipPermissions,omitempty"`

	// Plugins configures SDK plugins for extending functionality.
	// Plugins provide custom commands, agents, skills, and hooks that extend Claude Code's capabilities.
	// Currently only local plugins are supported via the 'local' type.
	Plugins []SdkPluginConfig `json:"plugins,omitempty"`

	// MCP servers
	McpServers      map[string]McpServerConfig
	StrictMcpConfig bool

	// Hooks and callbacks
	Hooks  map[HookEvent][]HookCallbackMatcher
	Stderr func(string)

	// Message handling
	IncludePartialMessages bool

	// SDK-specific
	PathToClaudeCodeExecutable string

	// Settings sources
	SettingSources []ConfigScope // validated scopes: local, user, project

	// Agents
	Agents map[string]AgentDefinition
}

// AgentDefinition defines a custom agent.
//
// Tools and DisallowedTools control which tools the agent can use:
//   - Tools: Explicitly lists allowed tools. If set, only these tools are available.
//   - DisallowedTools: Lists tools to exclude from the agent's available tools.
//
// These fields are mutually exclusive in practice - use one or the other, not both.
// If both are specified, the CLI will respect both constraints (allow only Tools,
// but exclude DisallowedTools from that set).
type AgentDefinition struct {
	Description     string   `json:"description"`
	Prompt          string   `json:"prompt"`
	Tools           []string `json:"tools,omitempty"`
	DisallowedTools []string `json:"disallowedTools,omitempty"`
	Model           string   `json:"model,omitempty"`
}

// ModelInfo represents model information.
type ModelInfo struct {
	Value       string `json:"value"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

// SlashCommand represents available slash commands.
type SlashCommand struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	ArgumentHint string `json:"argumentHint"`
}
