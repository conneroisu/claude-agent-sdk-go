package claude

import (
	"context"
	"errors"
	"sync"

	"github.com/conneroisu/claude/pkg/claude/adapters/cli"
	"github.com/conneroisu/claude/pkg/claude/adapters/jsonrpc"
	"github.com/conneroisu/claude/pkg/claude/adapters/parse"
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/streaming"
)

// Client provides a persistent bidirectional conversation interface.
// All methods are thread-safe.
type Client struct {
	opts  *options.AgentOptions
	hooks map[hooking.HookEvent][]hooking.HookMatcher
	perms *permissions.PermissionsConfig

	mu    sync.Mutex
	svc   *streaming.Service
	msgCh <-chan messages.Message
	errCh <-chan error
}

// NewClient creates a new Claude client with configuration.
// Call Connect() to establish the connection.
func NewClient(
	opts *options.AgentOptions,
	hooks map[hooking.HookEvent][]hooking.HookMatcher,
	perms *permissions.PermissionsConfig,
) *Client {
	clientOpts := opts
	if clientOpts == nil {
		clientOpts = &options.AgentOptions{}
	}

	return &Client{
		opts:  clientOpts,
		hooks: hooks,
		perms: perms,
	}
}

// Connect establishes the Claude CLI connection and initializes all services.
// It blocks until connection is established or fails.
// Returns error if already connected or if connection fails.
func (c *Client) Connect(
	ctx context.Context,
	initialPrompt *string,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.svc != nil {
		return errors.New("already connected")
	}

	c.svc = c.wireServices()

	return c.svc.Connect(ctx, initialPrompt)
}

func (c *Client) wireServices() *streaming.Service {
	transport := cli.NewAdapter(c.opts)
	protocol := jsonrpc.NewAdapter(transport)
	parser := parse.NewAdapter()

	hookSvc := hooking.NewService(c.hooks)
	permSvc := permissions.NewService(c.perms)

	return streaming.NewService(
		transport,
		protocol,
		parser,
		hookSvc,
		permSvc,
		nil,
	)
}

// SendMessage sends a user message to Claude in the active conversation.
// Returns error if not connected or if write fails.
func (c *Client) SendMessage(ctx context.Context, msg string) error {
	c.mu.Lock()
	svc := c.svc
	c.mu.Unlock()

	if svc == nil {
		return ErrNotConnected
	}

	return svc.SendMessage(ctx, msg)
}

// ReceiveMessages returns channels for streaming messages and errors.
// The same channels are returned on multiple calls.
// Returns error channel with ErrNotConnected if not connected.
func (c *Client) ReceiveMessages(
	ctx context.Context,
) (<-chan messages.Message, <-chan error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.svc == nil {
		errCh := make(chan error, 1)
		msgCh := make(chan messages.Message)
		errCh <- ErrNotConnected
		close(errCh)
		close(msgCh)

		return msgCh, errCh
	}

	if c.msgCh == nil {
		c.msgCh, c.errCh = c.svc.ReceiveMessages(ctx)
	}

	return c.msgCh, c.errCh
}

// Close disconnects from Claude and releases all resources.
// Safe to call multiple times.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.svc == nil {
		return nil
	}

	err := c.svc.Close()
	c.svc = nil
	c.msgCh = nil
	c.errCh = nil

	return err
}
