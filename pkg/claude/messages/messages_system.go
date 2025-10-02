package messages

// SystemMessage represents a system message with flexible data.
type SystemMessage struct {
	Subtype string         `json:"subtype"`
	Data    map[string]any `json:"data"` // Flexible - varies by subtype
}

func (SystemMessage) message() {}

// SystemMessageData is a discriminated union for SystemMessage.Data.
type SystemMessageData interface {
	systemMessageData()
}

// SystemMessageInit is sent at the start of a session.
type SystemMessageInit struct {
	Agents         []string          `json:"agents,omitempty"`
	APIKeySource   string            `json:"apiKeySource"`
	Cwd            string            `json:"cwd"`
	Tools          []string          `json:"tools"`
	MCPServers     []MCPServerStatus `json:"mcp_servers"`
	Model          string            `json:"model"`
	PermissionMode string            `json:"permissionMode"`
	SlashCommands  []string          `json:"slash_commands"`
	OutputStyle    string            `json:"output_style"`
}

func (SystemMessageInit) systemMessageData() {}

// CompactMetadata contains metadata about a compaction event.
type CompactMetadata struct {
	Trigger   string `json:"trigger"` // "manual" | "auto"
	PreTokens int    `json:"pre_tokens"`
}

// SystemMessageCompactBoundary marks a conversation compaction point.
type SystemMessageCompactBoundary struct {
	CompactMetadata CompactMetadata `json:"compact_metadata"`
}

func (SystemMessageCompactBoundary) systemMessageData() {}

// ServerInfo contains information about an MCP server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// MCPServerStatus represents the status of an MCP server.
type MCPServerStatus struct {
	Name       string      `json:"name"`
	Status     string      `json:"status"` // "connected" | "failed"...
	ServerInfo *ServerInfo `json:"serverInfo,omitempty"`
}
