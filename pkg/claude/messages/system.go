// System message types for Claude Agent.
package messages

// SystemMessage represents a system-level event or initialization.
//
// System messages are sent by Claude to communicate state changes,
// initialization parameters, and other system-level information.
// The Subtype field determines how to interpret the Data field.
type SystemMessage struct {
	Subtype string         `json:"subtype"`
	Data    map[string]any `json:"data"` // Parse into SystemMessageData
}

func (SystemMessage) message() {}

// SystemMessageData is a discriminated union for SystemMessage.Data.
//
// Parse from map[string]any based on the Subtype field.
// Subtypes: "init", "compact_boundary"
type SystemMessageData interface {
	systemMessageData()
}

// SystemMessageInit is sent at the start of a session.
//
// Contains initialization parameters including available tools,
// MCP servers, permission mode, and other session configuration.
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

// SystemMessageCompactBoundary marks a conversation compaction point.
//
// Indicates that the conversation history has been compacted
// (summarized or truncated) to stay within token limits.
type SystemMessageCompactBoundary struct {
	CompactMetadata struct {
		Trigger   string `json:"trigger"` // "manual" | "auto"
		PreTokens int    `json:"pre_tokens"`
	} `json:"compact_metadata"`
}

func (SystemMessageCompactBoundary) systemMessageData() {}

// MCPServerStatus represents the status of an MCP server.
type MCPServerStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "connected" | "failed" | etc.
	ServerInfo *struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo,omitempty"`
}
