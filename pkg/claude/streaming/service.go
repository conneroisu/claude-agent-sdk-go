// Package streaming provides domain services for managing streaming
// conversations with the Claude API.
package streaming

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Dependencies groups all external dependencies for the streaming service
type Dependencies struct {
	Transport   ports.Transport
	Protocol    ports.ProtocolHandler
	Parser      ports.MessageParser
	Hooks       *hooking.Service
	Permissions *permissions.Service
	MCPServers  map[string]ports.MCPServer
}

// Service handles streaming conversations
// This is a DOMAIN service - pure business logic for managing conversations
type Service struct {
	transport   ports.Transport
	protocol    ports.ProtocolHandler
	parser      ports.MessageParser
	hooks       *hooking.Service
	permissions *permissions.Service
	mcpServers  map[string]ports.MCPServer

	// Message routing channels (internal to service)
	msgCh chan map[string]any
	errCh chan error
}

// NewService creates a new streaming service
func NewService(deps Dependencies) *Service {
	return &Service{
		transport:   deps.Transport,
		protocol:    deps.Protocol,
		parser:      deps.Parser,
		hooks:       deps.Hooks,
		permissions: deps.Permissions,
		mcpServers:  deps.MCPServers,
		msgCh:       make(chan map[string]any),
		errCh:       make(chan error, 1),
	}
}

// Connect initializes the streaming connection
func (s *Service) Connect(ctx context.Context, prompt *string) error {
	if err := s.transport.Connect(ctx); err != nil {
		return fmt.Errorf("transport connect: %w", err)
	}

	hookCallbacks := s.buildHookCallbacks()

	deps := ports.ControlDependencies{
		Permissions: s.permissions,
		Hooks:       hookCallbacks,
		MCPServers:  s.mcpServers,
	}

	err := s.protocol.StartMessageRouter(ctx, s.msgCh, s.errCh, deps)
	if err != nil {
		return err
	}

	if prompt != nil {
		return s.SendMessage(ctx, *prompt)
	}

	return nil
}

// buildHookCallbacks creates hook callbacks map from hooking service
func (s *Service) buildHookCallbacks() map[string]ports.HookCallback {
	if s.hooks == nil {
		return nil
	}
	hookCallbacks := make(map[string]ports.HookCallback)
	for event, matchers := range s.hooks.GetHooks() {
		for _, matcher := range matchers {
			for i, callback := range matcher.Hooks {
				cb := callback // capture for closure
				callbackID := fmt.Sprintf("hook_%s_%d", string(event), i)
				hookCallbacks[callbackID] = func(
					input map[string]any,
					toolUseID *string,
					ctx any,
				) (map[string]any, error) {
					return cb(input, toolUseID, convertToHookContext(ctx))
				}
			}
		}
	}

	return hookCallbacks
}

// convertToHookContext converts context to hooking.HookContext
func convertToHookContext(ctx any) hooking.HookContext {
	if hookCtx, ok := ctx.(hooking.HookContext); ok {
		return hookCtx
	}
	if c, ok := ctx.(context.Context); ok {
		return hooking.HookContext{Signal: c}
	}

	return hooking.HookContext{Signal: context.Background()}
}

// SendMessage sends a message to the agent
func (s *Service) SendMessage(ctx context.Context, msg string) error {
	msgBytes, err := json.Marshal(map[string]any{
		"type":   "user",
		"prompt": msg,
	})
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	return s.transport.Write(ctx, string(msgBytes)+"\n")
}

// ReceiveMessages returns channels for receiving messages and errors
func (s *Service) ReceiveMessages(
	ctx context.Context,
) (<-chan messages.Message, <-chan error) {
	msgOutCh := make(chan messages.Message)
	errOutCh := make(chan error, 1)

	go s.messageLoop(ctx, msgOutCh, errOutCh)

	return msgOutCh, errOutCh
}

// messageLoop processes messages in a separate goroutine
func (s *Service) messageLoop(
	ctx context.Context,
	msgOutCh chan messages.Message,
	errOutCh chan error,
) {
	defer close(msgOutCh)
	defer close(errOutCh)

	for {
		if !s.processNextMessage(ctx, msgOutCh, errOutCh) {
			return
		}
	}
}

// processNextMessage handles one message cycle, returns false to exit loop
func (s *Service) processNextMessage(
	ctx context.Context,
	msgOutCh chan messages.Message,
	errOutCh chan error,
) bool {
	select {
	case <-ctx.Done():
		return false
	case msg, ok := <-s.msgCh:
		if !ok {
			return false
		}

		return s.handleReceivedMessage(msg, msgOutCh, errOutCh)
	case err := <-s.errCh:
		return s.handleStreamError(err, errOutCh)
	}
}

// handleReceivedMessage processes message, returns false to exit loop
func (s *Service) handleReceivedMessage(
	msg map[string]any,
	msgOutCh chan messages.Message,
	errOutCh chan error,
) bool {
	parsedMsg, err := s.parser.Parse(msg)
	if err != nil {
		errOutCh <- fmt.Errorf("parse message: %w", err)

		return false
	}

	msgOutCh <- parsedMsg

	return true
}

// handleStreamError processes stream error, returns false to exit loop
func (*Service) handleStreamError(err error, errOutCh chan error) bool {
	if err != nil {
		errOutCh <- err

		return false
	}

	return true
}

// Close closes the streaming connection
func (s *Service) Close() error {
	if s.transport != nil {
		return s.transport.Close()
	}

	return nil
}
