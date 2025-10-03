// Package streaming handles bidirectional streaming conversations.
//
// This is a domain service that manages conversation flow and state
// for interactive sessions with Claude.
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

// Service handles streaming conversations.
//
// This is a DOMAIN service - pure business logic for managing conversations.
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

// NewService creates a new streaming service.
func NewService(
	transport ports.Transport,
	protocol ports.ProtocolHandler,
	parser ports.MessageParser,
	hooks *hooking.Service,
	perms *permissions.Service,
	mcpServers map[string]ports.MCPServer,
) *Service {
	return &Service{
		transport:   transport,
		protocol:    protocol,
		parser:      parser,
		hooks:       hooks,
		permissions: perms,
		mcpServers:  mcpServers,
		msgCh:       make(chan map[string]any),
		errCh:       make(chan error, 1),
	}
}

// Connect establishes connection and optionally sends initial prompt.
//
// The prompt parameter is optional - pass nil for no initial prompt.
func (s *Service) Connect(
	ctx context.Context,
	prompt *string,
) error {
	// Connect transport
	if err := s.transport.Connect(ctx); err != nil {
		return fmt.Errorf("transport connect: %w", err)
	}

	// Build hook callbacks and permission service adapter
	hookCallbacks := s.buildHookCallbacks()
	permAdapter := s.wrapPermissionService()

	// Start message router
	if err := s.protocol.StartMessageRouter(
		ctx,
		s.msgCh,
		s.errCh,
		permAdapter,
		hookCallbacks,
		s.mcpServers,
	); err != nil {
		return fmt.Errorf("start message router: %w", err)
	}

	// Send initial prompt if provided
	if prompt != nil {
		if err := s.sendPrompt(ctx, *prompt); err != nil {
			return err
		}
	}

	return nil
}

// buildHookCallbacks constructs hook callback map from hook service.
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
				// Cast hooking.HookCallback to ports.HookCallback
				hookCallbacks[callbackID] = wrapHookCallback(callback)
			}
		}
	}

	return hookCallbacks
}

// wrapHookCallback wraps a hooking.HookCallback to match ports.HookCallback signature.
func wrapHookCallback(cb hooking.HookCallback) ports.HookCallback {
	return func(input map[string]any, toolUseID *string, ctx any) (map[string]any, error) {
		// Convert any to hooking.HookContext
		// The protocol handler passes context.Context as any
		var hookCtx hooking.HookContext
		switch c := ctx.(type) {
		case hooking.HookContext:
			hookCtx = c
		case context.Context:
			hookCtx = hooking.HookContext{Signal: c}
		default:
			hookCtx = hooking.HookContext{}
		}
		return cb(input, toolUseID, hookCtx)
	}
}

// wrapPermissionService wraps the permissions.Service to match ports.PermissionService.
func (s *Service) wrapPermissionService() ports.PermissionService {
	if s.permissions == nil {
		return nil
	}
	return &permissionServiceAdapter{service: s.permissions}
}

// permissionServiceAdapter adapts permissions.Service to ports.PermissionService.
type permissionServiceAdapter struct {
	service *permissions.Service
}

func (a *permissionServiceAdapter) CheckToolUse(
	ctx context.Context,
	toolName string,
	input map[string]any,
	suggestions any,
) (any, error) {
	// Convert suggestions from any to []permissions.PermissionUpdate
	var permSuggestions []permissions.PermissionUpdate
	if suggestions != nil {
		if suggList, ok := suggestions.([]permissions.PermissionUpdate); ok {
			permSuggestions = suggList
		}
	}
	return a.service.CheckToolUse(ctx, toolName, input, permSuggestions)
}

// sendPrompt sends a prompt message to the transport.
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

// SendMessage sends a user message in the conversation.
func (s *Service) SendMessage(ctx context.Context, msg string) error {
	return s.sendPrompt(ctx, msg)
}

// ReceiveMessages returns channels for streaming messages and errors.
//
// Messages are parsed from the raw transport messages into typed
// domain messages. The channels are kept open until the connection closes.
func (s *Service) ReceiveMessages(
	ctx context.Context,
) (<-chan messages.Message, <-chan error) {
	msgOutCh := make(chan messages.Message)
	errOutCh := make(chan error, 1)

	go s.streamMessages(ctx, msgOutCh, errOutCh)

	return msgOutCh, errOutCh
}

// streamMessages reads from internal channels and parses messages.
func (s *Service) streamMessages(
	ctx context.Context,
	msgOutCh chan<- messages.Message,
	errOutCh chan<- error,
) {
	defer close(msgOutCh)
	defer close(errOutCh)

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-s.msgCh:
			if !ok {
				return
			}
			if err := s.parseAndSend(msg, msgOutCh, errOutCh); err != nil {
				errOutCh <- err
				return
			}
		case err := <-s.errCh:
			if err != nil {
				errOutCh <- err
				return
			}
		}
	}
}

// parseAndSend parses a raw message and sends it to the output channel.
func (s *Service) parseAndSend(
	raw map[string]any,
	msgOutCh chan<- messages.Message,
	errOutCh chan<- error,
) error {
	parsedMsg, err := s.parser.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse message: %w", err)
	}
	msgOutCh <- parsedMsg
	return nil
}

// Close closes the transport connection.
func (s *Service) Close() error {
	if s.transport != nil {
		return s.transport.Close()
	}
	return nil
}
