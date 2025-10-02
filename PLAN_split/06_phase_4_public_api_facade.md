## Phase 4: Public API (Facade)
The public API acts as a facade over the domain services, hiding the complexity of ports and adapters.
### 4.1 Query Function (query.go)
Priority: Critical
```go
package claude
import (
"context"
"github.com/conneroisu/claude/pkg/claude/querying"
"github.com/conneroisu/claude/pkg/claude/hooking"
"github.com/conneroisu/claude/pkg/claude/permissions"
"github.com/conneroisu/claude/pkg/claude/ports"
"github.com/conneroisu/claude/pkg/claude/adapters/cli"
"github.com/conneroisu/claude/pkg/claude/adapters/jsonrpc"
"github.com/conneroisu/claude/pkg/claude/adapters/parse"
"github.com/conneroisu/claude/pkg/claude/messages"
"github.com/conneroisu/claude/pkg/claude/options"
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
// TODO: Create permissions config from options
var permissionsService *permissions.Service
// TODO: Create MCP servers from options
var mcpServers map[string]ports.MCPServer
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
"github.com/conneroisu/claude/pkg/claude/streaming"
"github.com/conneroisu/claude/pkg/claude/hooking"
"github.com/conneroisu/claude/pkg/claude/permissions"
"github.com/conneroisu/claude/pkg/claude/ports"
"github.com/conneroisu/claude/pkg/claude/adapters/cli"
"github.com/conneroisu/claude/pkg/claude/adapters/jsonrpc"
"github.com/conneroisu/claude/pkg/claude/adapters/parse"
"github.com/conneroisu/claude/pkg/claude/messages"
"github.com/conneroisu/claude/pkg/claude/options"
"sync"
)
// Client provides bidirectional, interactive conversations with Claude
// It's a facade that wires domain services with adapters
type Client struct {
opts             *options.AgentOptions
hooks            map[HookEvent][]HookMatcher
permissions      *PermissionsConfig
streamingService *streaming.Service
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
// TODO: Create MCP servers from options
var mcpServers map[string]ports.MCPServer
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
// Close disconnects from Claude
func (c *Client) Close() error {
c.mu.Lock()
defer c.mu.Unlock()
if c.streamingService == nil {
return nil
}
// Domain service handles cleanup
return c.streamingService.Close()
}
```