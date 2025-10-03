## Phase 5c: Permission Callbacks

### Priority: Medium

### Overview
Permission callbacks allow users to implement custom authorization logic for tool usage. These callbacks receive context about the tool being used and can allow, deny, or modify the request.

**Hexagonal Architecture Alignment:**
- Permissions are a distinct domain boundary (authorization/access control)
- Isolated in `internal/permissions/` package with clean separation of concerns
- Public API in `permissions.go` re-exports only consumer-facing types
- Core agent/tool logic depends on permission **ports** (abstractions), not implementations

### Package Structure

```
internal/permissions/
  ├── types.go       # Core types, interfaces, constants
  ├── evaluator.go   # Permission evaluation logic
  ├── updater.go     # Permission update application
  └── callback.go    # Callback orchestration

permissions.go       # Public API - re-exports from internal/permissions
```

---

### Public API (permissions.go)

```go
package claude

import (
	"context"
	"github.com/anthropics/claude-agent-sdk-go/internal/permissions"
)

// Re-export public-facing types for consumers
type (
	// ToolPermissionContext contains context for permission checks
	ToolPermissionContext = permissions.Context

	// PermissionResult is the result of a permission check
	PermissionResult = permissions.Result

	// PermissionResultAllow indicates permission granted
	PermissionResultAllow = permissions.ResultAllow

	// PermissionResultDeny indicates permission denied
	PermissionResultDeny = permissions.ResultDeny

	// CanUseToolFunc is a function that checks if a tool can be used
	CanUseToolFunc = permissions.CanUseToolFunc

	// PermissionUpdate represents a permission update
	PermissionUpdate = permissions.Update

	// PermissionUpdateType is the type of permission update
	PermissionUpdateType = permissions.UpdateType

	// PermissionRuleValue is a permission rule
	PermissionRuleValue = permissions.RuleValue

	// PermissionBehavior defines permission behavior (allow/deny/ask)
	PermissionBehavior = permissions.Behavior

	// PermissionMode defines permission mode
	PermissionMode = permissions.Mode

	// PermissionUpdateDestination defines where updates are stored
	PermissionUpdateDestination = permissions.UpdateDestination
)

// Re-export constants
const (
	PermissionUpdateTypeAddRules          = permissions.UpdateTypeAddRules
	PermissionUpdateTypeReplaceRules      = permissions.UpdateTypeReplaceRules
	PermissionUpdateTypeRemoveRules       = permissions.UpdateTypeRemoveRules
	PermissionUpdateTypeSetMode           = permissions.UpdateTypeSetMode
	PermissionUpdateTypeAddDirectories    = permissions.UpdateTypeAddDirectories
	PermissionUpdateTypeRemoveDirectories = permissions.UpdateTypeRemoveDirectories

	PermissionBehaviorAllow = permissions.BehaviorAllow
	PermissionBehaviorDeny  = permissions.BehaviorDeny
	PermissionBehaviorAsk   = permissions.BehaviorAsk

	PermissionDestinationUserSettings    = permissions.DestinationUserSettings
	PermissionDestinationProjectSettings = permissions.DestinationProjectSettings
	PermissionDestinationLocalSettings   = permissions.DestinationLocalSettings
	PermissionDestinationSession         = permissions.DestinationSession
)
```

---

### Internal Implementation (internal/permissions/)

