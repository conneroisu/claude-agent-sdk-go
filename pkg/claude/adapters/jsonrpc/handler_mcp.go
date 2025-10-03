package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// handleMCPMessage handles mcp_message control requests.
// It routes messages to the appropriate MCP server by name.
func (a *Adapter) handleMCPMessage(
	ctx context.Context,
	request map[string]any,
	mcpServers map[string]ports.MCPServer,
) (map[string]any, error) {
	// Unmarshal request into typed struct.
	var req MCPMessageRequest
	if err := unmarshalRequest(request, &req); err != nil {
		return nil, fmt.Errorf("unmarshal mcp_message request: %w", err)
	}

	// Look up the MCP server by name.
	server, exists := mcpServers[req.ServerName]
	if !exists {
		return a.mcpErrorResponse(
			req.Message,
			-32601,
			fmt.Sprintf("Server '%s' not found", req.ServerName),
		), nil
	}

	// Marshal the MCP message to JSON bytes.
	mcpMessageBytes, err := json.Marshal(req.Message)
	if err != nil {
		return a.mcpErrorResponse(
			req.Message,
			-32603,
			"failed to marshal mcp message",
		), nil
	}

	// Forward the message to the MCP server.
	responseBytes, err := server.HandleMessage(ctx, mcpMessageBytes)
	if err != nil {
		return a.mcpErrorResponse(
			req.Message,
			-32603,
			err.Error(),
		), nil
	}

	// Unmarshal the response from the MCP server.
	var mcpResponse map[string]any
	err = json.Unmarshal(responseBytes, &mcpResponse)
	if err != nil {
		return a.mcpErrorResponse(
			req.Message,
			-32603,
			"failed to unmarshal mcp response",
		), nil
	}

	// Return the MCP response wrapped in expected format.
	return map[string]any{
		"mcp_response": mcpResponse,
	}, nil
}

// mcpErrorResponse creates an MCP JSON-RPC error response.
// It follows the JSON-RPC 2.0 error format specification.
func (*Adapter) mcpErrorResponse(
	message map[string]any,
	code int,
	msg string,
) map[string]any {
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
