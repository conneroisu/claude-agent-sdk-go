package messages

// SystemMessage represents system-level status and initialization messages.
// System messages use a discriminated union pattern via the Subtype field.
type SystemMessage struct {
	// Subtype determines which SystemMessageData variant this contains
	Subtype string `json:"subtype"`

	// Data is flexible - parse into SystemMessageData based on Subtype
	Data map[string]any `json:"data"`
}

func (SystemMessage) message() {}

// SystemMessageInit is sent at the start of a session.
// It provides information about the current environment and configuration.
type SystemMessageInit struct {
	// Agents lists available subagent names
	Agents []string `json:"agents,omitempty"`

	// APIKeySource indicates where the API key comes from
	APIKeySource string `json:"apiKeySource"`

	// Cwd is the current working directory
	Cwd string `json:"cwd"`

	// Tools lists available tool names
	Tools []string `json:"tools"`

	// MCPServers shows status of configured MCP servers
	MCPServers []MCPServerStatus `json:"mcp_servers"`

	// Model is the currently selected AI model
	Model string `json:"model"`

	// PermissionMode indicates current permission handling mode
	PermissionMode string `json:"permissionMode"`

	// SlashCommands lists available slash commands
	SlashCommands []string `json:"slash_commands"`

	// OutputStyle indicates the output formatting style
	OutputStyle string `json:"output_style"`
}

func (SystemMessageInit) systemMessageData() {}

// SystemMessageCompactBoundary marks a conversation compaction point.
// This indicates where the conversation history was compacted
// to reduce token usage.
type SystemMessageCompactBoundary struct {
	CompactMetadata CompactMetadata `json:"compact_metadata"`
}

func (SystemMessageCompactBoundary) systemMessageData() {}

// CompactMetadata contains information about conversation compaction.
type CompactMetadata struct {
	// Trigger indicates what caused the compaction
	Trigger string `json:"trigger"` // "manual" | "auto"

	// PreTokens is the token count before compaction
	PreTokens int `json:"pre_tokens"`
}

// MCPServerStatus represents the status of an MCP server.
type MCPServerStatus struct {
	// Name is the server identifier
	Name string `json:"name"`

	// Status indicates connection state
	// Values: "connected", "failed", "needs-auth", "pending"
	Status string `json:"status"` //nolint:lll

	// ServerInfo contains server metadata (if connected)
	ServerInfo *MCPServerInfo `json:"serverInfo,omitempty"`
}

// MCPServerInfo contains metadata about an MCP server.
type MCPServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
