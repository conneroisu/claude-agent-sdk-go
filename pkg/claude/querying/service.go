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

// Dependencies groups all external dependencies for the querying service.
type Dependencies struct {
	Transport   ports.Transport
	Protocol    ports.ProtocolHandler
	Parser      ports.MessageParser
	Hooks       *hooking.Service
	Permissions *permissions.Service
	MCPServers  map[string]ports.MCPServer
}

// Service handles query execution
// This is a DOMAIN service - it contains only business logic,
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
func NewService(deps Dependencies) *Service {
	return &Service{
		transport:   deps.Transport,
		protocol:    deps.Protocol,
		parser:      deps.Parser,
		hooks:       deps.Hooks,
		permissions: deps.Permissions,
		mcpServers:  deps.MCPServers,
	}
}

// Execute executes a one-shot query.
func (s *Service) Execute(
	ctx context.Context,
	prompt string,
	_opts *options.AgentOptions,
) (<-chan messages.Message, <-chan error) {
	msgCh := make(chan messages.Message)
	errCh := make(chan error, 1)
	go func() {
		defer close(msgCh)
		defer close(errCh)
		if err := s.transport.Connect(ctx); err != nil {
			errCh <- fmt.Errorf("transport connect: %w", err)

			return
		}
		hookCallbacks := s.buildHookCallbacks()
		routerMsgCh, routerErrCh, err := s.startRouter(ctx, hookCallbacks)
		if err != nil {
			errCh <- err

			return
		}
		if err := s.sendPrompt(ctx, prompt); err != nil {
			errCh <- err

			return
		}
		channels := streamChannels{
			routerMsg: routerMsgCh,
			routerErr: routerErrCh,
			msg:       msgCh,
			err:       errCh,
		}
		s.streamMessages(ctx, channels)
	}()

	return msgCh, errCh
}

// startRouter initializes the protocol message router.
func (s *Service) startRouter(
	ctx context.Context,
	hookCallbacks map[string]ports.HookCallback,
) (chan map[string]any, chan error, error) {
	routerMsgCh := make(chan map[string]any)
	routerErrCh := make(chan error, 1)
	deps := ports.ControlDependencies{
		Permissions: s.permissions,
		Hooks:       hookCallbacks,
		MCPServers:  s.mcpServers,
	}
	err := s.protocol.StartMessageRouter(
		ctx,
		routerMsgCh,
		routerErrCh,
		deps,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"start message router: %w",
			err,
		)
	}

	return routerMsgCh, routerErrCh, nil
}

// sendPrompt sends the prompt message to the transport.
func (s *Service) sendPrompt(ctx context.Context, prompt string) error {
	promptMsg := map[string]any{"type": "user", "prompt": prompt}
	promptBytes, err := json.Marshal(promptMsg)
	if err != nil {
		return fmt.Errorf("marshal prompt: %w", err)
	}
	if err := s.transport.Write(ctx, string(promptBytes)+"\n"); err != nil {
		return fmt.Errorf("write prompt: %w", err)
	}

	return nil
}
