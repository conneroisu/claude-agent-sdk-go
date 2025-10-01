package jsonrpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.ProtocolHandler for control protocol
type Adapter struct {
	transport ports.Transport

	// Control protocol state
	pendingReqs    map[string]chan result
	requestCounter int
	mu             sync.Mutex
}

// Verify interface compliance at compile time
var _ ports.ProtocolHandler = (*Adapter)(nil)

type result struct {
	data map[string]any
	err  error
}

// NewAdapter creates a new JSON-RPC protocol adapter
func NewAdapter(transport ports.Transport) *Adapter {
	return &Adapter{
		transport:   transport,
		pendingReqs: make(map[string]chan result),
	}
}

// Initialize is a no-op - initialization happens implicitly
func (a *Adapter) Initialize(ctx context.Context, config any) (map[string]any, error) {
	return nil, nil
}

// SendControlRequest sends a control request and waits for response
func (a *Adapter) SendControlRequest(ctx context.Context, req map[string]any) (map[string]any, error) {
	// Generate unique request ID
	a.mu.Lock()
	a.requestCounter++
	requestID := fmt.Sprintf("req_%d_%s", a.requestCounter, randomHex(4))
	a.mu.Unlock()

	// Create result channel for this request
	resCh := make(chan result, 1)
	a.mu.Lock()
	a.pendingReqs[requestID] = resCh
	a.mu.Unlock()

	// Build control request envelope
	controlReq := map[string]any{
		"type":       "control_request",
		"request_id": requestID,
		"request":    req,
	}

	// Send via transport
	reqBytes, err := json.Marshal(controlReq)
	if err != nil {
		a.mu.Lock()
		delete(a.pendingReqs, requestID)
		a.mu.Unlock()

		return nil, fmt.Errorf("marshal control request: %w", err)
	}

	if err := a.transport.Write(ctx, string(reqBytes)+"\n"); err != nil {
		a.mu.Lock()
		delete(a.pendingReqs, requestID)
		a.mu.Unlock()

		return nil, fmt.Errorf("write control request: %w", err)
	}

	// Wait for response with 60s timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		a.mu.Lock()
		delete(a.pendingReqs, requestID)
		a.mu.Unlock()
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("control request timeout: %v", req["subtype"])
		}

		return nil, timeoutCtx.Err()

	case res := <-resCh:
		if res.err != nil {
			return nil, res.err
		}

		return res.data, nil
	}
}

// HandleControlRequest routes inbound control requests by subtype
func (a *Adapter) HandleControlRequest(ctx context.Context, req map[string]any, deps ports.ProtocolDependencies) (map[string]any, error) {
	request, _ := req["request"].(map[string]any)
	subtype, _ := request["subtype"].(string)

	switch subtype {
	case "can_use_tool":
		return a.handleCanUseTool(ctx, request, deps.Permissions)
	case "hook_callback":
		return a.handleHookCallback(ctx, request, deps.Hooks)
	case "mcp_message":
		return a.handleMCPMessage(ctx, request, deps.MCPServers)
	default:
		return nil, fmt.Errorf("unsupported control request subtype: %s", subtype)
	}
}

// StartMessageRouter continuously reads transport and partitions messages
func (a *Adapter) StartMessageRouter(ctx context.Context, msgCh chan<- map[string]any, errCh chan<- error, deps ports.ProtocolDependencies) error {
	go func() {
		transportMsgCh, transportErrCh := a.transport.ReadMessages(ctx)

		for {
			select {
			case <-ctx.Done():
				return

			case msg, ok := <-transportMsgCh:
				if !ok {
					return
				}

				msgType, _ := msg["type"].(string)
				switch msgType {
				case "control_response":
					// Route to pending request
					a.routeControlResponse(msg)

				case "control_request":
					// Handle inbound control request
					go a.handleControlRequestAsync(ctx, msg, deps)

				case "control_cancel_request":
					// TODO: Implement cancellation support
					continue

				default:
					// Forward SDK messages to public stream
					select {
					case msgCh <- msg:
					case <-ctx.Done():
						return
					}
				}

			case err := <-transportErrCh:
				if err != nil {
					select {
					case errCh <- err:
					case <-ctx.Done():
					}

					return
				}
			}
		}
	}()

	return nil
}

// routeControlResponse routes control_response messages to pending requests
func (a *Adapter) routeControlResponse(msg map[string]any) {
	response, _ := msg["response"].(map[string]any)
	requestID, _ := response["request_id"].(string)

	a.mu.Lock()
	defer a.mu.Unlock()

	if ch, exists := a.pendingReqs[requestID]; exists {
		subtype, _ := response["subtype"].(string)
		if subtype == "error" {
			errorMsg, _ := response["error"].(string)
			ch <- result{err: fmt.Errorf("control error: %s", errorMsg)}
		} else {
			responseData, _ := response["response"].(map[string]any)
			ch <- result{data: responseData}
		}
		delete(a.pendingReqs, requestID)
	}
}

