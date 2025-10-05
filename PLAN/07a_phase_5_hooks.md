## Phase 5a: Hooks Support

### Priority: Critical

### Overview
The facade re-exports domain hook types from the `hooking` package for public API convenience. This allows users to register callbacks that execute at specific points in the agent lifecycle.

### Public API (hooks.go)

```go
package claude

import (
	"fmt"
	"strings"

	"github.com/conneroisu/claude/pkg/claude/hooking"
)

// Re-export domain hook types for public API
type HookEvent = hooking.HookEvent
type HookContext = hooking.HookContext
type HookCallback = hooking.HookCallback
type HookMatcher = hooking.HookMatcher

// Re-export hook event constants
const (
	// HookEventPreToolUse is the event for pre-tool use hooks.
	HookEventPreToolUse       = hooking.HookEventPreToolUse
	// HookEventPostToolUse is the event for post-tool use hooks.
	HookEventPostToolUse      = hooking.HookEventPostToolUse
	// HookEventUserPromptSubmit is the event for user prompt submit hooks.
	HookEventUserPromptSubmit = hooking.HookEventUserPromptSubmit
	// HookEventStop is the event for stop hooks.
	HookEventStop             = hooking.HookEventStop
	// HookEventSubagentStop is the event for subagent stop hooks.
	HookEventSubagentStop     = hooking.HookEventSubagentStop
	// HookEventPreCompact is the event for pre-compact hooks.
	HookEventPreCompact       = hooking.HookEventPreCompact
)

// HookJSONOutput represents the output from a hook callback.
// This aligns with the Python SDK HookJSONOutput contract:
// - decision: "block" to prevent the action
// - systemMessage: Optional message saved in transcript (not visible to Claude)
// - hookSpecificOutput: Hook-specific data (see individual hook documentation)
//
// Reference: claude-agent-sdk-python/src/claude_agent_sdk/types.py:118
type HookJSONOutput struct {
	Decision           *string        `json:"decision,omitempty"`           // "block" to prevent action
	SystemMessage      *string        `json:"systemMessage,omitempty"`      // Message saved in transcript
	HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"` // Hook-specific data
}

