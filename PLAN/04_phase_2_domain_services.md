## Phase 2: Domain Services

### Overview

Domain services implement the core business logic for query execution, streaming conversations, hook orchestration, and permission management. They focus exclusively on **WHAT** needs to happen, delegating **HOW** (timeouts, request IDs, protocol routing) to adapter implementations.

### Key Architectural Boundaries

**Domain Services Responsibilities:**
- Orchestrate message flow and conversation lifecycle
- Coordinate hook execution and permission checks
- Parse and validate domain messages
- Manage service state (connected/disconnected)

**Adapter Responsibilities (NOT domain services):**
- Generate request IDs and callback IDs
- Track pending control protocol requests
- Implement timeout mechanisms
- Route messages by protocol type
- Handle transport-level concerns

**Critical Design Decision:** All control protocol mechanics (request ID generation, timeout handling, callback tracking) belong in the `jsonrpc` adapter. Domain services interact only through port interfaces.

### Control Protocol Context

The SDK uses a JSON-RPC control protocol with three message types:

1. **SDK Messages** - Regular messages (user, assistant, system, result, stream_event)
2. **Control Requests** - Bidirectional control messages (`type: "control_request"`)
3. **Control Responses** - Responses to control requests (`type: "control_response"`)

**Control Request Types:**

**SDK → CLI (Outbound):**
- `interrupt` - Stop current operation
- `set_permission_mode` - Change permission mode
- `set_model` - Change AI model
- `initialize` - Setup hooks (streaming only)

**CLI → SDK (Inbound):**
- `can_use_tool` - Ask permission for tool use
- `hook_callback` - Execute registered hook
- `mcp_message` - Proxy message to/from MCP server

**Note:** Request ID generation (`req_{counter}_{randomHex}`), callback ID tracking (`hook_{index}`), and 60s timeout enforcement are **adapter concerns**, implemented in Phase 3.

### Concurrency Policy

**Goroutine Safety Rules:**
- Domain services MAY launch goroutines for async operations
- All goroutines MUST respect context cancellation via `select` statements
- Shared state (maps, counters) belongs in adapters with proper synchronization
- Domain services use immutable data and channel communication

**Channel Ownership:**
- Services that create channels OWN them and MUST close them
- Receiver-only channels (`<-chan`) are consumed but not closed by domain services
- Buffered channels MAY be used for error channels (buffer=1) to prevent goroutine leaks
- Unbuffered channels provide natural backpressure for message flow

**Validation Strategy for `map[string]any` Inputs:**
- Domain services receive `map[string]any` from adapters (preserves protocol flexibility)
- Services MUST validate required fields before using data
- Type assertions MUST check success before using values
- Invalid data returns errors to caller; services do not panic
- Hook inputs include `hook_event_name` field for type discrimination

**Adapter State vs Domain State:**
- Request ID counters → Adapter state (protected by mutex)
- Callback ID maps → Adapter state (protected by mutex)
- Pending request channels → Adapter state (synchronized via map operations)
- Service lifecycle (connected/closed) → Domain state (single-threaded or atomic)

---

### 2.1 Querying Service (querying/service.go)
Priority: Critical

**Purpose:** Orchestrate one-shot query execution with automatic lifecycle management

**Responsibilities:**
1. Coordinate transport connection for query session
2. Format and send user prompt message
3. Stream parsed messages to caller via channels
4. Delegate control protocol handling to protocol adapter
5. Manage cleanup on completion or error

**Python SDK Parity:**
- Mirrors Python's `query()` function behavior
- One-shot execution (connect → prompt → stream → auto-cleanup)
- Uses channels instead of async generators
- Equivalent lifecycle management

**Dependencies (Ports Only):**
- `ports.Transport` - Connection and message I/O
- `ports.ProtocolHandler` - Control protocol routing
- `ports.MessageParser` - Raw message parsing
- `hooking.Service` - Hook orchestration (optional)
- `permissions.Service` - Permission checks (optional)
- `map[string]ports.MCPServer` - MCP server routing (optional)

**Concurrency Model:**
- Execute() launches background goroutine
- Returns `(<-chan messages.Message, <-chan error)` immediately
- Goroutine closes both channels on completion
- Context cancellation stops execution
- Protocol adapter manages its own routing goroutine

**Key Behaviors:**
- NO explicit initialization message (differs from streaming)
- Protocol adapter handles inbound control requests transparently
- Parser converts `map[string]any` to typed `messages.Message`
- Error channel buffered (size=1) to prevent goroutine leak on early return

**Implementation Guidance:**

**Package Structure:**
- `querying/service.go` - Service struct and constructor
- `querying/execute.go` - Execute implementation
- `querying/types.go` - Domain types if needed

