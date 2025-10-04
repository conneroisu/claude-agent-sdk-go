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

type HookJSONOutput struct {
	Decision           *string        `json:"decision,omitempty"`           // "block"
	SystemMessage      *string        `json:"systemMessage,omitempty"`
	HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"`
}

// BlockBashPatternHook returns a hook callback that blocks bash commands containing forbidden patterns
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

## Hook Callback Implementation

### Control Protocol Integration

Hooks are registered during the `initialize` control request and invoked via `hook_callback` control requests from the CLI.

#### Registration Flow (During Initialization)

When the SDK initializes, it registers hook callback IDs with the CLI:

```go
// 1. User provides hooks via QueryOptions
opts := &QueryOptions{
    Hooks: map[HookEvent]HookCallback{
        HookEventPreToolUse:  userPreToolUseCallback,
        HookEventPostToolUse: userPostToolUseCallback,
    },
}

// 2. SDK generates callback IDs and stores mapping
hookCallbackCounter := 0
hookCallbacks := make(map[string]HookCallback)  // callback_id → user function
hookCallbacksMap := make(map[string]string)     // callback_id → hook_name

for hookEvent, callback := range opts.Hooks {
    hookCallbackCounter++
    callbackID := fmt.Sprintf("hook_%d", hookCallbackCounter)

    // Store user callback for later invocation
    hookCallbacks[callbackID] = callback

    // Map callback ID to hook name for CLI
    hookCallbacksMap[callbackID] = string(hookEvent)
}

// 3. Send initialize request with callback IDs
initRequest := &InitializeRequest{
    Subtype:       "initialize",
    Version:       "1.0.0",
    HookCallbacks: hookCallbacksMap,  // {"hook_1": "PreToolUse", "hook_2": "PostToolUse"}
    // ... other fields
}
```

**Key Details:**
- Hook callback IDs follow format: `hook_{counter}` (e.g., `hook_1`, `hook_2`)
- Counter increments for each hook registered
- SDK maintains internal map of `callbackID → userCallback` for invocation
- CLI receives map of `callbackID → hookName` to know when to call

#### Invocation Flow (During Execution)

When the CLI needs to execute a hook, it sends a `hook_callback` control request:

```go
// 1. CLI sends hook_callback request
type HookCallbackRequest struct {
    Subtype    string         `json:"subtype"`     // "hook_callback"
    CallbackID string         `json:"callback_id"` // "hook_1"
    Input      map[string]any `json:"input"`       // Hook-specific input
    ToolUseID  *string        `json:"tool_use_id,omitempty"`
}

// Example incoming request:
{
    "type": "control_request",
    "request_id": "req_5_a3f2",
    "request": {
        "subtype": "hook_callback",
        "callback_id": "hook_1",
        "input": {
            "tool_name": "Bash",
            "tool_input": {"command": "rm -rf /"}
        }
    }
}

// 2. SDK looks up user callback by ID
callback, exists := hookCallbacks[request.CallbackID]
if !exists {
    return errors.New("unknown callback ID")
}

// 3. Build HookContext
ctx := HookContext{
    ConversationID: currentConversationID,
    // ... other context fields
}

// 4. Invoke user's callback function
output, err := callback(request.Input, request.ToolUseID, ctx)
if err != nil {
    // Return error response
    return &ControlResponse{
        Type:      "control_response",
        RequestID: request.RequestID,
        Error: &ControlError{
            Code:    "hook_execution_error",
            Message: err.Error(),
        },
    }
}

// 5. Send success response
return &ControlResponse{
    Type:      "control_response",
    RequestID: request.RequestID,
    Result:    output,  // Hook output (decision, systemMessage, etc.)
}
```

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

Based on the Python SDK implementation, hooks return structured output:

```go
type HookOutput struct {
    Decision           *string        `json:"decision,omitempty"`           // "block", "allow"
    SystemMessage      *string        `json:"systemMessage,omitempty"`      // Message to show user
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

// Example PostToolUse hook output (modifying result):
{
    "hookSpecificOutput": {
        "hookEventName":  "PostToolUse",
        "modifiedResult": "...",
    }
}
```

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
