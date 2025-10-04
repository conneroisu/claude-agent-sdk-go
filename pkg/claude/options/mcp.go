package options

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServerConfig represents configuration for MCP servers.
// There are two distinct types:
// 1. Client Configs (Stdio, SSE, HTTP): SDK connects TO external servers
// 2. SDK Config: SDK wraps user's in-process server to EXPOSE to Claude CLI.
type MCPServerConfig interface {
	mcpServerConfig()
}

// StdioServerConfig configures connection to external MCP server via stdio.
// The SDK will start the command and communicate over stdin/stdout.
type StdioServerConfig struct {
	// Type is "stdio" (optional for backwards compatibility)
	Type string

	// Command is the executable to run
	Command string

	// Args are command-line arguments
	Args []string

	// Env sets environment variables for the subprocess
	Env map[string]string
}

func (StdioServerConfig) mcpServerConfig() {}

// SSEServerConfig configures connection to external MCP server via SSE.
// The SDK will connect to the URL using Server-Sent Events transport.
type SSEServerConfig struct {
	// Type is "sse"
	Type string

	// URL is the SSE endpoint
	URL string

	// Headers are HTTP headers to send
	Headers map[string]string
}

func (SSEServerConfig) mcpServerConfig() {}

// HTTPServerConfig configures connection to external MCP server via HTTP.
// The SDK will make HTTP requests to the URL.
type HTTPServerConfig struct {
	// Type is "http"
	Type string

	// URL is the HTTP endpoint
	URL string

	// Headers are HTTP headers to send
	Headers map[string]string
}

func (HTTPServerConfig) mcpServerConfig() {}

// SDKServerConfig wraps a user-created in-process MCP server.
// The Instance field contains the actual *mcp.Server created by the user.
// The SDK wraps this server and exposes it to Claude CLI via control protocol.
type SDKServerConfig struct {
	// Type is "sdk"
	Type string

	// Name is the server identifier
	Name string

	// Instance is the user's MCP server (created with mcp.NewServer)
	Instance *mcpsdk.Server
}

func (SDKServerConfig) mcpServerConfig() {}
