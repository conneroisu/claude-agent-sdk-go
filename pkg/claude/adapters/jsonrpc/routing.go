package jsonrpc

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// routerChannels groups the channels used by the message router.
type routerChannels struct {
	transportMsg <-chan map[string]any
	transportErr <-chan error
	output       chan<- map[string]any
	errors       chan<- error
}

// StartMessageRouter implements ports.ProtocolHandler.
// It starts routing messages from the transport layer to appropriate handlers.
// Control messages are handled internally, while data messages are forwarded.
func (a *Adapter) StartMessageRouter(
	ctx context.Context,
	msgCh chan<- map[string]any,
	errCh chan<- error,
	deps ports.ControlDependencies,
) error {
	// Get transport message and error channels
	transportMsgCh, transportErrCh := a.transport.ReadMessages(ctx)

	channels := routerChannels{
		transportMsg: transportMsgCh,
		transportErr: transportErrCh,
		output:       msgCh,
		errors:       errCh,
	}

	// Start routing goroutine
	go a.routeMessages(ctx, channels, deps)

	return nil
}

// routeMessages is the main routing loop that processes incoming messages.
// It runs in a goroutine and routes messages based on their type.
func (a *Adapter) routeMessages(
	ctx context.Context,
	ch routerChannels,
	deps ports.ControlDependencies,
) {
	for {
		select {
		// Context cancellation - stop routing
		case <-ctx.Done():
			return

		// New message from transport
		case msg, ok := <-ch.transportMsg:
			if !ok {
				return
			}
			a.routeMessage(ctx, msg, ch.output, deps)

		// Error from transport
		case err, ok := <-ch.transportErr:
			if ok && err != nil {
				// Forward error non-blocking
				select {
				case ch.errors <- err:
				default:
				}
			}

			return
		}
	}
}

// routeMessage routes a single message based on its type.
// Control messages are handled internally via handleControlResponse/Request.
// All other messages are forwarded to the output channel.
func (a *Adapter) routeMessage(
	ctx context.Context,
	msg map[string]any,
	msgCh chan<- map[string]any,
	deps ports.ControlDependencies,
) {
	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	// Route based on message type
	switch msgType {
	case "control_response":
		a.handleControlResponse(msg)
	case "control_request":
		a.processControlRequest(ctx, msg, deps)
	default:
		msgCh <- msg
	}
}
