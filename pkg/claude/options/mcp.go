package options

// MCPServerConfig is configuration for MCP servers
type MCPServerConfig interface {
	mcpServerConfig()
}

// StdioServerConfig configures an MCP server using stdio transport
type StdioServerConfig struct {
	Type    string // "stdio"
	Command string
	Args    []string
	Env     map[string]string
}

func (StdioServerConfig) mcpServerConfig() {}

// SSEServerConfig configures an MCP server using Server-Sent Events
type SSEServerConfig struct {
	Type    string // "sse"
	URL     string
	Headers map[string]string
}

func (SSEServerConfig) mcpServerConfig() {}

// HTTPServerConfig configures an MCP server using HTTP transport
type HTTPServerConfig struct {
	Type    string // "http"
	URL     string
	Headers map[string]string
}

func (HTTPServerConfig) mcpServerConfig() {}

// SDKServerConfig is a marker for SDK-managed MCP servers
type SDKServerConfig struct {
	Type string // "sdk"
	Name string
}

func (SDKServerConfig) mcpServerConfig() {}
