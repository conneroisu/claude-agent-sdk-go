package ports

import "context"

// ProtocolHandler abstracts control protocol operations.
// This port defines what the domain needs for bidirectional control
// communication, without coupling to JSON-RPC implementation details.
//
// Request/Response Format: All requests and responses use map[string]any
// to remain agnostic to specific control protocol wire formats. Adapters
// handle marshaling/unmarshaling.
//
// Timeout Behavior: All control requests have a 60-second timeout
// enforced by the adapter, not by domain services. The context provided
// by callers is used for cancellation signals only.
type ProtocolHandler interface {
	// Initialize sends the initialize control request.
	// This must be called before any other operations to configure hooks,
	// MCP servers, and other session parameters.
	// Returns the initialize response or an error.
	Initialize(ctx context.Context, config map[string]any) (
		map[string]any,
		error,
	)

	// SendControlRequest sends a control request and waits for response.
	// The request map should contain the control request structure.
	// Returns the response map or an error if timeout (60s) or failure.
	SendControlRequest(ctx context.Context, req map[string]any) (
		map[string]any,
		error,
	)

	// HandleControlRequest processes inbound control requests.
	// Dependencies are bundled in ControlDependencies to avoid
	// circular imports and simplify parameter passing.
	// Returns the response map or an error.
	HandleControlRequest(
		ctx context.Context,
		req map[string]any,
		deps ControlDependencies,
	) (map[string]any, error)

	// StartMessageRouter partitions transport messages.
	// This method consumes from transport and writes to the provided channels.
	// Regular messages go to msgCh, errors go to errCh.
	// Dependencies are bundled in ControlDependencies.
	// Should be run in a goroutine.
	// Returns an error if routing fails fatally.
	StartMessageRouter(
		ctx context.Context,
		msgCh chan<- map[string]any,
		errCh chan<- error,
		deps ControlDependencies,
	) error
}

// PermissionsService checks tool permissions.
// This is passed to protocol handler to avoid circular dependencies.
type PermissionsService interface {
	// CanUseTool checks if a tool use is permitted.
	CanUseTool(
		ctx context.Context,
		toolName string,
		input map[string]any,
	) (allowed bool, reason string, err error)
}

// HookCallback is a function that executes during lifecycle hooks.
type HookCallback func(input map[string]any) (map[string]any, error)

// ControlDependencies bundles control protocol dependencies.
// This struct is passed to protocol handlers to provide access to
// hooks, permissions, and MCP servers without circular imports.
type ControlDependencies struct {
	Hooks      map[string]HookCallback
	Perms      PermissionsService
	MCPServers map[string]MCPServer
}
