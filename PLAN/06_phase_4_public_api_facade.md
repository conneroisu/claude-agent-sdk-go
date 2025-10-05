## Phase 4: Public API (Facade)

The public API acts as a facade over the domain services, hiding the complexity of ports and adapters.

### Overview: Facade Contracts and Behavior Specifications

This phase defines the public-facing API contracts, error semantics, and lifecycle management guarantees. **Focus is on behavioral contracts, not implementation.**

### 4.0 Core Contracts and Error Semantics

#### Error Handling Guarantees

**Query() Function:**
- **Channel Closure on Error:** If initialization fails (e.g., MCP server connection error), both message and error channels MUST be closed before returning
- **Blocking Behavior:** Initialization is **synchronous and blocking** - function does not return until all MCP servers are connected or initialization fails
- **Partial Failure Policy:** If ANY MCP server fails to initialize, ALL successfully connected servers MUST be closed before returning error
- **Error Channel Guarantees:** Error channel receives exactly ONE error on initialization failure, then closes

**Client Type:**
- **Connect() Blocking Behavior:** `Connect()` is **synchronous** - blocks until Claude CLI connection established or fails
- **Resource Cleanup Guarantee:** `Close()` MUST close all resources (streaming service, MCP servers) even if some closures fail
- **Concurrent Access:** All Client methods are **mutex-protected** - safe for concurrent access from multiple goroutines
- **State Validity:** Methods requiring connection (SendMessage, ReceiveMessages) return `ErrNotConnected` if called before Connect() succeeds

**MCP Server Initialization:**
- **Atomicity:** Initialization of multiple MCP servers is **atomic** - either all succeed or all fail (with cleanup)
- **Context Cancellation:** If context is cancelled during initialization, all in-progress connections MUST be cleaned up
- **Transport Lifecycle:** MCP transports are owned by their adapters - callers MUST NOT close transports directly
- **Connection Verification:** Initialization MUST verify MCP server responds to initial handshake before returning success

#### Channel Lifecycle Semantics

**Message Channels (from Query() and Client.ReceiveMessages()):**
- **Closure Guarantee:** Message channel closes when conversation ends normally OR on unrecoverable error
- **Error Coordination:** If error channel receives error, message channel MAY still deliver buffered messages before closing
- **No Sends After Close:** Implementation MUST NOT send on closed channels (panic prevention)

**Error Channels:**
- **Buffering:** Error channels MUST be buffered (at least size 1) to prevent goroutine leaks
- **Multiple Errors:** Error channel MAY receive multiple errors during streaming (one per failed operation)
- **Final State:** Error channel closes when no more errors possible (conversation ended, connection closed)

#### Open Questions and Design Decisions

**Lifecycle Management (To Be Resolved in Implementation):**
1. **MCP Server Reconnection:** Should ClientAdapter auto-reconnect on transport failure? Or fail fast and require user retry?
   - **Considerations:** Auto-reconnect adds complexity, fail-fast is more predictable
   - **Recommendation:** Start with fail-fast, add reconnect in future if needed

2. **Hook Execution Timeouts:** What default timeout for hook execution? User-configurable per hook or global?
   - **Considerations:** Python SDK uses 30s default, but Go users may prefer control
   - **Recommendation:** 30s default with per-hook override via HookMatcher config

3. **Permissions Callback Blocking:** Should permission callbacks block streaming, or queue decisions?
   - **Considerations:** Blocking is simpler, queuing allows async UX
   - **Recommendation:** Blocking (matches Python SDK), document clearly

4. **Resource Cleanup Order:** In `Client.Close()`, should MCP servers close before or after streaming service?
   - **Considerations:** Streaming may need MCP servers for final messages
   - **Recommendation:** Close streaming first (no new requests), then MCP servers

5. **Context Propagation:** Should individual MCP server calls inherit Query()/Connect() context, or use fresh background context?
   - **Considerations:** Inherited context enables cascading cancellation, background context prevents premature termination
   - **Recommendation:** Inherit context (standard Go practice)

#### Type Dependencies and References

**Types Defined in This Phase:**
- `Client` struct - defined in `client.go`
- `ErrNotConnected` error - defined in `errors.go`

