package ports

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
)

// ProtocolHandler defines what the domain needs for control
// protocol management. This interface handles JSON-RPC control
// messages while keeping domain services free of protocol details.
type ProtocolHandler interface {
	// Initialize sends the initialize control request with
	// hooks config
	Initialize(
		ctx context.Context,
		config map[string]any,
	) (map[string]any, error)

	// SendControlRequest sends a control request and waits
	// for response (60s timeout)
	SendControlRequest(
		ctx context.Context,
		req map[string]any,
	) (map[string]any, error)

	// HandleControlRequest routes inbound control requests
	// by subtype. Subtypes: can_use_tool, hook_callback,
	// mcp_message. Dependencies are passed as arguments to
	// avoid circular refs.
	HandleControlRequest(
		ctx context.Context,
		req map[string]any,
		perms *permissions.Service,
		hooks map[string]hooking.HookCallback,
		mcpServers map[string]MCPServer,
	) (map[string]any, error)

	// StartMessageRouter continuously reads transport and
	// partitions messages. Routes control_response,
	// control_request, control_cancel_request separately
	// from SDK messages. Dependencies (perms, hooks,
	// mcpServers) are passed by domain service for handling
	// inbound control requests.
	StartMessageRouter(
		ctx context.Context,
		msgCh chan<- map[string]any,
		errCh chan<- error,
		perms *permissions.Service,
		hooks map[string]hooking.HookCallback,
		mcpServers map[string]MCPServer,
	) error
}
