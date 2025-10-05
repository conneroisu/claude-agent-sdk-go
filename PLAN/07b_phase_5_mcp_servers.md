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

### ⚠️ Current Implementation Limitation

**IMPORTANT:** The current implementation uses **manual JSON-RPC method routing** because the Go MCP SDK does NOT provide in-memory transport abstractions.

**What this means:**
- The adapter manually inspects the `method` field of incoming JSON-RPC messages
- Routes to appropriate handler based on method name (initialize, tools/list, tools/call)
- Manually constructs JSON-RPC responses from handler results
- **NO automatic channel-based transport like the aspirational design**

**Why this approach:**
- The Go MCP SDK (like Python MCP SDK) lacks the Transport abstraction that TypeScript has
- TypeScript: `server.connect(transport)` allows pluggable custom transports
- Go/Python: Servers expect actual I/O streams, not abstract transports
- This forces manual routing similar to Python SDK (see `query.py:326-440`)

**Future migration path:**
When the Go MCP SDK adds Transport support (similar to TypeScript), we can:
1. Replace manual routing with `server.Connect(customTransport)`
2. Use channel-based in-memory transport for cleaner architecture
3. Eliminate switch statements and type assertions
4. Reduce adapter code significantly (~50% reduction)

**This is a pragmatic interim solution following the proven Python SDK approach** until the Go MCP ecosystem matures.

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
│    - Wraps server in SDKServerAdapter                               │
│    - Registers in internal server map                               │
│    - Server instance stored for manual routing                      │
│    ⚠️  NO automatic transport abstraction yet                       │
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
│ 6. SDKServerAdapter processes message (MANUAL ROUTING)              │
│    - Inspects JSON-RPC method field (initialize, tools/list, etc.)  │
│    - Manually routes to appropriate server.request_handlers entry   │
│    - Constructs JSON-RPC response from handler result               │
│    ⚠️  No in-memory transport abstraction in current MCP SDKs       │
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
Claude CLI ←→ Control Protocol ←→ SDK Server Adapter ←→ Manual Router ←→ User's mcp.Server
```

**Key components:**
- User creates `*mcp.Server` using `NewMCPServer()` and `AddTool()`
- User registers server via `options.SDKServerConfig` with server instance
- SDK extracts SDK servers during initialization
- SDK wraps in `SDKServerAdapter` implementing `ports.MCPSDKServer`
- Control protocol routes `mcp_message` requests to appropriate adapter
- **Adapter manually inspects JSON-RPC method and routes to handler** (no transport abstraction)

**⚠️ IMPORTANT - Current Limitation:**
The MCP Go SDK does NOT currently provide in-memory transport abstractions for custom routing. Unlike the TypeScript MCP SDK which supports `server.connect(transport)` with pluggable transports, the Go SDK (like Python) requires **manual method routing**.

This implementation follows the **same pattern as the Python SDK** (see `claude-agent-sdk-python/src/claude_agent_sdk/_internal/query.py:326-440`) where we:
1. Extract the JSON-RPC `method` field from incoming messages
2. Manually dispatch to the appropriate `server.request_handlers` entry
3. Convert handler results back to JSON-RPC responses

When the Go MCP SDK adds proper Transport abstractions, this can be refactored to use channel-based in-memory transports similar to TypeScript's approach.

**Note:** The `options.SDKServerConfig` type (defined in Phase 1) contains BOTH configuration AND the server instance. Unlike external server configs which only have connection params, SDK configs include the actual `*mcp.Server` instance to enable in-process communication.

---

## SDK Server Adapter Implementation

### adapters/mcp/sdk_server.go

This adapter wraps a user's `*mcp.Server` instance to integrate with the control protocol using **manual JSON-RPC method routing** (same approach as Python SDK):

```go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/ports"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// SDKServerAdapter wraps a user's *mcp.Server to expose it via control protocol
// This uses MANUAL routing because the Go MCP SDK lacks Transport abstraction
type SDKServerAdapter struct {
	name   string
	server *mcpsdk.Server
}

// Verify interface compliance
var _ ports.MCPServer = (*SDKServerAdapter)(nil)

// NewSDKServerAdapter creates an adapter for a user's SDK server
func NewSDKServerAdapter(name string, server *mcpsdk.Server) (*SDKServerAdapter, error) {
	if server == nil {
		return nil, fmt.Errorf("server cannot be nil")
	}

	adapter := &SDKServerAdapter{
		name:   name,
		server: server,
	}

	return adapter, nil
}

