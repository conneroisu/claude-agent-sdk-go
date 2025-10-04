package permissions

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// PermissionResult represents the outcome of a permission check.
// It is a discriminated union that can be either Allow or Deny.
type PermissionResult interface {
	permissionResult()
}

// PermissionResultAllow indicates tool use is allowed.
// It can optionally include updated tool input and permission changes.
type PermissionResultAllow struct {
	// UpdatedInput is intentionally flexible as tool inputs vary
	UpdatedInput       map[string]any
	UpdatedPermissions []PermissionUpdate
}

func (PermissionResultAllow) permissionResult() {}

// PermissionResultDeny indicates tool use is denied.
// It provides a denial message and controls whether to interrupt execution.
type PermissionResultDeny struct {
	Message   string
	Interrupt bool
}

func (PermissionResultDeny) permissionResult() {}

// PermissionUpdate represents a permission change.
// It encapsulates rule changes, mode updates, and storage destinations.
type PermissionUpdate struct {
	Type        string
	Rules       []PermissionRuleValue
	Behavior    *PermissionBehavior
	Mode        *options.PermissionMode
	Directories []string
	Destination *PermissionUpdateDestination
}

// PermissionRuleValue represents a single permission rule.
// It targets a specific tool with optional content matching.
type PermissionRuleValue struct {
	ToolName    string
	RuleContent *string
}

// PermissionBehavior defines how a permission rule behaves.
type PermissionBehavior string

const (
	// PermissionBehaviorAllow automatically allows matching operations.
	PermissionBehaviorAllow PermissionBehavior = "allow"
	// PermissionBehaviorDeny automatically denies matching operations.
	PermissionBehaviorDeny PermissionBehavior = "deny"
	// PermissionBehaviorAsk prompts for permission on matching.
	PermissionBehaviorAsk PermissionBehavior = "ask"
)

// PermissionUpdateDestination specifies where permission changes are stored.
type PermissionUpdateDestination string

const (
	// PermissionDestinationUserSettings stores in user settings.
	PermissionDestinationUserSettings PermissionUpdateDestination = "userSettings"
	// PermissionDestinationProjectSettings stores in project.
	PermissionDestinationProjectSettings PermissionUpdateDestination = "projectSettings"
	// PermissionDestinationLocalSettings stores locally.
	PermissionDestinationLocalSettings PermissionUpdateDestination = "localSettings"
	// PermissionDestinationSession stores for session.
	PermissionDestinationSession PermissionUpdateDestination = "session"
)

// ToolPermissionContext provides context for permission decisions.
// It includes CLI-provided suggestions for "always allow" workflows.
type ToolPermissionContext struct {
	Suggestions []PermissionUpdate
}

// CanUseToolFunc is a callback for permission checks.
// input is intentionally map[string]any as tool inputs vary by tool.
type CanUseToolFunc func(
	ctx context.Context,
	toolName string,
	input map[string]any,
	permCtx ToolPermissionContext,
) (PermissionResult, error)

// PermissionsConfig holds permission service configuration.
// It defines the initial permission mode and optional callback.
type PermissionsConfig struct {
	Mode       options.PermissionMode
	CanUseTool CanUseToolFunc
}
