package claude

import (
	"context"
	"sync"

	"github.com/conneroisu/claude/pkg/claude/adapters/cli"
	"github.com/conneroisu/claude/pkg/claude/adapters/jsonrpc"
	"github.com/conneroisu/claude/pkg/claude/adapters/parse"
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
	"github.com/conneroisu/claude/pkg/claude/streaming"
)

// Client provides bidirectional, interactive conversations with
// Claude. It's a facade that wires domain services with adapters.
type Client struct {
	opts             *options.AgentOptions
	hooks            map[hooking.HookEvent][]hooking.HookMatcher
	permissions      *permissions.Config
	streamingService *streaming.Service
	mu               sync.Mutex
}

// NewClient creates a new Claude client.
func NewClient(
	opts *options.AgentOptions,
	hooks map[hooking.HookEvent][]hooking.HookMatcher,
	perms *permissions.Config,
) *Client {
	if opts == nil {
		opts = &options.AgentOptions{} //nolint:revive // modifies-parameter: nil check pattern
	}

	return &Client{
		opts:        opts,
		hooks:       hooks,
		permissions: perms,
	}
}

// Connect establishes connection to Claude.
func (c *Client) Connect(
	ctx context.Context,
	prompt *string,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	transport := cli.NewAdapter(c.opts)
	protocol := jsonrpc.NewAdapter(transport)
	parser := parse.NewAdapter()

	var hookingService *hooking.Service
	if c.hooks != nil {
		hookingService = hooking.NewService(c.hooks)
	}

	var permissionsService *permissions.Service
	if c.permissions != nil {
		permissionsService = permissions.NewService(
			c.permissions,
		)
	}

	var mcpServers map[string]ports.MCPServer

	c.streamingService = streaming.NewService(
		transport,
		protocol,
		parser,
		hookingService,
		permissionsService,
		mcpServers,
	)

	return c.streamingService.Connect(ctx, prompt)
}

// SendMessage sends a message to Claude.
func (c *Client) SendMessage(
	ctx context.Context,
	msg string,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.streamingService == nil {
		return ErrNotConnected
	}

	return c.streamingService.SendMessage(ctx, msg)
}

// ReceiveMessages returns a channel of messages from Claude.
func (c *Client) ReceiveMessages(
	ctx context.Context,
) (<-chan messages.Message, <-chan error) {
	if c.streamingService == nil {
		errCh := make(chan error, 1)
		errCh <- ErrNotConnected
		close(errCh)

		return nil, errCh
	}

	return c.streamingService.ReceiveMessages(ctx)
}

// Close disconnects from Claude.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.streamingService == nil {
		return nil
	}

	return c.streamingService.Close()
}
