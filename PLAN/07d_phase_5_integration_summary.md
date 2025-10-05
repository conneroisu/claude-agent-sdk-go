## Phase 5: Integrations Summary

### Overview
Phase 5 adds three critical integration capabilities to the Claude Agent SDK, enabling users to customize agent behavior and add custom tools.

### Integration Components

#### **5a. Hooks Support** (Critical Priority)
[📄 Full Documentation](./07a_phase_5_hooks.md)

Lifecycle hooks that execute at key points in agent execution:
- `PreToolUse` - Before tool execution (can block/modify)
- `PostToolUse` - After tool execution
- `UserPromptSubmit` - When user submits a prompt
- `Stop` - When agent stops
- `SubagentStop` - When subagent stops
- `PreCompact` - Before context compaction

**Key file:** `hooks.go` in public API

#### **5b. MCP Server Support** (Critical Priority)
[📄 Full Documentation](./07b_phase_5_mcp_servers.md)

In-process user-defined tools via Model Context Protocol:
- **Two integration modes:**
  - External servers: Connect TO external MCP servers via stdio/HTTP/SSE
  - SDK servers: User creates in-process servers with direct state access
- Wraps official `github.com/modelcontextprotocol/go-sdk`
- Provides `NewMCPServer()` and `AddTool()` convenience functions
- Type-safe tool definitions using Go generics
- Automatic JSON schema inference from struct types
- In-memory transport for zero IPC overhead (SDK servers)
- Unified `ports.MCPServer` interface for both modes

**Key files:**
- `mcp.go` - Public API (NewMCPServer, AddTool)
- `adapters/mcp/client.go` - External server adapter
- `adapters/mcp/sdk_server.go` - SDK server adapter

#### **5c. Permission Callbacks** (Medium Priority)
[📄 Full Documentation](./07c_phase_5_permissions.md)

Custom authorization logic for tool usage:
- `CanUseToolFunc` callback interface
- Can allow, deny, or modify tool requests
- Supports permission updates and suggestions

**Key file:** `permissions.go` in public API

---

## Cross-Cutting Implementation Guidance

### Integration Points
- **Hooks** integrate with tool execution pipeline in agent core
- **MCP servers** integrate via adapter pattern in `adapters/mcp/`
- **Permissions** integrate with tool authorization in agent core

### Shared Patterns
- All three use callback/interface patterns for user extensibility
- All expose simple public APIs that wrap complex internal machinery
- All follow the SDK's adapter pattern for protocol translation

### Dependencies
- Hooks may trigger permission checks
- Permission callbacks may generate hook events
- MCP tools participate in both hooks and permissions

---

## Integration Validation Checklist

This section provides a step-by-step checklist for validating that all three integration features work together correctly.

### Step 1: SDK MCP Server Integration Validation

**Objective:** Verify SDK-based MCP servers expose custom tools correctly.

- [ ] **1.1** Create SDK MCP server using `claude.NewMCPServer()` (to be defined in Phase 5b)
- [ ] **1.2** Add at least one custom tool with `claude.AddTool()` (to be defined in Phase 5b)
- [ ] **1.3** Pass server instance to agent options via `MCPServers` map (to be defined in Phase 4)
- [ ] **1.4** Verify tool appears in agent's available tools list
- [ ] **1.5** Send message that triggers custom tool usage
- [ ] **1.6** Verify tool handler receives correct arguments and executes
- [ ] **1.7** Verify tool result returns to agent correctly

**Types referenced (to be defined):**
- `claude.NewMCPServer()` - Phase 5b (MCP servers)
- `claude.AddTool()` - Phase 5b (MCP servers)
- `options.MCPServerConfig` - Phase 4 or 5b (configuration)
- `options.SDKServerConfig` - Phase 4 or 5b (configuration)

---

### Step 2: Hooks Integration Validation

**Objective:** Verify lifecycle hooks execute at correct points and can modify behavior.

- [ ] **2.1** Define hook callbacks for at least two hook events:
  - `PreToolUse` - Before tool execution
  - `PostToolUse` - After tool execution
- [ ] **2.2** Register hooks with agent (mechanism to be defined in Phase 4 or 5a)
- [ ] **2.3** Trigger tool execution via agent message
- [ ] **2.4** Verify `PreToolUse` hook fires BEFORE tool executes
- [ ] **2.5** Verify hook receives correct input (tool name, arguments, etc.)
- [ ] **2.6** Test hook blocking capability:
  - Hook returns `Continue: false` (to be defined in Phase 5a)
  - Verify tool does NOT execute
  - Verify appropriate response sent to CLI
