## Phase 5c: Permission Callbacks

### Priority: Medium

### Overview
Permission callbacks allow users to implement custom authorization logic for tool usage. These callbacks receive context about the tool being used and can allow, deny, or modify the request.

**Hexagonal Architecture Alignment:**
- Permissions are a distinct domain boundary (authorization/access control)
- Located in `pkg/claude/permissions/` package as a domain service (consistent with Phase 2)
- Public API surface matches Phase 2's `permissions.Service` definition
- Core agent/tool logic depends on the permissions service through dependency injection

### Package Structure

The permissions service is defined in Phase 2 as `permissions.Service`. This phase provides implementation guidance and usage examples. The service is located at:

```
pkg/claude/permissions/
  └── service.go     # Service type with CheckToolUse, UpdateMode methods (defined in Phase 2)
```

---

### Implementation Reference

The `permissions.Service` type is fully defined in **Phase 2: Domain Services** (section 2.4). Key components include:

- `PermissionResult` interface with `PermissionResultAllow` and `PermissionResultDeny` implementations
- `PermissionUpdate` struct for permission modifications
- `ToolPermissionContext` for passing suggestion data
- `CanUseToolFunc` callback signature
- `PermissionsConfig` for service configuration
- `Service` struct with `CheckToolUse()` and `UpdateMode()` methods

Refer to Phase 2 for the complete implementation.

---

## Permission Request/Response Flow

### Control Protocol Integration

Permissions are requested by the CLI via `can_use_tool` control requests. The SDK evaluates the request and responds with either `PermissionResultAllow` or `PermissionResultDeny`.

#### Request Flow (CLI → SDK)

When Claude wants to use a tool, the CLI sends a permission request:

```go
// 1. CLI sends can_use_tool request
type CanUseToolRequest struct {
    Subtype              string             `json:"subtype"`     // "can_use_tool"
    ToolName             string             `json:"tool_name"`
    Input                map[string]any     `json:"input"`
    PermissionSuggestions []PermissionUpdate `json:"permission_suggestions,omitempty"`
    BlockedPath          *string            `json:"blocked_path,omitempty"`
}

// Example incoming request:
{
    "type": "control_request",
    "request_id": "req_3_b8c9",
    "request": {
        "subtype": "can_use_tool",
        "tool_name": "Bash",
        "input": {
            "command": "npm test",
            "description": "Run tests"
        },
        "permission_suggestions": [
            {
                "rule": {
                    "tool_pattern": "Bash",
                    "input_pattern": {"command": "npm *"}
                },
                "value": "always_allow"
            }
        ]
    }
}

// 2. Protocol handler routes to permissions service
result, err := p.permissionsService.CheckToolUse(
    ctx,
    request.ToolName,
    request.Input,
    &ToolPermissionContext{
        Suggestions: request.PermissionSuggestions,
        BlockedPath: request.BlockedPath,
    },
)

// 3. Permissions service evaluates based on mode and rules
// - If mode is "allow", auto-approve
// - If mode is "deny", auto-deny
// - If mode is "ask", invoke user callback
// - Check existing permission rules
// - Apply permission suggestions if auto-allowing

// 4. Build response based on result
if allowResult, ok := result.(*PermissionResultAllow); ok {
    response := &ControlResponse{
        Type:      "control_response",
        RequestID: request.RequestID,
        Result: map[string]any{
            "type":   "allow",
            "input":  allowResult.Input,  // May be modified by permissions
            "update": allowResult.Update, // New permission rules to apply
        },
    }
}

if denyResult, ok := result.(*PermissionResultDeny); ok {
    response := &ControlResponse{
        Type:      "control_response",
        RequestID: request.RequestID,
        Result: map[string]any{
            "type":   "deny",
            "reason": denyResult.Reason,
        },
    }
}
```

**Key Details:**
- CLI sends tool name and input for permission check
- May include suggestions for "always allow" rules
- SDK evaluates based on permission mode, rules, and user callback
- Response indicates allow/deny and may include modified input or new rules

#### Permission Modes

The SDK supports three permission modes (can be changed via `set_permission_mode` control request):

**1. Ask Mode (Default)**
- Invoke user's `CanUseToolFunc` callback for each tool use
- User callback returns allow/deny decision
- Most flexible but requires user interaction

**2. Allow Mode**
- Auto-approve all tool uses
- Still applies permission suggestions from CLI
- Useful for automated workflows

