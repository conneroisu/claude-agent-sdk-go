package querying

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Execute performs a one-shot query with automatic lifecycle management.
// It connects to the transport, sends the prompt, streams parsed messages,
// and automatically cleans up resources on completion or error.
//
// The function returns two channels:
//   - Message channel: Receives parsed messages from Claude
//   - Error channel (buffered): Receives errors during execution
//
// Both channels are closed when the query completes or fails.
func (s *Service) Execute(
	ctx context.Context,
	prompt string,
) (<-chan messages.Message, <-chan error) {
	msgCh := make(chan messages.Message)
	errCh := make(chan error, 1)

	go s.executeAsync(ctx, prompt, msgCh, errCh)

	return msgCh, errCh
}

func (s *Service) executeAsync(
	ctx context.Context,
	prompt string,
	msgCh chan<- messages.Message,
	errCh chan<- error,
) {
	defer close(msgCh)
	defer close(errCh)

	if err := s.transport.Connect(ctx); err != nil {
		errCh <- fmt.Errorf("transport connect: %w", err)

		return
	}
	defer func() {
		_ = s.transport.Close()
		// Ignore close error as channels are already closed
	}()

	routerMsgCh, routerErrCh := s.startRouter(ctx)
	if err := s.sendPrompt(ctx, prompt); err != nil {
		errCh <- fmt.Errorf("send prompt: %w", err)

		return
	}

	s.streamMessages(ctx, routerMsgCh, routerErrCh, msgCh, errCh)
}

func (s *Service) startRouter(
	ctx context.Context,
) (<-chan map[string]any, <-chan error) {
	msgCh := make(chan map[string]any)
	errCh := make(chan error, 1)

	// Convert hooks to ports.HookCallback map
	var hookCallbacks map[string]ports.HookCallback
	if s.hooks != nil {
		rawCallbacks := s.hooks.GetCallbacks()
		hookCallbacks = make(map[string]ports.HookCallback, len(rawCallbacks))
		for id, cb := range rawCallbacks {
			callback := cb // Capture for closure
			// Wrap hooking.HookCallback to match ports.HookCallback
			hookCallbacks[id] = func(
				input map[string]any,
			) (map[string]any, error) {
				// Call with nil toolUseID and empty context
				return callback(input, nil, hooking.HookContext{})
			}
		}
	}

	deps := ports.ControlDependencies{
		Hooks:      hookCallbacks,
		Perms:      s.perms,
		MCPServers: s.mcpServers,
	}

	go func() {
		_ = s.protocol.StartMessageRouter(ctx, msgCh, errCh, deps)
	}()

	return msgCh, errCh
}

func (s *Service) sendPrompt(
	ctx context.Context,
	prompt string,
) error {
	msg := map[string]any{
		"type":   "user",
		"prompt": prompt,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal prompt: %w", err)
	}

	return s.transport.Write(ctx, string(data)+"\n")
}

//nolint:revive // 5 params acceptable for stream coordination
func (s *Service) streamMessages(
	ctx context.Context,
	routerMsgCh <-chan map[string]any,
	routerErrCh <-chan error,
	msgCh chan<- messages.Message,
	errCh chan<- error,
) {
	for {
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()

			return

		case rawMsg, ok := <-routerMsgCh:
			if !ok {
				return
			}
			if err := s.processMessage(rawMsg, msgCh); err != nil {
				errCh <- err

				return
			}

		case err, ok := <-routerErrCh:
			if ok && err != nil {
				errCh <- err
			}

			return
		}
	}
}

func (s *Service) processMessage(
	raw map[string]any,
	msgCh chan<- messages.Message,
) error {
	msg, err := s.parser.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse message: %w", err)
	}

	msgCh <- msg

	return nil
}
