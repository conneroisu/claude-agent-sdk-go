## Phase 1: Core Domain & Ports

**Objective:** Establish the domain model and port interfaces that form the foundation of the SDK. This phase produces no executable code—only interfaces, data structures, and design decisions.

**Dependencies:** None (this is the foundation layer)

**Success Criteria:**
- All domain types compile without external dependencies
- Port interfaces are dependency-free (accept only stdlib or domain types)
- All public types have godoc comments
- Files respect 175-line limit
- No cyclic dependencies between packages

---

### 1.1 Domain Models (messages/, options/)

**Priority:** Critical
**Estimated Effort:** 3-4 days
**Prerequisites:** None

#### Design Decision: Type Safety vs Flexibility

Before implementing, understand when to use typed structs versus `map[string]any`:

**Use Typed Structs When:**
- Structure is well-defined and stable (e.g., `UsageStats`, hook input types)
- SDK needs to access specific fields (e.g., `ResultMessage.SessionID`)
- Type safety provides clear benefits (discriminated unions)

**Use `map[string]any` When:**
- Data varies by context and cannot be predetermined (e.g., tool inputs)
- SDK only passes data through without inspecting it (e.g., raw stream events)
- Flexibility is more important than compile-time validation

**Examples in this SDK:**
- ✅ Typed: `HookInput` variants, `ResultMessage` variants, `UsageStats`
- ❌ Flexible: `ToolUseBlock.Input` (varies by tool), `StreamEvent.Event` (raw API events)
- 🔄 Hybrid: `SystemMessage.Data` is `map[string]any` but parseable into typed variants

---

#### 1.1.1 Message Types (messages/ package)

**Work Items:**

- [ ] **Create package structure** (messages/):
  - Stub `messages.go` with core `Message` interface
  - Plan file split to respect 175-line limit (see decomposition below)

- [ ] **Define message type interfaces:**
  - `Message` - Root interface with marker method `message()`
  - `SystemMessageData` - Discriminated union for system message variants
  - `ResultMessage` - Discriminated union for result messages
  - `ContentBlock` - Interface for content block variants
  - `MessageContent` - Union of string or block list
  - `ToolResultContent` - Union of string or block list

- [ ] **Implement concrete message types** (split across files):
  - `UserMessage` - User input with optional parent tool use
  - `AssistantMessage` - Model response with content blocks
  - `SystemMessage` - Control messages (init, compact boundary)
  - `ResultMessageSuccess` - Successful query completion
  - `ResultMessageError` - Error during execution
  - `StreamEvent` - Real-time API events

- [ ] **Implement content block types:**
  - `TextBlock` - Plain text content
  - `ThinkingBlock` - Extended thinking with optional signature
  - `ToolUseBlock` - Tool invocation request
  - `ToolResultBlock` - Tool execution result

- [ ] **Define supporting types:**
  - `UsageStats` - Token usage metrics
  - `ModelUsage` - Per-model usage statistics
  - `PermissionDenial` - Denied tool use record
  - `MCPServerStatus` - MCP server connection state

**File Decomposition Strategy (175-line limit):**

Split `messages/` into these files:
- `messages.go` - Core interfaces (~50 lines)
- `user.go` - UserMessage (~40 lines)
- `assistant.go` - AssistantMessage (~60 lines)
- `system.go` - SystemMessage types (~80 lines)
- `result.go` - ResultMessage types (~90 lines)
- `stream.go` - StreamEvent (~30 lines)
- `content.go` - ContentBlock types (~70 lines)
- `usage.go` - Usage statistics (~40 lines)

**Validation Checkpoints:**

1. **Compile Check:** `go build ./pkg/claude/messages`
2. **No External Dependencies:** Verify no imports outside stdlib
3. **Godoc Coverage:** Run `go doc` on all exported types
4. **Linting:** `golangci-lint run --max-lines-per-file=175 ./pkg/claude/messages`

**Example (minimal, for clarity):**