**3. Deny Mode**
- Auto-deny all tool uses
- Useful for read-only or restricted environments

```go
// Permission mode can be changed at runtime
type SetPermissionModeRequest struct {
    Subtype string         `json:"subtype"` // "set_permission_mode"
    Mode    PermissionMode `json:"mode"`    // "ask", "allow", "deny"
}

// SDK updates internal mode
permissionsService.UpdateMode(request.Mode)
```

#### Permission Suggestions

The CLI can suggest permission rules to automatically allow certain patterns:

```go
type PermissionUpdate struct {
    Rule  PermissionRule      `json:"rule"`
    Value PermissionRuleValue `json:"value"` // "always_allow", "always_deny"
}

type PermissionRule struct {
    ToolPattern  string         `json:"tool_pattern"`   // "Bash", "Read", "*"
    InputPattern map[string]any `json:"input_pattern"`  // Pattern to match inputs
}

// Example: Always allow "npm test" commands
{
    "rule": {
        "tool_pattern": "Bash",
        "input_pattern": {
            "command": "npm test"
        }
    },
    "value": "always_allow"
}

// Example: Always allow reading files in /tmp
{
    "rule": {
        "tool_pattern": "Read",
        "input_pattern": {
            "file_path": "/tmp/*"
        }
    },
    "value": "always_allow"
}
```

**Suggestion Application:**

When permission mode is "allow", the SDK automatically applies suggestions:

```go
func (s *Service) CheckToolUse(
    ctx context.Context,
    toolName string,
    input map[string]any,
    permCtx *ToolPermissionContext,
) (PermissionResult, error) {
    // Check current mode
    if s.mode == PermissionModeAllow {
        // Auto-allow, but apply suggestions
        var updates []PermissionUpdate
        if permCtx != nil && len(permCtx.Suggestions) > 0 {
            updates = permCtx.Suggestions
            // Store suggestions as permanent rules
            s.applyPermissionUpdates(updates)
        }

        return &PermissionResultAllow{
            Input:  input,  // Unmodified
            Update: updates,
        }, nil
    }

    // ... other modes
}
```

#### Permission Result Types

Based on Phase 2 definitions:

```go
// Allow result
type PermissionResultAllow struct {
    Input  map[string]any     // Tool input (may be modified)
    Update []PermissionUpdate // Permission rules to apply
}

func (p *PermissionResultAllow) IsAllowed() bool { return true }

// Deny result
type PermissionResultDeny struct {
    Reason string // Explanation for denial
}

func (p *PermissionResultDeny) IsAllowed() bool { return false }

// Example usage in user callback:
func myPermissionCallback(
    toolName string,
    input map[string]any,
    ctx ToolPermissionContext,
) (PermissionResult, error) {
    if toolName == "Bash" {
        command := input["command"].(string)
        if strings.Contains(command, "rm -rf /") {
            return &PermissionResultDeny{
                Reason: "Dangerous command blocked",
            }, nil
        }
    }

    return &PermissionResultAllow{
        Input: input,  // Allow with original input
    }, nil
}
```

#### Complete Request/Response Example

**Scenario:** CLI asks to run `npm test`, suggests always allowing npm commands

```go
// 1. CLI → SDK: can_use_tool request
{
    "type": "control_request",
    "request_id": "req_7_f3a1",
    "request": {
        "subtype": "can_use_tool",
        "tool_name": "Bash",
        "input": {
            "command": "npm test",
            "description": "Run tests"
        },
        "permission_suggestions": [
            {
                "rule": {
                    "tool_pattern": "Bash",
                    "input_pattern": {"command": "npm *"}
                },
                "value": "always_allow"
            }
        ]
    }
}

// 2. SDK evaluates permission (assume mode = "ask")
// - Checks existing rules (no match)
// - Invokes user callback
// - User callback returns allow

// 3. SDK → CLI: control_response with allow
{
    "type": "control_response",
    "request_id": "req_7_f3a1",
    "result": {
        "type": "allow",
        "input": {
            "command": "npm test",
            "description": "Run tests"
        },
        "update": [
            {
                "rule": {
                    "tool_pattern": "Bash",
                    "input_pattern": {"command": "npm *"}
                },
                "value": "always_allow"
            }
        ]
    }
}

// 4. CLI executes tool with allowed input
// 5. Future npm commands auto-approved due to stored rule
```

