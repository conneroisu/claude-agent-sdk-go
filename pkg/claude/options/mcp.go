package options

import mcpserver "github.com/mark3labs/mcp-go/server"

// MCPServerConfig is the interface for all MCP server configurations.
// Different configuration types support external servers (stdio, HTTP, SSE)
// and in-process SDK servers.
type MCPServerConfig interface {
	mcpServerConfig()
	// GetName returns the server identifier for routing.
	GetName() string
}

// StdioServerConfig configures an external MCP server via subprocess.
// The server communicates over stdin/stdout using the stdio transport.
type StdioServerConfig struct {
	Type    string   // Always "stdio"
	Name    string   // Server identifier
	Command string   // Executable path
	Args    []string // Command arguments
	Env     map[string]string
}

func (*StdioServerConfig) mcpServerConfig()  {}
func (c *StdioServerConfig) GetName() string { return c.Name }

// SSEServerConfig configures an external MCP server via Server-Sent Events.
// The server communicates over HTTP using SSE for streaming.
type SSEServerConfig struct {
	Type string // Always "sse"
	Name string // Server identifier
	URL  string // SSE endpoint URL
}

func (*SSEServerConfig) mcpServerConfig()  {}
func (c *SSEServerConfig) GetName() string { return c.Name }

// HTTPServerConfig configures an external MCP server via HTTP.
// The server communicates using standard HTTP requests.
type HTTPServerConfig struct {
	Type string // Always "http"
	Name string // Server identifier
	URL  string // HTTP endpoint URL
}

func (*HTTPServerConfig) mcpServerConfig()  {}
func (c *HTTPServerConfig) GetName() string { return c.Name }

// SDKServerConfig configures an in-process MCP server.
// The server runs in the same process as the application and uses
// direct method invocation instead of IPC.
type SDKServerConfig struct {
	Type     string               // Always "sdk"
	Name     string               // Server identifier
	Instance *mcpserver.MCPServer // The user's MCP server instance
}

func (*SDKServerConfig) mcpServerConfig()  {}
func (c *SDKServerConfig) GetName() string { return c.Name }
