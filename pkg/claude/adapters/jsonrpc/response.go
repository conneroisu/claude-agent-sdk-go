package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// handleCancelRequest handles control_cancel_request messages.
func (a *Adapter) handleCancelRequest(msg map[string]any) {
	requestID, _ := msg[msgFieldRequestID].(string)
	a.mu.Lock()
	defer a.mu.Unlock()

	ch, exists := a.pendingReqs[requestID]
	if !exists {
		return
	}

	select {
	case ch <- result{err: errors.New(errRequestCancelled)}:
	default:
	}
	close(ch)
	delete(a.pendingReqs, requestID)
}

// routeControlResponse routes control_response messages to
// pending requests.
func (a *Adapter) routeControlResponse(msg map[string]any) {
	response, _ := msg[msgFieldResponse].(map[string]any)
	requestID, _ := response[msgFieldRequestID].(string)

	a.mu.Lock()
	defer a.mu.Unlock()

	ch, exists := a.pendingReqs[requestID]
	if !exists {
		return
	}

	subtype, _ := response[msgFieldSubtype].(string)
	if subtype == msgSubtypeError {
		errorMsg, _ := response[msgFieldError].(string)
		ch <- result{err: fmt.Errorf("control error: %s", errorMsg)}
	} else {
		responseData, _ := response[msgFieldResponse].(map[string]any)
		ch <- result{data: responseData}
	}
	delete(a.pendingReqs, requestID)
}

// handleControlRequestAsync handles inbound control requests
// asynchronously. Dependencies (perms, hooks, mcpServers) must be passed
// by the domain service that starts the router.
func (a *Adapter) handleControlRequestAsync(
	ctx context.Context,
	msg map[string]any,
	deps ports.ControlDependencies,
) {
	requestID, _ := msg[msgFieldRequestID].(string)

	// Handle the request
	responseData, err := a.HandleControlRequest(ctx, msg, deps)

	// Build response
	response := a.buildControlResponse(requestID, responseData, err)

	// Send response
	resBytes, _ := json.Marshal(response)
	// Best effort write
	_ = a.transport.Write(ctx, string(resBytes)+"\n")
}

// buildControlResponse creates a control protocol response message
func (*Adapter) buildControlResponse(
	requestID string,
	responseData map[string]any,
	err error,
) map[string]any {
	if err != nil {
		return map[string]any{
			msgFieldType: msgTypeControlResponse,
			msgFieldResponse: map[string]any{
				msgFieldSubtype:   msgSubtypeError,
				msgFieldRequestID: requestID,
				msgFieldError:     err.Error(),
			},
		}
	}

	return map[string]any{
		msgFieldType: msgTypeControlResponse,
		msgFieldResponse: map[string]any{
			msgFieldSubtype:   msgSubtypeSuccess,
			msgFieldRequestID: requestID,
			msgFieldResponse:  responseData,
		},
	}
}