**Types Referenced from Other Phases:**
- `options.AgentOptions` - **To be defined in Phase 4.2** (options package)
- `options.MCPServerConfig` interface - **To be defined in Phase 4.2**
- `options.StdioServerConfig`, `HTTPServerConfig`, `SSEServerConfig`, `SDKServerConfig` - **To be defined in Phase 4.2**
- `permissions.PermissionsConfig` - **To be defined in Phase 5c** (permissions)
- `permissions.Service` - **To be defined in Phase 5c**
- `hooking.HookEvent`, `hooking.HookMatcher` - **To be defined in Phase 5a** (hooks)
- `ports.MCPServer` - **Defined in Phase 1** (core domain ports)
- `messages.Message` - **Defined in Phase 1**
- All adapter types (cli, jsonrpc, parse, mcp) - **Defined in Phase 3**
- All domain services (querying, streaming) - **Defined in Phase 2**

### 4.1 Query Function (query.go)

**Priority:** Critical

**Signature:**
```go
func Query(
    ctx context.Context,
    prompt string,
    opts *options.AgentOptions,
    hooks map[hooking.HookEvent][]hooking.HookMatcher,
) (<-chan messages.Message, <-chan error)
```

**Contract:**

**Purpose:** One-shot query to Claude - wires up all layers and executes query via domain service

**Behavioral Guarantees:**
1. **Nil Options Handling:** If `opts` is nil, uses default `&options.AgentOptions{}`
2. **Layer Wiring Order:**
   - Infrastructure adapters created first (CLI transport, JSON-RPC protocol, parser)
   - Domain services created second (hooks, permissions)
   - MCP servers initialized third (may fail)
   - Query service created last with all dependencies
3. **Error Fast-Fail:** If MCP server initialization fails:
   - Creates CLOSED message channel (no messages will be sent)
   - Creates error channel with single error, then closes
   - Returns both closed channels immediately
4. **Success Path:** If initialization succeeds, delegates to `querying.Service.Execute()` and returns its channels

**Channel Contracts:**
- **Message Channel:** Delivers parsed messages from Claude; closes when conversation ends
- **Error Channel:** Buffered, delivers errors during streaming; closes when no more errors possible
- **Error State:** On init failure, both channels closed before return (safe to range over)

**Implementation Requirements:**
- MUST create all adapters before domain services (dependency order)
- MUST initialize MCP servers before creating query service
- MUST close both channels before returning on init error
- MUST NOT block on channel sends (goroutine leak prevention)
### 4.2 Client Type (client.go)

**Priority:** Critical

**Type Signature:**
```go
type Client struct {
    // Private fields (implementation detail)
}
```

**Public API:**
```go
func NewClient(
    opts *options.AgentOptions,
    hooks map[hooking.HookEvent][]hooking.HookMatcher,
    perms *permissions.PermissionsConfig,
) *Client

func (c *Client) Connect(ctx context.Context, prompt *string) error
func (c *Client) SendMessage(ctx context.Context, msg string) error
func (c *Client) ReceiveMessages(ctx context.Context) (<-chan messages.Message, <-chan error)
func (c *Client) Close() error
```

**Contract:**

**NewClient:**
- **Purpose:** Creates disconnected client with configuration
- **Nil Options:** If `opts` is nil, uses default `&options.AgentOptions{}`
- **No I/O:** Constructor does NOT connect to Claude (fail-fast principle)
- **Thread-Safety:** Returned client is safe for concurrent method calls

**Connect():**
- **Purpose:** Establishes Claude CLI connection and initializes all services
- **Blocking:** Synchronous - returns only after connection established or failure
- **State Transition:** On success, client transitions to "connected" state
- **Error Behavior:** On failure, client remains in "disconnected" state, can retry
- **Idempotency:** Calling Connect() on already-connected client returns error
- **Wiring Order:** (Same as Query - adapters, services, MCP servers, streaming service)
- **Resource Tracking:** MCP servers stored internally for cleanup in Close()

**SendMessage():**
- **Purpose:** Sends user message to Claude in established conversation
- **Pre-Condition:** MUST call Connect() first, else returns `ErrNotConnected`
- **Blocking:** Blocks until message sent to CLI transport or error
- **Thread-Safety:** Mutex-protected, safe for concurrent calls (though not typical usage)

**ReceiveMessages():**
- **Purpose:** Returns channels for streaming messages and errors from Claude
- **Pre-Condition:** MUST call Connect() first, else returns nil message channel + error channel with `ErrNotConnected`
- **Channel Lifecycle:** Channels remain valid until conversation ends or Close() called
- **Multiple Calls:** Calling multiple times returns SAME channels (not new ones)

