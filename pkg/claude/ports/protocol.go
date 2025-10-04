package ports

import "context"

// ProtocolHandler defines what the domain needs for control protocol.
//
// This port interface manages bidirectional communication with Claude CLI
// using the control protocol for permissions, hooks, and MCP routing.
//
// The control protocol uses JSON-RPC style requests and responses with
// unique request IDs for matching responses to requests.
//
// Example implementation: JSONRPCProtocolHandler that manages request
// tracking, timeouts, and message routing.
type ProtocolHandler interface {
	// Initialize sends the initialize control request with hooks config.
	//
	// Must be called at session start to register SDK hooks with the CLI.
	// The config parameter contains hook configurations.
	//
	// Returns the initialization response or an error.
	//
	// Example:
	//
	//	config := map[string]any{
	//	    "hooks": map[string]any{
	//	        "beforeToolUse": map[string]any{"enabled": true},
	//	    },
	//	}
	//	resp, err := protocol.Initialize(ctx, config)
	//	if err != nil {
	//	    return fmt.Errorf("initialize failed: %w", err)
	//	}
	Initialize(ctx context.Context, config any) (map[string]any, error)

	// SendControlRequest sends a control request and waits for response.
	//
	// Sends the request with a unique ID and blocks until the matching
	// response arrives or a timeout occurs (60 seconds).
	//
	// The req parameter is the complete control request message.
	// Returns the response payload or an error.
	//
	// Example:
	//
	//	req := map[string]any{
	//	    "type": "control_request",
	//	    "request_id": "req_1_a3f2",
	//	    "request": map[string]any{
	//	        "subtype": "set_model",
	//	        "model": "claude-opus-4-20250514",
	//	    },
	//	}
	//	resp, err := protocol.SendControlRequest(ctx, req)
	SendControlRequest(ctx context.Context, req map[string]any) (map[string]any, error)

	// HandleControlRequest routes inbound control requests by subtype.
	//
	// Called when the CLI sends a control request to the SDK.
	// Routes the request based on subtype to the appropriate handler:
	//   - "can_use_tool": Permissions check
	//   - "hook_callback": Hook execution
	//   - "mcp_message": MCP server routing
	//
	// Dependencies are passed as arguments to avoid circular imports:
	//   - perms: Permission service for can_use_tool requests
	//   - hooks: Hook callbacks for hook_callback requests
	//   - mcpServers: MCP server adapters for mcp_message requests
	//
	// Returns the response to send back to CLI, or an error.
	//
	// Example:
	//
	//	req := map[string]any{
	//	    "request": map[string]any{
	//	        "subtype": "can_use_tool",
	//	        "tool_name": "Bash",
	//	        "input": map[string]any{"command": "ls"},
	//	    },
	//	}
	//	resp, err := protocol.HandleControlRequest(ctx, req, perms, hooks, mcpServers)
	HandleControlRequest(
		ctx context.Context,
		req map[string]any,
		perms PermissionService,
		hooks map[string]HookCallback,
		mcpServers map[string]MCPServer,
	) (map[string]any, error)

	// StartMessageRouter continuously reads transport and partitions messages.
	//
	// Starts a goroutine that reads from the transport and routes messages:
	//   - Control responses: Matched to pending requests
	//   - Control requests: Handled via HandleControlRequest
	//   - Control cancel: Cancels pending requests
	//   - SDK messages: Forwarded to msgCh for domain processing
	//
	// Dependencies (perms, hooks, mcpServers) are passed for handling
	// inbound control requests from the CLI.
	//
	// Messages are sent to msgCh, errors to errCh.
	// Runs until context is cancelled or transport closes.
	//
	// Example:
	//
	//	msgCh := make(chan map[string]any)
	//	errCh := make(chan error)
	//	err := protocol.StartMessageRouter(ctx, msgCh, errCh, perms, hooks, mcpServers)
	//	if err != nil {
	//	    return fmt.Errorf("router start failed: %w", err)
	//	}
	StartMessageRouter(
		ctx context.Context,
		msgCh chan<- map[string]any,
		errCh chan<- error,
		perms PermissionService,
		hooks map[string]HookCallback,
		mcpServers map[string]MCPServer,
	) error
}

// PermissionService defines the interface for permission checks.
//
// This interface is used by ProtocolHandler to delegate permission
// decisions without depending on the concrete permissions package.
type PermissionService interface {
	// CanUseTool checks if a tool use is permitted.
	//
	// Returns the permission result (allow or deny) or an error.
	CanUseTool(ctx context.Context, req map[string]any) (map[string]any, error)
}
