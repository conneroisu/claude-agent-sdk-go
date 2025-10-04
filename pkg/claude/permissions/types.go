package permissions

import "github.com/conneroisu/claude/pkg/claude/options"

// PermissionResult represents the outcome of a permission check.
type PermissionResult interface {
	permissionResult()
}

// PermissionResultAllow indicates tool use is allowed.
type PermissionResultAllow struct {
	// UpdatedInput can modify tool input (varies by tool)
	UpdatedInput map[string]any

	// UpdatedPermissions can update permission rules
	UpdatedPermissions []PermissionUpdate
}

func (PermissionResultAllow) permissionResult() {}

// PermissionResultDeny indicates tool use is denied.
type PermissionResultDeny struct {
	// Message explains why permission was denied
	Message string

	// Interrupt stops execution completely if true
	Interrupt bool
}

func (PermissionResultDeny) permissionResult() {}

// PermissionUpdate represents a permission change.
type PermissionUpdate struct {
	// Type indicates the update operation
	Type string

	// Rules contains permission rules (for rule operations)
	Rules []PermissionRuleValue

	// Behavior is the permission action (for rule operations)
	Behavior *PermissionBehavior

	// Mode is the new permission mode (for setMode operation)
	Mode *options.PermissionMode

	// Directories are paths (for directory operations)
	Directories []string

	// Destination specifies where to save the update
	Destination *PermissionUpdateDestination
}

// PermissionRuleValue defines a permission rule.
type PermissionRuleValue struct {
	// ToolName is the tool this rule applies to
	ToolName string

	// RuleContent is the rule pattern (e.g., "git:*")
	RuleContent *string
}

// PermissionBehavior defines how a permission rule behaves.
type PermissionBehavior string

const (
	// PermissionBehaviorAllow automatically allows the tool.
	PermissionBehaviorAllow PermissionBehavior = "allow"

	// PermissionBehaviorDeny automatically denies the tool.
	PermissionBehaviorDeny PermissionBehavior = "deny"

	// PermissionBehaviorAsk prompts the user for permission.
	PermissionBehaviorAsk PermissionBehavior = "ask"
)

// PermissionUpdateDestination specifies where to save permission updates.
type PermissionUpdateDestination string

//revive:disable:line-length-limit Long type and const names required by API
const (
	// PermissionDestinationUserSettings saves to user settings.
	PermissionDestinationUserSettings PermissionUpdateDestination = "userSettings"

	// PermissionDestinationProjectSettings saves to project settings.
	PermissionDestinationProjectSettings PermissionUpdateDestination = "projectSettings"

	// PermissionDestinationLocalSettings saves to local settings.
	PermissionDestinationLocalSettings PermissionUpdateDestination = "localSettings"

	// PermissionDestinationSession saves for current session only.
	PermissionDestinationSession PermissionUpdateDestination = "session"
)

//revive:enable:line-length-limit

// ToolPermissionContext provides context for permission decisions.
type ToolPermissionContext struct {
	// Suggestions from CLI for "always allow" workflows
	Suggestions []PermissionUpdate
}
