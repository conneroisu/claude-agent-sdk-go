package options

// AgentOptions configures the Claude agent.
// Combines domain and infrastructure configuration.
type AgentOptions struct {
	// Domain settings (affect business logic)

	// AllowedTools restricts which tools can be used
	AllowedTools []string

	// DisallowedTools blocks specific tools
	DisallowedTools []string

	// Model specifies which Claude model to use
	Model *string

	// MaxTurns limits conversation turns
	MaxTurns *int

	// SystemPrompt configures the system prompt
	SystemPrompt SystemPromptConfig

	// PermissionMode sets permission handling behavior
	PermissionMode *PermissionMode

	// PermissionPromptToolName sets custom permission tool
	PermissionPromptToolName *string

	// Agents defines available subagents
	Agents map[string]AgentDefinition

	// Session management (domain concern)

	// ContinueConversation resumes previous session
	ContinueConversation bool

	// Resume specifies session ID to resume
	Resume *string

	// ForkSession creates new session from current
	ForkSession bool

	// IncludePartialMessages includes incomplete messages
	IncludePartialMessages bool

	// Infrastructure settings (how to connect/execute)

	// Cwd sets working directory
	Cwd *string

	// Settings path to settings file
	Settings *string

	// AddDirs adds directories to context
	AddDirs []string

	// Env sets environment variables
	Env map[string]string

	// User sets user identifier
	User *string

	// SettingSources specifies setting load order
	SettingSources []SettingSource

	// MaxBufferSize limits message buffer size
	MaxBufferSize *int

	// StderrCallback handles stderr output
	StderrCallback func(string)

	// ExtraArgs passes additional CLI arguments
	ExtraArgs map[string]*string

	// MCP server configuration (infrastructure)

	// MCPServers configures MCP server connections
	MCPServers map[string]MCPServerConfig

	// Internal flags (set by domain services, not users)

	// IsStreaming indicates streaming mode
	// nolint:unused
	IsStreaming bool
}
