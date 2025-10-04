## Architecture Overview

### Core Design Principles

- Idiomatic Go: Use Go conventions (interfaces, channels, contexts, errors)
- Type Safety: Leverage Go's strong typing with generics where appropriate
- Concurrency: Use goroutines and channels for async operations
- Error Handling: Explicit error returns following Go best practices
- Context Support: Full context.Context integration for cancellation and timeouts
- Zero Dependencies: Minimize external dependencies where possible
- Hexagonal Architecture: Strict separation between domain logic and infrastructure

Dependencies:

- github.com/modelcontextprotocol/go-sdk - For MCP server support

### Hexagonal Architecture (Ports and Adapters)

This SDK follows hexagonal architecture principles, also known as ports and adapters pattern.

This architectural style isolates the core business logic (domain) from external concerns (infrastructure) by defining clear boundaries and dependency rules.

Four Key Principles:

- Domain Independence: Core domain never imports adapters or infrastructure code
- Ports Define Contracts: Interfaces defined by domain needs, not external systems
- Adapters Implement Ports: Infrastructure code implements domain-defined interfaces
- Dependency Direction: Always flows inward (adapters → domain), never outward

### Package Structure (Hexagonal Architecture)

Following hexagonal architecture principles (ports and adapters), the SDK separates the core domain from external dependencies. Package names describe what they provide (functionality/context), not what they contain (generic types).

```
claude-agent-sdk-go/
├── cmd/                        # ═══ BINARIES (Entry Points) ═══
│   └── examples/               # Example applications
│       ├── quickstart/
│       ├── streaming/
│       ├── hooks/
│       └── mcp/
│
├── pkg/claude/
│   # LAYER 1: CORE DOMAIN (Business Logic)
│   # - Never imports from adapters/
│   # - Only imports from ports/ (interfaces it defines)
│   # - Pure business logic, no infrastructure concerns
│   ├── querying/               # Domain service: "Execute one-shot queries"
│   │   └── service.go
│   ├── streaming/              # Domain service: "Manage streaming conversations"
│   │   └── service.go
│   ├── hooking/                # Domain service: "Execute lifecycle hooks"
│   │   └── service.go
│   ├── permissions/            # Domain service: "Check tool permissions"
│   │   └── service.go
│   │
│   # LAYER 1B: DOMAIN MODELS
│   # - Shared types used across domain
│   # - No infrastructure dependencies
│   ├── messages/               # Domain models: Message types
│   │   ├── messages.go         # Core message interfaces
│   │   ├── control.go          # Control protocol types
│   │   └── ...
│   ├── options/                # Domain models: Configuration
│   │   ├── domain.go           # Pure domain options (PermissionMode, etc.)
│   │   ├── transport.go        # Transport configuration
│   │   ├── tools.go            # Built-in tool type definitions
│   │   └── mcp.go              # MCP server configuration (both client & SDK)
│   │
│   # LAYER 2: PORTS (Domain-Defined Interfaces)
│   # - Interfaces defined BY domain needs
│   # - NOT defined by external systems
│   # - This is the "contract" layer
│   ├── ports/
│   │   ├── transport.go        # What domain needs from transport
│   │   ├── protocol.go         # What domain needs from control protocol
│   │   ├── parser.go           # What domain needs from message parsing
│   │   └── mcp.go              # What domain needs from MCP servers (unified interface)
│   │
│   # LAYER 3: ADAPTERS (Infrastructure Implementations)
│   # - Implements port interfaces
│   # - Handles external concerns (CLI, JSON-RPC, parsing)
│   # - Can import from domain and ports
│   # - Domain NEVER imports from here
│   ├── adapters/
│   │   ├── cli/                # Adapter: CLI subprocess transport
│   │   │   └── transport.go    # Implements ports.Transport
│   │   ├── jsonrpc/            # Adapter: Control protocol handler
│   │   │   └── protocol.go     # Implements ports.ProtocolHandler
│   │   ├── parse/              # Adapter: Message parser
│   │   │   └── parser.go       # Implements ports.MessageParser
│   │   └── mcp/                # Adapters: MCP integration (2 types)
│   │       ├── client.go       # External servers: Implements ports.MCPServer
│   │       └── sdk_server.go   # SDK servers: Implements ports.MCPServer
│   │
│   # LAYER 4: PUBLIC API (Facade)
│   # - Wires domain services with adapters
│   # - Entry point for SDK users
│   # - Hides complexity of ports/adapters from users
│   ├── client.go               # Client for interactive conversations
│   ├── query.go                # Query() for one-shot requests
│   ├── mcp.go                  # NewMCPServer() and AddTool() for SDK servers
│   └── errors.go               # Public error types
│
├── go.mod
├── go.sum
├── README.md
├── CHANGELOG.md
├── LICENSE
└── .golangci.yaml
```