- [ ] **2.7** Verify `PostToolUse` hook fires AFTER tool completes
- [ ] **2.8** Verify hook receives tool result in input

**Types referenced (to be defined):**
- `hooking.HookEvent` - Re-exported in Phase 5a from domain
- `hooking.HookCallback` - Re-exported in Phase 5a from domain
- `hooking.HookInput` - To be defined in Phase 5a or domain layer
- `hooking.HookOutput` - To be defined in Phase 5a or domain layer
- `hooking.HookMatcher` - To be defined in Phase 5a

---

### Step 3: Permissions Integration Validation

**Objective:** Verify permission callbacks control tool authorization correctly.

- [ ] **3.1** Define `CanUseTool` callback function (to be defined in Phase 5c)
- [ ] **3.2** Register callback in permissions config (to be defined in Phase 5c)
- [ ] **3.3** Trigger tool usage that callback should ALLOW
- [ ] **3.4** Verify callback receives correct parameters:
  - Tool name
  - Tool input arguments
  - Context information
- [ ] **3.5** Verify callback returns `PermissionResultAllow` (defined in Phase 2/5c)
- [ ] **3.6** Verify tool executes normally
- [ ] **3.7** Trigger tool usage that callback should DENY
- [ ] **3.8** Verify callback returns `PermissionResultDeny` (defined in Phase 2/5c)
- [ ] **3.9** Verify tool does NOT execute
- [ ] **3.10** Verify denial message reaches CLI correctly

**Types referenced:**
- `PermissionResultAllow` - Defined in Phase 2 and 5c
- `PermissionResultDeny` - Defined in Phase 2 and 5c
- `PermissionsConfig` - To be defined in Phase 5c
- `CanUseTool` callback signature - To be defined in Phase 5c

---

### Step 4: Combined Integration Validation

**Objective:** Verify all three integrations work together without conflicts.

- [ ] **4.1** Set up agent with ALL three integrations enabled:
  - SDK MCP server with custom tool
  - Hooks for `PreToolUse` and `PostToolUse`
  - Permission callback
- [ ] **4.2** Trigger tool execution and verify execution order:
  1. Permission callback fires FIRST
  2. If allowed, `PreToolUse` hook fires
  3. If hook allows, tool executes
  4. `PostToolUse` hook fires
- [ ] **4.3** Test permission denial blocks hook execution:
  - Permission callback returns DENY
  - Verify `PreToolUse` hook does NOT fire
  - Verify tool does NOT execute
- [ ] **4.4** Test hook blocking after permission allows:
  - Permission callback returns ALLOW
  - `PreToolUse` hook returns `Continue: false`
  - Verify tool does NOT execute
  - Verify `PostToolUse` hook does NOT fire
- [ ] **4.5** Test successful execution with all integrations:
  - Permission ALLOWS
  - `PreToolUse` hook ALLOWS (returns `Continue: true`)
  - Tool executes successfully
  - `PostToolUse` hook receives result
- [ ] **4.6** Verify no resource leaks or deadlocks occur

---

### Step 5: Resource Cleanup Validation

**Objective:** Verify proper cleanup when agent shuts down with active integrations.

**Cleanup Order (to be implemented):**
1. **Stop accepting new messages** - Agent stops processing new requests
2. **Complete in-flight tool executions** - Let active tools finish
3. **Fire cleanup hooks** (if any) - `Stop`, `SubagentStop` hooks
4. **Close MCP server connections** - SDK servers and external servers
5. **Release permission callback resources** - If callback holds resources
6. **Close transport channels** - stdin/stdout/stderr
7. **Wait for goroutines** - Ensure all background work completes

**Validation Steps:**
- [ ] **5.1** Start agent with all integrations active
- [ ] **5.2** Initiate agent shutdown via context cancellation or `Close()`
- [ ] **5.3** Verify in-flight tool executions complete before shutdown
- [ ] **5.4** Verify `Stop` hooks fire (if registered)
- [ ] **5.5** Verify MCP server connections close gracefully:
  - SDK server handlers stop accepting requests
  - External server connections send close frames