// Connect initializes the server (no-op for SDK servers - already initialized)
func (a *SDKServerAdapter) Connect(ctx context.Context) error {
	// SDK servers are initialized by the user before registration
	// No connection needed - we route directly to request handlers
	return nil
}

// Name returns the server identifier
func (a *SDKServerAdapter) Name() string {
	return a.name
}

// HandleMessage routes a JSON-RPC message to the SDK server via manual method dispatch
// This follows the same pattern as Python SDK (query.py:326-440)
func (a *SDKServerAdapter) HandleMessage(ctx context.Context, messageBytes []byte) ([]byte, error) {
	// Parse JSON-RPC message
	var msg struct {
		JSONRPC string                 `json:"jsonrpc"`
		ID      any                    `json:"id"`
		Method  string                 `json:"method"`
		Params  map[string]any         `json:"params,omitempty"`
	}

	if err := json.Unmarshal(messageBytes, &msg); err != nil {
		return a.errorResponse(nil, -32700, "Parse error", err)
	}

	// Manual method routing (same as Python SDK)
	switch msg.Method {
	case "initialize":
		return a.handleInitialize(ctx, msg.ID)

	case "tools/list":
		return a.handleToolsList(ctx, msg.ID)

	case "tools/call":
		return a.handleToolsCall(ctx, msg.ID, msg.Params)

	case "notifications/initialized":
		// No-op notification
		return nil, nil

	default:
		return a.errorResponse(msg.ID, -32601, "Method not found", nil)
	}
}

// handleInitialize returns server capabilities (hardcoded like Python SDK)
func (a *SDKServerAdapter) handleInitialize(ctx context.Context, id any) ([]byte, error) {
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{}, // Tools capability without listChanged
			},
			"serverInfo": map[string]any{
				"name":    a.server.Name,
				"version": a.server.Version,
			},
		},
	}
	return json.Marshal(response)
}

// handleToolsList calls the server's ListTools handler
func (a *SDKServerAdapter) handleToolsList(ctx context.Context, id any) ([]byte, error) {
	// Get handler from server's request_handlers map
	handler := a.server.RequestHandlers[mcpsdk.ListToolsRequest]
	if handler == nil {
		return a.errorResponse(id, -32601, "tools/list handler not registered", nil)
	}

	// Create request and invoke handler
	req := &mcpsdk.ListToolsRequest{Method: "tools/list"}
	result, err := handler(ctx, req)
	if err != nil {
		return a.errorResponse(id, -32603, "Handler error", err)
	}

	// Convert MCP result to JSON-RPC response
	listResult, ok := result.(*mcpsdk.ListToolsResult)
	if !ok {
		return a.errorResponse(id, -32603, "Invalid handler result type", nil)
	}

	toolsData := make([]map[string]any, len(listResult.Tools))
	for i, tool := range listResult.Tools {
		toolsData[i] = map[string]any{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		}
	}

	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"tools": toolsData,
		},
	}
	return json.Marshal(response)
}

// handleToolsCall calls the server's CallTool handler
func (a *SDKServerAdapter) handleToolsCall(ctx context.Context, id any, params map[string]any) ([]byte, error) {
	// Extract call parameters
	name, _ := params["name"].(string)
	arguments, _ := params["arguments"].(map[string]any)

	if name == "" {
		return a.errorResponse(id, -32602, "Missing tool name", nil)
	}

	// Get handler from server's request_handlers map
	handler := a.server.RequestHandlers[mcpsdk.CallToolRequest]
	if handler == nil {
		return a.errorResponse(id, -32601, "tools/call handler not registered", nil)
	}

	// Create request and invoke handler
	req := &mcpsdk.CallToolRequest{
		Method: "tools/call",
		Params: mcpsdk.CallToolParams{
			Name:      name,
			Arguments: arguments,
		},
	}

	result, err := handler(ctx, req)
	if err != nil {
		return a.errorResponse(id, -32603, "Handler error", err)
	}

	// Convert MCP result to JSON-RPC response
	callResult, ok := result.(*mcpsdk.CallToolResult)
	if !ok {
		return a.errorResponse(id, -32603, "Invalid handler result type", nil)
	}

	// Convert content blocks to JSON
	content := make([]map[string]any, len(callResult.Content))
	for i, block := range callResult.Content {
		switch b := block.(type) {
		case *mcpsdk.TextContent:
			content[i] = map[string]any{
				"type": "text",
				"text": b.Text,
			}
		case *mcpsdk.ImageContent:
			content[i] = map[string]any{
				"type":     "image",
				"data":     b.Data,
				"mimeType": b.MimeType,
			}
		}
	}

	responseData := map[string]any{
		"content": content,
	}
	if callResult.IsError {
		responseData["is_error"] = true
	}

	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  responseData,
	}
	return json.Marshal(response)
}

