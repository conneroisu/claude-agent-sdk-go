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

// Service handles streaming conversations
type Service struct {
	transport   ports.Transport
	protocol    ports.ProtocolHandler
	parser      ports.MessageParser
	hooks       *hooking.Service
	permissions *permissions.Service
	mcpServers  map[string]ports.MCPServer

	// Message routing channels
	msgCh chan map[string]any
	errCh chan error
}

// NewService creates a new streaming service
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

// Connect establishes connection to Claude
func (s *Service) Connect(ctx context.Context, prompt any) error {
	// Connect transport
	if err := s.transport.Connect(ctx); err != nil {
		return fmt.Errorf("transport connect: %w", err)
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
	deps := ports.ProtocolDependencies{
		Permissions: s.permissions,
		Hooks:       hookCallbacks,
		MCPServers:  s.mcpServers,
	}

	if err := s.protocol.StartMessageRouter(ctx, s.msgCh, s.errCh, deps); err != nil {
		return fmt.Errorf("start message router: %w", err)
	}

	// Send initial prompt if provided
	if prompt != nil {
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
	}

	return nil
}

// SendMessage sends a message to Claude
func (s *Service) SendMessage(ctx context.Context, msg string) error {
	// Format message
	userMsg := map[string]any{
		"type":   "user",
		"prompt": msg,
	}

	// Send via transport
	msgBytes, err := json.Marshal(userMsg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	if err := s.transport.Write(ctx, string(msgBytes)+"\n"); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// ReceiveMessages returns a channel of messages from Claude
func (s *Service) ReceiveMessages(ctx context.Context) (<-chan messages.Message, <-chan error) {
	msgOutCh := make(chan messages.Message)
	errOutCh := make(chan error, 1)

	go func() {
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

				// Parse message using parser port
				parsedMsg, err := s.parser.Parse(msg)
				if err != nil {
					errOutCh <- fmt.Errorf("parse message: %w", err)

					return
				}
				msgOutCh <- parsedMsg

			case err := <-s.errCh:
				if err != nil {
					errOutCh <- err

					return
				}
			}
		}
	}()

	return msgOutCh, errOutCh
}

// Close disconnects from Claude
func (s *Service) Close() error {
	if s.transport != nil {
		return s.transport.Close()
	}

	return nil
}
