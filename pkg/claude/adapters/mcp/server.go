package mcp

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/ports"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerAdapter wraps a user-provided in-process MCP server.
// This adapter implements ports.MCPServer for *mcp.Server instances
// that run in the same process as the SDK.
type ServerAdapter struct {
	name   string
	server *mcp.Server
}

// Verify interface compliance at compile time.
var _ ports.MCPServer = (*ServerAdapter)(nil)

// NewServerAdapter creates a new MCP server adapter.
// The server should be created by the user using mcp.NewServer().
func NewServerAdapter(name string, server *mcp.Server) *ServerAdapter {
	return &ServerAdapter{
		name:   name,
		server: server,
	}
}

// Name returns the server identifier.
func (a *ServerAdapter) Name() string {
	return a.name
}

// HandleMessage routes a JSON-RPC message to the in-process MCP server.
// For SDK servers, this is handled by the server's built-in request handling.
func (a *ServerAdapter) HandleMessage(_ context.Context, _ []byte) ([]byte, error) {
	// SDK servers are handled differently - the control protocol
	// routes messages directly to the server's handlers via
	// the in-memory transport connection.
	// This method exists to satisfy the interface but isn't
	// called in practice for SDK servers.
	return nil, nil
}

// Close is a no-op for SDK servers.
// The server lifecycle is managed by the user, not the adapter.
func (_ *ServerAdapter) Close() error {
	return nil
}
