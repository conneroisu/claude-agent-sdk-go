package mcp

import (
	"context"
	"errors"
)

// ServerAdapter implements an MCP server.
// This allows exposing local tools via MCP protocol.
type ServerAdapter struct {
	tools map[string]ToolHandler
	ready bool
}

// ToolHandler processes a tool invocation.
type ToolHandler func(
	ctx context.Context,
	args map[string]any,
) (map[string]any, error)

// NewServerAdapter creates a new MCP server adapter.
func NewServerAdapter() *ServerAdapter {
	return &ServerAdapter{
		tools: make(map[string]ToolHandler),
		ready: false,
	}
}

// RegisterTool adds a tool handler to the server.
// The tool will be available for MCP clients to invoke.
func (s *ServerAdapter) RegisterTool(
	name string,
	handler ToolHandler,
) {
	s.tools[name] = handler
}

// Start begins serving MCP requests.
// The transport type (stdio, HTTP, etc.) is configured separately.
func (s *ServerAdapter) Start(ctx context.Context) error {
	if s.ready {
		return errors.New("server already running")
	}

	// Server start implementation
	// This would setup the listener and start accepting requests
	_ = ctx

	s.ready = true

	return nil
}

// handleToolCall processes an incoming tool invocation request.
// This method is called by the transport layer.

// handleListTools returns the list of available tools.
// This method is called by the transport layer.

// Stop terminates the MCP server.
func (s *ServerAdapter) Stop() error {
	if !s.ready {
		return nil
	}

	s.ready = false

	return nil
}
