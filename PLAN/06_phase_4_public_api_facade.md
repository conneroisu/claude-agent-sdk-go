## Phase 4: Public API (Facade)
The public API acts as a facade over the domain services, hiding the complexity of ports and adapters.
### 4.1 Query Function (query.go)
Priority: Critical
```go
package claude

import (
	"context"
	"github.com/conneroisu/claude/pkg/claude/adapters/cli"
	"github.com/conneroisu/claude/pkg/claude/adapters/jsonrpc"
	"github.com/conneroisu/claude/pkg/claude/adapters/parse"
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
	"github.com/conneroisu/claude/pkg/claude/querying"
)

// Query performs a one-shot query to Claude
// This is the main entry point that wires up domain services with adapters
func Query(ctx context.Context, prompt string, opts *options.AgentOptions, hooks map[HookEvent][]HookMatcher) (<-chan messages.Message, <-chan error) {
	if opts == nil {
		opts = &options.AgentOptions{}
	}
	// Wire up adapters (infrastructure layer)
	transport := cli.NewAdapter(opts)
	protocol := jsonrpc.NewAdapter(transport)
	parser := parse.NewAdapter()
	// Create domain services
	var hookingService *hooking.Service
	if hooks != nil {
		hookingService = hooking.NewService(hooks)
	}
	// Create permissions service from options
	var permissionsService *permissions.Service
	if opts.PermissionMode != nil {
		// TODO: Initialize permissions service based on permission mode
		// permissionsService = permissions.NewService(...)
	}
	// Initialize MCP servers from configuration
	mcpServers, err := initializeMCPServers(ctx, opts.MCPServers)
	if err != nil {
		errCh := make(chan error, 1)
		errCh <- fmt.Errorf("failed to initialize MCP servers: %w", err)
		close(errCh)
		return nil, errCh
	}
	queryService := querying.NewService(transport, protocol, parser, hookingService, permissionsService, mcpServers)
	// Execute domain logic
	return queryService.Execute(ctx, prompt, opts)
}
```
### 4.2 Client (client.go)
Priority: Critical
```go
package claude

import (
	"context"
	"github.com/conneroisu/claude/pkg/claude/adapters/cli"
	"github.com/conneroisu/claude/pkg/claude/adapters/jsonrpc"
	"github.com/conneroisu/claude/pkg/claude/adapters/parse"
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
	"github.com/conneroisu/claude/pkg/claude/streaming"
	"sync"
)

// Client provides bidirectional, interactive conversations with Claude
// It's a facade that wires domain services with adapters
type Client struct {
	opts             *options.AgentOptions
	hooks            map[HookEvent][]HookMatcher
	permissions      *PermissionsConfig
	streamingService *streaming.Service
	mcpServers       map[string]ports.MCPServer // Track for cleanup
	mu               sync.Mutex
}

// NewClient creates a new Claude client
func NewClient(opts *options.AgentOptions, hooks map[HookEvent][]HookMatcher, perms *PermissionsConfig) *Client {
	if opts == nil {
		opts = &options.AgentOptions{}
	}
	return &Client{
		opts:        opts,
		hooks:       hooks,
		permissions: perms,
	}
}

// Connect establishes connection to Claude
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
func (c *Client) ReceiveMessages(ctx context.Context) (<-chan messages.Message, <-chan error) {
	if c.streamingService == nil {
		errCh := make(chan error, 1)
		errCh <- ErrNotConnected
		close(errCh)
		return nil, errCh
	}
	return c.streamingService.ReceiveMessages(ctx)
}

// Close disconnects from Claude and cleans up resources
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
```

### 4.3 MCP Server Initialization (mcp_init.go)
Priority: Critical

This helper initializes MCP client connections from configuration:

```go
package claude

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/conneroisu/claude/pkg/claude/adapters/mcp"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/ports"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// initializeMCPServers creates MCP client connections from configuration
// Returns a map of server name -> connected MCP client adapter
func initializeMCPServers(
	ctx context.Context,
	configs map[string]options.MCPServerConfig,
) (map[string]ports.MCPServer, error) {
	if len(configs) == 0 {
		return nil, nil
	}

	servers := make(map[string]ports.MCPServer, len(configs))

	for name, cfg := range configs {
		server, err := initializeMCPServer(ctx, name, cfg)
		if err != nil {
			// Clean up already-connected servers
			for _, s := range servers {
				_ = s.Close()
			}
			return nil, fmt.Errorf("failed to initialize MCP server %q: %w", name, err)
		}
		servers[name] = server
	}

	return servers, nil
}

// initializeMCPServer creates a single MCP client connection
func initializeMCPServer(
	ctx context.Context,
	name string,
	cfg options.MCPServerConfig,
) (ports.MCPServer, error) {
	var transport mcpsdk.Transport

	switch config := cfg.(type) {
	case options.StdioServerConfig:
		// Create stdio transport using command
		cmd := exec.CommandContext(ctx, config.Command, config.Args...)
		if config.Env != nil {
			cmd.Env = append(cmd.Env, mapToEnvSlice(config.Env)...)
		}
		transport = &mcpsdk.CommandTransport{Command: cmd}

	case options.HTTPServerConfig:
		// Create HTTP streamable transport
		transport = &mcpsdk.StreamableClientTransport{
			Endpoint: config.URL,
			Headers:  config.Headers,
		}

	case options.SSEServerConfig:
		// SSE uses same streamable transport as HTTP
		transport = &mcpsdk.StreamableClientTransport{
			Endpoint: config.URL,
			Headers:  config.Headers,
		}

	case options.SDKServerConfig:
		// SDK-managed servers are handled by the MCP adapter layer
		// This case should be handled by a separate registry/factory
		return nil, fmt.Errorf("SDK-managed MCP servers not yet implemented")

	default:
		return nil, fmt.Errorf("unknown MCP server config type: %T", cfg)
	}

	// Create MCP client using official SDK
	client := mcpsdk.NewClient(
		&mcpsdk.Implementation{
			Name:    "claude-agent-sdk-go",
			Version: "0.1.0",
		},
		nil,
	)

	// Connect to the MCP server
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	// Wrap the session in our adapter that implements ports.MCPServer
	return mcp.NewAdapter(name, session), nil
}

// mapToEnvSlice converts map[string]string to []string in KEY=VALUE format
func mapToEnvSlice(m map[string]string) []string {
	result := make([]string, 0, len(m))
	for k, v := range m {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}
```

