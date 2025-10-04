package ports

import "context"

// ProtocolHandler defines what the domain needs for control protocol.
// This handles JSON-RPC bidirectional communication with Claude CLI.
type ProtocolHandler interface {
	// Initialize sends the initialize control request with hooks config
	Initialize(
		ctx context.Context,
		config any,
	) (map[string]any, error)

	// SendControlRequest sends a control request and waits for response.
	// Includes 60-second timeout protection.
	SendControlRequest(
		ctx context.Context,
		req map[string]any,
	) (map[string]any, error)

	// HandleControlRequest routes inbound control requests by subtype.
	// Subtypes: can_use_tool, hook_callback, mcp_message
	// Dependencies are passed as arguments to avoid circular references.
	HandleControlRequest(
		ctx context.Context,
		req map[string]any,
		deps ControlDependencies,
	) (map[string]any, error)

	// StartMessageRouter continuously reads transport and partitions messages.
	// Routes control_response, control_request separately from SDK messages.
	StartMessageRouter(
		ctx context.Context,
		msgCh chan<- map[string]any,
		errCh chan<- error,
		deps ControlDependencies,
	) error
}

// ControlDependencies contains dependencies needed by protocol handler.
// This struct avoids circular imports and keeps the port interface clean.
type ControlDependencies struct {
	// PermissionsService handles permission checks
	PermissionsService any // *permissions.Service

	// HookCallbacks maps callback IDs to hook functions
	HookCallbacks any // map[string]hooking.HookCallback

	// MCPServers maps server names to MCP server adapters
	MCPServers map[string]MCPServer
}
