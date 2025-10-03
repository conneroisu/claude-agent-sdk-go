package jsonrpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.ProtocolHandler for control protocol
// This is an INFRASTRUCTURE adapter - it handles protocol state management.
type Adapter struct {
	transport ports.Transport
	// Control protocol state (managed by adapter, not domain)
	pendingReqs    map[string]chan result
	requestCounter int
	mu             sync.Mutex
}

// Verify interface compliance at compile time.
var _ ports.ProtocolHandler = (*Adapter)(nil)

type result struct {
	data map[string]any
	err  error
}

func NewAdapter(transport ports.Transport) *Adapter {
	return &Adapter{
		transport:   transport,
		pendingReqs: make(map[string]chan result),
	}
}

// Initialize is a no-op - initialization happens implicitly in StartMessageRouter.
func (a *Adapter) Initialize(ctx context.Context, config any) (map[string]any, error) {
	return nil, nil
}

// SendControlRequest sends a control request and waits for response
// This method handles all request ID generation and timeout logic.
func (a *Adapter) SendControlRequest(ctx context.Context, req map[string]any) (map[string]any, error) {
	// Generate unique request ID: req_{counter}_{randomHex}
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
			return nil, fmt.Errorf("control request timeout: %s", req["subtype"])
		}

		return nil, timeoutCtx.Err()
	case res := <-resCh:
		if res.err != nil {
			return nil, res.err
		}

		return res.data, nil
	}
}

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