// handleControlRequestAsync handles inbound control requests asynchronously
func (a *Adapter) handleControlRequestAsync(ctx context.Context, msg map[string]any, deps ports.ProtocolDependencies) {
	requestID, _ := msg["request_id"].(string)

	// Handle the request
	responseData, err := a.HandleControlRequest(ctx, msg, deps)

	// Build response
	var response map[string]any
	if err != nil {
		response = map[string]any{
			"type": "control_response",
			"response": map[string]any{
				"subtype":    "error",
				"request_id": requestID,
				"error":      err.Error(),
			},
		}
	} else {
		response = map[string]any{
			"type": "control_response",
			"response": map[string]any{
				"subtype":    "success",
				"request_id": requestID,
				"response":   responseData,
			},
		}
	}

	// Send response
	resBytes, _ := json.Marshal(response)
	_ = a.transport.Write(ctx, string(resBytes)+"\n")
}

// handleCanUseTool handles can_use_tool control requests
func (a *Adapter) handleCanUseTool(ctx context.Context, request map[string]any, perms any) (map[string]any, error) {
	toolName, _ := request["tool_name"].(string)
	input, _ := request["input"].(map[string]any)

	if perms == nil {
		return nil, fmt.Errorf("permissions callback not provided")
	}

	// Use type assertion to call CheckToolUse
	type permChecker interface {
		CheckToolUse(ctx context.Context, toolName string, input map[string]any) (any, error)
	}

	checker, ok := perms.(permChecker)
	if !ok {
		return nil, fmt.Errorf("invalid permissions type")
	}

	result, err := checker.CheckToolUse(ctx, toolName, input)
	if err != nil {
		return nil, err
	}

	// Use type assertion to check result
	type permResult interface {
		IsAllowed() bool
		GetUpdatedInput() map[string]any
		GetDenyMessage() string
	}

	permRes, ok := result.(permResult)
	if !ok {
		return nil, fmt.Errorf("invalid permission result type")
	}

	// Convert PermissionResult to response format
	if permRes.IsAllowed() {
		response := map[string]any{"allow": true}
		if updatedInput := permRes.GetUpdatedInput(); updatedInput != nil {
			response["input"] = updatedInput
		}

		return response, nil
	}

	return map[string]any{
		"allow":  false,
		"reason": permRes.GetDenyMessage(),
	}, nil
}

// handleHookCallback handles hook_callback control requests
func (a *Adapter) handleHookCallback(ctx context.Context, request map[string]any, hooks map[string]any) (map[string]any, error) {
	callbackID, _ := request["callback_id"].(string)
	input, _ := request["input"].(map[string]any)
	toolUseID, _ := request["tool_use_id"].(*string)

	callbackAny, exists := hooks[callbackID]
	if !exists {
		return nil, fmt.Errorf("no hook callback found for ID: %s", callbackID)
	}

	// Type assert to hook callback function
	type hookCallback func(input map[string]any, toolUseID *string, ctx any) (map[string]any, error)
	callback, ok := callbackAny.(hookCallback)
	if !ok {
		return nil, fmt.Errorf("invalid hook callback type")
	}

	// Execute callback
	result, err := callback(input, toolUseID, struct{}{})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// handleMCPMessage handles mcp_message control requests
func (a *Adapter) handleMCPMessage(ctx context.Context, request map[string]any, mcpServers map[string]ports.MCPServer) (map[string]any, error) {
	serverName, _ := request["server_name"].(string)
	mcpMessage, _ := request["message"].(map[string]any)

	server, exists := mcpServers[serverName]
	if !exists {
		return a.mcpErrorResponse(mcpMessage, -32601, fmt.Sprintf("Server '%s' not found", serverName)), nil
	}

	method, _ := mcpMessage["method"].(string)
	messageID := mcpMessage["id"]
	params := mcpMessage["params"]

	var result any
	var err error

	switch method {
	case "initialize":
		result, err = server.Initialize(ctx, params)
	case "tools/list":
		result, err = server.ListTools(ctx)
	case "tools/call":
		callParams, _ := params.(map[string]any)
		toolName, _ := callParams["name"].(string)
		args, _ := callParams["arguments"].(map[string]any)
		result, err = server.CallTool(ctx, toolName, args)
	case "notifications/initialized":
		err = server.HandleNotification(ctx, method, params)
		result = nil
	default:
		return a.mcpErrorResponse(mcpMessage, -32601, "Method not found"), nil
	}

	if err != nil {
		return a.mcpErrorResponse(mcpMessage, -32603, err.Error()), nil
	}

	return map[string]any{
		"mcp_response": map[string]any{
			"jsonrpc": "2.0",
			"id":      messageID,
			"result":  result,
		},
	}, nil
}

// mcpErrorResponse creates an MCP JSON-RPC error response
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

// randomHex generates a random hex string of n bytes
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
