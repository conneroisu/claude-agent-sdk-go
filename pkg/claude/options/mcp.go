package options

// MCPServerConfig is configuration for MCP servers.
// These are infrastructure configurations for connecting
// to MCP servers (not runtime instances).
type MCPServerConfig interface {
	mcpServerConfig()
}

// StdioServerConfig configures an MCP server using stdio
// transport.
type StdioServerConfig struct {
	// Type is always "stdio"
	Type string

	// Command is the executable path
	Command string

	// Args are command-line arguments
	Args []string

	// Env sets environment variables
	Env map[string]string
}

// mcpServerConfig implements the MCPServerConfig interface.
func (StdioServerConfig) mcpServerConfig() {}

// SSEServerConfig configures an MCP server using Server-Sent
// Events.
type SSEServerConfig struct {
	// Type is always "sse"
	Type string

	// URL is the SSE endpoint
	URL string

	// Headers are HTTP headers to send
	Headers map[string]string
}

// mcpServerConfig implements the MCPServerConfig interface.
func (SSEServerConfig) mcpServerConfig() {}

// HTTPServerConfig configures an MCP server using HTTP
// transport.
type HTTPServerConfig struct {
	// Type is always "http"
	Type string

	// URL is the HTTP endpoint
	URL string

	// Headers are HTTP headers to send
	Headers map[string]string
}

// mcpServerConfig implements the MCPServerConfig interface.
func (HTTPServerConfig) mcpServerConfig() {}

// SDKServerConfig is a marker for SDK-managed MCP servers.
// The actual server instance is managed separately by the MCP
// adapter. This ONLY contains configuration, NOT the server
// instance itself.
type SDKServerConfig struct {
	// Type is always "sdk"
	Type string

	// Name identifies the server
	Name string
	// Note: Instance is NOT stored here to avoid circular deps.
	// The MCP adapter manages server instances separately.
}

// mcpServerConfig implements the MCPServerConfig interface.
func (SDKServerConfig) mcpServerConfig() {}