**Close():**
- **Purpose:** Disconnects from Claude and releases all resources
- **Cleanup Order:**
  1. Close streaming service first (stops new message processing)
  2. Close MCP servers second (after no more messages will use them)
- **Error Collection:** If multiple resources fail to close, collects all errors and returns combined error
- **Idempotency:** Calling Close() multiple times is safe (no-op after first)
- **Thread-Safety:** Mutex-protected

**Concurrency Guarantees:**
- All public methods protected by internal mutex
- Safe to call SendMessage() from one goroutine while ranging over ReceiveMessages() in another
- Connect() and Close() should NOT be called concurrently (undefined behavior)

### 4.3 MCP Server Initialization (mcp_init.go)

**Priority:** Critical

**Internal Helper Functions (Not Exported):**

```go
func initializeMCPServers(
    ctx context.Context,
    configs map[string]options.MCPServerConfig,
) (map[string]ports.MCPServer, error)

func initializeMCPServer(
    ctx context.Context,
    name string,
    cfg options.MCPServerConfig,
) (ports.MCPServer, error)

func mapToEnvSlice(m map[string]string) []string
```

**Behavioral Specification:**

**initializeMCPServers:**
- **Purpose:** Batch-initialize all configured MCP servers atomically
- **Empty Config:** If `configs` is nil or empty, returns `(nil, nil)` - not an error
- **Atomicity Guarantee:** If ANY server fails to initialize, ALL previously connected servers MUST be closed before returning error
- **Error Format:** Wraps errors with server name for debugging: `"failed to initialize MCP server %q: %w"`
- **Success Return:** Map of server name → connected `ports.MCPServer` adapter

**initializeMCPServer:**
- **Purpose:** Creates single MCP server connection based on config type
- **Type Dispatch:** Uses type switch on `options.MCPServerConfig` interface:
  - `StdioServerConfig`: Creates stdio subprocess transport, connects MCP SDK client
  - `HTTPServerConfig`: Creates HTTP streamable transport, connects MCP SDK client
  - `SSEServerConfig`: Uses same streamable transport as HTTP (SSE is transport detail)
  - `SDKServerConfig`: Creates SDK server adapter with in-memory transport (see Phase 5b)
- **Unknown Type:** Returns error for unrecognized config types (defensive)
- **Transport Ownership:** Created transports owned by MCP SDK client/session - do NOT close separately
- **Adapter Wrapping:** For stdio/HTTP/SSE, wraps MCP SDK `ClientSession` in `mcp.ClientAdapter` implementing `ports.MCPServer`
- **Context Respect:** Uses provided context for subprocess creation and connection timeout

**mapToEnvSlice:**
- **Purpose:** Converts environment variable map to slice format for `os/exec.Cmd.Env`
- **Format:** Each entry formatted as `"KEY=VALUE"`
- **Map Iteration:** Order non-deterministic (map iteration), but acceptable for env vars

**Error Propagation Semantics:**
- Connection failures (network, subprocess spawn) propagate immediately
- MCP handshake failures (protocol version mismatch) propagate as connection errors
- Context cancellation during init triggers cleanup and returns context error

**Dependencies:**
- MCP SDK types: `mcpsdk.Client`, `mcpsdk.Transport`, `mcpsdk.ClientSession`
- MCP SDK transports: `CommandTransport` (stdio), `StreamableClientTransport` (HTTP/SSE)
- Internal adapter: `mcp.ClientAdapter` (wraps external MCP servers)
- Internal adapter: `mcp.SDKServerAdapter` (wraps SDK-managed servers) - **defined in Phase 5b**

### 4.4 MCP Client Adapter (adapters/mcp/client.go)

**Priority:** Critical

**Purpose:** Adapter wrapping MCP SDK `ClientSession` to connect TO external MCP servers (stdio/HTTP/SSE)

**Type Signature:**
```go
type ClientAdapter struct {
    // Private fields
}

func NewClientAdapter(name string, session *mcpsdk.ClientSession) *ClientAdapter
```

**Interface Compliance:**
- MUST implement `ports.MCPServer` interface
- Compile-time verification: `var _ ports.MCPServer = (*ClientAdapter)(nil)`

**Behavioral Contract:**

**Name():**
- Returns server name provided at construction
- Used for logging and error messages

