// Package mcp provides adapters for SDK-managed MCP servers.
package mcp

import (
	"context"
	"errors"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// MessageHandler defines the interface that SDK MCP server instances
// must implement. This matches the expected interface from
// github.com/modelcontextprotocol/go-sdk.
type MessageHandler interface {
	// HandleMessage processes a raw JSON-RPC message and returns
	// a raw JSON-RPC response.
	HandleMessage(ctx context.Context, message []byte) ([]byte, error)
}

// Adapter wraps an SDK-managed MCP server and implements
// ports.MCPServer. It proxies messages between the Claude CLI and
// the user's MCP server instance.
type Adapter struct {
	name    string
	handler MessageHandler
}

// NewAdapter creates a new MCP server adapter.
// The instance must implement the MessageHandler interface.
func NewAdapter(name string, instance any) (ports.MCPServer, error) {
	handler, ok := instance.(MessageHandler)
	if !ok {
		return nil, errors.New(
			"MCP server instance must implement " +
				"HandleMessage(context.Context, []byte) " +
				"([]byte, error)",
		)
	}

	return &Adapter{
		name:    name,
		handler: handler,
	}, nil
}

// Name returns the identifier for this MCP server.
func (a *Adapter) Name() string {
	return a.name
}

// HandleMessage proxies a raw JSON-RPC message to the wrapped
// MCP server.
func (a *Adapter) HandleMessage(
	ctx context.Context,
	message []byte,
) ([]byte, error) {
	return a.handler.HandleMessage(ctx, message)
}