**Service Struct:**
- Holds port dependencies (transport, protocol, parser)
- Holds optional services (hooks, permissions)
- Holds MCP server map for routing
- NO stateful fields (counters, maps) - those belong in adapters

**Execute() Flow:**
1. Create message and error channels (error buffered, size=1)
2. Launch goroutine with deferred channel closes
3. Call `transport.Connect(ctx)`
4. Start protocol router via `protocol.StartMessageRouter()` (pass hooks, permissions, MCP servers)
5. Marshal and send user prompt message
6. Loop: select on context, router messages, router errors
7. Parse messages via `parser.Parse()` and forward to message channel
8. Return channels immediately to caller

**Test Checkpoints:**
- [ ] Transport connection failure propagates to error channel
- [ ] Context cancellation stops goroutine without leak
- [ ] Protocol router receives hook callbacks and permission service
- [ ] Prompt message formatted correctly (`type: "user"`, `prompt: "..."`)
- [ ] Parser errors propagate to error channel
- [ ] Message channel receives parsed messages in order
- [ ] Both channels closed on completion
- [ ] Error channel non-blocking (buffered) when goroutine exits early
- [ ] Mock transport failures trigger appropriate error handling
- [ ] Mock parser failures trigger appropriate error handling
### 2.2 Streaming Service (streaming/service.go)
Priority: Critical

**Purpose:** Manage persistent bidirectional streaming conversations

**Responsibilities:**
1. Establish persistent connection for multi-turn conversation
2. Send user messages on demand
3. Stream incoming messages continuously
4. Manage conversation lifecycle (connect/send/receive/close)
5. Delegate control protocol handling to protocol adapter

**Python SDK Parity:**
- Mirrors Python's streaming session behavior
- Persistent connection (connect → send/receive loop → explicit close)
- Uses channels for async message delivery
- Initialization message sent for hook registration (if hooks present)

**Dependencies (Ports Only):**
- `ports.Transport` - Connection and message I/O
- `ports.ProtocolHandler` - Control protocol routing
- `ports.MessageParser` - Raw message parsing
- `hooking.Service` - Hook orchestration (optional)
- `permissions.Service` - Permission checks (optional)
- `map[string]ports.MCPServer` - MCP server routing (optional)

**Concurrency Model:**
- Connect() establishes session synchronously
- ReceiveMessages() launches background goroutine for streaming
- SendMessage() writes synchronously (transport handles concurrency)
- Close() synchronously closes transport
- Internal channels owned by service (created in constructor or Connect)

**Key Behaviors:**
- MUST send initialization control request if hooks present (streaming only)
- Connect() may optionally send initial prompt
- SendMessage() can be called multiple times
- ReceiveMessages() returns new channels each call (or cached singleton?)
- Close() terminates transport and stops message flow

**Implementation Guidance:**

**Package Structure:**
- `streaming/service.go` - Service struct and constructor
- `streaming/connect.go` - Connect implementation
- `streaming/send.go` - SendMessage implementation
- `streaming/receive.go` - ReceiveMessages implementation
- `streaming/lifecycle.go` - Close and cleanup

**Service Struct:**
- Port dependencies (transport, protocol, parser)
- Optional services (hooks, permissions)
- MCP server map
- Internal routing channels (created in Connect)
- Connection state (connected bool or atomic)

**Connect() Flow:**
1. Call `transport.Connect(ctx)`
2. Create internal routing channels (msgCh, errCh)
3. Start protocol router via `protocol.StartMessageRouter()`
4. If hooks exist, send initialization control request via protocol
5. If initial prompt provided, format and send user message
6. Return nil on success

**SendMessage() Flow:**
1. Format user message (`type: "user"`, `prompt: "..."`)
2. Marshal to JSON
3. Call `transport.Write(ctx, jsonLine)`
4. Return error if any

**ReceiveMessages() Flow:**
1. Create output channels (msgCh, errCh with buffer=1)
2. Launch goroutine with deferred closes
3. Loop: select on context, internal msgCh, internal errCh
4. Parse messages via `parser.Parse()` and forward
5. Return output channels immediately

**Close() Flow:**
1. Call `transport.Close()`
2. Internal channels closed by protocol router
3. Return transport error if any

**Test Checkpoints:**
- [ ] Connect() without prompt succeeds without sending message
- [ ] Connect() with prompt sends formatted user message
- [ ] Connect() with hooks sends initialization control request
- [ ] SendMessage() formats and writes user message correctly
- [ ] ReceiveMessages() streams parsed messages continuously
- [ ] Context cancellation stops receive goroutine
- [ ] Close() terminates transport and stops message flow
- [ ] Multiple SendMessage() calls work correctly
- [ ] Parser errors in ReceiveMessages() propagate to error channel
- [ ] Transport write failures in SendMessage() return errors
- [ ] Hook initialization includes all callback IDs
- [ ] MCP servers available to protocol router for routing
### 2.3 Hooking Service (hooking/service.go)
Priority: High