- [ ] **5.6** Verify permission callback is NOT called after shutdown starts
- [ ] **5.7** Verify all goroutines exit within timeout (e.g., 5 seconds)
- [ ] **5.8** Verify no goroutine leaks using runtime checks
- [ ] **5.9** Verify no channel deadlocks occur
- [ ] **5.10** Verify cleanup is idempotent (calling `Close()` twice is safe)

**Resource Cleanup Semantics (to be implemented):**
- **MCP SDK Servers:** Close all active tool handlers, wait for in-flight calls
- **MCP External Servers:** Send close frames, close transport connections
- **Hooks:** No cleanup needed (stateless callbacks)
- **Permissions:** Callback-specific cleanup (if callback manages resources)
- **Channels:** Close message/error channels, drain buffered messages
- **Goroutines:** Cancel contexts, wait for worker pools to exit

---

### Step 6: Error Handling Validation

**Objective:** Verify error conditions are handled gracefully across integrations.

- [ ] **6.1** Verify hook callback errors propagate correctly:
  - Hook returns error instead of result
  - Verify error reaches agent error channel
  - Verify appropriate error sent to CLI
- [ ] **6.2** Verify permission callback errors propagate correctly:
  - Callback returns error
  - Verify tool execution blocked
  - Verify error message reaches CLI
- [ ] **6.3** Verify MCP tool handler errors propagate correctly:
  - Tool handler returns error
  - Verify error reaches agent
  - Verify `PostToolUse` hook still fires (if applicable)
  - Verify error sent to CLI
- [ ] **6.4** Verify panic recovery in integrations:
  - Hook callback panics
  - Verify panic caught and converted to error
  - Verify agent remains stable
- [ ] **6.5** Verify timeout handling:
  - Hook execution exceeds timeout (if timeouts implemented)
  - Verify hook cancelled
  - Verify timeout error propagates correctly

---

### Step 7: Concurrency Validation

**Objective:** Verify thread-safety and concurrent execution handling.

- [ ] **7.1** Verify concurrent tool executions don't interfere:
  - Trigger multiple tool calls in parallel
  - Verify each gets correct hook callbacks
  - Verify each gets correct permission checks
- [ ] **7.2** Verify hook callbacks are goroutine-safe:
  - Concurrent hooks don't corrupt shared state
  - Hook registration is thread-safe
- [ ] **7.3** Verify permission callback is goroutine-safe:
  - Concurrent permission checks don't interfere
  - Callback state (if any) is protected
- [ ] **7.4** Verify MCP server handlers are goroutine-safe:
  - Concurrent tool calls handled correctly
  - Server state properly synchronized

---

## Integration Flow Summary

**Conceptual execution flow when all integrations are active:**

```
User Message
    ↓
Agent receives tool use request
    ↓
[1] Permission callback fires
    ├─ DENY → Send denial to CLI, STOP
    └─ ALLOW → Continue
         ↓
[2] PreToolUse hook fires
    ├─ Continue: false → Send block message to CLI, STOP
    └─ Continue: true → Continue
         ↓
[3] MCP tool handler executes
    ├─ Success → Continue with result
    └─ Error → Continue with error
         ↓
[4] PostToolUse hook fires
    └─ Receives result/error
         ↓
Send result to CLI
```

**Key architectural notes:**
- Permissions checked BEFORE hooks (security boundary)
- PreToolUse can still block after permission allows (flexibility)
- PostToolUse always fires if tool executed (observability)
- Errors at any stage propagate to CLI gracefully

---

## Overall Checklist

### Code Organization
- [ ] All files under 175 lines (linting requirement)
- [ ] Functions under 25 lines where possible
- [ ] Complex logic extracted to helper functions
- [ ] Type switching replaced with handler maps

### Testing
- [ ] Unit tests for all hook types
- [ ] Integration tests for MCP server communication
- [ ] Permission callback authorization tests
- [ ] Cross-feature integration tests (hooks + permissions)

### Documentation
- [ ] Public API fully documented with examples
- [ ] Integration patterns documented
- [ ] Error handling documented
- [ ] Migration guide for users (if applicable)

### Performance
- [ ] Hook execution optimized (minimal overhead)
- [ ] MCP message routing efficient
- [ ] Permission checks don't block unnecessarily

---

## Implementation Order

**Recommended sequence:**
1. **Permissions** (simplest, foundational)
2. **Hooks** (depends on permissions)
3. **MCP** (can leverage hook/permission infrastructure)

This order minimizes rework and allows testing of each integration independently before combining them.
