package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// HandleControlRequest routes inbound control requests by subtype.
func (a *Adapter) HandleControlRequest(
	ctx context.Context,
	req map[string]any,
	perms ports.PermissionService,
	hooks map[string]ports.HookCallback,
	mcpServers map[string]ports.MCPServer,
) (map[string]any, error) {
	request, _ := req["request"].(map[string]any)
	subtype, _ := request["subtype"].(string)

	switch subtype {
	case "can_use_tool":
		return a.handleCanUseTool(ctx, request, perms)
	case "hook_callback":
		return a.handleHookCallback(ctx, request, hooks)
	case "mcp_message":
		return a.handleMCPMessage(ctx, request, mcpServers)
	default:
		return nil, fmt.Errorf("unsupported control request subtype: %s", subtype)
	}
}

func (*Adapter) handleCanUseTool(
	ctx context.Context,
	request map[string]any,
	perms ports.PermissionService,
) (map[string]any, error) {
	if perms == nil {
		return nil, fmt.Errorf("permissions callback not provided")
	}

	// Forward the entire request to the permission service
	// The service will handle parsing and validation
	return perms.CanUseTool(ctx, request)
}

func (*Adapter) handleHookCallback(
	ctx context.Context,
	request map[string]any,
	hooks map[string]ports.HookCallback,
) (map[string]any, error) {
	callbackID, _ := request["callback_id"].(string)
	input, _ := request["input"].(map[string]any)

	callback, exists := hooks[callbackID]
	if !exists {
		return nil, fmt.Errorf("no hook callback found for ID: %s", callbackID)
	}

	// Execute callback using the ports interface
	result, err := callback(ctx, input)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (a *Adapter) handleMCPMessage(
	ctx context.Context,
	request map[string]any,
	mcpServers map[string]ports.MCPServer,
) (map[string]any, error) {
	serverName, _ := request["server_name"].(string)
	mcpMessage, _ := request["message"].(map[string]any)

	server, exists := mcpServers[serverName]
	if !exists {
		return a.mcpErrorResponse(
			mcpMessage,
			-32601,
			fmt.Sprintf("Server '%s' not found", serverName),
		), nil
	}

	mcpMessageBytes, err := json.Marshal(mcpMessage)
	if err != nil {
		return a.mcpErrorResponse(mcpMessage, -32603, "failed to marshal mcp message"), nil
	}

	responseBytes, err := server.HandleMessage(ctx, mcpMessageBytes)
	if err != nil {
		return a.mcpErrorResponse(mcpMessage, -32603, err.Error()), nil
	}

	var mcpResponse map[string]any
	if err := json.Unmarshal(responseBytes, &mcpResponse); err != nil {
		return a.mcpErrorResponse(mcpMessage, -32603, "failed to unmarshal mcp response"), nil
	}

	return map[string]any{
		"mcp_response": mcpResponse,
	}, nil
}

func (a *Adapter) mcpErrorResponse(message map[string]any, code int, msg string) map[string]any {
	return map[string]any{
		"mcp_response": map[string]any{
			"jsonrpc": "2.0",
			"id":      message["id"],
			"error": map[string]any{
				"code":    code,
				"message": msg,
			},
		},
	}
}
