package ports

import (
	"context"
)

// ProtocolHandler defines what the domain needs for control protocol
type ProtocolHandler interface {
	// Initialize sends the initialize control request with hooks config
	Initialize(ctx context.Context, config any) (map[string]any, error)

	// SendControlRequest sends a control request and waits for response (60s timeout)
	SendControlRequest(ctx context.Context, req map[string]any) (map[string]any, error)

	// HandleControlRequest routes inbound control requests by subtype
	// Subtypes: can_use_tool, hook_callback, mcp_message
	HandleControlRequest(ctx context.Context, req map[string]any, deps ProtocolDependencies) (map[string]any, error)

	// StartMessageRouter continuously reads transport and partitions messages
	StartMessageRouter(ctx context.Context, msgCh chan<- map[string]any, errCh chan<- error, deps ProtocolDependencies) error
}

// ProtocolDependencies holds dependencies needed for protocol handling
type ProtocolDependencies struct {
	Permissions any                    // *permissions.Service
	Hooks       map[string]any         // map[string]HookCallback
	MCPServers  map[string]MCPServer
}
