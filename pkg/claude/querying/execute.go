// Package querying implements one-shot query execution against Claude CLI.
// It provides domain services for executing single queries that complete
// when the AI finishes responding, as opposed to streaming conversations.
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

// Execute performs a one-shot query against Claude CLI.
// It connects the transport, starts message routing, sends the prompt,
// and streams back parsed messages and errors.
// Returns channels for messages and errors that close when complete.
func (s *Service) Execute(
	ctx context.Context,
	prompt string,
	_ *options.AgentOptions,
) (<-chan messages.Message, <-chan error) {
	msgCh := make(chan messages.Message)
	errCh := make(chan error, 1)

	// Launch goroutine to handle query execution
	go func() {
		defer close(msgCh)
		defer close(errCh)

		if err := s.executeQuery(ctx, prompt, msgCh, errCh); err != nil {
			errCh <- err
		}
	}()

	return msgCh, errCh
}

// executeQuery runs the query execution logic.
// It orchestrates the connection, routing, and message streaming.
func (s *Service) executeQuery(
	ctx context.Context,
	prompt string,
	msgCh chan messages.Message,
	_ chan error,
) error {
	// Step 1: Connect transport
	if err := s.transport.Connect(ctx); err != nil {
		return fmt.Errorf("transport connect: %w", err)
	}

	// Step 2: Build hook callback map
	hookCallbacks := s.buildHookCallbacks()

	// Step 3: Create routing channels
	routerMsgCh := make(chan map[string]any)
	routerErrCh := make(chan error, 1)

	// Step 4: Start message router
	err := s.startRouter(ctx, routerMsgCh, routerErrCh, hookCallbacks)
	if err != nil {
		return err
	}

	// Step 5: Send the user prompt
	if err := s.sendPrompt(ctx, prompt); err != nil {
		return err
	}

	// Step 6: Stream messages back to caller
	return s.streamMessages(ctx, routerMsgCh, routerErrCh, msgCh)
}

// buildHookCallbacks creates the hook callback ID map for the protocol adapter.
// Each hook is assigned a unique ID that the CLI will reference when
// invoking callbacks during execution.
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
func (s *Service) startRouter(
	ctx context.Context,
	routerMsgCh chan map[string]any,
	routerErrCh chan error,
	hookCallbacks map[string]ports.HookCallback,
) error {
	err := s.protocol.StartMessageRouter(
		ctx,
		routerMsgCh,
		routerErrCh,
		s.permissions,
		hookCallbacks,
		s.mcpServers,
	)
	if err != nil {
		return fmt.Errorf("start message router: %w", err)
	}

	return nil
}

// sendPrompt sends the user prompt to the CLI.
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

// streamMessages reads and parses messages from the router.
func (s *Service) streamMessages(
	ctx context.Context,
	routerMsgCh chan map[string]any,
	routerErrCh chan error,
	msgCh chan messages.Message,
) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		case msg, ok := <-routerMsgCh:
			if !ok {
				return nil
			}
			if err := s.handleMessage(msg, msgCh); err != nil {
				return err
			}

		case err := <-routerErrCh:
			if err != nil {
				return err
			}
		}
	}
}

// handleMessage parses and sends a message to the output channel.
// It delegates parsing to the message parser.
func (s *Service) handleMessage(
	msg map[string]any,
	msgCh chan messages.Message,
) error {
	parsedMsg, err := s.parser.Parse(msg)
	if err != nil {
		return fmt.Errorf("parse message: %w", err)
	}
	msgCh <- parsedMsg

	return nil
}
