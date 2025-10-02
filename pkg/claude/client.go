package claude

import (
	"context"
	"fmt"
	"sync"

	"github.com/conneroisu/claude/pkg/claude/adapters/cli"
	"github.com/conneroisu/claude/pkg/claude/adapters/jsonrpc"
	"github.com/conneroisu/claude/pkg/claude/adapters/mcp"
	"github.com/conneroisu/claude/pkg/claude/adapters/parse"
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
	"github.com/conneroisu/claude/pkg/claude/streaming"
)

// Client provides bidirectional, interactive conversations with Claude
// It's a facade that wires domain services with adapters.
type Client struct {
	opts             *options.AgentOptions
	hooks            map[HookEvent][]HookMatcher
	permissions      *PermissionsConfig
	streamingService *streaming.Service
	mu               sync.Mutex
}

// NewClient creates a new Claude client for bidirectional
// streaming conversations.
func NewClient(
	opts *options.AgentOptions,
	hooks map[HookEvent][]HookMatcher,
	perms *PermissionsConfig,
) *Client {
	localOpts := opts
	if localOpts == nil {
		localOpts = &options.AgentOptions{}
	}

	// Mark as streaming for bidirectional conversation behavior
	localOpts.IsStreaming = true

	return &Client{
		opts:        localOpts,
		hooks:       hooks,
		permissions: perms,
	}
}

// Connect establishes connection to Claude.
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
	if c.permissions != nil {
		permissionsService = permissions.NewService(c.permissions)
	}

	// Create MCP servers from options
	mcpServers, err := initializeMCPServers(c.opts.MCPServers)
	if err != nil {
		return fmt.Errorf("initialize MCP servers: %w", err)
	}

	// Create streaming service with dependencies
	c.streamingService = streaming.NewService(streaming.Dependencies{
		Transport:   transport,
		Protocol:    protocol,
		Parser:      parser,
		Hooks:       hookingService,
		Permissions: permissionsService,
		MCPServers:  mcpServers,
	})

	// Execute domain logic
	return c.streamingService.Connect(ctx, prompt)
}

// SendMessage sends a message to Claude.
func (c *Client) SendMessage(ctx context.Context, msg string) error {
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

	// Domain service handles cleanup
	return c.streamingService.Close()
}

// initializeMCPServers creates runtime MCP server instances from configuration.
// Only SDK-managed servers are initialized; external servers (stdio, SSE, HTTP)
// are handled by the Claude CLI.
func initializeMCPServers(
	serverConfigs map[string]options.MCPServerConfig,
) (map[string]ports.MCPServer, error) {
	if len(serverConfigs) == 0 {
		return nil, nil
	}

	servers := make(map[string]ports.MCPServer)

	for name, config := range serverConfigs {
		// Only SDK servers need runtime instances
		sdkConfig, ok := config.(options.SDKServerConfig)
		if !ok {
			// Skip stdio, SSE, HTTP servers - they're managed by the CLI
			continue
		}

		if sdkConfig.Instance == nil {
			return nil, fmt.Errorf(
				"SDK MCP server '%s' has nil Instance",
				name,
			)
		}

		server, err := mcp.NewAdapter(name, sdkConfig.Instance)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create MCP adapter for '%s': %w",
				name,
				err,
			)
		}

		servers[name] = server
	}

	if len(servers) == 0 {
		return nil, nil
	}

	return servers, nil
}
