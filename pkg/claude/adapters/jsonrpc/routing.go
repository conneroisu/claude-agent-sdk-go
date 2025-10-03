//nolint:revive // comments-density: code is self-documenting
package jsonrpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// StartMessageRouter continuously reads transport and partitions
//nolint:revive // argument-limit,line-length-limit: all parameters required
// messages.
//nolint:revive // argument-limit: Hexagonal arch requires DI
func (a *Adapter) StartMessageRouter(
	ctx context.Context,
	msgCh chan<- map[string]any,
	errCh chan<- error,
	perms *permissions.Service,
	hooks map[string]hooking.HookCallback,
	mcpServers map[string]ports.MCPServer,
) error {
	go func() {
		transportMsgCh, transportErrCh := a.transport.ReadMessages(ctx) //nolint:lll

		for {
			select {
			case <-ctx.Done():
				return

			case msg, ok := <-transportMsgCh:
				if !ok {
					return
				}
				a.routeMessage(
					ctx,
					msg,
					msgCh,
					perms,
					hooks,
					mcpServers,
				)

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
//nolint:revive // argument-limit,early-return,line-length-limit: complexity justified
}

// routeMessage routes a message based on type.
//nolint:revive // argument-limit: Hexagonal arch requires DI
func (a *Adapter) routeMessage(
	ctx context.Context,
	msg map[string]any,
	msgCh chan<- map[string]any,
	perms *permissions.Service,
	hooks map[string]hooking.HookCallback,
	mcpServers map[string]ports.MCPServer,
) {
	msgType := extractOptionalString(msg, "type")

	switch msgType {
	case "control_response":
		a.routeControlResponse(msg)

	case "control_request":
		go a.handleControlRequestAsync(
			ctx,
			msg,
			perms,
			hooks,
			mcpServers,
		)

	case "control_cancel_request":
		a.handleCancelRequest(msg)

	default:
		select {
		case msgCh <- msg:
		case <-ctx.Done():
		}
	}
}

// routeControlResponse routes control_response messages.
func (a *Adapter) routeControlResponse(msg map[string]any) {
	response, _ := msg["response"].(map[string]any)
	requestID, _ := response["request_id"].(string)

	a.mu.Lock()
	defer a.mu.Unlock()

	if ch, exists := a.pendingReqs[requestID]; exists {
		subtype, _ := response["subtype"].(string)

		if subtype == "error" {
			errorMsg, _ := response["error"].(string)
			ch <- result{
				err: fmt.Errorf("control error: %s", errorMsg),
			}
		} else {
			responseData, _ := response["response"].(map[string]any) //nolint:lll
			ch <- result{data: responseData}
		}

		delete(a.pendingReqs, requestID)
	}
}

// handleCancelRequest handles control_cancel_request messages.
func (a *Adapter) handleCancelRequest(msg map[string]any) {
	requestID, _ := msg["request_id"].(string)

	a.mu.Lock()
	defer a.mu.Unlock()

	if ch, exists := a.pendingReqs[requestID]; exists {
		select {
		case ch <- result{err: errors.New("request cancelled")}:
		default:
		}
		close(ch)
		delete(a.pendingReqs, requestID)
	}
}
