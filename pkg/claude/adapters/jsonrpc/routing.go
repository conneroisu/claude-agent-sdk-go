package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// StartMessageRouter continuously reads transport and partitions messages.
// Control responses are routed to pending requests.
// Control requests are handled asynchronously.
// SDK messages are forwarded to the public stream.
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
				a.routeMessage(ctx, msg, msgCh, perms, hooks, mcpServers)
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

func (a *Adapter) routeMessage(
	ctx context.Context,
	msg map[string]any,
	msgCh chan<- map[string]any,
	perms ports.PermissionService,
	hooks map[string]ports.HookCallback,
	mcpServers map[string]ports.MCPServer,
) {
	msgType, _ := msg["type"].(string)
	switch msgType {
	case "control_response":
		a.routeControlResponse(msg)
	case "control_request":
		go a.handleControlRequestAsync(ctx, msg, perms, hooks, mcpServers)
	case "control_cancel_request":
		a.handleCancelRequest(msg)
	default:
		select {
		case msgCh <- msg:
		case <-ctx.Done():
		}
	}
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

func (a *Adapter) handleCancelRequest(msg map[string]any) {
	requestID, _ := msg["request_id"].(string)
	a.mu.Lock()
	defer a.mu.Unlock()

	if ch, exists := a.pendingReqs[requestID]; exists {
		select {
		case ch <- result{err: fmt.Errorf("request cancelled")}:
		default:
		}
		close(ch)
		delete(a.pendingReqs, requestID)
	}
}

// handleControlRequestAsync handles inbound control requests asynchronously.
func (a *Adapter) handleControlRequestAsync(
	ctx context.Context,
	msg map[string]any,
	perms ports.PermissionService,
	hooks map[string]ports.HookCallback,
	mcpServers map[string]ports.MCPServer,
) {
	requestID, _ := msg["request_id"].(string)

	responseData, err := a.HandleControlRequest(ctx, msg, perms, hooks, mcpServers)

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

	resBytes, _ := json.Marshal(response)
	_ = a.transport.Write(ctx, string(resBytes)+"\n")
}
