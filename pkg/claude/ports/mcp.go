package ports

import "context"

// MCPServer defines what the domain needs from an MCP server adapter.
// This port has TWO implementations:
//  1. ClientAdapter: Wraps an MCP client that connects TO external servers
//  2. ServerAdapter: Wraps a user's in-process *mcp.Server to expose
//     TO Claude CLI
//
// The domain doesn't care which type - it just routes JSON-RPC messages.
type MCPServer interface {
	// Name returns the server identifier.
	// Used for routing control protocol messages.
	Name() string

	// HandleMessage routes a raw JSON-RPC message and returns the response.
	// For ClientAdapter: Forwards to external server via MCP client session
	// For ServerAdapter: Routes to in-process server via in-memory transport
	HandleMessage(ctx context.Context, message []byte) ([]byte, error)

	// Close releases resources.
	// For ClientAdapter: Closes the client session connection
	// For ServerAdapter: Closes in-memory transport (server is user-managed)
	Close() error
}
