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

// Client provides bidirectional, interactive conversations with Claude
type Client struct {
	opts             *options.AgentOptions
	hooks            map[hooking.HookEvent][]hooking.HookMatcher
	permissions      *permissions.PermissionsConfig
	mcpServers       map[string]ports.MCPServer
	streamingService *streaming.Service
	mu               sync.Mutex
}

// NewClient creates a new Claude client
func NewClient(
	opts *options.AgentOptions,
	hooks map[hooking.HookEvent][]hooking.HookMatcher,
	perms *permissions.PermissionsConfig,
) *Client {
	if opts == nil {
		opts = &options.AgentOptions{} // nolint:revive
	}

	return &Client{
		opts:        opts,
		hooks:       hooks,
		permissions: perms,
		mcpServers:  make(map[string]ports.MCPServer),
	}
}

// RegisterMCPServer registers an MCP server instance with the client
func (c *Client) RegisterMCPServer(name string, server ports.MCPServer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mcpServers[name] = server
}

// Connect establishes connection to Claude
func (c *Client) Connect(ctx context.Context, prompt any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Mark as streaming
	c.opts.IsStreaming = true

	// Wire up adapters (infrastructure)
	transport := cli.NewAdapter(c.opts)
	protocol := jsonrpc.NewAdapter(transport)
	parser := parse.NewAdapter()

	// Wire up domain services
	var hookingService *hooking.Service
	if c.hooks != nil {
		hookingService = hooking.NewService(c.hooks)
	}

	var permissionsService *permissions.Service
	if c.permissions != nil {
		permissionsService = permissions.NewService(c.permissions)
	}

	// Use registered MCP servers
	mcpServers := c.mcpServers

	// Create streaming service with dependencies
	c.streamingService = streaming.NewService(
		transport,
		protocol,
		parser,
		hookingService,
		permissionsService,
		mcpServers,
	)

	// Execute domain logic
	return c.streamingService.Connect(ctx, prompt)
}

// SendMessage sends a message to Claude
func (c *Client) SendMessage(ctx context.Context, msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.streamingService == nil {
		return ErrNotConnected
	}

	return c.streamingService.SendMessage(ctx, msg)
}

// ReceiveMessages returns a channel of messages from Claude
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

// Close disconnects from Claude
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.streamingService == nil {
		return nil
	}

	return c.streamingService.Close()
}
