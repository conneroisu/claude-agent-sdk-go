package options

// AgentOptions configures the Claude agent
type AgentOptions struct {
	// Domain settings (affect business logic)
	AllowedTools             []string
	DisallowedTools          []string
	Model                    *string
	MaxTurns                 *int
	SystemPrompt             SystemPromptConfig
	PermissionMode           *PermissionMode
	PermissionPromptToolName *string
	Agents                   map[string]AgentDefinition

	// Session management (domain concern)
	ContinueConversation   bool
	Resume                 *string
	ForkSession            bool
	IncludePartialMessages bool

	// Infrastructure settings (how to connect/execute)
	Cwd            *string
	Settings       *string
	AddDirs        []string
	Env            map[string]string
	User           *string
	SettingSources []SettingSource
	MaxBufferSize  *int
	StderrCallback func(string)
	ExtraArgs      map[string]*string

	// MCP server configuration (infrastructure)
	MCPServers map[string]MCPServerConfig

	// Internal flags (set by domain services, not by users)
	IsStreaming bool // Internal: true for Client, false for Query
}
