package ports

import "context"

// ProtocolHandler defines what the domain needs for control protocol
type ProtocolHandler interface {
	// Initialize sends the initialize control request with hooks
	// config
	Initialize(
		ctx context.Context,
		config map[string]any,
	) (map[string]any, error)

	// SendControlRequest sends a control request and waits for
	// response (60s timeout)
	SendControlRequest(
		ctx context.Context,
		req map[string]any,
	) (map[string]any, error)

	// HandleControlRequest routes inbound control requests by subtype
	// Subtypes: can_use_tool, hook_callback, mcp_message
	// Dependencies are passed as arguments to avoid circular refs
	HandleControlRequest(
		ctx context.Context,
		req map[string]any,
		deps ControlDependencies,
	) (map[string]any, error)

	// StartMessageRouter continuously reads transport and partitions
	// messages
	// Routes control_response, control_request, control_cancel_request
	// separately from SDK messages
	// Dependencies (perms, hooks, mcpServers) are passed by domain
	// service for handling inbound control requests
	StartMessageRouter(
		ctx context.Context,
		msgCh chan<- map[string]any,
		errCh chan<- error,
		deps ControlDependencies,
	) error
}

// ControlDependencies bundles dependencies needed for control
// protocol handling
// This avoids circular dependencies and makes the interface cleaner
type ControlDependencies struct {
	Permissions PermissionChecker
	Hooks       map[string]HookCallback
	MCPServers  map[string]MCPServer
}

// PermissionChecker is a minimal interface for permission checking
// This allows the protocol adapter to check permissions without
// importing the full permissions package
type PermissionChecker interface {
	// CheckToolUse checks if a tool can be used with given parameters
	CheckToolUse(
		ctx context.Context,
		toolName string,
		input map[string]any,
		suggestions []any,
	) (any, error)
}

// HookCallback is a function that handles hook events
type HookCallback func(
	input map[string]any,
	toolUseID *string,
	ctx any,
) (map[string]any, error)
