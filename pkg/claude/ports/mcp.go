// Package ports defines interfaces for external integrations and services.
package ports

import "context"

// MCPServer defines an interface for an in-process MCP Server.
// It abstracts the underlying implementation, which should be a wrapper around
// the official MCP Go SDK (github.com/modelcontextprotocol/go-sdk).
// This allows the agent to route raw MCP messages from the Claude CLI
// to a user-defined tool server.
type MCPServer interface {
	// Name returns the identifier for this MCP server.
	Name() string
	// HandleMessage takes a raw JSON-RPC message, processes it, and returns
	// a raw JSON-RPC response. This is used to proxy messages.
	HandleMessage(ctx context.Context, message []byte) ([]byte, error)
}