Hexagonal Architecture Key Principles:

1. Core Domain Independence: Domain packages (`querying`, `streaming`, `hooking`, `permissions`) contain business logic and do NOT import adapters
1. Ports Define Contracts: The `ports/` package contains interfaces defined by the domain's needs
1. Adapters Implement Ports: Adapters in `adapters/` implement the port interfaces and handle external concerns
1. Dependency Direction: Always flows inward - adapters depend on domain, never the reverse
1. Context-Based Naming: Packages named for what they provide (`querying`, `streaming`) not what they contain (`models`, `handlers`)

### Control Protocol Layer

The control protocol layer enables bidirectional communication between the SDK and Claude CLI over stdin/stdout using JSON-RPC. This layer is implemented as an adapter (`adapters/jsonrpc`) that implements the `ports.ProtocolHandler` interface.

#### Message Flow Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       SDK User Application                       │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                ▼
                ┌───────────────────────────────┐
                │    Domain Services Layer      │
                │  (querying, streaming, etc.)  │
                └──────────┬────────────────────┘
                           │
                           ▼
                ┌──────────────────────────────┐
                │   ports.ProtocolHandler      │ ← Interface
                │   (defined by domain needs)  │
                └──────────┬───────────────────┘
                           │
                           ▼
                ┌──────────────────────────────┐
                │  adapters/jsonrpc/protocol   │ ← Implementation
                │   Control Protocol Handler   │
                └──────────┬───────────────────┘
                           │
                ┌──────────┴───────────┐
                ▼                      ▼
        ┌───────────────┐      ┌──────────────┐
        │  Outbound     │      │  Inbound     │
        │  Requests     │      │  Requests    │
        └───────┬───────┘      └──────┬───────┘
                │                     │
                ▼                     ▼
        ┌───────────────────────────────────┐
        │    ports.Transport (stdin/out)    │
        └───────────────┬───────────────────┘
                        │
                        ▼
        ┌───────────────────────────────────┐
        │         Claude CLI Process        │
        └───────────────────────────────────┘
```

#### Three Message Types

The control protocol handles three distinct message types:

**1. SDK Messages (Outbound)**
- Regular SDK messages sent to Claude CLI
- Examples: User queries, streaming requests, conversation updates
- Flow: Domain → Protocol → Transport → CLI

**2. Control Requests (Bidirectional)**

Outbound (SDK → CLI):
- `interrupt`: Cancel ongoing operations
- `set_permission_mode`: Change permission behavior (ask/allow/deny)
- `set_model`: Switch AI model
- `initialize`: Start session with hooks and config

Inbound (CLI → SDK):
- `can_use_tool`: Request permission to use a tool
- `hook_callback`: Execute registered lifecycle hooks
- `mcp_message`: Proxy MCP server requests

**3. Control Responses (Bidirectional)**
- Responses to control requests
- Include request_id for correlation
- May contain result data or errors

#### Request ID Format

All control requests use a consistent ID generation pattern:

```go
// Request ID: "req_{counter}_{randomHex}"
requestID := fmt.Sprintf("req_%d_%s", requestCounter, randomHex(4))
// Example: "req_1_a3f2", "req_2_b8c9"
```

This ensures unique identification for request/response correlation with a 60-second timeout on all outbound requests.

#### MCP Message Proxying

The control protocol includes a special routing layer for MCP (Model Context Protocol) messages:

```
┌─────────────────────────────────────────────────────────────┐
│                      Claude CLI                              │
│  Sends: {"type":"control_request","request":                │
│          {"subtype":"mcp_message","server":"myserver",...}}  │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
                ┌────────────────────────────┐
                │  jsonrpc Protocol Handler  │
                │  Receives control_request  │
                └────────────┬───────────────┘
                             │
                             ▼
                ┌────────────────────────────┐
                │  Routes to MCP Adapter     │
                │  based on server name      │
                └────────────┬───────────────┘
                             │
                ┌────────────┴────────────┐
                ▼                         ▼
        ┌───────────────┐       ┌────────────────┐
        │  MCP Client   │       │  MCP Server    │
        │  Adapter      │       │  Adapter       │
        └───────┬───────┘       └────────┬───────┘
                │                        │
                ▼                        ▼
        ┌──────────────┐         ┌──────────────┐
        │ Remote MCP   │         │ SDK MCP      │
        │ Server via   │         │ Server       │
        │ Transport    │         │ (in-process) │
        └──────────────┘         └──────────────┘
