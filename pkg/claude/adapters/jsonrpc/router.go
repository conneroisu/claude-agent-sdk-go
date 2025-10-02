// Package jsonrpc provides message routing functionality for the JSON-RPC
// protocol adapter. It handles routing of messages between the transport layer
// and control protocol handlers.
package jsonrpc

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// StartMessageRouter continuously reads transport and partitions messages.
func (a *Adapter) StartMessageRouter(
	ctx context.Context,
	msgCh chan<- map[string]any,
	errCh chan<- error,
	deps ports.ControlDependencies,
) error {
	go a.messageRouterLoop(ctx, msgCh, errCh, deps)

	return nil
}

// messageRouterLoop is the main loop for routing messages from the transport.
func (a *Adapter) messageRouterLoop(
	ctx context.Context,
	msgCh chan<- map[string]any,
	errCh chan<- error,
	deps ports.ControlDependencies,
) {
	defer close(msgCh)
	defer close(errCh)

	transportMsgCh, transportErrCh := a.transport.ReadMessages(ctx)
	channels := routerChannels{
		transportMsg: transportMsgCh,
		transportErr: transportErrCh,
		msg:          msgCh,
		err:          errCh,
	}
	for {
		if a.processRouterEvent(ctx, channels, deps) {
			return
		}
	}
}

// processRouterEvent handles a single event from the transport.
// Returns true if the router should exit.
func (a *Adapter) processRouterEvent(
	ctx context.Context,
	channels routerChannels,
	deps ports.ControlDependencies,
) bool {
	select {
	case <-ctx.Done():
		return true
	case msg, ok := <-channels.transportMsg:
		if !ok {
			return true
		}
		a.handleTransportMessage(ctx, msg, channels.msg, deps)

		return false
	case err := <-channels.transportErr:
		return handleTransportError(ctx, err, channels.err)
	}
}

// handleTransportMessage processes a single message from the transport.
func (a *Adapter) handleTransportMessage(
	ctx context.Context,
	msg map[string]any,
	msgCh chan<- map[string]any,
	deps ports.ControlDependencies,
) {
	msgType, _ := msg[msgFieldType].(string)
	switch msgType {
	case msgTypeControlResponse:
		a.routeControlResponse(msg)
	case msgTypeControlRequest:
		go a.handleControlRequestAsync(ctx, msg, deps)
	case msgTypeCancelRequest:
		a.handleCancelRequest(msg)
	default:
		forwardSDKMessage(ctx, msg, msgCh)
	}
}

// forwardSDKMessage forwards SDK messages to the public stream.
func forwardSDKMessage(
	ctx context.Context,
	msg map[string]any,
	msgCh chan<- map[string]any,
) {
	select {
	case msgCh <- msg:
	case <-ctx.Done():
	}
}

// handleTransportError processes errors from the transport.
// Returns true if the router should exit.
func handleTransportError(
	ctx context.Context,
	err error,
	errCh chan<- error,
) bool {
	if err == nil {
		return false
	}

	select {
	case errCh <- err:
	case <-ctx.Done():
	}

	return true
}
