## Hexagonal Architecture Summary

### Critical Architectural Boundary: Control Protocol Handling

**The most important design decision in this SDK is where control protocol logic lives.**

The SDK uses a JSON-RPC control protocol for bidirectional communication with the Claude CLI. This protocol has three message types:
1. **SDK Messages** - Regular conversation messages (user, assistant, system, result)
2. **Control Requests** - Bidirectional control messages (tool permissions, hook callbacks, mode changes)
3. **Control Responses** - Responses to control requests

**Architectural Split:**

**Domain Services (querying/, streaming/, hooking/, permissions/):**
- Orchestrate conversation flow and lifecycle
- Call user-provided callbacks (hooks, permission checks)
- Send formatted user messages
- DO NOT generate request IDs, track pending requests, or implement timeouts

**Protocol Adapter (adapters/jsonrpc/):**
- Generates request IDs (`req_{counter}_{randomHex}`)
- Tracks pending requests in `map[requestID]chan response`
- Generates callback IDs for hooks (`hook_{index}`)
- Implements 60-second timeout enforcement
- Routes messages by type (SDK vs control request vs control response)
- Handles request/response correlation

**This separation ensures:**
- Domain services are testable without subprocess or protocol complexity
- Protocol details can change without affecting business logic
- Services work with any transport/protocol implementation
- Clear responsibility boundaries

### Dependency Flow

The SDK strictly follows the dependency rule of hexagonal architecture:

```
LAYER 4: Public API (client.go, query.go)
- Entry point for SDK users
- Wires domain services with adapters
depends on ↓
LAYER 3: ADAPTERS (adapters/*)
- cli/      → implements ports.Transport
- jsonrpc/  → implements ports.ProtocolHandler
- parse/    → implements ports.MessageParser
- mcp/      → implements ports.MCPServer
         ↓ depends on
LAYER 2: PORTS (ports/*)
- Interfaces defined BY domain needs
- Contract layer between domain and infra
         ↑ depends on
LAYER 1: CORE DOMAIN (querying/, streaming/)
- Pure business logic
- No infrastructure dependencies
- Uses port interfaces, never adapters
```

Both adapters (Layer 3) and domain services (Layer 1) depend on ports (Layer 2).
Domain services never import adapters, ensuring clean separation.

### Key Architectural Decisions

#### 1. Ports Define Contracts

All interfaces are defined in `ports/` based on domain needs:

- `ports.Transport` - What domain needs for I/O
- `ports.ProtocolHandler` - What domain needs for control protocol
- `ports.MessageParser` - What domain needs for parsing
- `ports.MCPServer` - What domain needs from MCP servers

#### 2. Adapters Implement Ports

Infrastructure code implements these interfaces:

- `adapters/cli.Adapter` implements `ports.Transport`
- `adapters/jsonrpc.Adapter` implements `ports.ProtocolHandler`
- `adapters/parse.Adapter` implements `ports.MessageParser`
- `adapters/mcp.Adapter` implements `ports.MCPServer`

#### 3. Domain Services Orchestrate, Adapters Execute

Domain services focus on **WHAT** needs to happen (business logic), while adapters handle **HOW** it happens (protocol mechanics).

**Domain Service Responsibilities:**
- `querying.Service` - Orchestrates one-shot query lifecycle (connect → prompt → stream → cleanup)
- `streaming.Service` - Manages multi-turn conversation flow (connect → send/receive loop → close)
- `hooking.Service` - Executes user-defined lifecycle callbacks based on event matching
- `permissions.Service` - Evaluates tool permission decisions via user callbacks

**What Domain Services DO:**
- Coordinate message flow and conversation lifecycle
- Parse and validate domain messages via `ports.MessageParser`
- Call hook callbacks and permission callbacks (user-provided functions)
- Send formatted user messages through `ports.Transport`
- Manage service state (connected/disconnected, conversation context)

**What Domain Services DO NOT Do:**
- Generate request IDs or callback IDs (adapter concern)
- Track pending control protocol requests (adapter concern)
- Implement timeout mechanisms (adapter concern)
- Route messages by protocol type (adapter concern)
- Handle transport-level buffering or subprocess management (adapter concern)

#### 4. Control Protocol Mechanics Live in Adapters

All control protocol state and mechanics are implemented in `adapters/jsonrpc`:

**jsonrpc Adapter Responsibilities:**
- Generate unique request IDs: `req_{counter}_{randomHex(4)}`
- Track pending requests in map: `map[requestID]chan response`
- Generate callback IDs for hooks: `hook_{index}`
- Implement 60-second timeout enforcement on control requests
- Route inbound messages (SDK messages vs control requests vs control responses)
- Handle control protocol request/response correlation
- Manage protocol-level concurrency and synchronization

**Why This Split Matters:**
- Domain services remain testable without subprocess spawning or protocol complexity
- Protocol changes (e.g., different timeout values, ID formats) don't affect domain logic
- Services can work with any protocol implementation (JSON-RPC, gRPC, HTTP, etc.)
- Clear separation makes code easier to reason about and maintain

