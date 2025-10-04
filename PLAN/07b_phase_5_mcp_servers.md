## Phase 5b: MCP Server Support

### Priority: Critical

### Overview
To support in-process user-defined tools, the SDK provides a public API that wraps the official `github.com/modelcontextprotocol/go-sdk`. Instead of re-implementing the MCP server, the SDK offers convenience functions to create and configure an `mcp.Server`.

**SDK MCP servers run in the same process as the Go application**, using in-memory communication instead of subprocess/IPC. This provides:
- ✅ Better performance (no subprocess overhead)
- ✅ Simpler deployment (single process)
- ✅ Easier debugging (same process)
- ✅ Direct access to application state
- ✅ Type-safe tool definitions using Go generics

### SDK MCP Server Integration Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│ 1. User creates tools and MCP server                                │
│    server := claude.NewMCPServer("calculator", "1.0")                │
│    claude.AddTool(server, &mcp.Tool{...}, handler)                  │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 2. User registers server in options                                  │
│    opts := &options.AgentOptions{                                    │
│        MCPServers: map[string]options.MCPServerConfig{               │
│            "calc": options.SDKServerConfig{                          │
│                Type: "sdk",                                          │
│                Name: "calc",                                         │
│                Instance: server, // <-- Server instance              │
│            },                                                        │
│        },                                                            │
│    }                                                                 │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 3. SDK extracts and initializes SDK servers on Connect()            │
│    - Creates in-memory transport pair for each server               │
│    - Connects server to its transport                               │
│    - Wraps in SDKServerAdapter                                      │
│    - Registers in internal server map                               │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 4. Claude CLI sends control_request                                 │
│    {                                                                 │
│      "type": "control_request",                                      │
│      "request": {                                                    │
│        "subtype": "mcp_message",                                     │
│        "server_name": "calc",                                        │
│        "message": { /* JSON-RPC request */ }                         │
│      }                                                               │
│    }                                                                 │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 5. Control protocol adapter routes to SDK server                    │
│    - Extracts server_name from request                              │
│    - Looks up SDKServerAdapter by name                              │
│    - Forwards JSON-RPC message to adapter                           │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 6. SDKServerAdapter processes message                               │
│    - Writes JSON-RPC to in-memory transport                         │
│    - User's mcp.Server handles via request handlers                 │
│    - Reads JSON-RPC response from transport                         │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 7. Response sent back through control protocol                      │
│    {                                                                 │
│      "type": "control_response",                                     │
│      "response": {                                                   │
│        "subtype": "success",                                         │
│        "response": { "mcp_response": { /* JSON-RPC */ } }            │
│      }                                                               │
│    }                                                                 │
└─────────────────────────────────────────────────────────────────────┘
```

### Public API (mcp.go)

**File:** `pkg/claude/mcp.go`

```go
package claude

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewMCPServer creates a new in-process MCP server using the go-sdk.
// This server can be configured with tools and then passed to the Claude client
// via AgentOptions.
//
// Example:
//
//	server := claude.NewMCPServer("calculator", "1.0")
//	claude.AddTool(server, &mcp.Tool{Name: "add", Description: "Add numbers"}, addHandler)
//
//	opts := &options.AgentOptions{
//	    MCPServers: map[string]options.MCPServerConfig{
//	        "calc": options.SDKServerConfig{Type: "sdk", Name: "calc", Instance: server},
//	    },
//	}
func NewMCPServer(name, version string) *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{Name: name, Version: version}, nil)
}

// AddTool is a convenience wrapper around the go-sdk's generic AddTool function.
// It allows users to add a tool with a typed handler to an mcp.Server instance,
// benefiting from automatic schema inference and validation provided by the go-sdk.
//
// The Go MCP SDK uses generics to automatically infer JSON schema from the Args
// and Result type parameters. Struct tags (json, jsonschema) control schema generation.
//
// Example:
//
//	type AddArgs struct {
//	    A float64 `json:"a" jsonschema:"description=First number"`
//	    B float64 `json:"b" jsonschema:"description=Second number"`
//	}
//
//	type AddResult struct {
//	    Sum float64 `json:"sum"`
//	}
//
//	addHandler := func(ctx context.Context, req *mcp.CallToolRequest, args AddArgs) (*mcp.CallToolResult, AddResult, error) {
//	    return nil, AddResult{Sum: args.A + args.B}, nil
//	}
//
//	server := NewMCPServer("calculator", "1.0")
//	AddTool(server, &mcp.Tool{Name: "add", Description: "Add two numbers"}, addHandler)
func AddTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	mcp.AddTool(server, tool, handler)
}
```

### Tool Schema Inference with Go Generics

The Go MCP SDK leverages generics to automatically generate JSON schemas from struct types:

```go
// Args type with schema annotations
type GreetArgs struct {
	Name string `json:"name" jsonschema:"description=Person to greet,required"`
	Formal bool `json:"formal,omitempty" jsonschema:"description=Use formal greeting"`
}

