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

// Service handles query execution
type Service struct {
	transport   ports.Transport
	protocol    ports.ProtocolHandler
	parser      ports.MessageParser
	hooks       *hooking.Service
	permissions *permissions.Service
	mcpServers  map[string]ports.MCPServer
}

// NewService creates a new querying service
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

// Execute performs a one-shot query
func (s *Service) Execute(ctx context.Context, prompt string, opts *options.AgentOptions) (<-chan messages.Message, <-chan error) {
	msgCh := make(chan messages.Message)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		// Connect transport
		if err := s.transport.Connect(ctx); err != nil {
			errCh <- fmt.Errorf("transport connect: %w", err)

			return
		}

		// Build hook callbacks map
		var hookCallbacks map[string]any
		if s.hooks != nil {
			hookCallbacks = make(map[string]any)
			hooks := s.hooks.GetHooks()
			for event, matchers := range hooks {
				for _, matcher := range matchers {
					for i, callback := range matcher.Hooks {
						callbackID := fmt.Sprintf("hook_%s_%d", event, i)
						hookCallbacks[callbackID] = callback
					}
				}
			}
		}

		// Start message router
		routerMsgCh := make(chan map[string]any)
		routerErrCh := make(chan error, 1)

		deps := ports.ProtocolDependencies{
			Permissions: s.permissions,
			Hooks:       hookCallbacks,
			MCPServers:  s.mcpServers,
		}

		if err := s.protocol.StartMessageRouter(ctx, routerMsgCh, routerErrCh, deps); err != nil {
			errCh <- fmt.Errorf("start message router: %w", err)

			return
		}

		// Send prompt
		promptMsg := map[string]any{
			"type":   "user",
			"prompt": prompt,
		}
		promptBytes, err := json.Marshal(promptMsg)
		if err != nil {
			errCh <- fmt.Errorf("marshal prompt: %w", err)

			return
		}

		if err := s.transport.Write(ctx, string(promptBytes)+"\n"); err != nil {
			errCh <- fmt.Errorf("write prompt: %w", err)

			return
		}

		// Stream messages
		for {
			select {
			case <-ctx.Done():
				return

			case msg, ok := <-routerMsgCh:
				if !ok {

					return
				}

				// Parse message using parser port
				parsedMsg, err := s.parser.Parse(msg)
				if err != nil {
					errCh <- fmt.Errorf("parse message: %w", err)

					return
				}
				msgCh <- parsedMsg

			case err := <-routerErrCh:
				if err != nil {
					errCh <- err

					return
				}
			}
		}
	}()

	return msgCh, errCh
}
