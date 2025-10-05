package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Connect establishes the streaming session and optionally sends an
// initial prompt. It must be called before SendMessage or ReceiveMessages.
// The connection process includes transport setup, router initialization,
// and hook registration.
func (s *Service) Connect(
	ctx context.Context,
	initialPrompt *string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Prevent double connection
	if s.connected {
		return errors.New("already connected")
	}

	// Establish transport connection
	if err := s.transport.Connect(ctx); err != nil {
		return fmt.Errorf("transport connect: %w", err)
	}

	// Initialize message router with cleanup on failure
	if err := s.initializeRouter(ctx); err != nil {
		_ = s.transport.Close()

		return err
	}

	// Register hooks with cleanup on failure
	if err := s.initializeHooks(ctx); err != nil {
		_ = s.transport.Close()

		return err
	}

	// Send initial prompt if provided
	if initialPrompt != nil {
		if err := s.sendPromptUnsafe(ctx, *initialPrompt); err != nil {
			_ = s.transport.Close()

			return err
		}
	}

	s.connected = true

	return nil
}

// initializeRouter sets up the message routing infrastructure.
// It creates channels and converts hook callbacks for the protocol
// layer.
func (s *Service) initializeRouter(ctx context.Context) error {
	// Create message and error channels
	s.msgCh = make(chan map[string]any)
	s.errCh = make(chan error, 1)

	// Convert hooking callbacks to protocol callbacks
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

	// Build control dependencies
	deps := ports.ControlDependencies{
		Hooks:      hookCallbacks,
		Perms:      s.perms,
		MCPServers: s.mcpServers,
	}

	return s.protocol.StartMessageRouter(ctx, s.msgCh, s.errCh, deps)
}

// initializeHooks registers hook callbacks with the protocol layer.
// Returns nil if no hooks are configured.
func (s *Service) initializeHooks(ctx context.Context) error {
	if s.hooks == nil {
		return nil
	}

	// Get hook configuration mapping
	hookMap := s.hooks.GetHooks()
	if len(hookMap) == 0 {
		return nil
	}

	// Send initialization request to protocol
	req := map[string]any{
		"type":    "control_request",
		"subtype": "initialize",
		"request": map[string]any{
			"hook_callbacks": hookMap,
		},
	}

	_, err := s.protocol.SendControlRequest(ctx, req)

	return err
}

// sendPromptUnsafe sends a prompt without connection state checks.
// Used internally during connection setup.
func (s *Service) sendPromptUnsafe(
	ctx context.Context,
	prompt string,
) error {
	// Build user message
	msg := map[string]any{
		"type":   "user",
		"prompt": prompt,
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal prompt: %w", err)
	}

	// Send with newline delimiter
	return s.transport.Write(ctx, string(data)+"\n")
}