**HandleMessage(ctx, message []byte):**
- **Purpose:** Proxies raw JSON-RPC messages from Claude CLI to external MCP server
- **Message Flow:** Claude CLI → Domain → ClientAdapter → MCP SDK ClientSession → External Server
- **Protocol:** Decodes JSON-RPC message, routes to appropriate MCP SDK method, encodes response
- **Request Handling:**
  - Supported methods: `tools/list`, `tools/call`, `resources/list`, `resources/read`, `prompts/list`, `prompts/get`
  - Deserializes params into MCP SDK param types
  - Calls corresponding `session.ListTools()`, `session.CallTool()`, etc.
  - Wraps result in JSON-RPC response
- **Error Handling:**
  - Invalid JSON → returns error
  - Unsupported method → returns JSON-RPC error response (code -32603)
  - MCP SDK call fails → returns JSON-RPC error response with error message
- **Notification Handling:** One-way notifications processed but no response sent (per JSON-RPC spec)
- **Context Propagation:** Passes context to all MCP SDK calls (enables timeout/cancellation)

**Close():**
- Delegates to `session.Close()` - closes underlying transport
- MUST be idempotent (safe to call multiple times)
- Returns error if session close fails (e.g., subprocess kill error)

**Implementation Notes:**
- This adapter is for EXTERNAL MCP servers (user connects TO them)
- For SDK MCP servers (user HOSTS tools), see `SDKServerAdapter` in **Phase 5b**
- Adapter does NOT own session lifecycle - session created by `initializeMCPServer()`

**Error Semantics:**
- Decode errors → return Go error (caller handles)
- MCP method errors → return JSON-RPC error response (protocol-compliant)
- Transport errors → propagate from session methods (network failures, subprocess exit)

### 4.5 Helper Utilities (helpers/ package)

**Priority:** High

**Purpose:** Convenience utilities for common SDK operations (tool selection, prompt building)

**Package Structure:**
- `helpers/tools.go` - Tool selection helpers
- `helpers/prompts.go` - System prompt builders

#### Tool Selection Helpers (helpers/tools.go)

**Function Signatures:**
```go
func ToolsToString(tools []options.BuiltinTool) string
func AllowTools(specs ...string) string
func DenyTools(specs ...string) string
func AllToolsExcept(exclude ...options.BuiltinTool) []options.BuiltinTool
```

**Behavioral Contracts:**

**ToolsToString:**
- Converts slice of `BuiltinTool` to comma-separated string for CLI flags
- Example: `[ToolRead, ToolWrite]` → `"Read,Write"`
- Empty slice → empty string

**AllowTools / DenyTools:**
- Joins tool specs (names or patterns) into comma-separated string
- Supports matcher patterns: `"Bash(git:*)"`, `"Read"`, etc.
- `DenyTools` is alias for `AllowTools` (semantic naming for clarity)

**AllToolsExcept:**
- Returns all 18 builtin tools EXCEPT specified exclusions
- **Complete Tool List:** Bash, BashOutput, KillShell, Read, Write, Edit, Glob, Grep, Task, ExitPlanMode, WebFetch, WebSearch, ListMcpResources, ReadMcpResource, Mcp, NotebookEdit, TodoWrite, SlashCommand
- Uses map for O(1) exclusion lookup
- Order non-deterministic (map iteration) - acceptable for tool lists

**Design Rationale:**
- Simplifies tool configuration (avoid manually listing 18 tools)
- Follows "allow by default, deny explicitly" pattern
- Enables concise deny-list UX: `AllToolsExcept(ToolBash, ToolWebFetch)`

#### System Prompt Helpers (helpers/prompts.go)

**Function Signatures:**
```go
func BuildSystemPrompt(parts ...string) string
func AppendSystemPrompt(base, append string) string
```

**Behavioral Contracts:**

**BuildSystemPrompt:**
- Joins prompt parts with double newlines (`"\n\n"`)
- Variadic - accepts any number of strings
- Empty parts included as-is (caller responsible for filtering)

**AppendSystemPrompt:**
- Safely appends to base prompt with separator
- **Edge Cases:**
  - Base empty → returns append only (no leading separator)
  - Append empty → returns base only (no trailing separator)
  - Both empty → returns empty string
- Separator: double newline (`"\n\n"`)

**Design Rationale:**
- Provides consistent prompt formatting across SDK usage
- Handles edge cases (nil/empty strings) gracefully
- Matches common LLM prompt construction patterns

