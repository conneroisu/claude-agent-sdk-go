package claude

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/streaming"
)

// StreamSession represents an active streaming conversation.
type StreamSession struct {
	service *streaming.Service
}

// Stream creates a bidirectional streaming conversation.
// This is for interactive, multi-turn conversations.
func (c *Client) Stream(ctx context.Context) (*StreamSession, error) {
	cfg := &streaming.Config{
		Transport:   c.transport,
		Protocol:    c.protocol,
		Parser:      c.parser,
		Hooks:       c.hooks,
		Permissions: c.permissions,
		MCPServers:  c.mcpServers,
	}

	svc := streaming.NewService(cfg)

	if err := svc.Connect(ctx, nil); err != nil {
		return nil, err
	}

	return &StreamSession{service: svc}, nil
}

// Send sends a message to Claude.
func (s *StreamSession) Send(ctx context.Context, prompt string) error {
	return s.service.SendMessage(ctx, prompt)
}

// Receive listens for incoming messages from Claude.
func (s *StreamSession) Receive(
	ctx context.Context,
) (<-chan messages.Message, <-chan error) {
	return s.service.ReceiveMessages(ctx)
}

// Close terminates the streaming session.
func (s *StreamSession) Close() error {
	return s.service.Close()
}

// SendMessage sends a message in streaming mode.
// The client must be connected first.
func (c *Client) SendMessage(ctx context.Context, prompt string) error {
	cfg := &streaming.Config{
		Transport:   c.transport,
		Protocol:    c.protocol,
		Parser:      c.parser,
		Hooks:       c.hooks,
		Permissions: c.permissions,
		MCPServers:  c.mcpServers,
	}

	svc := streaming.NewService(cfg)

	return svc.SendMessage(ctx, prompt)
}

// ReceiveMessages receives messages in streaming mode.
func (c *Client) ReceiveMessages(
	ctx context.Context,
) (<-chan messages.Message, <-chan error) {
	cfg := &streaming.Config{
		Transport:   c.transport,
		Protocol:    c.protocol,
		Parser:      c.parser,
		Hooks:       c.hooks,
		Permissions: c.permissions,
		MCPServers:  c.mcpServers,
	}

	svc := streaming.NewService(cfg)

	return svc.ReceiveMessages(ctx)
}