```go
// messages/messages.go
package messages

// Message is the root interface for all SDK messages
type Message interface {
	message() // Marker method for type safety
}

// SystemMessageData discriminates system message variants
type SystemMessageData interface {
	systemMessageData()
}
```

**Open Design Questions:**

1. **Tool result unions:** Should `ToolResultContent` use a struct with a type field or remain as interface?
2. **JSON marshaling:** Do we need custom marshalers for discriminated unions, or rely on reflection?
3. **Validation:** Should message types validate themselves (e.g., `Validate() error`) or delegate to parsers?

---

#### 1.1.2 Options Types (options/ package)

**Work Items:**

- [ ] **Create options package structure:**
  - `domain.go` - Pure domain configuration (~80 lines)
  - `tools.go` - Built-in tool type definitions (~60 lines)
  - `transport.go` - Infrastructure configuration (~90 lines)
  - `mcp.go` - MCP server configuration (~70 lines)

- [ ] **Implement domain configuration (domain.go):**
  - `PermissionMode` type and constants (default, acceptEdits, plan, etc.)
  - `SettingSource` type (user, project, local)
  - `AgentDefinition` struct for subagent configuration
  - `SystemPromptConfig` interface with string and preset variants

- [ ] **Implement tool types (tools.go):**
  - `BuiltinTool` type for type-safe tool names
  - All 18 tool constants (Bash, Read, Write, Grep, etc.)
  - `WithMatcher()` method for pattern-based permissions
  - `String()` method for CLI serialization

- [ ] **Implement transport configuration (transport.go):**
  - `AgentOptions` struct (combines domain + infrastructure settings)
  - Session management fields (resume, fork, continue)
  - Infrastructure fields (cwd, settings, env)
  - Internal flags (e.g., `_isStreaming`)