#### Usage Example (from docs):
```go
// Exclude dangerous tools
opts := &options.AgentOptions{
    AllowedTools: helpers.AllToolsExcept(
        options.ToolBash,
        options.ToolWebFetch,
    ),
}

// Build multi-part system prompt
systemPrompt := helpers.BuildSystemPrompt(
    "You are a code review assistant.",
    "Focus on security and performance.",
)
```

### 4.6 Options Configuration Types (options/ package)

**Priority:** Critical

**Purpose:** Type-safe configuration structures for SDK initialization

**Package Structure:**
- `options/agent.go` - `AgentOptions` main config struct
- `options/mcp.go` - MCP server config types (discriminated union)
- `options/tools.go` - `BuiltinTool` enum and constants
- `options/permissions.go` - `PermissionsConfig` and `PermissionMode` - **Moved to Phase 5c** (permissions package owns this)

#### AgentOptions (options/agent.go)

**Type Definition:**
```go
type AgentOptions struct {
    CLIPath           string
    AllowedTools      []BuiltinTool
    DeniedTools       []BuiltinTool
    MCPServers        map[string]MCPServerConfig
    PermissionsConfig *permissions.PermissionsConfig  // See Phase 5c
    SystemPrompt      string
    // Additional fields TBD during implementation
}
```

**Field Semantics:**
- `CLIPath`: Absolute or relative path to Claude CLI binary; empty → search PATH
- `AllowedTools`: Explicit allow-list; nil → all tools allowed
- `DeniedTools`: Explicit deny-list; conflicts with AllowedTools checked at runtime
- `MCPServers`: Map of server name → config; nil → no MCP servers
- `PermissionsConfig`: Optional permission control; nil → allow all
- `SystemPrompt`: Custom system prompt; empty → use CLI default

**Validation Contract:**
- `AllowedTools` and `DeniedTools` MUST NOT both be non-empty (ambiguous intent)
- Empty options valid (all defaults)

#### MCP Server Config Types (options/mcp.go)

**Interface:**
```go
type MCPServerConfig interface {
    mcpServerConfig()  // Private marker method prevents external implementations
}
```

**Concrete Types:**

```go
type StdioServerConfig struct {
    Command string
    Args    []string
    Env     map[string]string  // Optional environment variables
}

type HTTPServerConfig struct {
    URL     string
    Headers map[string]string  // Optional HTTP headers
}

type SSEServerConfig struct {
    URL     string
    Headers map[string]string
}

type SDKServerConfig struct {
    Instance *mcp.Server  // User-hosted MCP server (Phase 5b)
}
```

**Design Rationale:**
- **Discriminated Union:** Private marker method ensures only these 4 types implement interface
- **Type Safety:** Config type determines transport type in `initializeMCPServer()`
- **Future-Proof:** Can add new transport types (WebSocket, gRPC) by adding new concrete types

**Behavioral Semantics:**
- `StdioServerConfig.Env` nil → inherit parent process env
- `HTTPServerConfig.Headers` / `SSEServerConfig.Headers` nil → no custom headers
- `SDKServerConfig.Instance` must be non-nil (validated in init)

#### BuiltinTool Enum (options/tools.go)

**Type Definition:**
```go
type BuiltinTool string

const (
    ToolBash             BuiltinTool = "Bash"
    ToolBashOutput       BuiltinTool = "BashOutput"
    ToolKillShell        BuiltinTool = "KillShell"
    ToolRead             BuiltinTool = "Read"
    ToolWrite            BuiltinTool = "Write"
    ToolEdit             BuiltinTool = "Edit"
    ToolGlob             BuiltinTool = "Glob"
    ToolGrep             BuiltinTool = "Grep"
    ToolTask             BuiltinTool = "Task"
    ToolExitPlanMode     BuiltinTool = "ExitPlanMode"
    ToolWebFetch         BuiltinTool = "WebFetch"
    ToolWebSearch        BuiltinTool = "WebSearch"
    ToolListMcpResources BuiltinTool = "ListMcpResources"
    ToolReadMcpResource  BuiltinTool = "ReadMcpResource"
    ToolMcp              BuiltinTool = "Mcp"
    ToolNotebookEdit     BuiltinTool = "NotebookEdit"
    ToolTodoWrite        BuiltinTool = "TodoWrite"
    ToolSlashCommand     BuiltinTool = "SlashCommand"
)

func (t BuiltinTool) WithMatcher(pattern string) string
```

