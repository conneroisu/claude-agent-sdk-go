// Package permissions manages tool permission checks.
//
// This is a domain service that determines whether Claude can use
// specific tools based on permission mode and user-defined callbacks.
package permissions

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// PermissionResult represents the outcome of a permission check.
//
// Can be either Allow (tool use permitted) or Deny (tool use blocked).
type PermissionResult interface {
	permissionResult()
}

// PermissionResultAllow indicates tool use is allowed.
//
// Can optionally include updated tool input and permission changes
// to be applied after the tool use.
type PermissionResultAllow struct {
	UpdatedInput       map[string]any // Tool inputs vary by tool
	UpdatedPermissions []PermissionUpdate
}

func (PermissionResultAllow) permissionResult() {}

// PermissionResultDeny indicates tool use is denied.
//
// Includes a message explaining the denial and whether to interrupt
// the conversation entirely.
type PermissionResultDeny struct {
	Message   string
	Interrupt bool
}

func (PermissionResultDeny) permissionResult() {}

// PermissionUpdate represents a permission change.
//
// Used to update permission rules, behaviors, modes, or directories.
type PermissionUpdate struct {
	Type        string
	Rules       []PermissionRuleValue
	Behavior    *PermissionBehavior
	Mode        *options.PermissionMode
	Directories []string
	Destination *PermissionUpdateDestination
}

// PermissionRuleValue defines a permission rule for a specific tool.
type PermissionRuleValue struct {
	ToolName    string
	RuleContent *string
}

// PermissionBehavior defines how to handle tool use requests.
type PermissionBehavior string

const (
	// PermissionBehaviorAllow always allows the tool.
	PermissionBehaviorAllow PermissionBehavior = "allow"
	// PermissionBehaviorDeny always denies the tool.
	PermissionBehaviorDeny PermissionBehavior = "deny"
	// PermissionBehaviorAsk prompts the user for each use.
	PermissionBehaviorAsk PermissionBehavior = "ask"
)

// PermissionUpdateDestination specifies where to save permission changes.
type PermissionUpdateDestination string

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

// ToolPermissionContext provides context for permission decisions.
//
// Includes suggestions from the CLI about how to handle the request.
type ToolPermissionContext struct {
	Suggestions []PermissionUpdate
}

// CanUseToolFunc is a callback for permission checks.
//
// Input is intentionally map[string]any as tool inputs vary by tool.
// The callback should return Allow or Deny with optional updates.
type CanUseToolFunc func(
	ctx context.Context,
	toolName string,
	input map[string]any,
	permCtx ToolPermissionContext,
) (PermissionResult, error)

// PermissionsConfig holds permission service configuration.
type PermissionsConfig struct {
	Mode       options.PermissionMode
	CanUseTool CanUseToolFunc
}

// Service manages tool permissions.
//
// This is a domain service - pure business logic for permission checks.
type Service struct {
	mode       options.PermissionMode
	canUseTool CanUseToolFunc
}

// NewService creates a new permissions service.
//
// If config is nil, defaults to Ask mode with no custom callback.
func NewService(config *PermissionsConfig) *Service {
	if config == nil {
		return &Service{
			mode: options.PermissionModeAsk,
		}
	}
	return &Service{
		mode:       config.Mode,
		canUseTool: config.CanUseTool,
	}
}

// CheckToolUse verifies if a tool can be used.
//
// The suggestions parameter comes from the control protocol's
// permission_suggestions field. Returns Allow/Deny result.
func (s *Service) CheckToolUse(
	ctx context.Context,
	toolName string,
	input map[string]any,
	suggestions []PermissionUpdate,
) (PermissionResult, error) {
	// Check permission mode
	switch s.mode {
	case options.PermissionModeBypassPermissions:
		return &PermissionResultAllow{}, nil
	case options.PermissionModeDefault,
		options.PermissionModeAcceptEdits,
		options.PermissionModePlan:
		return s.checkWithCallback(
			ctx,
			toolName,
			input,
			suggestions,
		)
	default:
		return s.denyUnknownMode()
	}
}

// checkWithCallback checks permission using the callback if set.
func (s *Service) checkWithCallback(
	ctx context.Context,
	toolName string,
	input map[string]any,
	suggestions []PermissionUpdate,
) (PermissionResult, error) {
	if s.canUseTool != nil {
		permCtx := ToolPermissionContext{
			Suggestions: suggestions,
		}
		result, err := s.canUseTool(ctx, toolName, input, permCtx)
		if err != nil {
			return nil, fmt.Errorf("permission callback: %w", err)
		}
		return result, nil
	}
	// Default: allow (CLI handles prompting in default mode)
	return &PermissionResultAllow{}, nil
}

// denyUnknownMode denies permission for unknown modes.
func (s *Service) denyUnknownMode() (PermissionResult, error) {
	return &PermissionResultDeny{
		Message:   fmt.Sprintf("unknown permission mode: %s", s.mode),
		Interrupt: false,
	}, nil
}

// UpdateMode changes the permission mode.
func (s *Service) UpdateMode(mode options.PermissionMode) {
	s.mode = mode
}

// CheckPermission implements ports.PermissionService interface.
//
// This is a wrapper around CheckToolUse for interface compatibility.
func (s *Service) CheckPermission(
	ctx context.Context,
	toolName string,
	toolInput map[string]any,
) (bool, error) {
	// Call CheckToolUse with empty suggestions
	result, err := s.CheckToolUse(ctx, toolName, toolInput, nil)
	if err != nil {
		return false, err
	}

	// Convert PermissionResult to boolean
	switch result.(type) {
	case *PermissionResultAllow:
		return true, nil
	case *PermissionResultDeny:
		return false, nil
	default:
		return false, fmt.Errorf("unknown permission result type")
	}
}