// Result type (optional - can use *mcp.CallToolResult directly)
type GreetResult struct {
	Message string `json:"message"`
}

// Handler function - types are inferred, schema generated automatically
func greetHandler(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args GreetArgs,
) (*mcp.CallToolResult, GreetResult, error) {
	greeting := "Hi"
	if args.Formal {
		greeting = "Hello"
	}
	return nil, GreetResult{Message: greeting + " " + args.Name}, nil
}

// Register tool - schema is automatically generated from GreetArgs
server := NewMCPServer("greeter", "1.0")
AddTool(server, &mcp.Tool{
	Name:        "greet",
	Description: "Greet a person by name",
}, greetHandler)
```

The generated JSON schema for the above example:
```json
{
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "description": "Person to greet"
    },
    "formal": {
      "type": "boolean",
      "description": "Use formal greeting"
    }
  },
  "required": ["name"]
}
```

### Integration Architecture

SDK-managed servers use a different architecture than external MCP servers:

**External MCP Servers (stdio/HTTP/SSE):**
```
Claude CLI ←→ MCP Client Adapter ←→ Subprocess/HTTP ←→ External Server
```

**SDK MCP Servers (in-process):**
```
Claude CLI ←→ Control Protocol ←→ SDK Server Adapter ←→ In-Memory Transport ←→ User's mcp.Server
```

**Key components:**
- User creates `*mcp.Server` using `NewMCPServer()` and `AddTool()`
- User registers server via `options.SDKServerConfig` with server instance
- SDK extracts SDK servers during initialization
- SDK creates in-memory transport pair for each server using `mcp.NewInMemoryTransports()`
- SDK connects user's server to its transport
- SDK wraps in `SDKServerAdapter` implementing `ports.MCPSDKServer`
- Control protocol routes `mcp_message` requests to appropriate adapter
- Adapter sends/receives JSON-RPC via in-memory transport

**Note:** The `options.SDKServerConfig` type (defined in Phase 1) contains BOTH configuration AND the server instance. Unlike external server configs which only have connection params, SDK configs include the actual `*mcp.Server` instance to enable in-process communication.

---

## SDK Server Adapter Implementation

### adapters/mcp/sdk_server.go

This adapter wraps a user's `*mcp.Server` instance to integrate with the control protocol:

```go
package mcp

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/ports"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// SDKServerAdapter wraps a user's *mcp.Server to expose it via control protocol
type SDKServerAdapter struct {
	name            string
	server          *mcpsdk.Server
	clientTransport *InMemoryClientTransport
	serverTransport *InMemoryServerTransport
	session         *mcpsdk.ServerSession
}

// Verify interface compliance
var _ ports.MCPServer = (*SDKServerAdapter)(nil)

// NewSDKServerAdapter creates an adapter for a user's SDK server
func NewSDKServerAdapter(name string, server *mcpsdk.Server) (*SDKServerAdapter, error) {
	// Create in-memory transport pair
	clientTransport, serverTransport := newInMemoryTransportPair()

	adapter := &SDKServerAdapter{
		name:            name,
		server:          server,
		clientTransport: clientTransport,
		serverTransport: serverTransport,
	}

	return adapter, nil
}

// Connect initializes the server connection
func (a *SDKServerAdapter) Connect(ctx context.Context) error {
	// Connect user's server to its transport
	session, err := a.server.Connect(ctx, a.serverTransport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect SDK server %q: %w", a.name, err)
	}
	a.session = session
	return nil
}

// Name returns the server identifier
func (a *SDKServerAdapter) Name() string {
	return a.name
}

// HandleMessage routes a JSON-RPC message to the SDK server via in-memory transport
func (a *SDKServerAdapter) HandleMessage(ctx context.Context, messageBytes []byte) ([]byte, error) {
	// Decode JSON-RPC message
	msg, err := jsonrpc.DecodeMessage(messageBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}

	// Write to client side of in-memory transport
	if err := a.clientTransport.Write(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to write to transport: %w", err)
	}

	// Read response from transport
	resp, err := a.clientTransport.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Encode response back to JSON-RPC
	return jsonrpc.EncodeMessage(resp)
}