#### 5. Configuration Separation

Configuration types are split by architectural layer:

- `options/domain.go` - Pure domain config (PermissionMode, AgentDefinition, hook registration)
- `options/transport.go` - Infrastructure config (Cwd, Env, MaxBufferSize, CLI paths)
- `options/mcp.go` - MCP server configurations (command, args, environment)

### Concrete Example: Tool Permission Flow

To illustrate the architectural split, here's how a tool permission request flows through the system:

**1. CLI sends inbound control request:**
```json
{"type": "control_request", "request_id": "req_1_a3f2", "method": "can_use_tool",
 "params": {"tool_name": "Bash", "input": {"command": "ls"}}}
```

**2. Transport adapter (cli.Adapter):**
- Reads JSON line from CLI stdout
- Passes raw `map[string]any` to protocol handler

**3. Protocol adapter (jsonrpc.Adapter):**
- Detects `type: "control_request"`
- Extracts `method: "can_use_tool"`
- Calls `permissions.Service.CheckToolUse()` with tool name and input
- Waits for result from domain service
- Formats control response with same `request_id`
- Sends response back through transport

**4. Domain service (permissions.Service):**
- Checks current permission mode (bypass, default, etc.)
- If mode requires check, calls user-provided callback
- Returns `PermissionResult` (allow/deny)
- NO knowledge of request IDs, timeouts, or JSON-RPC protocol

**5. Protocol adapter formats response:**
```json
{"type": "control_response", "request_id": "req_1_a3f2",
 "result": {"decision": "allow"}}
```

**6. Transport adapter writes to CLI stdin**

**Key Observations:**
- Domain service (`permissions.Service`) never sees request IDs
- Protocol adapter (`jsonrpc.Adapter`) handles all protocol mechanics
- Domain service is easily testable with mock callbacks
- Protocol could be replaced (gRPC, HTTP) without changing domain logic

### Benefits of This Architecture

1. Testability

- Domain services testable without infrastructure
- Mock adapters via interfaces
- No subprocess spawning in unit tests

2. Flexibility

- Swap CLI transport for HTTP transport
- Change JSON-RPC to gRPC
- Add new message parsers

3. Clarity

- Clear boundaries between layers
- Easy to understand where code belongs
- Package names describe purpose

4. Maintainability

- Infrastructure changes don't affect domain
- Domain changes don't ripple to all adapters
- Each layer has single responsibility

### Common Misunderstandings

**Misunderstanding 1: "Domain services don't interact with control protocol at all"**
- **Reality:** Domain services trigger control protocol interactions (e.g., permission checks, hook callbacks) but delegate the protocol mechanics to adapters
- **Example:** `streaming.Service` calls `protocol.StartMessageRouter()` which handles control protocol routing, but the service doesn't generate request IDs or track timeouts

**Misunderstanding 2: "All message handling is in domain services"**
- **Reality:** Domain services handle SDK messages (user, assistant, result), but the protocol adapter routes control requests/responses
- **Example:** When CLI asks "can I use this tool?", the jsonrpc adapter receives it, calls the domain service, and sends back the response

**Misunderstanding 3: "Adapters are just thin I/O wrappers"**
- **Reality:** The jsonrpc adapter contains significant logic for request/response correlation, timeout enforcement, and message routing
- **Example:** The adapter maintains a `map[requestID]chan response` to correlate async control responses with pending requests

**Misunderstanding 4: "Control protocol logic is duplicated between domain and adapters"**
- **Reality:** There is NO duplication. Domain services orchestrate WHAT happens (permission check, hook execution), adapters implement HOW it happens (request ID, timeout, routing)
- **Example:** Domain says "check permission for tool X", adapter says "send req_1_a3f2 to CLI, wait 60s for response, route answer back"

### Compile-Time Guarantees

All adapters verify interface compliance at compile time:

```go
// adapters/cli/transport.go
var _ ports.Transport = (*Adapter)(nil)
// adapters/jsonrpc/protocol.go
var _ ports.ProtocolHandler = (*Adapter)(nil)
// adapters/parse/parser.go
var _ ports.MessageParser = (*Adapter)(nil)
```

If an adapter doesn't fully implement its port interface, the code won't compile.

### Cross-References

This architectural summary is consistent with:

- **Phase 2 (Domain Services)** - Lines 5-22 explicitly state domain services delegate control protocol mechanics to adapters
- **Phase 3 (Adapters)** - Describes jsonrpc adapter implementing request ID generation, timeout handling, and message routing
- **Phase 4 (Public API)** - Shows how the public API wires domain services with adapter implementations
- **Key Design Decisions** - Documents the principle of keeping infrastructure concerns out of domain layer

**For Implementation Details:**
- Control protocol message types → See Phase 2, Control Protocol Context section
- Request ID format and generation → See Phase 3, jsonrpc adapter specification
- Hook callback ID format → See Phase 3, jsonrpc adapter specification
- Timeout enforcement mechanism → See Phase 3, jsonrpc adapter specification
- Domain service responsibilities → See Phase 2, section 2.1-2.4
