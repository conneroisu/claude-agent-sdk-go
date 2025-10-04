package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/ports"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ClientAdapter wraps an external MCP client session.
// This adapter implements ports.MCPServer for external MCP servers
// connected via stdio, HTTP, or SSE transports.
type ClientAdapter struct {
	name    string
	session *mcp.ClientSession
}

// Verify interface compliance at compile time.
var _ ports.MCPServer = (*ClientAdapter)(nil)

// NewClientAdapter creates a new MCP client adapter.
// The session should already be connected to an external MCP server.
func NewClientAdapter(name string, session *mcp.ClientSession) *ClientAdapter {
	return &ClientAdapter{
		name:    name,
		session: session,
	}
}

// Name returns the server identifier.
func (a *ClientAdapter) Name() string {
	return a.name
}

// HandleMessage forwards a JSON-RPC message to the external MCP server.
// It uses the session to send arbitrary JSON-RPC requests.
func (a *ClientAdapter) HandleMessage(ctx context.Context, message []byte) ([]byte, error) {
	// The message is already in JSON-RPC format
	// We need to forward it to the session's underlying connection
	// Since the SDK doesn't expose a generic Request method,
	// we parse the method and route to the appropriate typed method

	var req map[string]any
	if err := json.Unmarshal(message, &req); err != nil {
		return createErrorResponse(req, -32700, "Parse error")
	}

	method, _ := req["method"].(string)

	// Route to appropriate session method based on MCP protocol
	switch method {
	case "tools/list":
		result, err := a.session.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			return createErrorResponse(req, -32603, err.Error())
		}

		return createSuccessResponse(req, result)

	case "tools/call":
		var params mcp.CallToolParams
		if paramsData, ok := req["params"].(map[string]any); ok {
			paramsJSON, _ := json.Marshal(paramsData)
			_ = json.Unmarshal(paramsJSON, &params)
		}
		result, err := a.session.CallTool(ctx, &params)
		if err != nil {
			return createErrorResponse(req, -32603, err.Error())
		}

		return createSuccessResponse(req, result)

	default:
		return createErrorResponse(req, -32601, fmt.Sprintf("Method not found: %s", method))
	}
}

// Close terminates the connection to the external MCP server.
func (a *ClientAdapter) Close() error {
	if a.session != nil {
		return a.session.Close()
	}

	return nil
}
