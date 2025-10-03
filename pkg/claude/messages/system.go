package messages

// SystemMessage represents system-level messages from the
// Claude CLI, such as initialization or compaction boundaries.
//
// Data is intentionally kept as map[string]any to maintain
// flexibility. Users can parse it into specific
// SystemMessageData types based on Subtype.
type SystemMessage struct {
	// Subtype identifies the message variant
	// (e.g., "init", "compact_boundary")
	Subtype string `json:"subtype"`

	// Data contains the message payload as a flexible map.
	// Parse into SystemMessageInit or
	// SystemMessageCompactBoundary based on Subtype.
	Data map[string]any `json:"data"`
}

// message implements the Message interface.
func (SystemMessage) message() {}

// SystemMessageInit is sent at the start of a session.
// Parse from SystemMessage.Data when Subtype is "init".
type SystemMessageInit struct {
	// Agents lists available subagent names
	Agents []string `json:"agents,omitempty"`

	// APIKeySource indicates where API key comes from
	APIKeySource string `json:"apiKeySource"`

	// Cwd is the current working directory
	Cwd string `json:"cwd"`

	// Tools lists available tool names
	Tools []string `json:"tools"`

	// MCPServers lists MCP server status information
	MCPServers []MCPServerStatus `json:"mcp_servers"`

	// Model identifies the Claude model being used
	Model string `json:"model"`

	// PermissionMode indicates current permission settings
	PermissionMode string `json:"permissionMode"`

	// SlashCommands lists available slash command names
	SlashCommands []string `json:"slash_commands"`

	// OutputStyle indicates the CLI output format
	OutputStyle string `json:"output_style"`
}

// systemMessageData implements the SystemMessageData interface.
func (SystemMessageInit) systemMessageData() {}

// SystemMessageCompactBoundary marks a conversation compaction
// point. Parse from SystemMessage.Data when Subtype is
// "compact_boundary".
type SystemMessageCompactBoundary struct {
	// CompactMetadata contains compaction details
//nolint:revive // nested-structs: compact metadata structure
	CompactMetadata struct {
		// Trigger indicates why compaction occurred
		// ("manual" or "auto")
		Trigger string `json:"trigger"`

		// PreTokens is token count before compaction
		PreTokens int `json:"pre_tokens"`
	} `json:"compact_metadata"`
}

// systemMessageData implements the SystemMessageData interface.
func (SystemMessageCompactBoundary) systemMessageData() {}

// MCPServerStatus represents the connection status of an
// MCP server.
type MCPServerStatus struct {
	// Name is the MCP server identifier
	Name string `json:"name"`

	// Status indicates connection state
	// ("connected", "failed", "needs-auth", "pending")
	Status string `json:"status"`

	ServerInfo *struct { //nolint:revive // nested-structs: MCP spec requirement
		// Name is the server's reported name
		Name string `json:"name"`

		// Version is the server's version string
		Version string `json:"version"`
	} `json:"serverInfo,omitempty"`
}
