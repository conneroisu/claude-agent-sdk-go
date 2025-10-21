package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
)

// controlRequestEnvelope represents the envelope for control request messages.
type controlRequestEnvelope struct {
	Request struct {
		Subtype string `json:"subtype"`
	} `json:"request"`
	RequestID string `json:"request_id"`
}

// handleControlRequests processes incoming control requests from the CLI.
func (q *queryImpl) handleControlRequests() {
	for {
		select {
		case <-q.closeChan:
			return
		case data := <-q.controlRequestChan:
			// Parse the control request
			var envelope controlRequestEnvelope
			if err := json.Unmarshal(data, &envelope); err != nil {
				// Can't even parse the request ID, log and continue
				continue
			}

			// Handle the request in the background to avoid blocking
			go q.handleControlRequest(
				context.Background(),
				data,
				envelope.RequestID,
				envelope.Request.Subtype,
			)
		}
	}
}

// handleControlRequest handles a single control request from the CLI.
func (q *queryImpl) handleControlRequest(
	ctx context.Context,
	data json.RawMessage,
	requestID, subtype string,
) {
	var responseData map[string]any
	var err error

	switch subtype {
	case "can_use_tool":
		responseData, err = q.handleCanUseTool(ctx, data)
	case "hook_callback":
		responseData, err = q.handleHookCallback(ctx, data)
	case "mcp_message":
		// TODO: Handle SDK MCP requests when MCP servers are implemented
		err = clauderrs.NewProtocolError(
			clauderrs.ErrCodeProtocolError,
			"mcp_message handling not yet implemented",
			nil,
		).
			WithSessionID(q.sessionID).
			WithMessageType("control_request")
	default:
		err = clauderrs.NewProtocolError(
			clauderrs.ErrCodeProtocolError,
			fmt.Sprintf("unsupported control request subtype: %s", subtype),
			nil,
		).
			WithSessionID(q.sessionID).
			WithMessageType("control_request")
	}

	// Send response back to CLI
	if sendErr := q.sendControlResponse(ctx, requestID,
		responseData, err); sendErr != nil {
		// Log error but don't fail - the CLI will timeout
		if q.opts.Stderr != nil {
			q.opts.Stderr(fmt.Sprintf("Failed to send control response: %v", sendErr))
		}
	}
}

// handleCanUseTool processes can_use_tool control requests.
func (q *queryImpl) handleCanUseTool(
	ctx context.Context,
	data json.RawMessage,
) (map[string]any, error) {
	var req SDKControlPermissionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse permission request",
			err,
		).
			WithSessionID(q.sessionID).
			WithMessageType("control_request")
	}

	// Check if canUseTool callback is provided
	if q.opts.CanUseTool == nil {
		return nil, clauderrs.NewCallbackError(
			clauderrs.ErrCodeCallbackFailed,
			"canUseTool callback is not provided",
			nil,
			"canUseTool",
			false,
		).
			WithSessionID(q.sessionID)
	}

	// Convert JSONValue map to any map for the callback
	inputMap := make(map[string]JSONValue)
	maps.Copy(inputMap, req.Input)

	// Parse permission suggestions
	var suggestions []PermissionUpdate
	// TODO: Parse permission suggestions when needed

	// Call the user's callback
	result, err := q.opts.CanUseTool(ctx, req.ToolName, inputMap, suggestions)
	if err != nil {
		return nil, clauderrs.NewCallbackError(
			clauderrs.ErrCodeCallbackFailed,
			fmt.Sprintf("canUseTool failed for tool '%s'", req.ToolName),
			err,
			"canUseTool",
			false,
		).
			WithSessionID(q.sessionID)
	}

	// Convert PermissionResult to response format
	responseData := make(map[string]any)
	switch r := result.(type) {
	case *PermissionAllow:
		responseData["allow"] = true
		if r.UpdatedInput != nil {
			responseData["input"] = r.UpdatedInput
		}
		// TODO: Handle updatedPermissions when control protocol supports it
	case PermissionAllow:
		responseData["allow"] = true
		if r.UpdatedInput != nil {
			responseData["input"] = r.UpdatedInput
		}
	case *PermissionDeny:
		responseData["allow"] = false
		responseData["reason"] = r.Message
		// TODO: Handle interrupt flag when control protocol supports it
	case PermissionDeny:
		responseData["allow"] = false
		responseData["reason"] = r.Message
	default:
		return nil, clauderrs.NewCallbackError(clauderrs.ErrCodeCallbackFailed,
			fmt.Sprintf("canUseTool invalid return type %T", result),
			nil, "canUseTool", false).
			WithSessionID(q.sessionID)
	}

	return responseData, nil
}

// handleHookCallback processes hook_callback control requests.
func (q *queryImpl) handleHookCallback(
	ctx context.Context,
	data json.RawMessage,
) (map[string]any, error) {
	var req SDKHookCallbackRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse hook callback request",
			err,
		).
			WithSessionID(q.sessionID).
			WithMessageType("control_request")
	}

	// Look up the callback
	q.mu.Lock()
	callback, ok := q.hookCallbacks[req.CallbackID]
	q.mu.Unlock()

	if !ok {
		return nil, clauderrs.NewCallbackError(
			clauderrs.ErrCodeHookFailed,
			fmt.Sprintf("no hook callback found for ID: %s", req.CallbackID),
			nil,
			req.CallbackID,
			false,
		).
			WithSessionID(q.sessionID)
	}

	// Parse the hook input using the decoder
	hookInput, err := DecodeHookInput(req.Input)
	if err != nil {
		return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed,
			"failed to parse hook input", err).
			WithSessionID(q.sessionID).
			WithMessageType("hook_callback")
	}

	// Call the hook callback
	output, err := callback(ctx, hookInput, req.ToolUseID)
	if err != nil {
		toolUseID := ""
		if req.ToolUseID != nil {
			toolUseID = *req.ToolUseID
		}

		return nil, clauderrs.NewCallbackError(
			clauderrs.ErrCodeHookFailed,
			fmt.Sprintf("hook execution failed for tool use ID: %s", toolUseID),
			err,
			req.CallbackID,
			false,
		).
			WithSessionID(q.sessionID)
	}

	// Convert hook output to response format
	// The hook output should already be in the correct format
	// (JSON-serializable)
	// Marshal and unmarshal to convert to map[string]any
	outputBytes, err := json.Marshal(output)
	if err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to marshal hook output",
			err,
		).
			WithSessionID(q.sessionID).
			WithMessageType("hook_callback")
	}

	var responseData map[string]any
	if err := json.Unmarshal(outputBytes, &responseData); err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to unmarshal hook output",
			err,
		).
			WithSessionID(q.sessionID).
			WithMessageType("hook_callback")
	}

	return responseData, nil
}