- [ ] **Implement MCP configuration (mcp.go):**
  - `MCPServerConfig` interface
  - Client configs: `StdioServerConfig`, `SSEServerConfig`, `HTTPServerConfig`
  - SDK config: `SDKServerConfig` (wraps user's in-process server)

**Validation Checkpoints:**

1. **Compile Check:** `go build ./pkg/claude/options`
2. **Dependency Check:** Verify MCP SDK import (`github.com/modelcontextprotocol/go-sdk/mcp`) is only dependency
3. **Type Safety:** Test `BuiltinTool` constants prevent typos at compile time
4. **Linting:** Verify all files under 175 lines

**Example (minimal):**

```go
// options/tools.go
package options

// BuiltinTool provides type safety for Claude built-in tools
type BuiltinTool string

const (
	ToolBash  BuiltinTool = "Bash"
	ToolRead  BuiltinTool = "Read"
	ToolWrite BuiltinTool = "Write"
	// ... (15 more tools)
)

// WithMatcher creates a pattern-based permission rule
func (t BuiltinTool) WithMatcher(pattern string) string {
	return fmt.Sprintf("%s(%s)", t, pattern)
}
```

**Open Design Questions:**

1. **System prompt presets:** Should we validate preset names at compile time or runtime?
2. **Tool matchers:** Should `WithMatcher` return a typed `ToolMatcher` instead of string?
3. **MCP config validation:** Should `MCPServerConfig` types validate themselves or rely on adapters?

---

#### 1.1.3 Control Protocol Types (messages/control.go)

**Work Items:**

- [ ] **Define SDK → CLI control messages:**
  - `ControlRequest` - Wrapper for outbound requests
  - `InterruptRequest` - Stop execution
  - `SetPermissionModeRequest` - Change permission mode
  - `SetModelRequest` - Switch model mid-conversation
  - `InitializeRequest` - Send hook configurations

- [ ] **Define CLI → SDK control messages:**
  - `InboundControlRequest` - Wrapper for inbound requests
  - `CanUseToolRequest` - Permission check callback
  - `HookCallbackRequest` - Hook execution request
  - `MCPMessageRequest` - MCP JSON-RPC routing

- [ ] **Define control responses:**
  - `ControlResponse` - Generic response wrapper
  - `ResponseUnion` - Success/error discriminated union
  - `ControlCancelRequest` - Cancel pending requests

- [ ] **Define permission types:**
  - `PermissionUpdate` - Rule updates
  - `PermissionRuleValue` - Tool-specific rules
  - `PermissionBehavior` - Allow/deny/ask
  - `PermissionResult` - Callback return types (allow/deny variants)

**Validation Checkpoints:**

1. **JSON Compatibility:** Test round-trip encoding/decoding
2. **Request ID Format:** Verify `req_{counter}_{hex}` generation
3. **Discriminated Unions:** Ensure subtype field correctly identifies variants

**Example (minimal):**

```go
// messages/control.go
package messages

// ControlRequest wraps SDK → CLI control requests
type ControlRequest struct {
	Type      string `json:"type"` // Always "control_request"
	RequestID string `json:"request_id"`
	Request   any    `json:"request"` // Discriminated by subtype
}

// PermissionResult discriminates allow/deny decisions
type PermissionResult interface {
	permissionResult()
}
```

**Open Design Questions:**

1. **Request ID generation:** Should this be in the domain or adapter layer?
2. **Timeout handling:** Should control request timeouts (60s) be configurable?
3. **Type assertions:** How do we safely extract subtypes from `any` fields?

---

### 1.2 Ports (Interfaces)

**Priority:** Critical
**Estimated Effort:** 1-2 days
**Prerequisites:** Domain models complete

#### Overview

Ports define what the domain NEEDS from external systems. They are defined BY the domain, not by infrastructure. This inverts the dependency direction—adapters implement ports, not the other way around.

**Key Principle:** Ports accept only stdlib types or domain types. No infrastructure types (e.g., no `*exec.Cmd`, no `net.Conn`).

---

#### 1.2.1 Transport Port (ports/transport.go)

**Work Items:**

- [ ] **Define Transport interface:**
  - `Connect(ctx) error` - Establish connection to CLI process
  - `Write(ctx, data) error` - Send data to CLI stdin
  - `ReadMessages(ctx) (<-chan map[string]any, <-chan error)` - Stream raw messages
  - `EndInput() error` - Signal end of input (EOF)
  - `Close() error` - Clean up resources
  - `IsReady() bool` - Connection health check

**Validation Checkpoints:**

1. **No External Dependencies:** Interface uses only stdlib types
2. **Channel Semantics:** Document ownership and closure guarantees
3. **Context Propagation:** Ensure all blocking operations accept `context.Context`

**Example:**

```go
// ports/transport.go
package ports

import "context"

// Transport abstracts connection to Claude CLI process
type Transport interface {
	Connect(ctx context.Context) error
	Write(ctx context.Context, data string) error
	ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error)
	EndInput() error
	Close() error
	IsReady() bool
}
```

**Open Design Questions:**

1. **Buffering:** Should `ReadMessages` buffer messages internally, or is that the adapter's responsibility?
2. **Error Recovery:** Should `Transport` auto-reconnect, or leave that to the domain service?
3. **Resource Cleanup:** Who closes the channels returned by `ReadMessages`?

---

#### 1.2.2 Protocol Handler Port (ports/protocol.go)

**Work Items:**

- [ ] **Define ProtocolHandler interface:**
  - `Initialize(ctx, config) (map[string]any, error)` - Send initialize control request
  - `SendControlRequest(ctx, req) (map[string]any, error)` - Send control request, wait for response (60s timeout)
  - `HandleControlRequest(ctx, req, deps...) (map[string]any, error)` - Route inbound control requests
  - `StartMessageRouter(ctx, msgCh, errCh, deps...) error` - Partition transport messages

**Validation Checkpoints:**

1. **Timeout Specification:** Document 60-second control request timeout
2. **Dependency Injection:** Verify `HandleControlRequest` accepts permissions/hooks/MCP as arguments (no circular refs)
3. **Concurrency Safety:** Document goroutine ownership and channel semantics

**Example:**

```go
// ports/protocol.go
package ports

import "context"

// ProtocolHandler abstracts control protocol operations
type ProtocolHandler interface {
	Initialize(ctx context.Context, config any) (map[string]any, error)
	SendControlRequest(ctx context.Context, req map[string]any) (map[string]any, error)
	// HandleControlRequest depends on permissions, hooks, MCP servers
	// These are passed as arguments to avoid circular dependencies
	HandleControlRequest(ctx context.Context, req map[string]any,
		perms PermissionsService, hooks map[string]HookCallback,
		mcpServers map[string]MCPServer) (map[string]any, error)
}
```

**Open Design Questions:**

1. **Type Safety:** Should we define typed request/response structs instead of `map[string]any`?
2. **Router Lifecycle:** Should `StartMessageRouter` run in a goroutine, or expect the caller to spawn it?
3. **Cancellation:** How do we cancel pending control requests when context is canceled?

---

#### 1.2.3 Message Parser Port (ports/parser.go)

**Work Items:**

- [ ] **Define MessageParser interface:**
  - `Parse(raw map[string]any) (messages.Message, error)` - Convert raw JSON to typed message

**Validation Checkpoints:**

1. **Error Handling:** Document which errors are returned (e.g., unknown message type, malformed JSON)
2. **Type Discrimination:** Ensure parser can distinguish all message variants

**Example:**

```go
// ports/parser.go
package ports

import "github.com/conneroisu/claude/pkg/claude/messages"

// MessageParser converts raw transport messages to domain types
type MessageParser interface {
	Parse(raw map[string]any) (messages.Message, error)
}
```

**Open Design Questions:**

1. **Streaming Optimization:** Should parser support incremental parsing for large messages?
2. **Validation:** Should parser validate message structure, or trust the CLI output?

---

#### 1.2.4 MCP Server Port (ports/mcp.go)

**Work Items:**

- [ ] **Define MCPServer interface:**
  - `Name() string` - Server identifier for routing
  - `HandleMessage(ctx, message []byte) ([]byte, error)` - Route JSON-RPC messages
  - `Close() error` - Clean up resources

**Note:** This port has TWO implementations:
1. **ClientAdapter:** Routes messages TO external MCP servers
2. **ServerAdapter:** Routes messages TO user's in-process server

**Validation Checkpoints:**

1. **Implementation Agnostic:** Ensure interface doesn't leak client/server distinctions
2. **Error Semantics:** Document which errors indicate retryable vs fatal failures

**Example:**

```go
// ports/mcp.go
package ports

import "context"

// MCPServer abstracts MCP message routing
// Implemented by both client adapters (connect TO servers)
// and server adapters (wrap user's in-process server)
type MCPServer interface {
	Name() string
	HandleMessage(ctx context.Context, message []byte) ([]byte, error)
	Close() error
}
```

**Open Design Questions:**

1. **Bidirectional Messages:** Do we need a callback for server-initiated messages?
2. **Connection State:** Should the port expose connection status, or hide that in adapters?

---

### 1.3 Error Taxonomy

**Priority:** Critical
**Estimated Effort:** 1 day
**Prerequisites:** Domain models complete

#### Overview

Define a structured error hierarchy that supports:
- Programmatic error handling (type assertions, error unwrapping)
- User-friendly error messages
- Debugging context (error chains, stack traces)

---

#### Work Items

- [ ] **Define sentinel errors (errors.go):**
  - `ErrNotConnected` - Transport not connected
  - `ErrCLINotFound` - Claude Code executable not found
  - `ErrCLIConnection` - CLI connection failed
  - `ErrProcessFailed` - CLI process exited with error
  - `ErrJSONDecode` - Invalid JSON from CLI
  - `ErrMessageParse` - Message parsing failed
  - `ErrControlTimeout` - Control request timed out (60s)
  - `ErrInvalidInput` - User provided invalid input

- [ ] **Define structured error types:**
  - `CLINotFoundError` - Includes searched path
  - `ProcessError` - Includes exit code and stderr
  - `JSONDecodeError` - Includes failed line and underlying error

- [ ] **Implement error wrapping:**
  - All structured errors implement `Unwrap() error`
  - Support `errors.Is()` and `errors.As()` for error chains

**Validation Checkpoints:**

1. **Error Wrapping:** Test `errors.Is(err, ErrJSONDecode)` works correctly
2. **Context Preservation:** Verify structured errors include actionable details
3. **User Messages:** Ensure all `Error()` strings are helpful to users

**Example:**

```go
// errors.go
package claude

import (
	"errors"
	"fmt"
)

var (
	ErrNotConnected   = errors.New("claude: not connected")
	ErrCLINotFound    = errors.New("claude: CLI not found")
	ErrControlTimeout = errors.New("claude: control request timeout")
)

// CLINotFoundError provides context about where we searched
type CLINotFoundError struct {
	Path string
}

func (e *CLINotFoundError) Error() string {
	return fmt.Sprintf("Claude Code not found: %s", e.Path)
}

func (e *CLINotFoundError) Unwrap() error {
	return ErrCLINotFound
}

// ProcessError captures CLI execution failures
type ProcessError struct {
	ExitCode int
	Stderr   string
}

func (e *ProcessError) Error() string {
	return fmt.Sprintf("process failed with exit code %d: %s", e.ExitCode, e.Stderr)
}

func (e *ProcessError) Unwrap() error {
	return ErrProcessFailed
}
```

**Error Handling Guidelines:**

1. **Wrap, Don't Replace:** Use `fmt.Errorf("context: %w", err)` to preserve error chains
2. **Sentinel for Types:** Use sentinel errors for programmatic checks
3. **Structured for Context:** Use structured errors when additional data is needed
4. **User-Facing Messages:** All `Error()` methods should be readable by end users

**Open Design Questions:**

1. **Stack Traces:** Should we integrate with `pkg/errors` or similar for stack capture?
2. **Error Codes:** Should errors have numeric codes for API consumers?
3. **Localization:** Do error messages need i18n support?

---

### 1.4 Implementation Sequencing

**Recommended Order:**

1. **Day 1-2:** Message types (messages/ package)
   - Start with core interfaces
   - Implement concrete types
   - Split files to respect 175-line limit
   - **Checkpoint:** `go build ./pkg/claude/messages` succeeds

2. **Day 2-3:** Options types (options/ package)
   - Domain configuration first
   - Tool types second
   - Transport and MCP configs last
   - **Checkpoint:** `go build ./pkg/claude/options` succeeds

3. **Day 3:** Control protocol types (messages/control.go)
   - SDK → CLI requests
   - CLI → SDK requests
   - Permission types
   - **Checkpoint:** JSON round-trip tests pass

4. **Day 4:** Port interfaces (ports/ package)
   - Transport port
   - Protocol handler port
   - Parser and MCP ports
   - **Checkpoint:** All interfaces compile, no circular dependencies

5. **Day 4:** Error taxonomy (errors.go)
   - Sentinel errors
   - Structured error types
   - **Checkpoint:** Error wrapping tests pass

**Dependencies for Next Phase:**

- ✅ All domain types compile
- ✅ Port interfaces defined
- ✅ No external dependencies except MCP SDK
- ✅ All files under 175 lines
- ✅ Godoc coverage >15%

---

### 1.5 Testing Strategy

**Unit Tests (phase 1):**

- **Message Parsing:** Table-driven tests for JSON deserialization
  - Valid messages of each type
  - Invalid/malformed messages
  - Edge cases (nil fields, empty arrays)

- **Type Discrimination:** Test discriminated unions
  - `SystemMessageData` variants
  - `ResultMessage` variants
  - `PermissionResult` variants

- **Tool Types:** Test `BuiltinTool` API
  - `WithMatcher()` formatting
  - String conversion
  - Type safety (compile-time check, not runtime)

- **Error Wrapping:** Test error chains
  - `errors.Is()` works for all sentinel errors
  - `errors.As()` extracts structured error types
  - Context is preserved through wrapping

**Example Test Structure:**

```go
// messages/messages_test.go
func TestMessageParsing(t *testing.T) {
	tests := []struct{
		name string
		json string
		want messages.Message
		wantErr bool
	}{
		{
			name: "UserMessage",
			json: `{"content":"test","parent_tool_use_id":null}`,
			want: &messages.UserMessage{Content: messages.StringContent("test")},
		},
		// ... more cases
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test logic
		})
	}
}
```

**No Integration Tests Yet:** Phase 1 has no executable components to integrate.

---

## Phase 1 Completion Checklist

- [ ] All files compile without errors
- [ ] All files under 175 lines (excluding comments/blanks)
- [ ] All exported types have godoc comments
- [ ] Godoc coverage ≥15% per file
- [ ] No cyclic dependencies between packages
- [ ] No external dependencies except MCP SDK (in options/mcp.go only)
- [ ] Unit tests cover:
  - [ ] Message type parsing (all variants)
  - [ ] Discriminated union behavior
  - [ ] Tool type API
  - [ ] Error wrapping and unwrapping
- [ ] golangci-lint passes with project configuration
- [ ] All open design questions documented for team review

**Deliverables:**

- `pkg/claude/messages/` - 8 files defining all message types
- `pkg/claude/options/` - 4 files defining configuration types
- `pkg/claude/ports/` - 4 files defining port interfaces
- `pkg/claude/errors.go` - Error taxonomy
- Tests covering all domain types and error handling

**Next Phase Dependencies:**

Phase 2 (Domain Services) can begin once:
- All port interfaces are defined
- Message types are complete and tested
- Error taxonomy is established

---

## Appendix: Reference Code Examples

This appendix provides minimal code snippets for clarity. **Do not implement these verbatim**—they serve as reference only. Follow the work items and validation checkpoints above.

### Built-in Tools Reference (18 tools)

```go
// options/tools.go
const (
	ToolBash, ToolBashOutput, ToolKillShell  // Execution
	ToolRead, ToolWrite, ToolEdit, ToolGlob, ToolGrep  // File operations
	ToolTask, ToolExitPlanMode  // Agent tools
	ToolWebFetch, ToolWebSearch  // Web tools
	ToolListMcpResources, ToolReadMcpResource, ToolMcp  // MCP tools
	ToolNotebookEdit, ToolTodoWrite, ToolSlashCommand  // Other tools
)
```

### Helper Function Note

The `[]BuiltinTool` to string conversion helper will be defined in Phase 4 (`helpers/tools.go`):

```go
// Phase 4 deliverable - DO NOT implement in Phase 1
func ToolsToString(tools []options.BuiltinTool) string {
	strs := make([]string, len(tools))
	for i, t := range tools {
		strs[i] = string(t)
	}
	return strings.Join(strs, ",")
}
```

---

## Linting Compliance Notes

### File Size Requirements (175 line limit)

**messages/ package decomposition:**
- `messages.go` - Core interfaces (~50 lines)
- `user.go` - UserMessage (~40 lines)
- `assistant.go` - AssistantMessage (~60 lines)
- `system.go` - SystemMessage types (~80 lines)
- `result.go` - ResultMessage types (~90 lines)
- `stream.go` - StreamEvent (~30 lines)
- `content.go` - ContentBlock types (~70 lines)
- `usage.go` - Usage statistics (~40 lines)
- `control.go` - Control protocol types (~150 lines)

**options/ package:**
- `domain.go` (~80 lines)
- `tools.go` (~60 lines)
- `transport.go` (~90 lines)
- `mcp.go` (~70 lines)

**ports/ package:**
- `transport.go` (~50 lines)
- `protocol.go` (~60 lines)
- `parser.go` (~25 lines)
- `mcp.go` (~50 lines)

### Complexity Mitigation

- Message type parsing → Extract per-type parsers
- Content block switching → Use type-specific functions
- Validation logic → Extract to dedicated functions

### Compliance Checklist

- [ ] All files under 175 lines (excl. comments/blanks)
- [ ] All functions under 25 lines
- [ ] Max 4 parameters per function
- [ ] Max 3 return values per function
- [ ] 15% minimum comment density per file
- [ ] All exported items have godoc comments