### 4.4 MCP Adapter (adapters/mcp/adapter.go)
Priority: Critical

This adapter wraps the MCP SDK's ClientSession to implement our ports.MCPServer interface:

**Note:** The `ports.MCPServer` interface needs a `Close() error` method added to support resource cleanup. Update Phase 1 port definition accordingly.

```go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/ports"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Adapter wraps an MCP ClientSession to implement ports.MCPServer
type Adapter struct {
	name    string
	session *mcpsdk.ClientSession
}

// Verify interface compliance at compile time
var _ ports.MCPServer = (*Adapter)(nil)

// NewAdapter creates a new MCP adapter wrapping the given session
func NewAdapter(name string, session *mcpsdk.ClientSession) *Adapter {
	return &Adapter{
		name:    name,
		session: session,
	}
}

// Name returns the server name
func (a *Adapter) Name() string {
	return a.name
}

// HandleMessage forwards a raw JSON-RPC message to the MCP server
// and returns the response. This is used by the domain to proxy
// messages from Claude CLI to the MCP server.
func (a *Adapter) HandleMessage(ctx context.Context, message []byte) ([]byte, error) {
	// Decode the raw JSON-RPC message
	msg, err := jsonrpc.DecodeMessage(message)
	if err != nil {
		return nil, err
	}

	// Route the message based on type
	switch m := msg.(type) {
	case *jsonrpc.Request:
		// Forward request to MCP server via session
		// The session handles method routing internally
		result, err := a.handleRequest(ctx, m)
		if err != nil {
			// Return JSON-RPC error response
			errResp := &jsonrpc.Response{
				ID:    m.ID,
				Error: &jsonrpc.Error{Code: -32603, Message: err.Error()},
			}
			return jsonrpc.EncodeMessage(errResp)
		}
		// Return successful response
		resp := &jsonrpc.Response{
			ID:     m.ID,
			Result: result,
		}
		return jsonrpc.EncodeMessage(resp)

	case *jsonrpc.Notification:
		// Handle notifications (no response expected)
		return nil, a.handleNotification(ctx, m)

	default:
		return nil, fmt.Errorf("unsupported message type: %T", msg)
	}
}

// handleRequest routes JSON-RPC requests to appropriate MCP SDK methods
func (a *Adapter) handleRequest(ctx context.Context, req *jsonrpc.Request) (any, error) {
	switch req.Method {
	case "tools/list":
		return a.session.ListTools(ctx, nil)

	case "tools/call":
		var params mcpsdk.CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		return a.session.CallTool(ctx, &params)

	case "resources/list":
		return a.session.ListResources(ctx, nil)

	case "resources/read":
		var params mcpsdk.ReadResourceParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		return a.session.ReadResource(ctx, &params)

	case "prompts/list":
		return a.session.ListPrompts(ctx, nil)

	case "prompts/get":
		var params mcpsdk.GetPromptParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		return a.session.GetPrompt(ctx, &params)

	default:
		return nil, fmt.Errorf("unsupported method: %s", req.Method)
	}
}

// handleNotification routes JSON-RPC notifications
func (a *Adapter) handleNotification(ctx context.Context, notif *jsonrpc.Notification) error {
	// MCP notifications are typically one-way, no response needed
	// Could implement logging/monitoring here
	return nil
}

// Close closes the MCP session
func (a *Adapter) Close() error {
	return a.session.Close()
}
```

---

## Linting Compliance Notes

### File Size Requirements (175 line limit)

**Public API files:**
- ✅ `client.go` - Estimated 130 lines (compliant with error handling)
- ✅ `query.go` - Estimated 95 lines (compliant with error handling)
- ✅ `mcp_init.go` - 95 lines (compliant)
- ❌ `adapters/mcp/adapter.go` - 125 lines (compliant, but close to limit)
- ✅ `errors.go` - Estimated 60 lines (compliant)

**If adapter.go exceeds limit later, split into:**
- `adapter.go` - Adapter struct + constructor + Name/Close (40 lines)
- `handler.go` - HandleMessage + routing logic (85 lines)

### Complexity Considerations

**Parameter limits (4 max):**
- Use config structs for complex initialization
- Option functions for flexible configuration

**Example compliant API:**
```go
// GOOD: 2-3 parameters
func NewClient(
    opts *options.AgentOptions,
    cfg *ClientConfig,
) *Client

// GOOD: Variadic options (counts as 2 effective params)
func NewClient(
    opts *options.AgentOptions,
    clientOpts ...ClientOption,
) *Client
```

### Checklist

- [ ] All files under 175 lines
- [ ] Constructor uses option pattern or config struct (≤4 params)
- [ ] Public API fully documented with godoc examples
- [ ] Error types have clear documentation
- [ ] Builder pattern for complex init if needed
- [ ] MCP server initialization handles all config types (stdio, HTTP, SSE)
- [ ] MCP adapter correctly implements ports.MCPServer interface
- [ ] Proper cleanup on initialization errors (close partial connections)
- [ ] MCP client sessions properly closed in Client.Close()
- [ ] Context cancellation properly handled in MCP connections