### Permission Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Claude CLI                                                   │
│ • Needs to use tool: Bash                                   │
│ • Generates permission suggestions                          │
└──────────────────────┬──────────────────────────────────────┘
                       │ can_use_tool request
                       ▼
        ┌──────────────────────────────────┐
        │ Control Protocol Handler          │
        │ • Receives can_use_tool request  │
        │ • Extracts tool name, input      │
        └──────────────┬───────────────────┘
                       │
                       ▼
        ┌──────────────────────────────────┐
        │ Permissions Service               │
        │ • Check permission mode          │
        │ • Evaluate existing rules        │
        │ • Invoke user callback (if ask)  │
        └──────────────┬───────────────────┘
                       │
                       ▼
        ┌──────────────────────────────────┐
        │ User Callback (Optional)          │
        │ • Custom authorization logic     │
        │ • Return allow/deny decision     │
        └──────────────┬───────────────────┘
                       │
                       ▼
        ┌──────────────────────────────────┐
        │ PermissionResult                  │
        │ • Allow: input + updates         │
        │ • Deny: reason                   │
        └──────────────┬───────────────────┘
                       │
                       ▼
        ┌──────────────────────────────────┐
        │ Control Response                  │
        │ • Sent back to CLI               │
        └──────────────────────────────────┘
```

### Implementation Requirements

**1. Permission Evaluation (permissions/service.go)**

```go
func (s *Service) CheckToolUse(
    ctx context.Context,
    toolName string,
    input map[string]any,
    permCtx *ToolPermissionContext,
) (PermissionResult, error) {
    s.mu.RLock()
    mode := s.mode
    s.mu.RUnlock()

    // Mode-based evaluation
    switch mode {
    case PermissionModeAllow:
        updates := s.applySuggestions(permCtx)
        return &PermissionResultAllow{Input: input, Update: updates}, nil

    case PermissionModeDeny:
        return &PermissionResultDeny{Reason: "Permission mode is deny"}, nil

    case PermissionModeAsk:
        // Check existing rules
        if result := s.checkRules(toolName, input); result != nil {
            return result, nil
        }

        // Invoke user callback
        if s.callback != nil {
            return s.callback(toolName, input, *permCtx)
        }

        // Default deny if no callback
        return &PermissionResultDeny{Reason: "No permission callback configured"}, nil
    }
}
```

**2. Control Protocol Integration (adapters/jsonrpc/protocol.go)**

```go
case "can_use_tool":
    result, err := p.permissionsService.CheckToolUse(
        ctx,
        request.ToolName,
        request.Input,
        &ToolPermissionContext{
            Suggestions: request.PermissionSuggestions,
            BlockedPath: request.BlockedPath,
        },
    )

    if err != nil {
        return &ControlResponse{
            Type:      "control_response",
            RequestID: req.RequestID,
            Error: &ControlError{
                Code:    "permission_check_error",
                Message: err.Error(),
            },
        }, nil
    }

    return &ControlResponse{
        Type:      "control_response",
        RequestID: req.RequestID,
        Result:    serializePermissionResult(result),
    }, nil
```

---

## Implementation Notes

### Hexagonal Architecture Benefits

**Clear Domain Boundaries:**
- Permissions isolated as a domain service
- No coupling between permissions and other domain services
- Clean dependency injection through constructor

**Dependency Injection:**
- Other services receive `*permissions.Service` through their constructors
- Callback function (`CanUseToolFunc`) passed via `PermissionsConfig`
- Easy to test with mock callbacks

### File Size Requirements

**pkg/claude/permissions/ package:**
- See Phase 2 for linting compliance notes
- Service implementation defined in Phase 2 follows all file size constraints

### Integration with Domain Services

The permissions service is injected into querying and streaming services as shown in Phase 2:

```go
// Example from Phase 2
queryService := querying.NewService(
	transport,
	protocol,
	parser,
	hooks,
	permissionsService,  // Injected here
	mcpServers,
)
```

The JSON-RPC adapter calls the permissions service when handling `can_use_tool` control requests (see Phase 3 for details).

### Integration Points


**Dependency Flow:**
```
Domain Services (querying, streaming)
    ↓ depend on
permissions.Service
    ↓ uses
