package querying

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Execute runs a one-shot query and returns message channels.
// The returned channels stream messages from Claude until completion or error.
// opts parameter is reserved for future use and currently unused.
func (s *Service) Execute(
	ctx context.Context,
	prompt string,
	_opts *options.AgentOptions,
) (<-chan messages.Message, <-chan error) {
	msgCh := make(chan messages.Message)
	errCh := make(chan error, 1)

	// Execute query in background goroutine to enable async streaming
	go func() {
		defer close(msgCh)
		defer close(errCh)

		if err := s.executeQuery(ctx, prompt, msgCh, errCh); err != nil {
			errCh <- err
		}
	}()

	return msgCh, errCh
}

// executeQuery handles the query execution logic.
// Connects transport, initializes hooks, starts router, and sends prompt.
func (s *Service) executeQuery(
	ctx context.Context,
	prompt string,
	msgCh chan<- messages.Message,
	errCh chan<- error,
) error {
	if err := s.transport.Connect(ctx); err != nil {
		return fmt.Errorf("transport connect: %w", err)
	}

	hookCallbacks := s.buildHookCallbacks()
	if err := s.startRouter(ctx, msgCh, errCh, hookCallbacks); err != nil {
		return err
	}

	return s.sendPrompt(ctx, prompt)
}

// buildHookCallbacks creates the hook callback map.
// Generates unique callback IDs for control protocol routing.
func (s *Service) buildHookCallbacks() map[string]hooking.HookCallback {
	if s.hooks == nil {
		return nil
	}

	callbacks := make(map[string]hooking.HookCallback)
	hooks := s.hooks.GetHooks()

	// Generate callback IDs in format: hook_{event}_{index}
	for event, matchers := range hooks {
		for _, matcher := range matchers {
			for i, callback := range matcher.Hooks {
				callbackID := fmt.Sprintf("hook_%s_%d", event, i)
				callbacks[callbackID] = callback
			}
		}
	}

	return callbacks
}

// startRouter initializes the message router.
// Creates control dependencies and starts protocol handler.
func (s *Service) startRouter(
	ctx context.Context,
	msgCh chan<- messages.Message,
	errCh chan<- error,
	hookCallbacks map[string]hooking.HookCallback,
) error {
	routerMsgCh := make(chan map[string]any)
	routerErrCh := make(chan error, 1)

	deps := ports.ControlDependencies{
		PermissionsService: s.permissions,
		HookCallbacks:      hookCallbacks,
		MCPServers:         s.mcpServers,
	}

	if err := s.protocol.StartMessageRouter(
		ctx,
		routerMsgCh,
		routerErrCh,
		deps,
	); err != nil {
		return fmt.Errorf("start message router: %w", err)
	}

	go s.routeMessages(ctx, routeMessagesParams{
		routerMsgCh: routerMsgCh,
		routerErrCh: routerErrCh,
		msgCh:       msgCh,
		errCh:       errCh,
	})

	return nil
}

// routeMessagesParams groups routing parameters to comply with argument limits.
type routeMessagesParams struct {
	routerMsgCh <-chan map[string]any
	routerErrCh <-chan error
	msgCh       chan<- messages.Message
	errCh       chan<- error
}

// routeMessages routes parsed messages to output channels.
// Continuously reads from router channels and forwards to output.
func (s *Service) routeMessages(
	ctx context.Context,
	params routeMessagesParams,
) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg, ok := <-params.routerMsgCh:
			if !ok {
				return
			}

			if err := s.parseAndSendMessage(msg, params.msgCh); err != nil {
				params.errCh <- err

				return
			}

		case err := <-params.routerErrCh:
			if err != nil {
				params.errCh <- err

				return
			}
		}
	}
}

// parseAndSendMessage parses and sends a message.
func (s *Service) parseAndSendMessage(
	raw map[string]any,
	msgCh chan<- messages.Message,
) error {
	parsedMsg, err := s.parser.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse message: %w", err)
	}
	msgCh <- parsedMsg

	return nil
}

// sendPrompt sends the initial prompt.
func (s *Service) sendPrompt(ctx context.Context, prompt string) error {
	promptMsg := map[string]any{
		"type":   "user",
		"prompt": prompt,
	}

	promptBytes, err := json.Marshal(promptMsg)
	if err != nil {
		return fmt.Errorf("marshal prompt: %w", err)
	}

	if err := s.transport.Write(ctx, string(promptBytes)+"\n"); err != nil {
		return fmt.Errorf("write prompt: %w", err)
	}

	return nil
}
