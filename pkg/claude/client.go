// Package claude provides the public API for Claude Agent SDK.
// This is the main entry point for SDK users.
package claude

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/adapters/cli"
	"github.com/conneroisu/claude/pkg/claude/adapters/jsonrpc"
	"github.com/conneroisu/claude/pkg/claude/adapters/mcp"
	"github.com/conneroisu/claude/pkg/claude/adapters/parse"
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Public type aliases for convenience.
type (
	HookEvent    = hooking.HookEvent
	HookMatcher  = hooking.HookMatcher
	HookCallback = hooking.HookCallback
	HookContext  = hooking.HookContext
)

// Hook event constants.
const (
	HookEventPreToolUse       = hooking.HookEventPreToolUse
	HookEventPostToolUse      = hooking.HookEventPostToolUse
	HookEventUserPromptSubmit = hooking.HookEventUserPromptSubmit
	HookEventNotification     = hooking.HookEventNotification
	HookEventSessionStart     = hooking.HookEventSessionStart
	HookEventSessionEnd       = hooking.HookEventSessionEnd
	HookEventStop             = hooking.HookEventStop
	HookEventSubagentStop     = hooking.HookEventSubagentStop
	HookEventPreCompact       = hooking.HookEventPreCompact
)

// Client provides the main interface for interacting with Claude.
// This facade orchestrates all domain services and adapters.
type Client struct {
	options     *options.AgentOptions
	transport   ports.Transport
	protocol    ports.ProtocolHandler
	parser      ports.MessageParser
	hooks       *hooking.Service
	permissions *permissions.Service
	mcpServers  map[string]ports.MCPServer
}

// NewClient creates a new Claude client with the given options.
// This is the primary constructor for SDK users.
func NewClient(opts *options.AgentOptions) (*Client, error) {
	// Create adapters
	transport := cli.NewAdapter(opts)
	parser := parse.NewAdapter()

	// Create domain services
	hooksService := hooking.NewService(nil)
	permsConfig := &permissions.Config{}
	if opts.PermissionMode != nil {
		permsConfig.Mode = *opts.PermissionMode
	}
	permsService := permissions.NewService(permsConfig)

	// Create protocol handler
	protocol := jsonrpc.NewAdapter(transport)

	// Initialize MCP servers
	mcpServers := make(map[string]ports.MCPServer)
	for name, config := range opts.MCPServers {
		mcpServers[name] = mcp.NewClientAdapter(name, &config)
	}

	return &Client{
		options:     opts,
		transport:   transport,
		protocol:    protocol,
		parser:      parser,
		hooks:       hooksService,
		permissions: permsService,
		mcpServers:  mcpServers,
	}, nil
}

// Connect establishes connection to Claude CLI.
// This must be called before Query or Stream.
// The hooks parameter is optional and can be nil.
func (c *Client) Connect(
	ctx context.Context,
	hooks map[HookEvent][]HookMatcher,
) error {
	for event, matchers := range hooks {
		for _, matcher := range matchers {
			for _, callback := range matcher.Hooks {
				c.hooks.RegisterHook(string(event), func(
					ctx context.Context,
					input map[string]any,
				) (map[string]any, error) {
					return callback(
						input,
						nil,
						HookContext{Signal: ctx},
					)
				})
			}
		}
	}

	return c.transport.Connect(ctx)
}

// RegisterHook adds a lifecycle hook callback.
// Hooks allow intercepting and modifying tool calls.
func (c *Client) RegisterHook(
	event string,
	handler func(context.Context, map[string]any) (map[string]any, error),
) {
	c.hooks.RegisterHook(event, handler)
}

// Close terminates all connections and cleans up resources.
func (c *Client) Close() error {
	return c.transport.Close()
}
