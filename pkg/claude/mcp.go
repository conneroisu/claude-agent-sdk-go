package claude

// This file defines the core abstractions for MCP servers and tools, along with SDK
// implementations that allow users to create custom MCP servers in Go.

import (
	"context"
	"errors"
	"sync"
)

// McpServer represents an MCP server instance.
type McpServer interface {
	// Name returns the server name.
	Name() string
	// Version returns the server version.
	Version() string
	// Tools returns the tools provided by this server.
	Tools() []McpTool
	// Start starts the MCP server.
	Start(ctx context.Context) error
	// Stop stops the MCP server.
	Stop(ctx context.Context) error
}

// McpTool represents a tool provided by an MCP server.
type McpTool interface {
	// Name returns the tool name.
	Name() string
	// Description returns the tool description.
	Description() string
	// InputSchema returns the JSON schema for tool inputs.
	InputSchema() map[string]any
	// Execute executes the tool with the given input.
	Execute(ctx context.Context, input map[string]any) (*McpToolResult, error)
}

// McpToolResult represents the result of tool execution.
type McpToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ToolFunc is the handler function for SDK MCP tools.
type ToolFunc func(ctx context.Context, args map[string]any) (*McpToolResult, error)

// sdkMcpTool implements McpTool.
type sdkMcpTool struct {
	name        string
	description string
	inputSchema map[string]any
	handler     ToolFunc
}

func (t *sdkMcpTool) Name() string                { return t.name }
func (t *sdkMcpTool) Description() string         { return t.description }
func (t *sdkMcpTool) InputSchema() map[string]any { return t.inputSchema }

func (t *sdkMcpTool) Execute(ctx context.Context, input map[string]any) (*McpToolResult, error) {
	return t.handler(ctx, input)
}

// Tool creates a new SDK MCP tool.
func Tool(name, description string, inputSchema map[string]any, handler ToolFunc) McpTool {
	return &sdkMcpTool{
		name:        name,
		description: description,
		inputSchema: inputSchema,
		handler:     handler,
	}
}

// sdkMcpServer implements McpServer.
type sdkMcpServer struct {
	name    string
	version string
	tools   []McpTool
	running bool
	mu      sync.Mutex
}

func (s *sdkMcpServer) Name() string     { return s.name }
func (s *sdkMcpServer) Version() string  { return s.version }
func (s *sdkMcpServer) Tools() []McpTool { return s.tools }

func (s *sdkMcpServer) Start(_ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return errors.New("server already running")
	}

	s.running = true

	return nil
}

func (s *sdkMcpServer) Stop(_ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	return nil
}

// CreateSdkMcpServer creates an SDK MCP server.
func CreateSdkMcpServer(name, version string, tools []McpTool) McpServerConfig {
	server := &sdkMcpServer{
		name:    name,
		version: version,
		tools:   tools,
	}

	return McpSdkServerConfig{
		Type:     "sdk",
		Name:     name,
		Instance: server,
	}
}