// Note: The following helper is provided as an example but depends on types
// not yet defined in earlier phases. Consider this a reference implementation
// for future use once all domain types are established.
//
// BlockBashPatternHook returns a hook callback that blocks bash commands containing forbidden patterns
// This example demonstrates the HookJSONOutput structure and hook callback pattern.
func BlockBashPatternHook(patterns []string) HookCallback {
	return func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
		toolName, _ := input["tool_name"].(string)
		if toolName != "Bash" {
			return map[string]any{}, nil
		}
		toolInput, _ := input["tool_input"].(map[string]any)
		command, _ := toolInput["command"].(string)
		for _, pattern := range patterns {
			if strings.Contains(command, pattern) {
				return map[string]any{
					"hookSpecificOutput": map[string]any{
						"hookEventName":            "PreToolUse",
						"permissionDecision":       "deny",
						"permissionDecisionReason": fmt.Sprintf("Command contains forbidden pattern: %s", pattern),
					},
				}, nil
			}
		}
		return map[string]any{}, nil
	}
}
```

---

## Hook Lifecycle: Sequencing and Critical Edge Cases

### Execution Sequence

**Phase 1: Registration (SDK Initialization)**
1. User provides hooks map → `Query()` or `Client.Connect()`
2. Hook service generates sequential callback IDs: `hook_1`, `hook_2`, etc.
3. Internal storage: `callbackID → user function` (for execution)
4. CLI mapping: `callbackID → hook event name` (sent in initialize request)
5. Initialize request transmitted with `HookCallbacks` field

**Phase 2: CLI Triggers Hook (Runtime)**
1. CLI detects hook event (e.g., before tool use)
2. Matches event to registered callback ID via mapping
3. Sends `hook_callback` control request with callback ID + input data

**Phase 3: SDK Executes Callback**
1. Protocol handler receives `hook_callback` request
2. Extracts `callback_id`, looks up user function
3. Builds `HookContext` from current conversation state
4. Invokes user callback with timeout protection (goroutine + select)
5. Returns hook output or error to CLI

**Phase 4: CLI Acts on Response**
- If hook returns `decision: "block"` → Tool execution prevented
- If hook returns modified data → CLI uses modified values
- If hook errors → CLI may retry or abort based on policy

### Critical Edge Cases

**1. Hook Registration Race Conditions**
- **Problem:** Multiple goroutines calling `RegisterHooks` concurrently
- **Mitigation:** Mutex protection around callback counter and storage
- **Implementation:** `sync.RWMutex` in `Service` struct, lock during registration

**2. Callback Timeout Protection**
- **Problem:** User callback blocks indefinitely
- **Default Limit:** 30 seconds (configurable via context)
- **User Override:** Pass custom timeout via parent context
- **Mitigation:** Execute in goroutine with context timeout
- **Implementation:**
  ```go
  // Default timeout is 30 seconds, but users can override by passing a context
  // with a custom deadline/timeout when calling Query() or Client methods.
  // Example user override:
  //   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
  //   defer cancel()
  //   client.Query(ctx, "prompt", options)

  // Internal implementation applies default if no deadline set:
  hookCtx := ctx
  if _, hasDeadline := ctx.Deadline(); !hasDeadline {
      var cancel context.CancelFunc
      hookCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
      defer cancel()
  }

  select {
  case result := <-resultCh:
      return result.output, result.err
  case <-hookCtx.Done():
      return nil, fmt.Errorf("hook execution timeout: %w", hookCtx.Err())
  }
  ```

**3. Unknown Callback ID**
- **Problem:** CLI sends callback ID not in registry (shouldn't happen, but defensive)
- **Mitigation:** Return structured error response, don't panic
- **Response:** `ControlError{Code: "unknown_callback", Message: "..."}`

**4. Hook Panic Recovery**
- **Problem:** User callback panics
- **Recovery Behavior:**
  - Panic is caught via defer/recover in execution goroutine
  - Converted to a structured error returned to CLI
  - Stack trace is logged for debugging (if logger available)
  - Hook is marked as failed, but execution continues for other hooks
- **Mitigation:** Recover in execution goroutine, convert to error
- **Implementation:**
  ```go
  defer func() {
      if r := recover(); r != nil {
          // Log stack trace for debugging
          stack := debug.Stack()
          if logger != nil {
              logger.Error("hook panicked", "panic", r, "stack", string(stack))
          }
          // Return structured error to caller
          resultCh <- hookResult{
              err: fmt.Errorf("hook panicked: %v (see logs for stack trace)", r),
          }
      }
  }()
  ```

**5. Nil Hook Context Fields**
- **Problem:** Context fields may be nil during initialization or edge states
- **Mitigation:** Explicitly document which fields are optional, provide zero values
- **Convention:** Use pointers for optional fields, document in `HookContext` godoc

**6. Hook Output Validation**
- **Problem:** Hook returns invalid output structure
- **Mitigation:** Validate output schema before sending to CLI
- **Implementation:** Check for required fields based on hook type, return error if invalid

### Hook Execution Architecture

```
┌───────────────────────────────────────────────────────────┐
│ User Application                                          │
│ ┌───────────────────────────────────────────────────────┐ │
│ │ User Provides Hooks:                                  │ │
│ │   PreToolUse  → func(input, toolUseID, ctx) output   │ │
│ │   PostToolUse → func(input, toolUseID, ctx) output   │ │
│ └───────────────────────────────────────────────────────┘ │
└──────────────────────┬────────────────────────────────────┘
                       │ Initialize
                       ▼
        ┌──────────────────────────────────┐
        │ SDK Initialization                │
        │ • Generate IDs: hook_1, hook_2   │
        │ • Store: callbackID → userFunc   │
        │ • Send to CLI: callbackID → name│
        └──────────────┬───────────────────┘
                       │
                       ▼
        ┌──────────────────────────────────┐
        │ Claude CLI                        │
        │ • Knows: hook_1 = PreToolUse     │
        │ • Triggers hook before tool use  │
        └──────────────┬───────────────────┘
                       │ hook_callback request
                       ▼
        ┌──────────────────────────────────┐
        │ Control Protocol Handler          │
        │ • Receives: hook_callback request │
        │ • Extracts: callback_id = hook_1  │
        └──────────────┬───────────────────┘
                       │
                       ▼
        ┌──────────────────────────────────┐
        │ Hook Execution Service            │
        │ • Lookup: hook_1 → userFunc      │
        │ • Invoke: userFunc(input, ctx)   │
        │ • Return: output to CLI          │
        └──────────────────────────────────┘
