// MCP server configuration for Claude Agent.
package options

// MCPServerConfig is configuration for MCP servers.
//
// These are infrastructure configurations for connecting to MCP servers.
// Not to be confused with runtime MCP server instances (see ports.MCPServer).
type MCPServerConfig interface {
	mcpServerConfig()
}

// StdioServerConfig configures an MCP server using stdio transport.
//
// The CLI will spawn the specified command as a subprocess and
// communicate via stdin/stdout.
type StdioServerConfig struct {
	Type    string // "stdio"
	Command string
	Args    []string
	Env     map[string]string
}

func (StdioServerConfig) mcpServerConfig() {}

// SSEServerConfig configures an MCP server using Server-Sent Events.
//
// The CLI will connect to the specified URL and receive events
// via SSE (one-way communication from server to client).
type SSEServerConfig struct {
	Type    string // "sse"
	URL     string
	Headers map[string]string
}

func (SSEServerConfig) mcpServerConfig() {}

// HTTPServerConfig configures an MCP server using HTTP transport.
//
// The CLI will make HTTP requests to the specified URL.
type HTTPServerConfig struct {
	Type    string // "http"
	URL     string
	Headers map[string]string
}

func (HTTPServerConfig) mcpServerConfig() {}

// SDKServerConfig is a marker for SDK-managed MCP servers.
//
// This ONLY contains configuration, NOT the server instance itself.
// The actual server instance is managed separately by the MCP adapter
// to avoid circular dependencies.
type SDKServerConfig struct {
	Type string // "sdk"
	Name string
	// Instance is NOT stored here - managed by MCP adapter
}

func (SDKServerConfig) mcpServerConfig() {}