```

**MCP Message Flow:**

1. CLI sends `mcp_message` control request with server name
2. Protocol handler extracts server name and message
3. Routes to appropriate MCP adapter (client or server)
4. MCP adapter processes the message:
   - **Client adapter**: Forwards to remote MCP server via transport
   - **Server adapter**: Routes to in-process MCP server instance
5. Response flows back through the same path
6. Protocol handler wraps response in control_response
7. Sends back to CLI

This architecture allows the SDK to:
- Host MCP servers directly (using go-sdk's mcp.Server)
- Connect to remote MCP servers as a client
- All communication proxied through the control protocol
- CLI remains unaware of MCP implementation details

#### Permission Suggestions

When the CLI requests tool permission, it can include suggestions for "always allow" workflows:

```go
type CanUseToolRequest struct {
    Subtype              string             `json:"subtype"` // "can_use_tool"
    ToolName             string             `json:"tool_name"`
    Input                map[string]any     `json:"input"`
    PermissionSuggestions []PermissionUpdate `json:"permission_suggestions,omitempty"`
    BlockedPath          *string            `json:"blocked_path,omitempty"`
}
```

The SDK can choose to apply these suggestions automatically (e.g., when permission mode is "allow") or present them to the user for approval.

#### Hook Callback Registration

During initialization, the SDK registers hook callback IDs that the CLI will use later:

```go
// During initialize request
type InitializeRequest struct {
    Subtype       string              `json:"subtype"` // "initialize"
    Version       string              `json:"version"`
    HookCallbacks map[string]string   `json:"hook_callbacks"` // callback_id → hook_name
    // ... other fields
}

// Hook callback ID format: "hook_{counter}"
callbackID := fmt.Sprintf("hook_%d", nextCallbackID)
hookCallbacksMap[callbackID] = userProvidedCallback
```

When the CLI needs to execute a hook, it sends a `hook_callback` control request with the callback ID, and the SDK looks up the corresponding user function.

#### Timeout Protection

All outbound control requests include 60-second timeout protection:

```go
ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
defer cancel()

response, err := protocol.SendControlRequest(ctx, request)
if err != nil {
    // Handle timeout or other errors
}
```

This prevents the SDK from hanging indefinitely if the CLI becomes unresponsive.

### SDK MCP Server Integration Architecture

The SDK supports two types of MCP integration:

#### 1. External MCP Servers (Client Mode)
The SDK connects TO external MCP servers running as separate processes:

```
┌──────────────────────────────────────────────────────────────┐
│ User Application                                             │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ Claude SDK                                               │ │
│ │ ┌─────────────┐    ┌──────────────┐                     │ │
│ │ │   Domain    │───>│ ports.       │                     │ │
│ │ │   Services  │    │ MCPServer    │                     │ │
│ │ └─────────────┘    └──────────────┘                     │ │
│ │                           │                              │ │
│ │                           ▼                              │ │
│ │                    ┌──────────────┐                     │ │
│ │                    │ MCP Client   │                     │ │
│ │                    │   Adapter    │                     │ │
│ │                    └──────────────┘                     │ │
│ └──────────────────────────│─────────────────────────────┘ │
└────────────────────────────┼───────────────────────────────┘
                             │ Subprocess/HTTP/SSE
                             ▼
                   ┌─────────────────────┐
                   │  External MCP Server │
                   │  (Separate Process)  │
                   └─────────────────────┘
