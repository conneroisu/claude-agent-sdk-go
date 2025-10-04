package options

// AgentOptions configures the Claude agent's behavior.
// This struct combines domain configuration (business logic) with
// infrastructure settings (connection details, file paths, env vars).
type AgentOptions struct {
	// === Domain Settings (affect business logic) ===

	// AllowedTools specifies which built-in tools the agent can use.
	// When set, only these tools are available for agent execution.
	AllowedTools []BuiltinTool

	// DisallowedTools specifies which built-in tools the agent cannot use.
	// Takes precedence over AllowedTools if a tool appears in both lists.
	DisallowedTools []BuiltinTool

	// Model optionally specifies which Claude model to use.
	// If nil, the CLI's default model is used.
	Model *string

	// MaxTurns limits the maximum number of conversation turns.
	// If nil, no limit is enforced beyond the CLI's default.
	MaxTurns *int

	// SystemPrompt configures the system prompt for the agent.
	// Can be a simple string or a preset-based configuration.
	SystemPrompt SystemPromptConfig

	// PermissionMode controls how tool permissions are handled.
	// If nil, uses the CLI's default permission mode.
	PermissionMode *PermissionMode

	// PermissionPromptToolName customizes the tool name in prompts.
	// Useful for providing context-specific tool descriptions.
	PermissionPromptToolName *string

	// Agents defines subagent configurations for task delegation.
	// Map keys are agent names, values are their definitions.
	Agents map[string]AgentDefinition

	// === Session Management (domain concern) ===

	// ContinueConversation continues the previous conversation session.
	// When true, appends to the existing conversation history.
	ContinueConversation bool

	// Resume specifies a session ID to resume from a specific point.
	// When set, loads conversation state from the specified session.
	Resume *string

	// ForkSession creates a new session branch from the current point.
	// Allows exploring alternative conversation paths without losing history.
	ForkSession bool

	// IncludePartialMessages includes incomplete messages in the conversation.
	// Useful for debugging or analyzing interrupted agent executions.
	IncludePartialMessages bool

	// === Infrastructure Settings (connection and execution) ===

	// Cwd specifies the working directory for the agent.
	// If nil, uses the current process working directory.
	Cwd *string

	// Settings specifies the path to a custom settings file.
	// If nil, uses the CLI's default settings location.
	Settings *string

	// AddDirs specifies additional directories in agent's context.
	// These directories are added to the file access scope.
	AddDirs []string

	// Env provides additional environment variables for the agent process.
	// These are merged with the current process environment.
	Env map[string]string

	// User specifies a custom user identifier for the session.
	// Useful for multi-user scenarios or session tracking.
	User *string

	// SettingSources specifies which sources to load.
	// Controls the precedence of configuration levels.
	SettingSources []SettingSource

	// MaxBufferSize limits the size of internal message buffers.
	// If nil, uses the CLI's default buffer size.
	MaxBufferSize *int

	// StderrCallback is called with stderr output from the Claude CLI process.
	// Useful for capturing diagnostic information or errors.
	StderrCallback func(string)

	// ExtraArgs provides additional command-line arguments to pass to the CLI.
	// Map keys are flag names (without dashes), values are flag values.
	// Nil values represent boolean flags without arguments.
	ExtraArgs map[string]*string

	// === MCP Server Configuration (infrastructure) ===

	// MCPServers configures MCP server connections.
	// Map keys are server names, values are configurations.
	// Supports client connections (Stdio, SSE, HTTP) and SDK servers.
	MCPServers map[string]MCPServerConfig

	// === Internal Flags (set by domain services, not by users) ===

	// Internal: isStreaming indicates whether this is a streaming session.
	// Set by domain services: true for Client, false for Query.
	// nolint:unused // Set by streaming/querying services
	isStreaming bool
}
