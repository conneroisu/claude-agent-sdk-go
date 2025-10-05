package options

// AgentOptions combines domain and infrastructure settings.
// This is the main configuration struct passed to agent operations.
type AgentOptions struct {
	// Domain Configuration
	Model               string
	SystemPrompt        string
	PermissionMode      PermissionMode
	AllowedTools        []string
	DeniedTools         []string
	Subagents           []AgentDefinition
	CustomInstructions  string
	OnlyWriteableIfSafe bool
	ContextSize         *int
	MaxTurns            *int

	// Session Management
	SessionID  *string
	ForkFrom   *string
	ContinueOn *string
	Resume     bool

	// Infrastructure Configuration
	WorkingDirectory string
	Cwd              string
	SettingsPath     *string
	Environment      map[string]string
	CLIPath          *string

	// MCP Servers
	MCPServers []MCPServerConfig

	// Hook Callbacks
	HookCallbacks map[string]HookCallback

	// Permission Callbacks
	PermissionCallback PermissionCallback

	// Internal Flags (not exposed to users)
	IsStreaming bool
}

// HookCallback is a function that executes during lifecycle hooks.
// Hooks receive input data and return result data or an error.
type HookCallback func(input map[string]any) (map[string]any, error)

// PermissionCallback is a function that checks tool permissions.
// Permission callbacks receive tool name and input, and return a decision.
type PermissionCallback func(
	toolName string,
	input map[string]any,
) (allowed bool, reason string, err error)

// DefaultOptions returns sensible default options.
func DefaultOptions() *AgentOptions {
	return &AgentOptions{
		Model:               "claude-sonnet-4-5",
		PermissionMode:      PermissionModeDefault,
		AllowedTools:        nil, // nil means all tools allowed
		OnlyWriteableIfSafe: true,
		Resume:              false,
		IsStreaming:         false,
		Environment:         make(map[string]string),
		HookCallbacks:       make(map[string]HookCallback),
		MCPServers:          make([]MCPServerConfig, 0),
	}
}
