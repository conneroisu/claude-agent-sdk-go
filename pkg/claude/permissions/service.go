package permissions

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// PermissionResult represents the outcome of a permission check
type PermissionResult interface {
	permissionResult()
}

// PermissionResultAllow indicates tool use is allowed
type PermissionResultAllow struct {
	// Intentionally flexible - tool inputs vary by tool
	UpdatedInput       map[string]any
	UpdatedPermissions []PermissionUpdate
}

func (PermissionResultAllow) permissionResult() {}

// PermissionResultDeny indicates tool use is denied
type PermissionResultDeny struct {
	Message   string
	Interrupt bool
}

func (PermissionResultDeny) permissionResult() {}

// PermissionUpdate represents a permission change
type PermissionUpdate struct {
	Type        string
	Rules       []PermissionRuleValue
	Behavior    *PermissionBehavior
	Mode        *options.PermissionMode
	Directories []string
	Destination *PermissionUpdateDestination
}

// PermissionRuleValue represents a single permission rule
type PermissionRuleValue struct {
	ToolName    string
	RuleContent *string
}

// PermissionBehavior defines how permissions behave
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

// PermissionUpdateDestination specifies where permission updates are saved
type PermissionUpdateDestination string

const (
	PermissionDestinationUserSettings PermissionUpdateDestination = (
		"userSettings")
	PermissionDestinationProjectSettings PermissionUpdateDestination = (
		"projectSettings")
	PermissionDestinationLocalSettings PermissionUpdateDestination = (
		"localSettings")
	PermissionDestinationSession PermissionUpdateDestination = (
		"session")
)

// ToolPermissionContext provides context for permission decisions
type ToolPermissionContext struct {
	Suggestions []PermissionUpdate
}

// CanUseToolFunc is a callback for permission checks
// input is intentionally map[string]any as tool inputs vary by tool
type CanUseToolFunc func(
	ctx context.Context,
	toolName string,
	input map[string]any,
	permCtx ToolPermissionContext,
) (PermissionResult, error)

// PermissionsConfig holds permission service configuration
type PermissionsConfig struct {
	Mode       options.PermissionMode
	CanUseTool CanUseToolFunc
}

// Service manages tool permissions
type Service struct {
	mode       options.PermissionMode
	canUseTool CanUseToolFunc
}

// NewService creates a new permissions service
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

// CheckToolUse verifies if a tool can be used
// suggestions parameter comes from the control protocol's
// permission_suggestions field
func (s *Service) CheckToolUse(
	ctx context.Context,
	toolName string,
	input map[string]any,
	suggestions []any,
) (any, error) {
	parsedSuggestions := parseSuggestions(suggestions)

	// 1. Check permission mode
	switch s.mode {
	case options.PermissionModeBypassPermissions:
		return handleBypassMode()

	case options.PermissionModeDefault,
		options.PermissionModeAcceptEdits,
		options.PermissionModePlan,
		options.PermissionModeAsk:
		return s.handleStandardMode(ctx, toolName, input, parsedSuggestions)

	default:
		return s.handleUnknownMode()
	}
}

// handleBypassMode returns allow result for bypass permission mode
func handleBypassMode() (any, error) {
	return &PermissionResultAllow{}, nil
}

// handleStandardMode handles standard permission modes with optional callback
func (s *Service) handleStandardMode(
	ctx context.Context,
	toolName string,
	input map[string]any,
	parsedSuggestions []PermissionUpdate,
) (any, error) {
	// Call canUseTool callback if set
	if s.canUseTool != nil {
		permCtx := ToolPermissionContext{
			Suggestions: parsedSuggestions,
		}
		result, err := s.canUseTool(ctx, toolName, input, permCtx)
		if err != nil {
			return nil, fmt.Errorf(
				"permission callback failed: %w",
				err,
			)
		}

		return result, nil
	}

	// Apply default behavior (ask user via CLI)
	// In default mode without callback, we allow but this should be
	// handled by CLI
	return &PermissionResultAllow{}, nil
}

// handleUnknownMode returns deny result for unknown permission modes
func (s *Service) handleUnknownMode() (any, error) {
	return &PermissionResultDeny{
		Message: fmt.Sprintf(
			"unknown permission mode: %s",
			s.mode,
		),
		Interrupt: false,
	}, nil
}

// UpdateMode changes the permission mode
func (s *Service) UpdateMode(mode options.PermissionMode) {
	s.mode = mode
}