// errorResponse constructs a JSON-RPC error response
func (a *SDKServerAdapter) errorResponse(id any, code int, message string, err error) ([]byte, error) {
	errData := map[string]any{
		"code":    code,
		"message": message,
	}
	if err != nil {
		errData["data"] = err.Error()
	}

	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   errData,
	}
	return json.Marshal(response)
}

// Close releases resources (no-op for SDK servers)
func (a *SDKServerAdapter) Close() error {
	// User's server instance is user-managed - we don't close it
	return nil
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

## Error Handling for Misconfigured Servers

### Common Error Scenarios

1. **Server Not Found**
   - User references a server name that wasn't registered
   - Return JSON-RPC error -32601 "Server not found"
   - Control protocol response includes error details

2. **Missing Handler Registration**
   - User's server doesn't register required handlers (tools/list, tools/call)
   - Return JSON-RPC error -32601 "Handler not registered"
   - Provide clear error message indicating which handler is missing

3. **Invalid Handler Results**
   - Handler returns wrong result type
   - Return JSON-RPC error -32603 "Invalid handler result type"
   - Include type information in error data

4. **Malformed JSON-RPC Messages**
   - CLI sends invalid JSON or missing required fields
   - Return JSON-RPC error -32700 "Parse error"
   - Include parsing error details

5. **Tool Execution Errors**
   - User's tool handler returns an error
   - Propagate error through JSON-RPC error response
   - Include error message and stack trace in error data

### Error Response Format

All errors follow JSON-RPC 2.0 error specification:

```json
{
  "jsonrpc": "2.0",
  "id": 123,
  "error": {
    "code": -32601,
    "message": "tools/list handler not registered",
    "data": "server 'calculator' did not register a ListToolsRequest handler"
  }
}
```

### Validation Checks

The adapter performs these validation checks before routing:

```go
// In NewSDKServerAdapter
if server == nil {
    return nil, fmt.Errorf("server cannot be nil")
}

// In HandleMessage
if msg.Method == "" {
    return a.errorResponse(msg.ID, -32600, "Invalid Request",
        fmt.Errorf("missing method field"))
}

// In handleToolsCall
if name == "" {
    return a.errorResponse(id, -32602, "Missing tool name", nil)
}
```

---

## Testing Requirements

### Test Coverage for Malformed MCP Responses

The adapter MUST have tests covering these scenarios:

1. **Invalid JSON-RPC Message**
   ```go
   func TestSDKServerAdapter_InvalidJSON(t *testing.T) {
       adapter, _ := NewSDKServerAdapter("test", server)
       _, err := adapter.HandleMessage(ctx, []byte("{invalid json"))
       require.Error(t, err)
       // Verify error response contains parse error
   }
   ```

2. **Unknown Method**
   ```go
   func TestSDKServerAdapter_UnknownMethod(t *testing.T) {
       msg := `{"jsonrpc":"2.0","id":1,"method":"unknown/method"}`
       resp, err := adapter.HandleMessage(ctx, []byte(msg))
       require.NoError(t, err)
       // Verify response contains method not found error
   }
   ```

3. **Missing Required Parameters**
   ```go
   func TestSDKServerAdapter_MissingToolName(t *testing.T) {
       msg := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{}}`
       resp, err := adapter.HandleMessage(ctx, []byte(msg))
       // Verify error response for missing tool name
   }
   ```

4. **Handler Not Registered**
   ```go
   func TestSDKServerAdapter_HandlerNotRegistered(t *testing.T) {
       emptyServer := mcpsdk.NewServer("test", "1.0")
       adapter, _ := NewSDKServerAdapter("test", emptyServer)
       msg := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
       resp, err := adapter.HandleMessage(ctx, []byte(msg))
       // Verify handler not found error
   }
   ```

5. **Server Returns Error**
   ```go
   func TestSDKServerAdapter_HandlerReturnsError(t *testing.T) {
       // Register handler that returns error
       server.RequestHandlers[CallToolRequest] = func(ctx, req) (any, error) {
           return nil, errors.New("tool execution failed")
       }
       msg := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"test"}}`
       resp, err := adapter.HandleMessage(ctx, []byte(msg))
       // Verify error propagated to JSON-RPC response
   }
   ```

