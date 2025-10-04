// Package streaming implements bidirectional streaming conversations.
package streaming

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Connect establishes the streaming connection with Claude CLI.
// It connects the transport, builds hook callbacks, starts message routing,
// and optionally sends an initial prompt to begin the conversation.
// Returns error if connection or initialization fails.
func (s *Service) Connect(ctx context.Context, prompt *string) error {
	if err := s.transport.Connect(ctx); err != nil {
		return fmt.Errorf("transport connect: %w", err)
	}

	hookCallbacks := s.buildHookCallbacks()

	if err := s.startRouter(ctx, hookCallbacks); err != nil {
		return err
	}

	if prompt != nil {
		if err := s.sendInitialPrompt(ctx, *prompt); err != nil {
			return err
		}
	}

	return nil
}

// buildHookCallbacks creates the hook callback ID map for the protocol adapter.
// It generates unique IDs for each hook callback that the CLI will reference.
func (s *Service) buildHookCallbacks() map[string]ports.HookCallback {
	if s.hooks == nil {
		return nil
	}

	hookCallbacks := make(map[string]ports.HookCallback)
	hooks := s.hooks.GetHooks()

	for event, matchers := range hooks {
		for _, matcher := range matchers {
			for i, callback := range matcher.Hooks {
				callbackID := fmt.Sprintf("hook_%s_%d", event, i)
				hookCallbacks[callbackID] = adaptHookCallback(callback)
			}
		}
	}

	return hookCallbacks
}

// adaptHookCallback converts a hooking.HookCallback to ports.HookCallback.
func adaptHookCallback(callback hooking.HookCallback) ports.HookCallback {
	return func(
		ctx context.Context,
		input map[string]any,
	) (map[string]any, error) {
		hookCtx := hooking.HookContext{Signal: ctx}
		toolUseID, _ := input["tool_use_id"].(*string)

		return callback(input, toolUseID, hookCtx)
	}
}

// startRouter initializes the protocol message router.
// Protocol adapter handles all control protocol concerns.
func (s *Service) startRouter(
	ctx context.Context,
	hookCallbacks map[string]ports.HookCallback,
) error {
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

	return nil
}

// sendInitialPrompt sends the initial prompt if provided.
func (s *Service) sendInitialPrompt(ctx context.Context, prompt string) error {
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