// StartMessageRouter continuously reads transport and partitions messages.
func (a *Adapter) StartMessageRouter(
	ctx context.Context,
	msgCh chan<- map[string]any,
	errCh chan<- error,
	perms ports.PermissionService,
	hooks map[string]ports.HookCallback,
	mcpServers map[string]ports.MCPServer,
) error {
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
					go a.handleControlRequestAsync(ctx, msg, perms, hooks, mcpServers)
				case "control_cancel_request":
					// Cancel a pending control request
					requestID, _ := msg["request_id"].(string)
					a.mu.Lock()
					if ch, exists := a.pendingReqs[requestID]; exists {
						// Send error to indicate cancellation
						select {
						case ch <- result{err: fmt.Errorf("request cancelled")}:
						default:
						}
						close(ch)
						delete(a.pendingReqs, requestID)
					}
					a.mu.Unlock()

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

// routeControlResponse routes control_response messages to pending requests.
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
// Dependencies (perms, hooks, mcpServers) must be passed by the domain service that starts the router.
func (a *Adapter) handleControlRequestAsync(
	ctx context.Context,
	msg map[string]any,
	perms ports.PermissionService,
	hooks map[string]ports.HookCallback,
	mcpServers map[string]ports.MCPServer,
) {
	requestID, _ := msg["request_id"].(string)
	// Handle the request
	responseData, err := a.HandleControlRequest(ctx, msg, perms, hooks, mcpServers)
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
	a.transport.Write(ctx, string(resBytes)+"\n")
}

// handleCanUseTool handles can_use_tool control requests.
func (a *Adapter) handleCanUseTool(ctx context.Context, request map[string]any, perms ports.PermissionService) (map[string]any, error) {
	toolName, _ := request["tool_name"].(string)
	input, _ := request["input"].(map[string]any)

	// Parse permission suggestions from control request
	var suggestions []permissions.PermissionUpdate
	if suggestionsData, ok := request["permission_suggestions"].([]any); ok {
		for _, sugData := range suggestionsData {
			if sugMap, ok := sugData.(map[string]any); ok {
				update := parsePermissionUpdate(sugMap)
				suggestions = append(suggestions, update)
			}
		}
	}

	if perms == nil {
		return nil, fmt.Errorf("permissions callback not provided")
	}

	result, err := perms.CheckToolUse(ctx, toolName, input, suggestions)
	if err != nil {
		return nil, err
	}

	// Convert PermissionResult to response format
	switch r := result.(type) {
	case *permissions.PermissionResultAllow:
		response := map[string]any{"allow": true}
		if r.UpdatedInput != nil {
			response["input"] = r.UpdatedInput
		}
		// Include updated permissions in response (for "always allow" flow)
		if r.UpdatedPermissions != nil && len(r.UpdatedPermissions) > 0 {
			response["updated_permissions"] = r.UpdatedPermissions
		}

		return response, nil
	case *permissions.PermissionResultDeny:
		return map[string]any{
			"allow":  false,
			"reason": r.Message,
		}, nil
	default:
		return nil, fmt.Errorf("unknown permission result type")
	}
}

// parsePermissionUpdate parses a single permission update from raw data.
func parsePermissionUpdate(data map[string]any) permissions.PermissionUpdate {
	update := permissions.PermissionUpdate{}

	if updateType, ok := data["type"].(string); ok {
		update.Type = updateType
	}

	if rulesData, ok := data["rules"].([]any); ok {
		for _, ruleData := range rulesData {
			if ruleMap, ok := ruleData.(map[string]any); ok {
				toolName, _ := ruleMap["toolName"].(string)
				ruleContent := getStringPtr(ruleMap, "ruleContent")
				update.Rules = append(update.Rules, permissions.PermissionRuleValue{
					ToolName:    toolName,
					RuleContent: ruleContent,
				})
			}
		}
	}

	if behaviorStr, ok := data["behavior"].(string); ok {
		behavior := permissions.PermissionBehavior(behaviorStr)
		update.Behavior = &behavior
	}

	if modeStr, ok := data["mode"].(string); ok {
		mode := options.PermissionMode(modeStr)
		update.Mode = &mode
	}

	if dirsData, ok := data["directories"].([]any); ok {
		for _, dirData := range dirsData {
			if dir, ok := dirData.(string); ok {
				update.Directories = append(update.Directories, dir)
			}
		}
	}

	if destStr, ok := data["destination"].(string); ok {
		dest := permissions.PermissionUpdateDestination(destStr)
		update.Destination = &dest
	}

	return update
}

// handleHookCallback handles hook_callback control requests.
func (a *Adapter) handleHookCallback(ctx context.Context, request map[string]any, hooks map[string]ports.HookCallback) (map[string]any, error) {
	callbackID, _ := request["callback_id"].(string)
	input, _ := request["input"].(map[string]any)
	// toolUseID is a string, not *string - JSON decoding produces plain string values
	var toolUseID *string
	if id, ok := request["tool_use_id"].(string); ok {
		toolUseID = &id
	}
	callback, exists := hooks[callbackID]
	if !exists {
		return nil, fmt.Errorf("no hook callback found for ID: %s", callbackID)
	}
	// Execute callback with context for cancellation support
	// The port interface expects any, so we pass ctx directly
	result, err := callback(input, toolUseID, ctx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// handleMCPMessage handles mcp_message control requests by proxying
// the raw message to the appropriate in-process MCPServer.
func (a *Adapter) handleMCPMessage(ctx context.Context, request map[string]any, mcpServers map[string]ports.MCPServer) (map[string]any, error) {
	serverName, _ := request["server_name"].(string)
	mcpMessage, _ := request["message"].(map[string]any)
	server, exists := mcpServers[serverName]
	if !exists {
		return a.mcpErrorResponse(mcpMessage, -32601, fmt.Sprintf("Server '%s' not found", serverName)), nil
	}
	// Marshal the message to be sent to the server wrapper.
	mcpMessageBytes, err := json.Marshal(mcpMessage)
	if err != nil {
		return a.mcpErrorResponse(mcpMessage, -32603, "failed to marshal mcp message"), nil
	}
	// The MCPServer port handles the message and returns a raw response.
	responseBytes, err := server.HandleMessage(ctx, mcpMessageBytes)
	if err != nil {
		return a.mcpErrorResponse(mcpMessage, -32603, err.Error()), nil
	}
	// Unmarshal the response to be embedded in the control protocol response.
	var mcpResponse map[string]any
	if err := json.Unmarshal(responseBytes, &mcpResponse); err != nil {
		return a.mcpErrorResponse(mcpMessage, -32603, "failed to unmarshal mcp response"), nil
	}

	return map[string]any{
		"mcp_response": mcpResponse,
	}, nil
}

// mcpErrorResponse creates an MCP JSON-RPC error response.
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

// randomHex generates a random hex string of n bytes.
func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)

	return hex.EncodeToString(b)
}

// getStringPtr extracts an optional string pointer from a map.
func getStringPtr(m map[string]any, key string) *string {
	if v, ok := m[key].(string); ok {
		return &v
	}

	return nil
}