```

### Implementation Requirements

**1. Callback ID Generation (hooking/service.go)**

```go
type Service struct {
    callbackCounter int
    callbacks       map[string]HookCallback  // callback_id → user function
    mu              sync.RWMutex
}

func (s *Service) RegisterHooks(hooks map[HookEvent]HookCallback) map[string]string {
    s.mu.Lock()
    defer s.mu.Unlock()

    callbackMap := make(map[string]string)  // callback_id → hook_name

    for event, callback := range hooks {
        s.callbackCounter++
        callbackID := fmt.Sprintf("hook_%d", s.callbackCounter)

        s.callbacks[callbackID] = callback
        callbackMap[callbackID] = string(event)
    }

    return callbackMap
}
```

**2. Callback Invocation (hooking/execute.go)**

```go
func (s *Service) ExecuteCallback(
    ctx context.Context,
    callbackID string,
    input map[string]any,
    toolUseID *string,
    hookCtx HookContext,
) (map[string]any, error) {
    s.mu.RLock()
    callback, exists := s.callbacks[callbackID]
    s.mu.RUnlock()

    if !exists {
        return nil, fmt.Errorf("unknown callback ID: %s", callbackID)
    }

    // Invoke user's callback with timeout protection
    resultCh := make(chan hookResult, 1)

    go func() {
        output, err := callback(input, toolUseID, hookCtx)
        resultCh <- hookResult{output: output, err: err}
    }()

    select {
    case result := <-resultCh:
        return result.output, result.err
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

type hookResult struct {
    output map[string]any
    err    error
}
```

**3. Control Protocol Integration (adapters/jsonrpc/protocol.go)**

The protocol handler must route `hook_callback` requests to the hook service:

```go
func (p *ProtocolHandler) handleInboundRequest(req *ControlRequest) (*ControlResponse, error) {
    var request struct {
        Subtype    string         `json:"subtype"`
        CallbackID string         `json:"callback_id,omitempty"`
        Input      map[string]any `json:"input,omitempty"`
        ToolUseID  *string        `json:"tool_use_id,omitempty"`
    }

    if err := json.Unmarshal(req.Request, &request); err != nil {
        return nil, err
    }

    switch request.Subtype {
    case "hook_callback":
        // Route to hook service
        output, err := p.hookService.ExecuteCallback(
            ctx,
            request.CallbackID,
            request.Input,
            request.ToolUseID,
            buildHookContext(),
        )

        if err != nil {
            return &ControlResponse{
                Type:      "control_response",
                RequestID: req.RequestID,
                Error: &ControlError{
                    Code:    "hook_execution_error",
                    Message: err.Error(),
                },
            }, nil
        }

        return &ControlResponse{
            Type:      "control_response",
            RequestID: req.RequestID,
            Result:    output,
        }, nil

    // ... other cases
    }
}
```

### Hook-Specific Output Format

Based on the Python SDK implementation (claude-agent-sdk-python/src/claude_agent_sdk/types.py:118),
hooks return structured output matching the HookJSONOutput contract:

```go
// HookJSONOutput aligns with Python SDK contract:
// class HookJSONOutput(TypedDict):
//     decision: NotRequired[Literal["block"]]
//     systemMessage: NotRequired[str]
//     hookSpecificOutput: NotRequired[Any]
type HookJSONOutput struct {
    Decision           *string        `json:"decision,omitempty"`           // "block" to prevent action
    SystemMessage      *string        `json:"systemMessage,omitempty"`      // Transcript message (not shown to Claude)
    HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"` // Hook-specific data
}

// Example PreToolUse hook output (blocking a command):
{
    "hookSpecificOutput": {
        "hookEventName":            "PreToolUse",
        "permissionDecision":       "deny",
        "permissionDecisionReason": "Command contains forbidden pattern: rm -rf"
    }
}

// Example PostToolUse hook output (adding context):
{
    "hookSpecificOutput": {
        "hookEventName":      "PostToolUse",
        "additionalContext":  "Tool execution completed successfully",
    }
}

// Example UserPromptSubmit hook output (adding custom instructions):
{
    "hookSpecificOutput": {
        "hookEventName":      "UserPromptSubmit",
        "additionalContext":  "My favorite color is hot pink",
    }
}
```

### Hook Timeout and Streaming Cancellation Interaction

**Context Cancellation Hierarchy:**
1. **Stream Cancellation**: User cancels the Query/stream → top-level context cancelled
2. **Hook Timeout**: Individual hook exceeds timeout → hook context cancelled
3. **Propagation**: Stream cancellation always propagates to active hooks

**Behavior Matrix:**

| Scenario | Hook Timeout | Stream Cancelled | Result |
|----------|--------------|------------------|--------|
| Hook completes normally | No | No | Hook output returned to CLI |
| Hook exceeds timeout | Yes | No | Hook fails with timeout error, CLI receives error |
| Stream cancelled during hook | No | Yes | Hook context cancelled immediately, cleanup runs |
| Both timeout and cancel | Yes | Yes | Whichever fires first (usually cancellation) |

**Implementation Notes:**
```go
// Hook execution respects both timeouts and cancellation
func (s *Service) ExecuteCallback(
    ctx context.Context,  // Inherits stream cancellation
    callbackID string,
    input map[string]any,
    toolUseID *string,
    hookCtx HookContext,
) (map[string]any, error) {
    // Apply hook-specific timeout (default 30s) if no deadline set
    hookCtx := ctx
    if _, hasDeadline := ctx.Deadline(); !hasDeadline {
        var cancel context.CancelFunc
        hookCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
        defer cancel()
    }

    // Execute hook in goroutine
    resultCh := make(chan hookResult, 1)
    go func() {
        defer func() {
            if r := recover(); r != nil {
                resultCh <- hookResult{err: fmt.Errorf("hook panicked: %v", r)}
            }
        }()
        output, err := callback(input, toolUseID, hookCtx)
        resultCh <- hookResult{output: output, err: err}
    }()

    // Wait for completion, timeout, or stream cancellation
    select {
    case result := <-resultCh:
        return result.output, result.err
    case <-hookCtx.Done():
        // Could be timeout or parent cancellation
        return nil, fmt.Errorf("hook execution cancelled: %w", hookCtx.Err())
    }
}
```

**User-Facing Behavior:**
- When a stream is cancelled (e.g., via `AbortController` in JS, context cancellation in Go), any in-flight hooks are immediately cancelled
- Hook callbacks should check `ctx.Done()` if performing long operations
- Timeout errors are distinguishable from cancellation errors via `errors.Is(err, context.DeadlineExceeded)` vs `errors.Is(err, context.Canceled)`

---

## Implementation Notes

### File Size Requirements (175 line limit)

**hooking/ package must be split into 4 files:**
- `service.go` - Service struct + constructor (~50 lines)
- `execute.go` - Execution logic (~80 lines)
- `registry.go` - Hook registry management (~60 lines)
- `types.go` - Hook input/output types (~60 lines)

❌ Do NOT create a single `service.go` (would be 250+ lines)

### Complexity Hotspots

**Hook execution logic requires extraction:**
- Type switching for hook inputs → Extract per-type handlers
- Callback invocation → Extract wrapper function
- Error handling → Extract error wrapper

**Recommended patterns:**
- Use handler map/registry instead of large type switches
- Create callback wrapper functions to reduce boilerplate
- Extract error formatting into dedicated functions

### Checklist

- [ ] Hook execution functions under 25 lines each
- [ ] Type switching extracted to handler map/registry
- [ ] Callback handling simplified with wrappers
- [ ] All hooking/ files under 175 lines

---

## Related Files
- [Phase 5b: MCP Server Support](./07b_phase_5_mcp_servers.md)
- [Phase 5c: Permission Callbacks](./07c_phase_5_permissions.md)
- [Phase 5: Integration Summary](./07d_phase_5_integration_summary.md)
