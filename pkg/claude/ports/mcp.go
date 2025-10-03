// MCP server port definition.
package ports

import "context"

// MCPServer defines an interface for an in-process MCP Server.
//
// It abstracts the underlying implementation, which should be a wrapper
// around the official MCP Go SDK (github.com/modelcontextprotocol/go-sdk).
// This allows the agent to route raw MCP messages from the Claude CLI
// to a user-defined tool server.
type MCPServer interface {
	// Name returns the server name (used for routing).
	Name() string

	// HandleMessage processes a raw JSON-RPC message and returns response.
	// This is used to proxy messages between Claude CLI and the server.
	HandleMessage(ctx context.Context, message []byte) ([]byte, error)

	// Close closes the MCP server connection and releases resources.
	Close() error
}