#### types.go
```go
package permissions

import "context"

// Context contains context for permission evaluation
type Context struct {
	Suggestions []Update
}

// Result is the result of a permission check
type Result interface {
	result()
}

// ResultAllow indicates permission is granted
type ResultAllow struct {
	UpdatedInput       map[string]any
	UpdatedPermissions []Update
}

// ResultDeny indicates permission is denied
type ResultDeny struct {
	Message   string
	Interrupt bool
}

func (ResultAllow) result() {}
func (ResultDeny) result()  {}

// CanUseToolFunc is a function that checks if a tool can be used
type CanUseToolFunc func(ctx context.Context, toolName string, input map[string]any, permCtx Context) (Result, error)

// Update represents a permission update
type Update struct {
	Type        UpdateType
	Rules       []RuleValue
	Behavior    *Behavior
	Mode        *Mode
	Directories []string
	Destination *UpdateDestination
}

// UpdateType is the type of permission update
type UpdateType string

const (
	UpdateTypeAddRules          UpdateType = "addRules"
	UpdateTypeReplaceRules      UpdateType = "replaceRules"
	UpdateTypeRemoveRules       UpdateType = "removeRules"
	UpdateTypeSetMode           UpdateType = "setMode"
	UpdateTypeAddDirectories    UpdateType = "addDirectories"
	UpdateTypeRemoveDirectories UpdateType = "removeDirectories"
)

// RuleValue is a permission rule
type RuleValue struct {
	ToolName    string
	RuleContent *string
}

// Behavior defines permission behavior
type Behavior string

const (
	BehaviorAllow Behavior = "allow"
	BehaviorDeny  Behavior = "deny"
	BehaviorAsk   Behavior = "ask"
)

// Mode is the permission mode
type Mode string

// UpdateDestination defines where permission updates are stored
type UpdateDestination string

const (
	DestinationUserSettings    UpdateDestination = "userSettings"
	DestinationProjectSettings UpdateDestination = "projectSettings"
	DestinationLocalSettings   UpdateDestination = "localSettings"
	DestinationSession         UpdateDestination = "session"
)
```

#### evaluator.go
```go
package permissions

import "context"

// Evaluator handles permission evaluation logic
type Evaluator struct {
	callback CanUseToolFunc
}

// NewEvaluator creates a new permission evaluator
func NewEvaluator(callback CanUseToolFunc) *Evaluator {
	return &Evaluator{callback: callback}
}

// Evaluate checks if a tool can be used
func (e *Evaluator) Evaluate(ctx context.Context, toolName string, input map[string]any, permCtx Context) (Result, error) {
	if e.callback == nil {
		return ResultAllow{UpdatedInput: input}, nil
	}
	return e.callback(ctx, toolName, input, permCtx)
}
```

#### updater.go
```go
package permissions

// Updater applies permission updates to settings
type Updater struct {
	// TODO: Add settings manager dependency
}

// NewUpdater creates a new permission updater
func NewUpdater() *Updater {
	return &Updater{}
}

// Apply applies permission updates to the appropriate destination
func (u *Updater) Apply(updates []Update) error {
	// TODO: Implementation will route updates to correct settings layer
	// based on Destination field (userSettings, projectSettings, etc.)
	return nil
}
```

#### callback.go
```go
package permissions

import "context"

// Handler orchestrates permission checks and updates
type Handler struct {
	evaluator *Evaluator
	updater   *Updater
}

// NewHandler creates a new permission handler
func NewHandler(callback CanUseToolFunc) *Handler {
	return &Handler{
		evaluator: NewEvaluator(callback),
		updater:   NewUpdater(),
	}
}

// CheckToolUse evaluates permission and applies any updates
func (h *Handler) CheckToolUse(ctx context.Context, toolName string, input map[string]any, permCtx Context) (Result, error) {
	result, err := h.evaluator.Evaluate(ctx, toolName, input, permCtx)
	if err != nil {
		return nil, err
	}

	// Apply permission updates if present
	if allow, ok := result.(ResultAllow); ok && len(allow.UpdatedPermissions) > 0 {
		if err := h.updater.Apply(allow.UpdatedPermissions); err != nil {
			return nil, err
		}
	}

	return result, nil
}
```

---

## Implementation Notes

### Hexagonal Architecture Benefits

**Clear Domain Boundaries:**
- Permissions isolated as distinct subdomain
- No coupling to tool execution or agent orchestration
- Explicit, unidirectional dependencies

