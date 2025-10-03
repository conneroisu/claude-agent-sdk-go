package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// HandleControlRequest routes inbound control requests by subtype.
// It dispatches to the appropriate handler based on the request subtype.
// This is the main entry point for handling control protocol requests.
//nolint:revive // argument-limit: all parameters required for handler
func (a *Adapter) HandleControlRequest(
	ctx context.Context,
	req map[string]any,
	perms *permissions.Service,
	hooks map[string]hooking.HookCallback,
	mcpServers map[string]ports.MCPServer,
) (map[string]any, error) {
	// Extract the request envelope (required).
	request, err := extractRequiredMap(req, "request")
	if err != nil {
		return nil, fmt.Errorf(
			"control request: %w",
			err,
		)
	}

	// Extract subtype to route the request (required).
	subtype, err := extractRequiredString(request, "subtype")
	if err != nil {
		return nil, fmt.Errorf(
			"control request: %w",
			err,
		)
	}

	// Route to appropriate handler based on subtype.
	switch subtype {
	case "can_use_tool":
		return a.handleCanUseTool(ctx, request, perms)
	case "hook_callback":
		return a.handleHookCallback(ctx, request, hooks)
	case "mcp_message":
		return a.handleMCPMessage(ctx, request, mcpServers)
	default:
		return nil, fmt.Errorf(
			"unsupported control request subtype: %s",
			subtype,
		)
	}
}

// handleControlRequestAsync handles inbound control requests async.
// It processes the request and sends a control response back through
// the transport. This method is designed to be run in a goroutine.
//nolint:revive // argument-limit: all parameters required for async handler
func (a *Adapter) handleControlRequestAsync(
	ctx context.Context,
	msg map[string]any,
	perms *permissions.Service,
	hooks map[string]hooking.HookCallback,
	mcpServers map[string]ports.MCPServer,
) {
	// Extract request ID for response correlation (required).
	requestID := extractOptionalString(msg, "request_id")

	// Process the control request.
	responseData, err := a.HandleControlRequest(
		ctx,
		msg,
		perms,
		hooks,
		mcpServers,
	)

	// Build response based on success or error.
	var response map[string]any
	if err != nil {
		// Error response.
		response = map[string]any{
			"type": "control_response",
			"response": map[string]any{
				"subtype":    "error",
				"request_id": requestID,
				"error":      err.Error(),
			},
		}
	} else {
		// Success response.
		response = map[string]any{
			"type": "control_response",
			"response": map[string]any{
				"subtype":    "success",
				"request_id": requestID,
				"response":   responseData,
			},
		}
	}

	// Send the response back through the transport.
	resBytes, _ := json.Marshal(response)
	_ = a.transport.Write(ctx, string(resBytes)+"\n")
}
