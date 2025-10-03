// Protocol handler port definition.
package ports

import "context"

// ProtocolHandler defines what the domain needs for control protocol.
//
// The control protocol handles JSON-RPC style request/response messages
// for initialization, permissions, hooks, and MCP message routing.
type ProtocolHandler interface {
	// Initialize sends the initialize control request with hooks config.
	Initialize(ctx context.Context, cfg any) (map[string]any, error)

	// SendControlRequest sends a control request and waits for response.
	// Has a 60-second timeout for responses.
	SendControlRequest(
		ctx context.Context,
		req map[string]any,
	) (map[string]any, error)

	// HandleControlRequest routes inbound control requests by subtype.
	//
	// Subtypes:
	//   - can_use_tool: Permission check
	//   - hook_callback: Lifecycle hook execution
	//   - mcp_message: MCP server message proxying
	//
	// Dependencies are passed as arguments to avoid circular imports.
	HandleControlRequest(
		ctx context.Context,
		req map[string]any,
		perms PermissionService,
		hooks map[string]HookCallback,
		mcpServers map[string]MCPServer,
	) (map[string]any, error)

	// StartMessageRouter continuously reads transport and partitions msgs.
	//
	// Routes control_response, control_request, control_cancel_request
	// separately from SDK messages (user, assistant, system, result, etc.).
	//
	// Dependencies (perms, hooks, mcpServers) are passed for handling
	// inbound control requests.
	StartMessageRouter(
		ctx context.Context,
		msgCh chan<- map[string]any,
		errCh chan<- error,
		perms PermissionService,
		hooks map[string]HookCallback,
		mcpServers map[string]MCPServer,
	) error
}