**Dependency Inversion:**
- Tool executors depend on `CanUseToolFunc` (port/abstraction)
- Implementation details hidden in `internal/permissions/`
- Easy to test and mock

**Separation of Concerns:**
- `evaluator.go` - Permission checking logic
- `updater.go` - Settings update application
- `callback.go` - Orchestration layer
- `types.go` - Domain types and constants

### File Size Requirements

**internal/permissions/ package:**
- `types.go` - ~100 lines (types, interfaces, constants)
- `evaluator.go` - ~25 lines (evaluation logic)
- `updater.go` - ~25 lines (update logic)
- `callback.go` - ~40 lines (orchestration)
- ✅ All files under 175 lines
- Well-organized, focused responsibilities

### Usage Patterns

**Consumer Usage (using public API):**
```go
package main

import (
	"context"
	"strings"

	"github.com/anthropics/claude-agent-sdk-go"
)

func restrictedBashCallback(ctx context.Context, toolName string, input map[string]any, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
	if toolName == "Bash" {
		command := input["command"].(string)
		if strings.Contains(command, "rm -rf") {
			return claude.PermissionResultDeny{
				Message: "Destructive commands not allowed",
				Interrupt: true,
			}, nil
		}
	}
	return claude.PermissionResultAllow{
		UpdatedInput: input,
	}, nil
}

func main() {
	client := claude.NewClient("api-key")

	agent := client.NewAgent(
		claude.WithPermissionCallback(restrictedBashCallback),
	)

	// Permission callback is automatically invoked during tool execution
	agent.Execute(context.Background(), "run a bash command")
}
```

**Integration with Tool Execution:**
```go
// In internal tool executor
type ToolExecutor struct {
	permissionHandler *permissions.Handler
}

func (te *ToolExecutor) Execute(ctx context.Context, tool *Tool, input map[string]any) (any, error) {
	// Check permissions before execution
	permCtx := permissions.Context{
		Suggestions: buildSuggestions(tool),
	}

	result, err := te.permissionHandler.CheckToolUse(ctx, tool.Name, input, permCtx)
	if err != nil {
		return nil, err
	}

	// Handle deny
	if deny, ok := result.(permissions.ResultDeny); ok {
		if deny.Interrupt {
			return nil, fmt.Errorf("permission denied: %s", deny.Message)
		}
		// Log but continue if not interrupting
	}

	// Handle allow (potentially with modifications)
	allow := result.(permissions.ResultAllow)
	return tool.Execute(ctx, allow.UpdatedInput)
}
```

### Integration Points

**Dependency Flow (Hexagonal):**
```
Agent/Tool Executor (Core)
    ↓ depends on
CanUseToolFunc (Port/Interface)
    ↑ implemented by
permissions.Handler (Adapter/Implementation)
```

**Key Points:**
- Core domain (agent/tool execution) depends on **abstraction** (`CanUseToolFunc`)
- Implementation lives in `internal/permissions/`
- No reverse dependencies - permissions cannot depend on agent
- Easy to test with mock implementations

### Checklist

**Architecture:**
- [ ] `internal/permissions/` package created with proper structure
- [ ] Public API re-exports types from internal package
- [ ] Tool executor depends on `CanUseToolFunc` port, not implementation

**Functionality:**
- [ ] Permission callback properly integrated with tool execution flow
- [ ] Permission updates correctly propagate to settings
- [ ] Deny interrupts properly halt execution
- [ ] Allow modifications correctly update tool input

**Testing:**
- [ ] Unit tests for `Evaluator` with mock callbacks
- [ ] Unit tests for `Updater` with mock settings
- [ ] Integration tests with real tool execution
- [ ] Test permission update propagation

---

## Related Files
- [Phase 5a: Hooks Support](./07a_phase_5_hooks.md)
- [Phase 5b: MCP Server Support](./07b_phase_5_mcp_servers.md)
- [Phase 5: Integration Summary](./07d_phase_5_integration_summary.md)
