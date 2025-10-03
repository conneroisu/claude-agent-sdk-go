## Phase 5c: Permission Callbacks

### Priority: Medium

### Overview
Permission callbacks allow users to implement custom authorization logic for tool usage. These callbacks receive context about the tool being used and can allow, deny, or modify the request.

### Public API (permissions.go)

```go
package claude

import (
	"context"
)

type ToolPermissionContext struct {
	Suggestions []PermissionUpdate
}

type PermissionResult interface {
	permissionResult()
}

type PermissionResultAllow struct {
	UpdatedInput       map[string]any
	UpdatedPermissions []PermissionUpdate
}

type PermissionResultDeny struct {
	Message   string
	Interrupt bool
}

func (PermissionResultAllow) permissionResult() {}
func (PermissionResultDeny) permissionResult()  {}

// CanUseToolFunc is a function that can be used to check if a tool can be used.
type CanUseToolFunc func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error)

// PermissionUpdate is a permission update.
type PermissionUpdate struct {
	Type        PermissionUpdateType
	Rules       []PermissionRuleValue
	Behavior    *PermissionBehavior
	Mode        *PermissionMode
	Directories []string
	Destination *PermissionUpdateDestination
}

// PermissionUpdateType is the type of a permission update.
type PermissionUpdateType string

const (
	// PermissionUpdateTypeAddRules is the type for adding rules to a permission update.
	PermissionUpdateTypeAddRules PermissionUpdateType = "addRules"
	// PermissionUpdateTypeReplaceRules is the type for replacing rules in a permission update.
	PermissionUpdateTypeReplaceRules PermissionUpdateType = "replaceRules"
	// PermissionUpdateTypeRemoveRules is the type for removing rules from a permission update.
	PermissionUpdateTypeRemoveRules PermissionUpdateType = "removeRules"
	// PermissionUpdateTypeSetMode is the type for setting the mode of a permission update.
	PermissionUpdateTypeSetMode PermissionUpdateType = "setMode"
	// PermissionUpdateTypeAddDirectories is the type for adding directories to a permission update.
	PermissionUpdateTypeAddDirectories PermissionUpdateType = "addDirectories"
	// PermissionUpdateTypeRemoveDirectories is the type for removing directories from a permission update.
	PermissionUpdateTypeRemoveDirectories PermissionUpdateType = "removeDirectories"
)

// PermissionRuleValue is a rule for a permission update.
type PermissionRuleValue struct {
	ToolName    string
	RuleContent *string
}

// PermissionBehavior is the behavior for a permission update.
type PermissionBehavior string

const (
	// PermissionBehaviorAllow is the behavior for allowing a permission.
	PermissionBehaviorAllow PermissionBehavior = "allow"
	// PermissionBehaviorDeny is the behavior for denying a permission.
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	// PermissionBehaviorAsk is the behavior for asking the user for permission.
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

// PermissionMode is the mode for a permission update.
type PermissionMode string

// PermissionUpdateDestination is the destination for a permission update.
type PermissionUpdateDestination string

const (
	// PermissionDestinationUserSettings is the destination for user-specific settings.
	PermissionDestinationUserSettings    PermissionUpdateDestination = "userSettings"
	// PermissionDestinationProjectSettings is the destination for project-specific settings.
	PermissionDestinationProjectSettings PermissionUpdateDestination = "projectSettings"
	// PermissionDestinationLocalSettings is the destination for local settings.
	PermissionDestinationLocalSettings   PermissionUpdateDestination = "localSettings"
	// PermissionDestinationSession is the destination for session-specific settings.
	PermissionDestinationSession         PermissionUpdateDestination = "session"
)
```

---

## Implementation Notes

### File Size Requirements

**permissions/ package:**
- ✅ Already appropriately sized (under 175 lines)
- Types and interfaces are well-organized
- No splitting required

### Usage Patterns

**CanUseToolFunc implementation:**
- Check tool name and input parameters
- Return `PermissionResultAllow` to proceed (optionally with modifications)
- Return `PermissionResultDeny` to block with a message
- Can suggest permission updates via `UpdatedPermissions`

**Example:**
```go
func restrictedBashCallback(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error) {
	if toolName == "Bash" {
		command := input["command"].(string)
		if strings.Contains(command, "rm -rf") {
			return PermissionResultDeny{
				Message: "Destructive commands not allowed",
				Interrupt: true,
			}, nil
		}
	}
	return PermissionResultAllow{
		UpdatedInput: input,
	}, nil
}
```

### Checklist

- [ ] Permission callback properly integrated with tool execution flow
- [ ] Permission updates correctly propagate to settings
- [ ] Deny interrupts properly halt execution
- [ ] Allow modifications correctly update tool input

---

## Related Files
- [Phase 5a: Hooks Support](./07a_phase_5_hooks.md)
- [Phase 5b: MCP Server Support](./07b_phase_5_mcp_servers.md)
- [Phase 5: Integration Summary](./07d_phase_5_integration_summary.md)