6. **Malformed Response from Handler**
   ```go
   func TestSDKServerAdapter_InvalidHandlerResultType(t *testing.T) {
       // Register handler that returns wrong type
       server.RequestHandlers[ListToolsRequest] = func(ctx, req) (any, error) {
           return "wrong type", nil
       }
       msg := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
       resp, err := adapter.HandleMessage(ctx, []byte(msg))
       // Verify invalid result type error
   }
   ```

### Integration Tests

Test complete flow including control protocol:

```go
func TestSDKServer_EndToEnd(t *testing.T) {
    // 1. Create SDK server with tool
    server := claude.NewMCPServer("test", "1.0")
    claude.AddTool(server, &mcp.Tool{Name: "echo"}, echoHandler)

    // 2. Register server in options
    opts := &options.AgentOptions{
        MCPServers: map[string]options.MCPServerConfig{
            "test": options.SDKServerConfig{
                Type: "sdk",
                Name: "test",
                Instance: server,
            },
        },
    }

    // 3. Execute query that uses the tool
    msgCh, errCh := claude.Query(ctx, "Use the echo tool", opts, nil)

    // 4. Verify tool was called and returned correct result
    // ...
}
```

---

## Implementation Notes

### File Size Requirements

**MCP integration in adapters/mcp/:**
- ✅ `sdk_server.go` - ~175 lines (manual routing adapter)
- ✅ `client.go` - 125 lines (external MCP client adapter)

Both files are under the 175-line limit.

### Complexity Hotspots

**Manual method routing:**
- No automatic transport abstraction available
- Must inspect JSON-RPC `method` field and dispatch manually
- Similar to Python SDK implementation at query.py:326-440
- Type assertions required for handler results

**Message routing:**
- JSON-RPC parsing/encoding handled manually with encoding/json
- Control protocol wraps messages in `mcp_response` field
- Error handling propagates from user's tool handlers
- All JSON-RPC error codes follow RFC specification

**Future improvements when Go MCP SDK adds Transport support:**
- Replace manual routing with channel-based in-memory transport
- Use `server.Connect(transport)` pattern similar to TypeScript
- Eliminate switch statement and type assertions
- Reduce adapter code by ~50%

**Recommended patterns:**
- Follow Python SDK's manual routing approach (proven in production)
- Comprehensive error handling for all edge cases
- Extensive test coverage for malformed messages
- Document limitation clearly for users

### Checklist

- [ ] SDK server adapter implements `ports.MCPServer`
- [ ] Manual method routing implemented (initialize, tools/list, tools/call)
- [ ] JSON-RPC messages parsed and routed correctly
- [ ] Control protocol handles `mcp_message` requests
- [ ] Error responses follow JSON-RPC 2.0 specification
- [ ] All error scenarios tested (server not found, handler missing, malformed JSON)
- [ ] Test coverage for malformed MCP responses (>90%)
- [ ] Integration tests verify end-to-end flow
- [ ] Resources cleaned up in Close()
- [ ] User's server instance NOT closed by adapter
- [ ] Adapter file under 175 lines
- [ ] Complete example in cmd/examples/mcp/
- [ ] Documentation clearly states manual routing limitation
- [ ] Migration path documented for when Transport abstraction arrives

---

## Related Files
- [Phase 5a: Hooks Support](./07a_phase_5_hooks.md)
- [Phase 5c: Permission Callbacks](./07c_phase_5_permissions.md)
- [Phase 5: Integration Summary](./07d_phase_5_integration_summary.md)
- [Phase 4: Public API Facade](./06_phase_4_public_api_facade.md) - MCP initialization
- [Phase 1: Core Domain & Ports](./03_phase_1_core_domain_ports.md) - MCP port definition
