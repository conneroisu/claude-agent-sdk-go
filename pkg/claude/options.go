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
// Tools and DisallowedTools are mutually exclusive - specify one or the other.
// Tools is an allowlist (agent can only use these tools), while DisallowedTools
// is a denylist (agent can use any tool except these).
type AgentDefinition struct {
	Description     string   `json:"description"`
	Prompt          string   `json:"prompt"`
	Tools           []string `json:"tools"`
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
