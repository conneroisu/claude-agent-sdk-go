// Package permissions manages tool permission checks for the
// Claude Agent SDK.
//
// Permissions control which tools can be used and allow custom
// permission logic via callbacks.
package permissions

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// PermissionResult represents the outcome of a permission check.
type PermissionResult interface {
	permissionResult()
}

// PermissionResultAllow indicates tool use is allowed.
type PermissionResultAllow struct {
	// UpdatedInput optionally modifies tool input parameters.
	// Intentionally flexible - tool inputs vary by tool.
	UpdatedInput map[string]any

	// UpdatedPermissions contains permission changes to apply
	UpdatedPermissions []PermissionUpdate
}

// permissionResult implements the PermissionResult interface.
func (PermissionResultAllow) permissionResult() {}

// PermissionResultDeny indicates tool use is denied.
type PermissionResultDeny struct {
	// Message explains why permission was denied
	Message string

	// Interrupt indicates if execution should stop
	Interrupt bool
}

// permissionResult implements the PermissionResult interface.
func (PermissionResultDeny) permissionResult() {}

// PermissionUpdate represents a permission change.
type PermissionUpdate struct {
	// Type identifies the update type
	Type string

	// Rules contains permission rule values
	Rules []PermissionRuleValue

	// Behavior specifies allow/deny/ask behavior
	Behavior *PermissionBehavior

	// Mode sets permission mode
	Mode *options.PermissionMode

	// Directories lists affected directories
	Directories []string

	// Destination specifies where to save permission
	Destination *PermissionUpdateDestination
}

// PermissionRuleValue defines a single permission rule.
type PermissionRuleValue struct {
	// ToolName identifies the tool
	ToolName string

	// RuleContent contains the rule definition
	RuleContent *string
}

// PermissionBehavior defines permission behavior.
type PermissionBehavior string

const (
	// PermissionBehaviorAllow always allows the tool.
	PermissionBehaviorAllow PermissionBehavior = "allow"

	// PermissionBehaviorDeny always denies the tool.
	PermissionBehaviorDeny PermissionBehavior = "deny"

	// PermissionBehaviorAsk prompts for permission.
	PermissionBehaviorAsk PermissionBehavior = "ask"
)

// PermissionUpdateDestination specifies where permissions are
// saved.
type PermissionUpdateDestination string

const (
	// PermissionDestinationUserSettings saves to user settings.
//nolint:revive // line-length-limit: constant name clarity
	PermissionDestinationUserSettings PermissionUpdateDestination = "userSettings" //nolint:lll

//nolint:revive // line-length-limit: constant name clarity
	// PermissionDestinationProjectSettings saves to project.
//nolint:revive,lll // line-length-limit: constant name clarity
//nolint:revive // line-length-limit: constant name clarity

	// PermissionDestinationLocalSettings saves to local settings.
//nolint:revive // line-length-limit: constant name clarity
	PermissionDestinationLocalSettings PermissionUpdateDestination = "localSettings" //nolint:lll

	// PermissionDestinationSession saves for session only.
//nolint:revive,lll // line-length-limit: constant name clarity
)

// ToolPermissionContext provides context for permission
// decisions.
type ToolPermissionContext struct {
	// Suggestions contains recommended permission updates
	Suggestions []PermissionUpdate
}

// CanUseToolFunc is a callback for permission checks.
// input is intentionally map[string]any as tool inputs vary.
type CanUseToolFunc func(
	ctx context.Context,
	toolName string,
	input map[string]any,
	permCtx ToolPermissionContext,
) (PermissionResult, error)

// Config holds permission service configuration.
type Config struct {
	// Mode sets the permission mode
	Mode options.PermissionMode

	// CanUseTool is optional callback for custom permission logic
	CanUseTool CanUseToolFunc
}
