package claude

import (
	"context"
	"fmt"
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

// Client provides bidirectional streaming conversations with Claude.
// It maintains a persistent connection for sending multiple messages
// and receiving responses in real-time.
//
// Example:
//
//	client := claude.NewClient(nil, nil, nil)
//	defer client.Close()
//	err := client.Connect(ctx, nil)
//	msgCh, errCh := client.ReceiveMessages(ctx)
//	client.SendMessage(ctx, "Hello!")
type Client struct {
	opts              *options.AgentOptions
	hooks             map[hooking.HookEvent][]hooking.HookMatcher
	permissionsConfig *permissions.PermissionsConfig
	streamingService  *streaming.Service
	mcpServers        map[string]ports.MCPServer
	mu                sync.Mutex
}

// NewClient creates a new Claude client.
// All parameters are optional and can be nil for defaults.
func NewClient(
	opts *options.AgentOptions,
	hooks map[hooking.HookEvent][]hooking.HookMatcher,
	perms *permissions.PermissionsConfig,
) *Client {
	if opts == nil {
		opts = &options.AgentOptions{}
	}

	return &Client{
		opts:              opts,
		hooks:             hooks,
		permissionsConfig: perms,
	}
}

// Connect establishes connection to Claude.
// It wires up adapters, initializes MCP servers, and optionally
// sends an initial prompt to begin the conversation.
func (c *Client) Connect(ctx context.Context, prompt *string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Wire up infrastructure adapters
	transport := cli.NewAdapter(c.opts)
	protocol := jsonrpc.NewAdapter(transport)
	parser := parse.NewAdapter()

	// Wire up domain services
	var hookingService *hooking.Service
	if c.hooks != nil {
		hookingService = hooking.NewService(c.hooks)
	}

	var permissionsService *permissions.Service
	if c.permissionsConfig != nil {
		permissionsService = permissions.NewService(c.permissionsConfig)
	}

	// Initialize MCP servers from configuration
	mcpServers, err := initializeMCPServers(ctx, c.opts.MCPServers)
	if err != nil {
		return fmt.Errorf("failed to initialize MCP servers: %w", err)
	}
	c.mcpServers = mcpServers

	// Create streaming service with all dependencies
	c.streamingService = streaming.NewService(
		transport,
		protocol,
		parser,
		hookingService,
		permissionsService,
		mcpServers,
	)

	// Execute connection with optional initial prompt
	return c.streamingService.Connect(ctx, prompt)
}

// SendMessage sends a message to Claude.
// The client must be connected before calling this method.
func (c *Client) SendMessage(ctx context.Context, msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.streamingService == nil {
		return ErrNotConnected
	}

	return c.streamingService.SendMessage(ctx, msg)
}

// ReceiveMessages returns channels for receiving messages from Claude.
// The client must be connected before calling this method.
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

// Close disconnects from Claude and cleans up all resources.
// Always call this when done, typically using defer.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error

	if c.streamingService != nil {
		if err := c.streamingService.Close(); err != nil {
			errs = append(errs, fmt.Errorf("streaming service: %w", err))
		}
	}

	for name, server := range c.mcpServers {
		if err := server.Close(); err != nil {
			errs = append(errs, fmt.Errorf("MCP server %q: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}
