package permissions

import "context"

// PermissionMode represents the permission checking mode.
type PermissionMode string

// Permission modes.
const (
	PermissionModeBypass      PermissionMode = "bypass"
	PermissionModeDefault     PermissionMode = "default"
	PermissionModePlan        PermissionMode = "plan"
	PermissionModeAcceptEdits PermissionMode = "acceptEdits"
)

// PermissionsConfig configures the permissions service.
type PermissionsConfig struct {
	Mode       PermissionMode
	CanUseTool CanUseToolFunc
}

// CanUseToolFunc is a callback that checks tool permissions.
type CanUseToolFunc func(
	ctx context.Context,
	toolName string,
	input map[string]any,
	permCtx ToolPermissionContext,
) (PermissionResult, error)

// ToolPermissionContext provides context for permission checks.
type ToolPermissionContext struct {
	Suggestions []PermissionUpdate
}

// PermissionResult is a sealed union of allow/deny results.
type PermissionResult interface {
	permissionResult()
}

// PermissionResultAllow indicates the tool is allowed.
type PermissionResultAllow struct {
	ModifiedInput      map[string]any
	UpdatedPermissions []PermissionUpdate
}

func (PermissionResultAllow) permissionResult() {}

// PermissionResultDeny indicates the tool is denied.
type PermissionResultDeny struct {
	Message   string
	Interrupt bool
}

func (PermissionResultDeny) permissionResult() {}

// PermissionUpdate represents a permission rule update.
type PermissionUpdate struct {
	Type        string
	Rules       []PermissionRule
	Behavior    string
	Mode        *string
	Directories []string
	Destination string
}

// PermissionRule represents a tool permission rule.
type PermissionRule struct {
	ToolName string
	Content  []string
}
