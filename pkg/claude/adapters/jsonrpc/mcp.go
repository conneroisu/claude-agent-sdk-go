package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// handleMCPMessage handles mcp_message control requests by proxying
// the raw message to the appropriate in-process MCPServer.
func (*Adapter) handleMCPMessage(
	ctx context.Context,
	request map[string]any,
	mcpServers map[string]ports.MCPServer,
) (map[string]any, error) {
	serverName, _ := request[msgFieldServerName].(string)
	mcpMessage, _ := request[msgFieldMessage].(map[string]any)

	server, exists := mcpServers[serverName]
	if !exists {
		return buildMCPErrorResponse(
			mcpMessage,
			-32601,
			fmt.Sprintf("Server '%s' not found", serverName),
		), nil
	}

	// Marshal the message to be sent to the server wrapper.
	mcpMessageBytes, err := json.Marshal(mcpMessage)
	if err != nil {
		return buildMCPErrorResponse(
			mcpMessage,
			-32603,
			"failed to marshal mcp message",
		), nil
	}

	// The MCPServer port handles the message and returns a raw response.
	responseBytes, err := server.HandleMessage(ctx, mcpMessageBytes)
	if err != nil {
		return buildMCPErrorResponse(
			mcpMessage,
			-32603,
			err.Error(),
		), nil
	}

	// Unmarshal the response to be embedded in the control protocol
	// response.
	var mcpResponse map[string]any
	unmarshalErr := json.Unmarshal(responseBytes, &mcpResponse)
	if unmarshalErr != nil {
		return buildMCPErrorResponse(
			mcpMessage,
			-32603,
			"failed to unmarshal mcp response",
		), nil
	}

	return map[string]any{
		msgFieldMCPResponse: mcpResponse,
	}, nil
}

// buildMCPErrorResponse creates an MCP JSON-RPC error response.
func buildMCPErrorResponse(
	message map[string]any,
	code int,
	msg string,
) map[string]any {
	return map[string]any{
		msgFieldMCPResponse: map[string]any{
			"jsonrpc": "2.0",
			"id":      message["id"],
			msgFieldError: map[string]any{
				"code":          code,
				msgFieldMessage: msg,
			},
		},
	}
}
