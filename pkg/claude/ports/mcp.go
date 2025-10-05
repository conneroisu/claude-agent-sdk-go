package ports

import "context"

// MCPServer abstracts MCP message routing.
// This port has TWO implementations:
// 1. ClientAdapter: Routes messages TO external MCP servers
// 2. ServerAdapter: Routes messages TO user's in-process server
//
// The interface is deliberately implementation-agnostic - it doesn't
// expose whether the server is external or in-process. This allows
// the domain to treat both types uniformly.
//
// Message Format: Messages are raw JSON-RPC bytes. The adapter is
// responsible for parsing, routing, and serializing responses.
//
// Error Semantics:
// - Retryable errors: Connection failures, timeouts
// - Fatal errors: Invalid JSON-RPC, unknown methods.
type MCPServer interface {
	// Name returns the server identifier for routing.
	// This name is used to route mcp_message control requests
	// to the correct server instance.
	Name() string

	// HandleMessage routes a JSON-RPC message to the server.
	// For client adapters: Forwards to external server via transport
	// For server adapters: Manually dispatches to in-process handlers
	// Returns the JSON-RPC response or an error.
	HandleMessage(ctx context.Context, message []byte) ([]byte, error)

	// Close cleans up server resources.
	// For client adapters: Disconnects from external server
	// For server adapters: Performs cleanup (if needed)
	Close() error
}
