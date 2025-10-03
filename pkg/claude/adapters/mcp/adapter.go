package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/ports"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Adapter wraps an MCP ClientSession to implement ports.MCPServer.
type Adapter struct {
	name    string
	session *mcpsdk.ClientSession
}

// Verify interface compliance at compile time.
var _ ports.MCPServer = (*Adapter)(nil)

// NewAdapter creates a new MCP adapter wrapping the given session.
func NewAdapter(name string, session *mcpsdk.ClientSession) *Adapter {
	return &Adapter{
		name:    name,
		session: session,
	}
}

// Name returns the server name.
func (a *Adapter) Name() string {
	return a.name
}

// HandleMessage forwards a raw JSON-RPC message to the MCP server.
//
// This parses the incoming message and routes it to the appropriate
// ClientSession method based on the JSON-RPC method name.
func (a *Adapter) HandleMessage(
	ctx context.Context,
	message []byte,
) ([]byte, error) {
	// Parse the raw message to extract method and params
	var rawMsg map[string]any
	if err := json.Unmarshal(message, &rawMsg); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}

	method, ok := rawMsg["method"].(string)
	if !ok {
		return nil, fmt.Errorf("missing method field")
	}

	// Route to appropriate handler
	result, err := a.handleMethod(ctx, method, rawMsg["params"])
	if err != nil {
		// Return JSON-RPC error response
		return a.errorResponse(rawMsg["id"], -32603, err.Error())
	}

	// Return success response
	return a.successResponse(rawMsg["id"], result)
}

// handleMethod routes to the appropriate ClientSession method.
func (a *Adapter) handleMethod(
	ctx context.Context,
	method string,
	paramsRaw any,
) (any, error) {
	switch method {
	case "tools/list":
		return a.session.ListTools(ctx, nil)

	case "tools/call":
		var params mcpsdk.CallToolParams
		if err := unmarshalParams(paramsRaw, &params); err != nil {
			return nil, err
		}
		return a.session.CallTool(ctx, &params)

	case "resources/list":
		return a.session.ListResources(ctx, nil)

	case "resources/read":
		var params mcpsdk.ReadResourceParams
		if err := unmarshalParams(paramsRaw, &params); err != nil {
			return nil, err
		}
		return a.session.ReadResource(ctx, &params)

	case "prompts/list":
		return a.session.ListPrompts(ctx, nil)

	case "prompts/get":
		var params mcpsdk.GetPromptParams
		if err := unmarshalParams(paramsRaw, &params); err != nil {
			return nil, err
		}
		return a.session.GetPrompt(ctx, &params)

	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}
}

// unmarshalParams converts raw params to typed struct.
func unmarshalParams(raw any, target any) error {
	if raw == nil {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal params: %w", err)
	}
	return nil
}

// successResponse creates a JSON-RPC success response.
func (a *Adapter) successResponse(
	id any,
	result any,
) ([]byte, error) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	return json.Marshal(resp)
}

// errorResponse creates a JSON-RPC error response.
func (a *Adapter) errorResponse(
	id any,
	code int,
	message string,
) ([]byte, error) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	return json.Marshal(resp)
}

// Close closes the MCP session.
func (a *Adapter) Close() error {
	return a.session.Close()
}
