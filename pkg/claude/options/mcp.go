package options

// MCPServerConfig is configuration for MCP servers (not runtime instances)
// These are infrastructure configurations for connecting to MCP servers.
type MCPServerConfig interface {
	mcpServerConfig()
}

// StdioServerConfig configures an MCP server using stdio transport.
type StdioServerConfig struct {
	Type    string // "stdio"
	Command string
	Args    []string
	Env     map[string]string
}

func (StdioServerConfig) mcpServerConfig() {}

// SSEServerConfig configures an MCP server using Server-Sent Events.
type SSEServerConfig struct {
	Type    string // "sse"
	URL     string
	Headers map[string]string
}

func (SSEServerConfig) mcpServerConfig() {}

// HTTPServerConfig configures an MCP server using HTTP transport.
type HTTPServerConfig struct {
	Type    string // "http"
	URL     string
	Headers map[string]string
}

func (HTTPServerConfig) mcpServerConfig() {}

// SDKServerConfig configures an SDK-managed MCP server
// The Instance field holds the actual MCP server implementation.
type SDKServerConfig struct {
	Type string // "sdk"
	Name string
	// Instance holds the MCP server (e.g., *mcp.Server from go-sdk)
	Instance any
}

func (SDKServerConfig) mcpServerConfig() {}