**WithMatcher Behavior:**
- **Purpose:** Creates pattern-based tool spec for CLI flags
- **Format:** `"ToolName(pattern)"` - e.g., `"Bash(git:*)"`, `"Read(*.go)"`
- **Return Type:** String (not BuiltinTool) - represents CLI spec, not enum value
- **Usage:** Combine with `helpers.AllowTools()` for granular control

**Tool Count Verification:**
- Total: **18 tools** (matches evaluation criteria)
- Categories: Shell (3), File (3), Search (2), Planning (2), Web (2), MCP (3), Other (3)

---

## Linting Compliance Notes

### File Size Requirements (175 line limit)

**Public API files:**
- ✅ `client.go` - Estimated 130 lines (compliant with error handling)
- ✅ `query.go` - Estimated 95 lines (compliant with error handling)
- ✅ `mcp.go` - 35 lines (SDK MCP server public API)
- ✅ `mcp_init.go` - 95 lines (compliant, includes SDK server handling)
- ✅ `errors.go` - Estimated 60 lines (compliant)

**MCP adapter files:**
- ✅ `adapters/mcp/client.go` - 125 lines (external MCP client adapter)
- ✅ `adapters/mcp/sdk_server.go` - 175 lines (SDK server adapter + in-memory transport)

**Helper utilities files:**
- ✅ `helpers/tools.go` - 60 lines (compliant)
- ✅ `helpers/prompts.go` - 20 lines (compliant)

All files are under the 175-line limit.

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

### Implementation Checklist

**Behavioral Contract Verification:**
- [ ] Query() closes both channels on initialization error (test with failing MCP server)
- [ ] Query() cleanup on partial MCP failure tested (connect 2 servers, fail 2nd, verify 1st closed)
- [ ] Client.Connect() is idempotent (calling twice returns error on 2nd call)
- [ ] Client.ReceiveMessages() returns same channels on multiple calls (pointer equality check)
- [ ] Client.Close() collects all errors (test with multiple failing MCP servers)
- [ ] Client.Close() is idempotent (calling twice succeeds, 2nd is no-op)

**Error Semantics Verification:**
- [ ] Error channels are buffered (no goroutine leaks on unread errors)
- [ ] Message channels close before error channels on normal termination
- [ ] Context cancellation during MCP init triggers cleanup (verify with timeout context)
- [ ] MCP connection failure returns descriptive error with server name

**Lifecycle Management:**
- [ ] MCP server cleanup order tested: streaming first, then MCP servers
- [ ] Context propagation verified: parent context cancellation propagates to MCP calls
- [ ] Resource leak test: Connect() → immediate Close() → no leaked goroutines/file descriptors

**Type System Verification:**
- [ ] All 18 BuiltinTool constants defined (count verification)
- [ ] MCPServerConfig private marker method prevents external implementations
- [ ] AgentOptions validation rejects both AllowedTools AND DeniedTools non-empty
- [ ] BuiltinTool.WithMatcher() returns string (not BuiltinTool) - compile-time check

**MCP Client Integration (External Servers):**
- [ ] initializeMCPServer() handles all 4 config types (stdio, HTTP, SSE, SDK)
- [ ] ClientAdapter implements ports.MCPServer (compile-time check)
- [ ] ClientAdapter.HandleMessage() supports all 6 MCP methods (tools/list, tools/call, etc.)
- [ ] ClientAdapter returns JSON-RPC error response on method failure (not Go error)
- [ ] Unsupported JSON-RPC method returns error code -32603

**Helper Utilities:**
- [ ] AllToolsExcept() with empty exclusions returns all 18 tools
- [ ] AllToolsExcept() with all tools excluded returns empty slice
- [ ] AppendSystemPrompt() edge cases tested (both empty, one empty, both non-empty)
- [ ] BuildSystemPrompt() joins with double newlines (verify separator)

**Documentation Requirements:**
- [ ] Query() godoc explains channel closure guarantees
- [ ] Client.Connect() godoc warns about NOT calling concurrently with Close()
- [ ] initializeMCPServers() documents atomicity guarantee
- [ ] All 5 open questions documented in package-level comment (for future resolution)

**Code Quality:**
- [ ] All files under 175 lines (enforcement via linter)
- [ ] Public constructors ≤4 parameters (Query: 4, NewClient: 3 - compliant)
- [ ] No full implementations in plan (this doc is contracts only)
