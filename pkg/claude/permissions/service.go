package permissions

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// PermissionResult represents the outcome of a permission check
type PermissionResult interface {
	IsAllowed() bool
	GetUpdatedInput() map[string]any
	GetDenyMessage() string
}

// PermissionResultAllow indicates tool use is allowed
type PermissionResultAllow struct {
	UpdatedInput       map[string]any
	UpdatedPermissions []PermissionUpdate
}

func (p *PermissionResultAllow) IsAllowed() bool           { return true }
func (p *PermissionResultAllow) GetUpdatedInput() map[string]any { return p.UpdatedInput }
func (p *PermissionResultAllow) GetDenyMessage() string    { return "" }

// PermissionResultDeny indicates tool use is denied
type PermissionResultDeny struct {
	Message   string
	Interrupt bool
}

func (p *PermissionResultDeny) IsAllowed() bool           { return false }
func (p *PermissionResultDeny) GetUpdatedInput() map[string]any { return nil }
func (p *PermissionResultDeny) GetDenyMessage() string    { return p.Message }

// PermissionUpdate represents a permission change
type PermissionUpdate struct {
	Type        string
	Rules       []PermissionRuleValue
	Behavior    *PermissionBehavior
	Mode        *options.PermissionMode
	Directories []string
	Destination *PermissionUpdateDestination
}

// PermissionRuleValue represents a permission rule
type PermissionRuleValue struct {
	ToolName    string
	RuleContent *string
}

// PermissionBehavior defines permission behavior
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

// PermissionUpdateDestination defines where permission updates are stored
type PermissionUpdateDestination string

const (
	PermissionDestinationUserSettings    PermissionUpdateDestination = "userSettings"
	PermissionDestinationProjectSettings PermissionUpdateDestination = "projectSettings"
	PermissionDestinationLocalSettings   PermissionUpdateDestination = "localSettings"
	PermissionDestinationSession         PermissionUpdateDestination = "session"
)

// ToolPermissionContext provides context for permission decisions
type ToolPermissionContext struct {
	Suggestions []PermissionUpdate
}

// CanUseToolFunc is a callback for permission checks
type CanUseToolFunc func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error)

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
			mode: options.PermissionModeDefault,
		}
	}

	return &Service{
		mode:       config.Mode,
		canUseTool: config.CanUseTool,
	}
}

// CheckToolUse verifies if a tool can be used
func (s *Service) CheckToolUse(ctx context.Context, toolName string, input map[string]any) (PermissionResult, error) {
	// Check permission mode
	switch s.mode {
	case options.PermissionModeBypassPermissions:
		// Always allow
		return &PermissionResultAllow{}, nil

	case options.PermissionModeDefault, options.PermissionModeAcceptEdits, options.PermissionModePlan:
		// Call canUseTool callback if set
		if s.canUseTool != nil {
			permCtx := ToolPermissionContext{
				// TODO: Extract suggestions from control request if available
				Suggestions: []PermissionUpdate{},
			}
			result, err := s.canUseTool(ctx, toolName, input, permCtx)
			if err != nil {
				return nil, fmt.Errorf("permission callback failed: %w", err)
			}

			return result, nil
		}

		// Apply default behavior (ask user via CLI)
		// In default mode without callback, we allow but this should be handled by CLI
		return &PermissionResultAllow{}, nil

	default:
		// Unknown mode - deny for safety
		return &PermissionResultDeny{
			Message:   fmt.Sprintf("unknown permission mode: %s", s.mode),
			Interrupt: false,
		}, nil
	}
}

// UpdateMode changes the permission mode
func (s *Service) UpdateMode(mode options.PermissionMode) {
	s.mode = mode
}
