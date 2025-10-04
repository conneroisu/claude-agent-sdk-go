# Architecture Documentation

## Hexagonal Architecture (Ports and Adapters)

This SDK follows hexagonal architecture to keep the domain logic isolated from infrastructure concerns.

### Core Principles

1. **Domain Independence**: Business logic doesn't depend on frameworks or external services
2. **Dependency Inversion**: Infrastructure depends on domain, not vice versa
3. **Testability**: Easy to test with mocks and fixtures
4. **Flexibility**: Swap implementations without changing domain code

## Layer Structure

```
┌─────────────────────────────────────────────────────────┐
│                    Public API Layer                      │
│                  (pkg/claude/client.go)                  │
│         Facade providing convenience functions           │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│                  Domain Services Layer                   │
│   ┌─────────────┐  ┌────────────┐  ┌────────────────┐  │
│   │  Querying   │  │ Streaming  │  │  Hooking       │  │
│   │  Service    │  │  Service   │  │  Service       │  │
│   └─────────────┘  └────────────┘  └────────────────┘  │
│   ┌─────────────┐                                       │
│   │ Permissions │                                       │
│   │  Service    │                                       │
│   └─────────────┘                                       │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│                    Ports (Interfaces)                    │
│   - Transport                                            │
│   - ProtocolHandler                                      │
│   - MessageParser                                        │
│   - MCPServer                                            │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│                 Adapters (Infrastructure)                │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐             │
│   │   CLI    │  │ JSON-RPC │  │   MCP    │             │
│   │ Adapter  │  │ Adapter  │  │ Adapter  │             │
│   └──────────┘  └──────────┘  └──────────┘             │
│   ┌──────────┐                                          │
│   │  Parser  │                                          │
│   │ Adapter  │                                          │
│   └──────────┘                                          │
└─────────────────────────────────────────────────────────┘
```

## Domain Models

### Message Types (Discriminated Unions)

All message types implement the `Message` interface with a marker method:

```go
type Message interface {
    message()  // Marker method
}
```

**Variants:**
- `UserMessage` - User input
- `AssistantMessage` - Claude's responses
- `SystemMessage` - System notifications
- `ResultMessage` - Execution results
- `StreamEvent` - Streaming events

### Content Blocks (Discriminated Union)

```go
type ContentBlock interface {
    contentBlock()  // Marker method
}
```

**Variants:**
- `TextBlock` - Text content
- `ThinkingBlock` - Claude's thinking process
- `ToolUseBlock` - Tool invocation
- `ToolResultBlock` - Tool execution result

## Port Definitions

### Transport Port

Handles communication with Claude CLI subprocess:

```go
type Transport interface {
    Connect(ctx context.Context) error
    Write(ctx context.Context, data string) error
    ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error)
    EndInput() error
    Close() error
    IsReady() bool
}
```

### ProtocolHandler Port

Manages JSON-RPC control protocol:

```go
type ProtocolHandler interface {
    Initialize(ctx context.Context, config any) (map[string]any, error)
    SendControlRequest(ctx context.Context, req map[string]any) (map[string]any, error)
    HandleControlRequest(ctx context.Context, req map[string]any, deps ControlDependencies) (map[string]any, error)
    StartMessageRouter(ctx context.Context, msgCh chan<- map[string]any, errCh chan<- error, deps ControlDependencies) error
}
```

### MessageParser Port

Converts raw JSON to typed domain messages:

```go
type MessageParser interface {
    Parse(data map[string]any) (messages.Message, error)
}
```

## Adapter Implementations

### CLI Adapter

- **File**: `pkg/claude/adapters/cli/`
- **Purpose**: Manages Claude CLI subprocess
- **Components**:
  - `adapter.go` - Main struct and interface impl
  - `discovery.go` - Find CLI binary
  - `command.go` - Build CLI command
  - `connect.go` - Establish connection
  - `process.go` - Process lifecycle
  - `io.go` - I/O handling

### JSON-RPC Adapter

- **File**: `pkg/claude/adapters/jsonrpc/`
- **Purpose**: Implements control protocol
- **Components**:
  - `protocol.go` - Protocol handler implementation
  - Request ID generation
  - Control request routing
  - Message routing to handlers

### Parser Adapter

- **File**: `pkg/claude/adapters/parse/`
- **Purpose**: Parse raw JSON to typed messages
- **Component**: `parser.go` - Monolithic parser with type-specific methods

### MCP Adapter

- **File**: `pkg/claude/adapters/mcp/`
- **Purpose**: MCP client/server integration
- **Components**:
  - `client.go` - MCP client adapter
  - `server.go` - MCP server adapter
  - `helpers.go` - Serialization utilities

## Domain Services

### Querying Service

- **File**: `pkg/claude/querying/`
- **Purpose**: One-shot query execution
- **Flow**:
  1. Connect transport
  2. Start message router
  3. Send query
  4. Close input
  5. Stream results

### Streaming Service

- **File**: `pkg/claude/streaming/`
- **Purpose**: Bidirectional conversation
- **Flow**:
  1. Connect transport
  2. Start message router
  3. Send/receive messages continuously
  4. Close when done

### Hooking Service

- **File**: `pkg/claude/hooking/`
- **Purpose**: Manage lifecycle hooks
- **Features**:
  - Register hooks for events
  - Execute hooks with context
  - Pass results to protocol

### Permissions Service

- **File**: `pkg/claude/permissions/`
- **Purpose**: Control tool usage
- **Modes**:
  - `bypass` - Allow all
  - `ask` - Require approval
  - `default` - Standard checking

## Data Flow

### Query Flow

```
User
  ↓
Query() function
  ↓
QueryService.Execute()
  ↓
Transport.Connect()
  ↓
Protocol.StartMessageRouter()
  ↓
Transport.Write(query)
  ↓
Transport.ReadMessages() → Parser.Parse()
  ↓
Message channels → User
```

### Hook Flow

```
Claude wants to use tool
  ↓
Control Request (can_use_tool)
  ↓
Protocol.HandleControlRequest()
  ↓
PermissionsService.CheckToolUse()
  ↓
HookingService (if pre-tool hook registered)
  ↓
Execute hook callback
  ↓
Return result to Claude
  ↓
Claude executes tool
  ↓
HookingService (if post-tool hook registered)
  ↓
Return to Claude
```

## Testing Strategy

### Unit Tests

- **Mock all ports** using `internal/testutil/mocks.go`
- **Test domain services** in isolation
- **Use fixtures** from `internal/testutil/fixtures.go`

### Integration Tests

- **Require Claude CLI** installed
- **Test with real subprocess**
- **Tagged with `//go:build integration`**

## Code Quality Constraints

All code must adhere to:

- **File size**: ≤175 lines
- **Function size**: ≤25 lines
- **Line length**: ≤80 characters
- **Cognitive complexity**: ≤20
- **Nesting depth**: ≤3
- **Comment density**: ≥15%

## Best Practices

1. **Always use Config structs** for service dependencies
2. **Channel-based communication** for async operations
3. **Context propagation** for cancellation
4. **Discriminated unions** with marker methods
5. **Interface segregation** - small, focused ports
6. **Dependency injection** via constructors