CanUseToolFunc callback (provided by user)
```

**Key Points:**
- Domain services receive `*permissions.Service` via constructor
- Service is optional (can be nil)
- Callback function customizes permission logic
- See Phase 2 and Phase 3 for complete integration details

### Permission Persistence and Durability

**Important:** The SDK does not persist permission rules or mode changes to disk. All permission state is managed in-memory for the session lifetime.

#### Responsibility Split

**SDK Responsibilities (In-Memory Only):**
- Store current permission mode (`ask`, `allow`, `deny`) in memory
- Maintain in-memory permission rule cache during session
- Return `PermissionUpdate` suggestions to CLI in responses
- Apply mode changes from `set_permission_mode` control requests

**CLI Responsibilities (Persistent Storage):**
- Persist permission mode preferences across sessions
- Store permission rules (e.g., "always allow npm *") to disk/database
- Load saved rules when starting new sessions
- Apply saved rules via control protocol on session start

#### Session Lifecycle

```go
// Session Start
// 1. SDK initializes with default mode: "ask"
permissionsService := permissions.NewService(&permissions.PermissionsConfig{
    Mode:     permissions.PermissionModeAsk,  // Default
    Callback: userCallback,
})

// 2. CLI may send set_permission_mode to restore user's saved preference
// SDK updates in-memory mode:
permissionsService.UpdateMode(permissions.PermissionModeAllow)

// 3. During session, permission updates returned to CLI
// SDK returns updates in PermissionResultAllow:
return &PermissionResultAllow{
    Input: input,
    Update: []PermissionUpdate{  // CLI will persist these
        {
            Rule: PermissionRule{
                ToolPattern: "Bash",
                InputPattern: map[string]any{"command": "npm *"},
            },
            Value: PermissionRuleValueAlwaysAllow,
        },
    },
}

// 4. Session End
// SDK discards all in-memory state
// CLI retains saved rules for future sessions
```

#### Durability Guarantees

**What Survives Process Restart:**
- ❌ Permission mode (`ask`, `allow`, `deny`) - SDK defaults to `ask`
- ❌ Permission rule cache - SDK starts with empty rules
- ✅ User's saved preferences - managed by CLI, not SDK

**What Persists Within Session:**
- ✅ Current permission mode (until `set_permission_mode` called)
- ✅ Permission rules added via suggestions (in-memory cache)
- ✅ User callback function (registered at initialization)

#### Design Rationale

**Why SDK Doesn't Persist:**
1. **Separation of Concerns** - SDK focuses on protocol handling, CLI handles user preferences
2. **Flexibility** - Different CLIs can implement different storage strategies (file, database, cloud)
3. **Hexagonal Architecture** - Persistence is an infrastructure concern, not domain logic
4. **Parity with Python SDK** - Python SDK also uses in-memory-only permission state

**Implications for Users:**
- Permission decisions are fresh each session (unless CLI restores state)
- Mode changes via `set_permission_mode` are temporary per session
- Permission updates returned to CLI enable persistent rule storage
- Users building custom CLIs control persistence strategy

#### Permission Rule Caching (In-Memory)

The SDK maintains an in-memory cache of permission rules during the session:

```go
// In permissions/service.go
type Service struct {
    mu       sync.RWMutex
    mode     PermissionMode
    callback CanUseToolFunc
    rules    []PermissionUpdate  // In-memory cache
}

func (s *Service) CheckToolUse(...) (PermissionResult, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // Check cached rules first (fast path)
    for _, rule := range s.rules {
        if s.matchesRule(toolName, input, rule.Rule) {
            if rule.Value == PermissionRuleValueAlwaysAllow {
                return &PermissionResultAllow{Input: input}, nil
            }
            if rule.Value == PermissionRuleValueAlwaysDeny {
                return &PermissionResultDeny{Reason: "Blocked by rule"}, nil
            }
        }
    }

    // No cached rule matched, invoke callback or apply mode logic
    // ...
}