```

**Flow:**
1. User configures external server in `options.MCPServers` with stdio/HTTP/SSE config
2. SDK creates MCP client via go-sdk and connects to external process
3. ClientAdapter wraps the client session implementing `ports.MCPServer`
4. Control protocol routes `mcp_message` requests to ClientAdapter
5. ClientAdapter forwards JSON-RPC to external server via subprocess/HTTP

#### 2. SDK MCP Servers (In-Process Mode)
User creates MCP servers in the same process as their application:

```
┌──────────────────────────────────────────────────────────────┐
│ User Application                                             │
│                                                              │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ User's MCP Server (In-Process)                           │ │
│ │ ┌────────────┐  ┌────────────┐  ┌────────────┐          │ │
│ │ │  Tool 1    │  │  Tool 2    │  │  Tool 3    │          │ │
│ │ │  Handler   │  │  Handler   │  │  Handler   │          │ │
│ │ └────────────┘  └────────────┘  └────────────┘          │ │
│ │        │               │               │                 │ │
│ │        └───────────────┴───────────────┘                 │ │
│ │                        │                                 │ │
│ │                        ▼                                 │ │
│ │                ┌──────────────┐                         │ │
│ │                │ *mcp.Server  │ (from go-sdk)           │ │
│ │                └──────────────┘                         │ │
│ └─────────────────────┼──────────────────────────────────┘ │
│                       │ In-Memory Transport                 │
│                       ▼                                     │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ Claude SDK                                               │ │
│ │ ┌─────────────┐    ┌──────────────┐                     │ │
│ │ │   Domain    │───>│ ports.       │                     │ │
│ │ │   Services  │    │ MCPServer    │                     │ │
│ │ └─────────────┘    └──────────────┘                     │ │
│ │                           │                              │ │
│ │                           ▼                              │ │
│ │                    ┌──────────────┐                     │ │
│ │                    │ SDK Server   │                     │ │
│ │                    │   Adapter    │                     │ │
│ │                    └──────────────┘                     │ │
│ │                           │                              │ │
│ │                           ▼                              │ │
│ │                  Control Protocol                        │ │
│ │                           │                              │ │
│ │                           ▼                              │ │
│ │                    CLI Transport                         │ │
│ └──────────────────────────│─────────────────────────────┘ │
└────────────────────────────┼───────────────────────────────┘
                             │ Subprocess (stdio)
                             ▼
                   ┌─────────────────────┐
                   │   Claude CLI        │
                   └─────────────────────┘
```

**Flow:**
1. User creates `*mcp.Server` with `claude.NewMCPServer()`
2. User registers tools with `claude.AddTool()`
3. User passes server instance via `options.SDKServerConfig{Instance: server}`
4. SDK creates in-memory transport pair (channels)
5. SDK connects user's server to in-memory transport
6. SDKServerAdapter wraps server implementing `ports.MCPServer`
7. Control protocol routes `mcp_message` requests to SDKServerAdapter
8. SDKServerAdapter sends/receives JSON-RPC via in-memory channels
9. User's tool handlers execute in same process (no IPC overhead)

**Key Differences:**

| Aspect | External Servers | SDK Servers |
|--------|------------------|-------------|
| **Process** | Separate process | Same process as app |
| **Transport** | stdio/HTTP/SSE | In-memory channels |
| **Adapter** | ClientAdapter | SDKServerAdapter |
| **User Creates** | Config (command/URL) | `*mcp.Server` instance |
| **Performance** | IPC overhead | Zero IPC overhead |
| **Deployment** | Multiple processes | Single process |
| **State Access** | No direct access | Direct access to app state |

**MCP Configuration Types:**

The SDK supports four configuration types in `options.MCPServerConfig`:

1. **StdioServerConfig** - External server via subprocess (command + args)
2. **HTTPServerConfig** - External server via HTTP transport
3. **SSEServerConfig** - External server via Server-Sent Events
4. **SDKServerConfig** - In-process server with `Instance *mcp.Server` field

**Critical Detail:** `SDKServerConfig` MUST store the server instance (confirmed from TypeScript/Python implementations):

```go
type SDKServerConfig struct {
    Type     string        // "sdk"
    Name     string        // Server name
    Instance *mcp.Server   // The user's MCP server instance (MUST be stored)
}
```

This differs from external configs which only store connection details. The SDK needs the instance to create in-memory transport and connect the server.

**Unified Port Interface:**

Both adapter types implement the same `ports.MCPServer` interface, allowing the domain to treat them uniformly:

```go
type MCPServer interface {
    Name() string
    HandleMessage(ctx context.Context, message []byte) ([]byte, error)
    Close() error
}
```

This unified interface is a key benefit of hexagonal architecture - the domain doesn't care whether an MCP server is external or in-process, it just routes JSON-RPC messages through the port interface.
