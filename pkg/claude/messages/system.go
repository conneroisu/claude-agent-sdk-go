package messages

// SystemMessage represents system notifications and state changes.
//
// System messages communicate session initialization, compaction boundaries,
// and other system-level events. The Data field contains subtype-specific
// information as a flexible map that can be parsed into typed structures.
//
// Example:
//
//	msg := SystemMessage{
//	    Subtype: "init",
//	    Data: map[string]any{
//	        "model": "claude-sonnet-4-20250514",
//	        "cwd": "/home/user/project",
//	    },
//	}
type SystemMessage struct {
	// Subtype identifies the type of system message.
	// Known values: "init", "compact_boundary"
	Subtype string `json:"subtype"`

	// Data contains subtype-specific information.
	// Parse into SystemMessageData types based on Subtype.
	Data map[string]any `json:"data"`
}

// message implements the Message interface.
func (SystemMessage) message() {}

// SystemMessageData is a discriminated union for SystemMessage.Data.
//
// Parse the Data map into one of these concrete types based on the
// SystemMessage.Subtype field.
type SystemMessageData interface {
	systemMessageData()
}

// SystemMessageInit is sent at the start of a session.
//
// Contains initialization information about the Claude environment,
// including available tools, MCP servers, model, and configuration.
//
// Example:
//
//	init := SystemMessageInit{
//	    Model: "claude-sonnet-4-20250514",
//	    Cwd: "/home/user/project",
//	    Tools: []string{"Bash", "Read", "Write"},
//	}
type SystemMessageInit struct {
	// Agents lists available sub-agent names.
	Agents []string `json:"agents,omitempty"`

	// APIKeySource indicates where the API key came from.
	// Example: "environment", "config"
	APIKeySource string `json:"apiKeySource"`

	// Cwd is the current working directory for the session.
	Cwd string `json:"cwd"`

	// Tools lists available built-in tool names.
	Tools []string `json:"tools"`

	// MCPServers lists connected MCP servers and their status.
	MCPServers []MCPServerStatus `json:"mcp_servers"`

	// Model identifies which Claude model is being used.
	Model string `json:"model"`

	// PermissionMode indicates the permission handling mode.
	// Values: "default", "acceptEdits", "plan", "bypassPermissions", "ask"
	PermissionMode string `json:"permissionMode"`

	// SlashCommands lists available slash commands.
	SlashCommands []string `json:"slash_commands"`

	// OutputStyle indicates the output formatting style.
	OutputStyle string `json:"output_style"`
}

// systemMessageData implements the SystemMessageData interface.
func (SystemMessageInit) systemMessageData() {}

// CompactMetadata contains information about a compaction event.
type CompactMetadata struct {
	// Trigger indicates what triggered the compaction.
	// Values: "manual", "auto"
	Trigger string `json:"trigger"`

	// PreTokens is the token count before compaction.
	PreTokens int `json:"pre_tokens"`
}

// SystemMessageCompactBoundary marks a conversation compaction point.
//
// Sent when Claude compacts the conversation history to stay within
// context limits. Contains metadata about the compaction trigger and
// token counts before compaction.
//
// Example:
//
//	boundary := SystemMessageCompactBoundary{
//	    CompactMetadata: CompactMetadata{
//	        Trigger: "auto",
//	        PreTokens: 50000,
//	    },
//	}
type SystemMessageCompactBoundary struct {
	// CompactMetadata contains information about the compaction event.
	CompactMetadata CompactMetadata `json:"compact_metadata"`
}

// systemMessageData implements the SystemMessageData interface.
func (SystemMessageCompactBoundary) systemMessageData() {}

// MCPServerInfo contains metadata about an MCP server.
type MCPServerInfo struct {
	// Name is the server's self-reported name.
	Name string `json:"name"`

	// Version is the server's version string.
	Version string `json:"version"`
}

// MCPServerStatus represents the status of an MCP server.
//
// Contains the server name, connection status, and optional server
// information (name and version) when successfully connected.
//
// Example:
//
//	status := MCPServerStatus{
//	    Name: "filesystem",
//	    Status: "connected",
//	    ServerInfo: &MCPServerInfo{
//	        Name: "filesystem-server",
//	        Version: "1.0.0",
//	    },
//	}
type MCPServerStatus struct {
	// Name is the configured name for this MCP server.
	Name string `json:"name"`

	// Status indicates the connection state.
	// Values: "connected", "failed", "needs-auth", "pending"
	Status string `json:"status"`

	// ServerInfo contains server metadata when connected.
	ServerInfo *MCPServerInfo `json:"serverInfo,omitempty"`
}
