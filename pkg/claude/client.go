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

// Client provides bidirectional, interactive conversations with Claude.
// It manages the lifecycle of a streaming session, including connection setup,
// message sending/receiving, and resource cleanup. Client is safe for
// concurrent use.
//
// Example:
//
//	client := claude.NewClient(&options.AgentOptions{MaxTurns: intPtr(10)})
//	if err := client.Connect(ctx, nil); err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	client.SendMessage(ctx, "Hello")
//	msgCh, errCh := client.ReceiveMessages(ctx)
type Client struct {
	opts              *options.AgentOptions
	hooks             map[hooking.HookEvent][]hooking.HookMatcher
	permissionsConfig *permissions.PermissionsConfig
	streamingService  *streaming.Service
	mcpServers        map[string]ports.MCPServer // Track for cleanup
	mu                sync.Mutex
}

// NewClient creates a new Claude client for streaming conversations.
// Pass nil for opts to use defaults. The hooks and perms parameters allow
// customizing lifecycle hooks and permission checks respectively.
func NewClient(opts *options.AgentOptions, hooks map[hooking.HookEvent][]hooking.HookMatcher, perms *permissions.PermissionsConfig) *Client {
	if opts == nil {
		opts = &options.AgentOptions{}
	}
	return &Client{
		opts:              opts,
		hooks:             hooks,
		permissionsConfig: perms,
	}
}

// Connect establishes a connection to Claude and optionally starts with an
// initial prompt. Must be called before SendMessage or ReceiveMessages.
// Returns an error if the connection fails or if already connected.
func (c *Client) Connect(ctx context.Context, prompt *string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
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
	if c.permissionsConfig != nil {
		permissionsService = permissions.NewService(c.permissionsConfig)
	}
	// Initialize MCP servers from configuration
	mcpServers, err := initializeMCPServers(ctx, c.opts.MCPServers)
	if err != nil {
		return fmt.Errorf("failed to initialize MCP servers: %w", err)
	}
	c.mcpServers = mcpServers // Store for cleanup
	// Create streaming service with dependencies
	c.streamingService = streaming.NewService(transport, protocol, parser, hookingService, permissionsService, mcpServers)
	// Execute domain logic
	return c.streamingService.Connect(ctx, prompt)
}

// SendMessage sends a message to Claude in the current conversation.
// Returns ErrNotConnected if Connect has not been called first.
func (c *Client) SendMessage(ctx context.Context, msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.streamingService == nil {
		return ErrNotConnected
	}
	return c.streamingService.SendMessage(ctx, msg)
}

// ReceiveMessages returns channels for receiving messages and errors from
// Claude. The message channel will close when the conversation ends normally.
// Returns ErrNotConnected via error channel if Connect has not been called.
func (c *Client) ReceiveMessages(ctx context.Context) (<-chan messages.Message, <-chan error) {
	if c.streamingService == nil {
		errCh := make(chan error, 1)
		errCh <- ErrNotConnected
		close(errCh)
		return nil, errCh
	}
	return c.streamingService.ReceiveMessages(ctx)
}

// Close disconnects from Claude and cleans up all resources including
// MCP servers. Should be called when done with the client, typically via defer.
// Returns any errors encountered during cleanup.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error

	// Close streaming service
	if c.streamingService != nil {
		if err := c.streamingService.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing streaming service: %w", err))
		}
	}

	// Close all MCP server connections
	for name, server := range c.mcpServers {
		if err := server.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing MCP server %q: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}
	return nil
}