**Purpose:** Orchestrate hook callback execution based on event matchers

**Responsibilities:**
1. Store hook configurations (event → matchers → callbacks)
2. Execute matching hooks for events
3. Aggregate hook results (later hooks override earlier)
4. Handle blocking decisions (decision="block" stops execution)
5. Validate hook inputs and handle errors

**Python SDK Parity:**
- Mirrors Python's hook execution model
- Supports 9 hook events (PreToolUse, PostToolUse, etc.)
- Pattern matching for tool-specific hooks
- Aggregated results from multiple hooks

**Hook Events (Constants):**
- `PreToolUse` - Before tool execution
- `PostToolUse` - After tool execution
- `UserPromptSubmit` - User submits prompt
- `Notification` - CLI notification
- `SessionStart` - Session begins
- `SessionEnd` - Session ends
- `Stop` - User stops operation
- `SubagentStop` - Subagent stops
- `PreCompact` - Before transcript compaction

**Hook Input Types:**
- Define types for each hook event (PreToolUseHookInput, etc.)
- All include BaseHookInput (session_id, transcript_path, cwd, permission_mode)
- Use `map[string]any` at callback level for flexibility
- Include `hook_event_name` field for type discrimination

**Dependencies:**
- NONE - Pure domain service with no external dependencies
- Callbacks provided by user at construction time

**Implementation Guidance:**

**Package Structure:**
- `hooking/service.go` - Service struct and core methods
- `hooking/types.go` - Hook input types and constants
- `hooking/matching.go` - Pattern matching logic
- `hooking/execution.go` - Hook execution logic

**Service Struct:**
- `hooks map[HookEvent][]HookMatcher` - Hook configuration
- NO other state

**HookCallback Signature:**
```
func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error)
```

**Execute() Flow:**
1. Check if service and hooks are nil (return nil, nil)
2. Find matchers for event type
3. For each matcher, check pattern match against input
4. For each matching callback:
   - Check context cancellation
   - Execute callback with input, toolUseID, HookContext
   - If result contains `decision: "block"`, return immediately
   - Otherwise aggregate results (merge into map)
5. Return aggregated result

**Pattern Matching Logic:**
- Empty pattern or "*" matches all
- For PreToolUse/PostToolUse: match against `tool_name` field in input
- Exact string match (no regex in initial version)
- Failed type assertion on tool_name → no match

**GetHooks() Method:**
- Returns hook configuration map
- Used by protocol adapter for initialization

**Register() Method:**
- Adds new hook matcher to event
- Initializes map if nil

**Test Checkpoints:**
- [ ] Nil service returns nil, nil from Execute()
- [ ] Empty hooks return nil, nil from Execute()
- [ ] Pattern "*" matches all events
- [ ] Pattern matching works for tool names (PreToolUse/PostToolUse)
- [ ] Hook execution respects context cancellation
- [ ] decision="block" stops execution and returns immediately
- [ ] Multiple hooks aggregate results correctly
- [ ] Later hooks override earlier hooks for same keys
- [ ] Hook errors propagate to caller
- [ ] Register() adds hooks correctly
- [ ] GetHooks() returns configuration
- [ ] Type assertions on input fields handle missing fields gracefully
### 2.4 Permissions Service (permissions/service.go)
Priority: High

**Purpose:** Manage tool permission checks and mode updates

**Responsibilities:**
1. Store permission mode and callback configuration
2. Execute permission checks via user callback
3. Handle permission mode logic (bypass, default, etc.)
4. Pass permission suggestions to user callback
5. Return allow/deny results to protocol adapter

**Python SDK Parity:**
- Mirrors Python's permission checking model
- Supports permission modes (bypass, default, plan, acceptEdits)
- Permission suggestions for "always allow" workflows
- Allow/deny result types

**Permission Modes:**
- `bypass` - Always allow, no checks
- `default` - Standard permission flow with callback
- `plan` - Planning mode (callback decides)
- `acceptEdits` - Accept edits mode (callback decides)

**Permission Result Types:**
- `PermissionResultAllow` - Tool allowed, optional input modifications, optional permission updates
- `PermissionResultDeny` - Tool denied, message, interrupt flag

**Permission Update Structure:**
- Type (e.g., "addRules")
- Rules (tool name + rule content patterns)
- Behavior (allow/deny/ask)
- Mode (optional)
- Directories (optional)
- Destination (userSettings/projectSettings/localSettings/session)

