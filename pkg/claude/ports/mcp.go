package ports

import "context"

// MCPServer defines what the domain needs from an MCP server adapter.
//
// This port interface abstracts MCP server communication, supporting
// two distinct implementation types:
//
//  1. ClientAdapter: Wraps an MCP client session that connects TO
//     external MCP servers (stdio, SSE, HTTP transports).
//
//  2. ServerAdapter: Wraps a user's in-process *mcp.Server to EXPOSE
//     it TO Claude CLI via the control protocol.
//
// The domain doesn't care which type - it only needs to route JSON-RPC
// messages to the appropriate server and get responses back.
//
// Example usage in control protocol:
//
//	// Route message to the appropriate MCP server
//	server := mcpServers["filesystem"]
//	response, err := server.HandleMessage(ctx, jsonRpcMessage)
type MCPServer interface {
	// Name returns the server identifier.
	//
	// Used for routing control protocol messages to the correct server.
	// The name must be unique within a session.
	//
	// For ClientAdapter: The configured server name from MCPServerConfig.
	// For ServerAdapter: The name from SDKServerConfig.Name.
	//
	// Example:
	//
	//	name := server.Name()
	//	// Returns: "filesystem", "database", etc.
	Name() string

	// HandleMessage routes a raw JSON-RPC message and returns the response.
	//
	// The message parameter contains the complete JSON-RPC request.
	// Returns the JSON-RPC response or an error.
	//
	// For ClientAdapter:
	//   - Forwards the message to external server via MCP client session
	//   - Uses the configured transport (stdio/SSE/HTTP)
	//   - Returns the server's response
	//
	// For ServerAdapter:
	//   - Routes message to user's in-process server via in-memory transport
	//   - Executes the user's tool/resource/prompt handlers
	//   - Returns the server's response
	//
	// Example:
	//
	//	jsonRpc := []byte(`{
	//	    "jsonrpc": "2.0",
	//	    "method": "tools/list",
	//	    "id": 1
	//	}`)
	//	response, err := server.HandleMessage(ctx, jsonRpc)
	//	if err != nil {
	//	    return fmt.Errorf("MCP request failed: %w", err)
	//	}
	HandleMessage(ctx context.Context, message []byte) ([]byte, error)

	// Close releases resources.
	//
	// For ClientAdapter:
	//   - Closes the client session connection to external server
	//   - Terminates the transport (kills process, closes socket, etc.)
	//
	// For ServerAdapter:
	//   - Closes the in-memory transport
	//   - Does NOT close the server itself (user manages lifecycle)
	//
	// Returns an error if cleanup fails.
	//
	// Example:
	//
	//	defer server.Close()
	Close() error
}
