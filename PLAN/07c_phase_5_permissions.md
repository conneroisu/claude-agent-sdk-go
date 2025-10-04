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

### Checklist

**Architecture:**
- [ ] `permissions.Service` implemented as defined in Phase 2
- [ ] Service properly integrated into domain services via constructor injection
- [ ] Permission types match Phase 2 definitions

**Functionality:**
- [ ] Permission callback properly invoked via `CheckToolUse()` method
- [ ] Permission results (allow/deny) correctly handled
- [ ] Permission updates applied when present
- [ ] Tool inputs modified when permissions return updates

**Testing:**
- [ ] Unit tests for `Service` with mock callbacks
- [ ] Integration tests with querying and streaming services
- [ ] Test various permission modes and scenarios

---

## Related Files
- [Phase 5a: Hooks Support](./07a_phase_5_hooks.md)
- [Phase 5b: MCP Server Support](./07b_phase_5_mcp_servers.md)
- [Phase 5: Integration Summary](./07d_phase_5_integration_summary.md)