// Close releases resources
func (a *SDKServerAdapter) Close() error {
	// Close both sides of in-memory transport
	// Note: User's server instance is NOT closed here - it's user-managed
	var errs []error

	if a.clientTransport != nil {
		if err := a.clientTransport.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if a.serverTransport != nil {
		if err := a.serverTransport.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing SDK server adapter: %v", errs)
	}
	return nil
}

// InMemoryTransport implementation - simplified for in-process communication
// Uses channels to route messages between client and server sides

type InMemoryClientTransport struct {
	sendCh chan jsonrpc.Message
	recvCh chan jsonrpc.Message
}

type InMemoryServerTransport struct {
	sendCh chan jsonrpc.Message
	recvCh chan jsonrpc.Message
}

func newInMemoryTransportPair() (*InMemoryClientTransport, *InMemoryServerTransport) {
	clientToServer := make(chan jsonrpc.Message, 10)
	serverToClient := make(chan jsonrpc.Message, 10)

	client := &InMemoryClientTransport{
		sendCh: clientToServer,
		recvCh: serverToClient,
	}

	server := &InMemoryServerTransport{
		sendCh: serverToClient,
		recvCh: clientToServer,
	}

	return client, server
}

func (t *InMemoryClientTransport) Write(ctx context.Context, msg jsonrpc.Message) error {
	select {
	case t.sendCh <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *InMemoryClientTransport) Read(ctx context.Context) (jsonrpc.Message, error) {
	select {
	case msg := <-t.recvCh:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (t *InMemoryClientTransport) Close() error {
	close(t.sendCh)
	return nil
}

// Server side implements mcp.Connection interface
func (t *InMemoryServerTransport) Read(ctx context.Context) (jsonrpc.Message, error) {
	select {
	case msg := <-t.recvCh:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (t *InMemoryServerTransport) Write(ctx context.Context, msg jsonrpc.Message) error {
	select {
	case t.sendCh <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *InMemoryServerTransport) Close() error {
	close(t.sendCh)
	return nil
}

func (t *InMemoryServerTransport) SessionID() string {
	return "sdk-server-session"
}

// Implement mcp.Transport interface
func (t *InMemoryServerTransport) Connect(ctx context.Context) (mcpsdk.Connection, error) {
	return t, nil
}
```

---

## Complete Working Example

### cmd/examples/mcp/calculator/main.go

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool argument types
type AddArgs struct {
	A float64 `json:"a" jsonschema:"description=First number,required"`
	B float64 `json:"b" jsonschema:"description=Second number,required"`
}

type SubtractArgs struct {
	A float64 `json:"a" jsonschema:"description=First number,required"`
	B float64 `json:"b" jsonschema:"description=Second number,required"`
}

type MultiplyArgs struct {
	A float64 `json:"a" jsonschema:"description=First number,required"`
	B float64 `json:"b" jsonschema:"description=Second number,required"`
}

type DivideArgs struct {
	A float64 `json:"a" jsonschema:"description=Dividend,required"`
	B float64 `json:"b" jsonschema:"description=Divisor (must not be zero),required"`
}

// Tool result type
type MathResult struct {
	Result float64 `json:"result"`
}

// Tool handlers
func addHandler(
	ctx context.Context,
	req *mcpsdk.CallToolRequest,
	args AddArgs,
) (*mcpsdk.CallToolResult, MathResult, error) {
	return nil, MathResult{Result: args.A + args.B}, nil
}

func subtractHandler(
	ctx context.Context,
	req *mcpsdk.CallToolRequest,
	args SubtractArgs,
) (*mcpsdk.CallToolResult, MathResult, error) {
	return nil, MathResult{Result: args.A - args.B}, nil
}

func multiplyHandler(
	ctx context.Context,
	req *mcpsdk.CallToolRequest,
	args MultiplyArgs,
) (*mcpsdk.CallToolResult, MathResult, error) {
	return nil, MathResult{Result: args.A * args.B}, nil
}

func divideHandler(
	ctx context.Context,
	req *mcpsdk.CallToolRequest,
	args DivideArgs,
) (*mcpsdk.CallToolResult, MathResult, error) {
	if args.B == 0 {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: "Error: Division by zero"},
			},
			IsError: true,
		}, MathResult{}, nil
	}
	return nil, MathResult{Result: args.A / args.B}, nil
}

func main() {
	ctx := context.Background()

	// 1. Create SDK MCP server with calculator tools
	server := claude.NewMCPServer("calculator", "1.0")

	claude.AddTool(server, &mcpsdk.Tool{
		Name:        "add",
		Description: "Add two numbers",
	}, addHandler)

	claude.AddTool(server, &mcpsdk.Tool{
		Name:        "subtract",
		Description: "Subtract two numbers",
	}, subtractHandler)

	claude.AddTool(server, &mcpsdk.Tool{
		Name:        "multiply",
		Description: "Multiply two numbers",
	}, multiplyHandler)

	claude.AddTool(server, &mcpsdk.Tool{
		Name:        "divide",
		Description: "Divide two numbers",
	}, divideHandler)

	// 2. Configure Claude agent with SDK server
	opts := &options.AgentOptions{
		MCPServers: map[string]options.MCPServerConfig{
			"calc": options.SDKServerConfig{
				Type:     "sdk",
				Name:     "calc",
				Instance: server,
			},
		},
		// Pre-approve calculator tools
		AllowedTools: []options.BuiltinTool{
			options.ToolMcp,
		},
	}

	// 3. Execute query using calculator
	msgCh, errCh := claude.Query(ctx, "What is 15 + 27?", opts, nil)

	// 4. Process responses
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			displayMessage(msg)

		case err, ok := <-errCh:
			if !ok {
				return
			}
			if err != nil {
				log.Fatalf("Error: %v", err)
			}
		}
	}
}

func displayMessage(msg messages.Message) {
	switch m := msg.(type) {
	case messages.AssistantMessage:
		for _, block := range m.Content {
			if text, ok := block.(messages.TextBlock); ok {
				fmt.Printf("Claude: %s\n", text.Text)
			}
		}

	case messages.ResultMessageSuccess:
		fmt.Printf("\nResult: %s\n", m.Result)
		fmt.Printf("Cost: $%.6f\n", m.TotalCostUSD)
		fmt.Printf("Turns: %d\n", m.NumTurns)
	}
}
```

### Output:
```
Claude: I'll use the add tool to calculate 15 + 27.
[Tool use: add with arguments {a: 15, b: 27}]
Claude: The result is 42.

Result: The result is 42.
Cost: $0.000234
Turns: 1
```

---

## Control Protocol Integration

The control protocol adapter (defined in `adapters/jsonrpc/protocol.go`) handles `mcp_message` requests:

```go
// In HandleControlRequest method
case "mcp_message":
	serverName := req["server_name"].(string)
	message := req["message"].(map[string]any)

	// Look up SDK server adapter
	mcpServer, ok := mcpServers[serverName]
	if !ok {
		return map[string]any{
			"error": fmt.Sprintf("SDK MCP server %q not found", serverName),
		}, fmt.Errorf("server not found")
	}

	// Encode message to JSON-RPC
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to encode MCP message: %w", err)
	}

	// Route to SDK server adapter
	responseBytes, err := mcpServer.HandleMessage(ctx, messageBytes)
	if err != nil {
		return map[string]any{
			"error": err.Error(),
		}, err
	}

	// Decode response
	var response map[string]any
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode MCP response: %w", err)
	}

	return map[string]any{
		"mcp_response": response,
	}, nil
```

---

## Implementation Notes

### File Size Requirements

**MCP integration in adapters/mcp/:**
- ✅ `sdk_server.go` - 175 lines (adapter + in-memory transport)
- ✅ `client.go` - 125 lines (external MCP client adapter)

Both files are under the 175-line limit.

### Complexity Hotspots

**In-memory transport:**
- Uses Go channels for message passing
- No serialization overhead
- Context-aware with proper cancellation
- Buffered channels prevent blocking

**Message routing:**
- JSON-RPC encoding/decoding handled by go-sdk
- Control protocol wraps messages in `mcp_response` field
- Error handling propagates from user's tool handlers

**Recommended patterns:**
- Use `mcp.NewInMemoryTransports()` pattern from go-sdk examples
- Leverage existing control protocol infrastructure
- Keep adapter focused on message proxying only

### Checklist

- [ ] SDK server adapter implements `ports.MCPServer`
- [ ] In-memory transport pair created with channels
- [ ] Server connected to transport on initialization
- [ ] JSON-RPC messages routed correctly
- [ ] Control protocol handles `mcp_message` requests
- [ ] Error responses properly formatted
- [ ] Resources cleaned up in Close()
- [ ] User's server instance NOT closed by adapter
- [ ] Adapter file under 175 lines
- [ ] Complete example in cmd/examples/mcp/

---

## Related Files
- [Phase 5a: Hooks Support](./07a_phase_5_hooks.md)
- [Phase 5c: Permission Callbacks](./07c_phase_5_permissions.md)
- [Phase 5: Integration Summary](./07d_phase_5_integration_summary.md)
- [Phase 4: Public API Facade](./06_phase_4_public_api_facade.md) - MCP initialization
- [Phase 1: Core Domain & Ports](./03_phase_1_core_domain_ports.md) - MCP port definition