**Dependencies:**
- NONE - Pure domain service
- User callback provided at construction time
- `options.PermissionMode` enum from options package

**Implementation Guidance:**

**Package Structure:**
- `permissions/service.go` - Service struct and core methods
- `permissions/types.go` - Result types, update types, constants

**Service Struct:**
- `mode options.PermissionMode` - Current permission mode
- `canUseTool CanUseToolFunc` - User callback (optional)

**CanUseToolFunc Signature:**
```
func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error)
```

**CheckToolUse() Flow:**
1. Switch on permission mode
2. If `bypass`: return PermissionResultAllow immediately
3. If other modes:
   - If callback exists: call with ToolPermissionContext{Suggestions: suggestions}
   - If no callback: return PermissionResultAllow (CLI handles default behavior)
4. Handle callback errors
5. Return result

**ToolPermissionContext:**
- Contains `Suggestions []PermissionUpdate`
- Suggestions come from control protocol `permission_suggestions` field
- User can return these in UpdatedPermissions for "always allow" flow

**UpdateMode() Method:**
- Updates service mode field
- Used when CLI sends `set_permission_mode` control request
- No validation (mode values come from CLI)

**Test Checkpoints:**
- [ ] Bypass mode always returns allow
- [ ] Default mode calls user callback if provided
- [ ] Default mode returns allow if no callback
- [ ] Callback receives suggestions from protocol
- [ ] Callback errors propagate to caller
- [ ] UpdateMode() changes mode correctly
- [ ] PermissionResultAllow can include updated input
- [ ] PermissionResultAllow can include permission updates
- [ ] PermissionResultDeny includes message and interrupt flag
- [ ] Nil config creates service with default mode
- [ ] Unknown mode returns deny for safety
- [ ] Tool input as map[string]any passed to callback unchanged

---

## Linting Compliance Notes

### File Size Requirements (175 line limit)

**Package decomposition follows implementation guidance above:**

**querying/ package:**
- `service.go` - Service struct + constructor (~40 lines)
- `execute.go` - Execute implementation (~80 lines)
- `types.go` - Domain types if needed (~30 lines)

**streaming/ package:**
- `service.go` - Service struct + constructor (~40 lines)
- `connect.go` - Connect implementation (~70 lines)
- `send.go` - SendMessage implementation (~30 lines)
- `receive.go` - ReceiveMessages implementation (~60 lines)
- `lifecycle.go` - Close implementation (~20 lines)

**hooking/ package:**
- `service.go` - Service struct + core methods (~60 lines)
- `types.go` - Hook input types + constants (~80 lines)
- `matching.go` - Pattern matching logic (~40 lines)
- `execution.go` - Hook execution logic (~60 lines)

**permissions/ package:**
- `service.go` - Service struct + core methods (~60 lines)
- `types.go` - Result types + update types + constants (~80 lines)

### Complexity Management (25 line limit, complexity limits)

**Strategies to meet constraints:**
- **Early returns:** Reduce nesting depth in permission checks and hook execution
- **Extracted validation:** Separate functions for map[string]any field extraction
- **Small helper functions:** Break down Execute() flow into composable steps
- **Minimal switch statements:** Permission mode switch is simple (3-4 cases)
- **No complex routing:** Message routing delegated to protocol adapter

**Specific extractions:**
- Hook pattern matching → `matchesPattern()` helper (~10 lines)
- Hook aggregation → `aggregateResults()` helper (~15 lines)
- Permission mode check → `checkMode()` helper (~20 lines)
- Message formatting → `formatUserMessage()` helper (~10 lines)

### Dependency Verification

**Port dependencies only:**
- ✅ `ports.Transport` - Interface for I/O
- ✅ `ports.ProtocolHandler` - Interface for control protocol
- ✅ `ports.MessageParser` - Interface for parsing
- ✅ `ports.MCPServer` - Interface for MCP routing
- ✅ Domain services (hooking, permissions) - Within domain layer
- ✅ `options` package - Domain configuration types

**No adapter dependencies:**
- ❌ NO imports from `adapters/` packages
- ❌ NO direct transport implementations (cli, stdio)
- ❌ NO direct protocol implementations (jsonrpc)
- ❌ NO request ID generation logic
- ❌ NO timeout implementation details

### Checklist

- [ ] All services depend only on port interfaces
- [ ] No adapter implementation imports in domain services
- [ ] File sizes ≤ 175 lines
- [ ] Function complexity ≤ 15 (cyclomatic)
- [ ] Function complexity ≤ 20 (cognitive)
- [ ] Max nesting depth ≤ 3 levels
- [ ] All functions ≤ 25 lines
- [ ] Early return pattern used throughout
- [ ] Validation extracted to separate functions
- [ ] map[string]any inputs validated before use