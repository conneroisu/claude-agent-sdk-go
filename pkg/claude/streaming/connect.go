// Package streaming provides connection management for streaming conversations.
package streaming

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Connect establishes a streaming connection.
// Initializes transport, hooks, router, and sends initial prompt if provided.
func (s *Service) Connect(ctx context.Context, prompt *string) error {
	if err := s.transport.Connect(ctx); err != nil {
		return fmt.Errorf("transport connect: %w", err)
	}

	hookCallbacks := s.buildHookCallbacks()
	if err := s.startRouter(ctx, hookCallbacks); err != nil {
		return err
	}

	if prompt != nil {
		return s.sendInitialPrompt(ctx, *prompt)
	}

	return nil
}

// buildHookCallbacks creates the hook callback map.
// This maps generated callback IDs to hook functions.
func (s *Service) buildHookCallbacks() map[string]hooking.HookCallback {
	if s.hooks == nil {
		return nil
	}

	callbacks := make(map[string]hooking.HookCallback)
	hooks := s.hooks.GetHooks()

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
func (s *Service) startRouter(
	ctx context.Context,
	hookCallbacks map[string]hooking.HookCallback,
) error {
	deps := ports.ControlDependencies{
		PermissionsService: s.permissions,
		HookCallbacks:      hookCallbacks,
		MCPServers:         s.mcpServers,
	}

	if err := s.protocol.StartMessageRouter(
		ctx,
		s.msgCh,
		s.errCh,
		deps,
	); err != nil {
		return fmt.Errorf("start message router: %w", err)
	}

	return nil
}

// sendInitialPrompt sends the first message.
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

	if err := s.transport.Write(ctx, string(promptBytes)+"\n"); err != nil {
		return fmt.Errorf("write prompt: %w", err)
	}

	return nil
}