func (s *Service) applyPermissionUpdates(updates []PermissionUpdate) {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Add to in-memory cache
    s.rules = append(s.rules, updates...)
}
```

**Cache Behavior:**
- Rules added when CLI sends permission suggestions
- Rules checked before invoking user callback (performance optimization)
- Rules cleared when service is destroyed (end of session)
- No disk persistence or durability guarantees

---

## Testing Requirements

### Unit Tests

**permissions/service_test.go:**
- [ ] Test permission mode switching (`ask` → `allow` → `deny`)
- [ ] Test callback invocation in `ask` mode
- [ ] Test auto-allow/deny in `allow`/`deny` modes
- [ ] Test permission rule matching (exact match, wildcard patterns)
- [ ] Test concurrent permission checks (race conditions)
- [ ] Test permission update application to in-memory cache
- [ ] Test thread-safety with multiple goroutines

**Example Test:**
```go
func TestPermissionModeSwitch(t *testing.T) {
    callbackInvoked := false
    callback := func(toolName string, input map[string]any, ctx ToolPermissionContext) (PermissionResult, error) {
        callbackInvoked = true
        return &PermissionResultAllow{Input: input}, nil
    }

    svc := permissions.NewService(&permissions.PermissionsConfig{
        Mode:     permissions.PermissionModeAsk,
        Callback: callback,
    })

    // Test ask mode invokes callback
    result, err := svc.CheckToolUse(context.Background(), "Bash", map[string]any{"command": "ls"}, nil)
    assert.NoError(t, err)
    assert.True(t, callbackInvoked)
    assert.True(t, result.IsAllowed())

    // Switch to allow mode
    callbackInvoked = false
    svc.UpdateMode(permissions.PermissionModeAllow)

    // Test allow mode skips callback
    result, err = svc.CheckToolUse(context.Background(), "Bash", map[string]any{"command": "rm -rf /"}, nil)
    assert.NoError(t, err)
    assert.False(t, callbackInvoked) // Callback not invoked
    assert.True(t, result.IsAllowed())

    // Switch to deny mode
    svc.UpdateMode(permissions.PermissionModeDeny)

    // Test deny mode blocks everything
    result, err = svc.CheckToolUse(context.Background(), "Bash", map[string]any{"command": "ls"}, nil)
    assert.NoError(t, err)
    assert.False(t, result.IsAllowed())
    assert.Contains(t, result.(*PermissionResultDeny).Reason, "deny")
}
```

### Integration Tests

**adapters/jsonrpc/protocol_test.go:**
- [ ] Test `can_use_tool` request/response flow
- [ ] Test `set_permission_mode` control request handling
- [ ] Test permission mode persistence within session (multiple requests)
- [ ] Test permission mode does NOT persist across protocol instances
- [ ] Test permission update serialization in responses
- [ ] Test concurrent permission requests

**Example Integration Test:**
```go
func TestSetPermissionModePersistsInSession(t *testing.T) {
    // Create protocol with permissions service
    permSvc := permissions.NewService(&permissions.PermissionsConfig{
        Mode: permissions.PermissionModeAsk,
        Callback: func(toolName string, input map[string]any, ctx ToolPermissionContext) (PermissionResult, error) {
            return &PermissionResultDeny{Reason: "User denied"}, nil
        },
    })

    protocol := jsonrpc.NewProtocol(transport, parser, hooks, permSvc, mcpServers)

    // Send set_permission_mode to "allow"
    setModeReq := &ControlRequest{
        Type:      "control_request",
        RequestID: "req_1",
        Request: map[string]any{
            "subtype": "set_permission_mode",
            "mode":    "allow",
        },
    }
    resp, err := protocol.HandleControlRequest(context.Background(), setModeReq)
    assert.NoError(t, err)
    assert.Nil(t, resp.Error)

    // Verify mode change persists for subsequent requests
    canUseReq := &ControlRequest{
        Type:      "control_request",
        RequestID: "req_2",
        Request: map[string]any{
            "subtype":   "can_use_tool",
            "tool_name": "Bash",
            "input":     map[string]any{"command": "ls"},
        },
    }
    resp, err = protocol.HandleControlRequest(context.Background(), canUseReq)
    assert.NoError(t, err)
    result := resp.Result.(map[string]any)
    assert.Equal(t, "allow", result["type"]) // Auto-allowed due to mode
}

