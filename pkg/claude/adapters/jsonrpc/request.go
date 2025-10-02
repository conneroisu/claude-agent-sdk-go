package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

const (
	// randomHexBytes is the number of bytes to generate for random hex strings.
	randomHexBytes = 4
	// controlRequestTimeout is the default timeout for control requests.
	controlRequestTimeout = 60 * time.Second

	// Message type constants.
	msgTypeControlResponse     = "control_response"
	msgTypeControlRequest      = "control_request"
	msgTypeCancelRequest       = "control_cancel_request"
	msgSubtypeError            = "error"
	msgSubtypeSuccess          = "success"
	msgFieldType               = "type"
	msgFieldResponse           = "response"
	msgFieldRequestID          = "request_id"
	msgFieldSubtype            = "subtype"
	msgFieldError              = "error"
	msgFieldRequest            = "request"
	msgFieldToolName           = "tool_name"
	msgFieldInput              = "input"
	msgFieldPermissionSuggests = "permission_suggestions"
	msgFieldCallbackID         = "callback_id"
	msgFieldToolUseID          = "tool_use_id"
	msgFieldServerName         = "server_name"
	msgFieldMessage            = "message"
	msgFieldMCPResponse        = "mcp_response"
	msgFieldAllow              = "allow"
	msgFieldReason             = "reason"
	msgFieldUpdatedPerms       = "updated_permissions"

	// Error messages.
	errRequestCancelled = "request cancelled"
)

// SendControlRequest sends a control request and waits for response.
// This method handles all request ID generation and timeout logic.
func (a *Adapter) SendControlRequest(
	ctx context.Context,
	req map[string]any,
) (map[string]any, error) {
	requestID := a.generateRequestID()
	resCh := a.registerPendingRequest(requestID)

	if err := a.sendControlRequestEnvelope(ctx, requestID, req); err != nil {
		a.cleanupPendingRequest(requestID)

		return nil, err
	}

	return a.waitForResponse(ctx, requestID, req, resCh)
}

// HandleControlRequest routes inbound control requests by subtype.
func (a *Adapter) HandleControlRequest(
	ctx context.Context,
	req map[string]any,
	deps ports.ControlDependencies,
) (map[string]any, error) {
	request, _ := req[msgFieldRequest].(map[string]any)
	subtype, _ := request[msgFieldSubtype].(string)

	switch subtype {
	case "can_use_tool":
		return handleCanUseTool(ctx, request, deps.Permissions)
	case "hook_callback":
		return handleHookCallback(ctx, request, deps.Hooks)
	case "mcp_message":
		return a.handleMCPMessage(ctx, request, deps.MCPServers)
	default:
		return nil, fmt.Errorf(
			"unsupported control request subtype: %s",
			subtype,
		)
	}
}

// generateRequestID creates a unique request ID.
func (a *Adapter) generateRequestID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.requestCounter++

	return fmt.Sprintf("req_%d_%s", a.requestCounter, randomHex(randomHexBytes))
}

// registerPendingRequest creates and registers a result channel for a request.
func (a *Adapter) registerPendingRequest(requestID string) chan result {
	resCh := make(chan result, 1)
	a.mu.Lock()
	a.pendingReqs[requestID] = resCh
	a.mu.Unlock()

	return resCh
}

// cleanupPendingRequest removes a pending request from the map.
func (a *Adapter) cleanupPendingRequest(requestID string) {
	a.mu.Lock()
	delete(a.pendingReqs, requestID)
	a.mu.Unlock()
}

// sendControlRequestEnvelope marshals and sends the control request.
func (a *Adapter) sendControlRequestEnvelope(
	ctx context.Context,
	requestID string,
	req map[string]any,
) error {
	controlReq := map[string]any{
		msgFieldType:      msgTypeControlRequest,
		msgFieldRequestID: requestID,
		msgFieldRequest:   req,
	}

	reqBytes, err := json.Marshal(controlReq)
	if err != nil {
		return fmt.Errorf("marshal control request: %w", err)
	}

	if err := a.transport.Write(ctx, string(reqBytes)+"\n"); err != nil {
		return fmt.Errorf("write control request: %w", err)
	}

	return nil
}

// waitForResponse waits for a response with timeout.
func (a *Adapter) waitForResponse(
	ctx context.Context,
	requestID string,
	req map[string]any,
	resCh chan result,
) (map[string]any, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, controlRequestTimeout)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		a.cleanupPendingRequest(requestID)
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf(
				"control request timeout: %s",
				req[msgFieldSubtype],
			)
		}

		return nil, timeoutCtx.Err()
	case res := <-resCh:
		if res.err != nil {
			return nil, res.err
		}

		return res.data, nil
	}
}
