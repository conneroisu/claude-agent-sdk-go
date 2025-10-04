package options

// AgentOptions configures the Claude agent.
// This combines domain and infrastructure configuration.
type AgentOptions struct {
	// === Domain Settings (affect business logic) ===

	// AllowedTools lists tools the agent can use
	AllowedTools []BuiltinTool

	// DisallowedTools lists tools the agent cannot use
	DisallowedTools []BuiltinTool

	// Model specifies the AI model (optional)
	Model *string

	// MaxTurns limits conversation turns (optional)
	MaxTurns *int

	// SystemPrompt configures the system prompt
	SystemPrompt SystemPromptConfig

	// PermissionMode sets permission handling mode
	PermissionMode *PermissionMode

	// PermissionPromptToolName customizes permission prompts (optional)
	PermissionPromptToolName *string

	// Agents defines subagent configurations
	Agents map[string]AgentDefinition

	// === Session Management (domain concern) ===

	// ContinueConversation continues from previous session
	ContinueConversation bool

	// Resume resumes from a specific session ID (optional)
	Resume *string

	// ForkSession creates a fork of the current session
	ForkSession bool

	// IncludePartialMessages includes incomplete messages
	IncludePartialMessages bool

	// === Infrastructure Settings (how to connect/execute) ===

	// Cwd sets the working directory (optional)
	Cwd *string

	// Settings specifies settings file path (optional)
	Settings *string

	// AddDirs adds additional directories to the context
	AddDirs []string

	// Env sets environment variables
	Env map[string]string

	// User specifies the user identifier (optional)
	User *string

	// SettingSources specifies which setting sources to use
	SettingSources []SettingSource

	// MaxBufferSize sets the maximum buffer size (optional)
	MaxBufferSize *int

	// StderrCallback is called with stderr output
	StderrCallback func(string)

	// ExtraArgs passes additional CLI arguments
	ExtraArgs map[string]*string

	// MCPServers configures MCP server connections
	MCPServers map[string]MCPServerConfig

	// === Internal Flags (set by domain services, not by users) ===

	// isStreaming is true for Client, false for Query
	IsStreaming bool
}
