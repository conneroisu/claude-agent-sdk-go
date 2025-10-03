//nolint:revive // comments-density: code is self-documenting
package streaming

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
)

// Connect establishes connection to Claude.
func (s *Service) Connect(
	ctx context.Context,
	prompt *string,
) error {
	if err := s.transport.Connect(ctx); err != nil {
		return fmt.Errorf("transport connect: %w", err)
	}

	hookCallbacks := s.buildHookCallbacks()

	if err := s.protocol.StartMessageRouter(
		ctx,
		s.msgCh,
		s.errCh,
		s.permissions,
		hookCallbacks,
		s.mcpServers,
	); err != nil {
		return fmt.Errorf("start message router: %w", err)
	}

	if prompt != nil {
		return s.sendInitialPrompt(ctx, *prompt)
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

// sendInitialPrompt sends the initial prompt if provided.
func (s *Service) sendInitialPrompt(
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
