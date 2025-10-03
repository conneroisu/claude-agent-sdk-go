// Package querying handles one-shot query execution.
//
// This is a domain service that encapsulates the business logic for
// executing one-shot queries to Claude without managing conversation state.
package querying

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Service handles query execution.
//
// This is a DOMAIN service - contains only business logic,
// no infrastructure concerns like protocol state management.
type Service struct {
	transport   ports.Transport
	protocol    ports.ProtocolHandler
	parser      ports.MessageParser
	hooks       *hooking.Service
	permissions *permissions.Service
	mcpServers  map[string]ports.MCPServer
}

// NewService creates a new querying service.
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
	}
}

// Execute runs a one-shot query and streams results.
//
// Returns channels for messages and errors. The channels are closed
// when the query completes or an error occurs.
func (s *Service) Execute(
	ctx context.Context,
	prompt string,
	opts *options.AgentOptions,
) (<-chan messages.Message, <-chan error) {
	msgCh := make(chan messages.Message)
	errCh := make(chan error, 1)

	go s.executeAsync(ctx, prompt, opts, msgCh, errCh)

	return msgCh, errCh
}

// executeAsync runs the query execution in a goroutine.
func (s *Service) executeAsync(
	ctx context.Context,
	prompt string,
	opts *options.AgentOptions,
	msgCh chan<- messages.Message,
	errCh chan<- error,
) {
	defer close(msgCh)
	defer close(errCh)

	// Connect transport
	if err := s.transport.Connect(ctx); err != nil {
		errCh <- fmt.Errorf("transport connect: %w", err)

		return
	}

	// Build hook callbacks and permission service adapter, then start routing
	if err := s.startRouting(ctx, msgCh, errCh); err != nil {
		errCh <- err

		return
	}

	// Send prompt
	if err := s.sendPrompt(ctx, prompt); err != nil {
		errCh <- err
	}
}

// startRouting builds hook callbacks and starts message routing.
func (s *Service) startRouting(
	ctx context.Context,
	msgCh chan<- messages.Message,
	errCh chan<- error,
) error {
	hookCallbacks := s.buildHookCallbacks()
	permAdapter := s.wrapPermissionService()

	routerMsgCh := make(chan map[string]any)
	routerErrCh := make(chan error, 1)

	if err := s.protocol.StartMessageRouter(
		ctx,
		routerMsgCh,
		routerErrCh,
		permAdapter,
		hookCallbacks,
		s.mcpServers,
	); err != nil {
		return fmt.Errorf("start message router: %w", err)
	}

	go s.streamMessages(ctx, routerMsgCh, routerErrCh, msgCh, errCh)

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

// sendPrompt sends the user prompt to the transport.
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

// streamMessages reads from router and parses messages.
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
			return
		case msg, ok := <-routerMsgCh:
			if !ok {
				return
			}
			if err := s.parseAndSend(msg, msgCh, errCh); err != nil {
				errCh <- err

				return
			}
		case err := <-routerErrCh:
			if err != nil {
				errCh <- err

				return
			}
		}
	}
}

// parseAndSend parses a raw message and sends it to the output channel.
func (s *Service) parseAndSend(
	raw map[string]any,
	msgCh chan<- messages.Message,
	errCh chan<- error,
) error {
	parsedMsg, err := s.parser.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse message: %w", err)
	}
	msgCh <- parsedMsg

	return nil
}
