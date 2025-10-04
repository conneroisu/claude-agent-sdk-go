package options

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServerConfig represents configuration for MCP servers.
// This is a discriminated union with two usage patterns:
//
// 1. Client Configs (Stdio, SSE, HTTP): SDK connects TO external servers
// 2. SDK Config: SDK wraps user's in-process server to EXPOSE to CLI
//
// The domain uses the same interface for both patterns.
type MCPServerConfig interface {
	mcpServerConfig()
}

// === CLIENT MCP SERVERS (SDK connects TO them) ===

// StdioServerConfig configures connection via stdio transport.
// The SDK spawns the specified command and communicates via
// stdin/stdout using JSON-RPC.
type StdioServerConfig struct {
	// Type is the transport type ("stdio"), optional for compatibility
	Type string
	// Command is the executable path or name to spawn
	Command string
	// Args are command-line arguments to pass to the command
	Args []string
	// Env provides additional environment variables for the server process
	Env map[string]string
}

func (StdioServerConfig) mcpServerConfig() {}

// SSEServerConfig configures connection via Server-Sent Events.
// The SDK establishes an HTTP connection and receives SSE streams.
type SSEServerConfig struct {
	// Type is the transport type ("sse")
	Type string
	// URL is the SSE endpoint URL to connect to
	URL string
	// Headers are optional HTTP headers to include in the connection request
	Headers map[string]string
}

func (SSEServerConfig) mcpServerConfig() {}

// HTTPServerConfig configures connection to an external MCP server via HTTP.
// The SDK sends JSON-RPC messages as HTTP POST requests and receives responses.
type HTTPServerConfig struct {
	// Type is the transport type ("http")
	Type string
	// URL is the HTTP endpoint URL to send requests to
	URL string
	// Headers are optional HTTP headers to include in requests
	Headers map[string]string
}

func (HTTPServerConfig) mcpServerConfig() {}

// === SDK MCP SERVERS (User creates, SDK wraps and exposes to CLI) ===

// SDKServerConfig wraps a user-created in-process MCP server instance.
// The user creates an *mcp.Server using the go-sdk, and the SDK wraps it
// to expose it to Claude CLI via the control protocol.
//
// This enables users to build custom MCP servers in Go and use them directly
// with Claude without spawning separate processes.
type SDKServerConfig struct {
	// Type is the transport type ("sdk")
	Type string
	// Name is the server identifier used for routing control protocol messages
	Name string
	// Instance is the user's MCP server created with mcp.NewServer()
	// The SDK wraps this server and handles message routing
	Instance *mcpsdk.Server
}

func (SDKServerConfig) mcpServerConfig() {}