func TestPermissionModeDoesNotPersistAcrossInstances(t *testing.T) {
    // First instance
    permSvc1 := permissions.NewService(&permissions.PermissionsConfig{
        Mode: permissions.PermissionModeAsk,
    })
    permSvc1.UpdateMode(permissions.PermissionModeAllow)

    // Second instance (simulating new session)
    permSvc2 := permissions.NewService(&permissions.PermissionsConfig{
        Mode: permissions.PermissionModeAsk,
    })

    // Verify second instance has default mode (not "allow")
    assert.Equal(t, permissions.PermissionModeAsk, permSvc2.GetMode())
}
```

### Coverage Expectations

**Minimum Coverage Targets:**
- `permissions/service.go`: 90% line coverage
  - All permission modes tested (`ask`, `allow`, `deny`)
  - All callback scenarios (nil callback, callback returns allow/deny)
  - All rule matching logic (exact match, wildcard, no match)
  - Concurrent access scenarios

- `set_permission_mode` flows: 100% coverage
  - Valid mode changes (`ask`, `allow`, `deny`)
  - Invalid mode values (error handling)
  - Mode changes during active permission checks

- `can_use_tool` flows: 95% coverage
  - Permission granted scenarios
  - Permission denied scenarios
  - Permission updates applied
  - Input modification scenarios

**Table-Driven Test Example:**
```go
func TestPermissionRuleMatching(t *testing.T) {
    tests := []struct {
        name        string
        rule        PermissionRule
        toolName    string
        input       map[string]any
        shouldMatch bool
    }{
        {
            name: "exact match",
            rule: PermissionRule{
                ToolPattern:  "Bash",
                InputPattern: map[string]any{"command": "npm test"},
            },
            toolName:    "Bash",
            input:       map[string]any{"command": "npm test"},
            shouldMatch: true,
        },
        {
            name: "wildcard match",
            rule: PermissionRule{
                ToolPattern:  "Bash",
                InputPattern: map[string]any{"command": "npm *"},
            },
            toolName:    "Bash",
            input:       map[string]any{"command": "npm install"},
            shouldMatch: true,
        },
        {
            name: "no match - different command",
            rule: PermissionRule{
                ToolPattern:  "Bash",
                InputPattern: map[string]any{"command": "npm *"},
            },
            toolName:    "Bash",
            input:       map[string]any{"command": "ls -la"},
            shouldMatch: false,
        },
        {
            name: "no match - different tool",
            rule: PermissionRule{
                ToolPattern:  "Read",
                InputPattern: map[string]any{"file_path": "/tmp/*"},
            },
            toolName:    "Bash",
            input:       map[string]any{"command": "ls"},
            shouldMatch: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := permissions.NewService(&permissions.PermissionsConfig{
                Mode: permissions.PermissionModeAsk,
            })

            // Apply rule
            svc.applyPermissionUpdates([]PermissionUpdate{
                {Rule: tt.rule, Value: PermissionRuleValueAlwaysAllow},
            })

            // Check if rule matches
            matched := svc.matchesRule(tt.toolName, tt.input, tt.rule)
            assert.Equal(t, tt.shouldMatch, matched)
        })
    }
}
```

---

## Checklist

**Architecture:**
- [ ] `permissions.Service` implemented as defined in Phase 2
- [ ] Service properly integrated into domain services via constructor injection
- [ ] Permission types match Phase 2 definitions
- [ ] In-memory state management documented and tested
- [ ] No persistence layer (design decision documented)

**Functionality:**
- [ ] Permission callback properly invoked via `CheckToolUse()` method
- [ ] Permission results (allow/deny) correctly handled
- [ ] Permission updates returned to CLI in responses
- [ ] Tool inputs modified when permissions return updates
- [ ] Permission mode changes applied in-memory
- [ ] Permission rules cached in-memory for session duration

**Persistence & Durability:**
- [ ] Documentation clearly states SDK does not persist permission state
- [ ] Session lifecycle behavior documented
- [ ] Responsibility split between SDK and CLI explained
- [ ] In-memory rule caching implemented and tested
- [ ] Mode changes do NOT persist across service instances

**Testing:**
- [ ] Unit tests for `Service` with mock callbacks (90% coverage target)
- [ ] Integration tests with querying and streaming services
- [ ] Test various permission modes and scenarios
- [ ] Test mode switching across multiple requests within session
- [ ] Test mode does NOT persist across service instances (new session)
- [ ] Test concurrent permission checks (thread safety)
- [ ] Table-driven tests for rule matching logic
- [ ] Coverage for `set_permission_mode` flows (100% target)
- [ ] Coverage for `can_use_tool` flows (95% target)

---

## Related Files
- [Phase 5a: Hooks Support](./07a_phase_5_hooks.md)
- [Phase 5b: MCP Server Support](./07b_phase_5_mcp_servers.md)
- [Phase 5: Integration Summary](./07d_phase_5_integration_summary.md)
