//nolint:revive // comments-density: code is self-documenting
package querying

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
)

// Execute performs a one-shot query to Claude.
func (s *Service) Execute(
	ctx context.Context,
	prompt string,
	opts *options.AgentOptions,
) (<-chan messages.Message, <-chan error) {
	msgCh := make(chan messages.Message)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		if err := s.executeQuery(
			ctx,
			prompt,
			opts,
			msgCh,
			errCh,
		); err != nil {
			errCh <- err
		}
	}()

	return msgCh, errCh
}

//nolint:revive // argument-limit,unused-parameter,line-length-limit: justified
// executeQuery runs the query execution logic.
func (s *Service) executeQuery(
	ctx context.Context,
	prompt string,
	opts *options.AgentOptions,
	msgCh chan<- messages.Message,
	errCh chan<- error,
) error {
	if err := s.transport.Connect(ctx); err != nil {
		return fmt.Errorf("transport connect: %w", err)
	}

	hookCallbacks := s.buildHookCallbacks()

	routerMsgCh := make(chan map[string]any)
	routerErrCh := make(chan error, 1)

	if err := s.protocol.StartMessageRouter(
		ctx,
		routerMsgCh,
		routerErrCh,
		s.permissions,
		hookCallbacks,
		s.mcpServers,
	); err != nil {
		return fmt.Errorf("start message router: %w", err)
	}

	if err := s.sendPrompt(ctx, prompt); err != nil {
		return err
	}

	return s.streamMessages(
		ctx,
		routerMsgCh,
		routerErrCh,
		msgCh,
		errCh,
	)
}

// sendPrompt sends the user prompt to Claude.
func (s *Service) sendPrompt(
	ctx context.Context,
	prompt string,
) error {
	promptMsg := map[string]any{
		"type":   "user",
		"prompt": prompt,
	}

	promptBytes, err := json.Marshal(promptMsg)
	if err != nil {
		return fmt.Errorf("marshal prompt: %w", err)
	}

	if err := s.transport.Write(
		ctx,
		string(promptBytes)+"\n",
	); err != nil {
		return fmt.Errorf("write prompt: %w", err)
	}

	return nil
}

// buildHookCallbacks creates hook callback map.
func (s *Service) buildHookCallbacks() map[string]hooking.HookCallback { //nolint:lll
	if s.hooks == nil {
		return nil
	}

	callbacks := make(map[string]hooking.HookCallback)
	hooks := s.hooks.GetHooks()

	for event, matchers := range hooks {
		for _, matcher := range matchers {
			for i, callback := range matcher.Hooks {
				callbackID := fmt.Sprintf(
					"hook_%s_%d",
					event,
					i,
				)
				callbacks[callbackID] = callback
			}
		}
	}

	return callbacks
}
