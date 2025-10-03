package permissions

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// Service manages tool permissions.
// This is a DOMAIN service containing only business logic.
type Service struct {
	mode       options.PermissionMode
	canUseTool CanUseToolFunc
}

// NewService creates a new permissions service.
func NewService(config *Config) *Service {
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

// CheckToolUse verifies if a tool can be used.
// suggestions parameter comes from the control protocol's
// permission_suggestions field.
func (s *Service) CheckToolUse(
	ctx context.Context,
	toolName string,
	input map[string]any,
	suggestions []PermissionUpdate,
) (PermissionResult, error) {
	switch s.mode {
	case options.PermissionModeBypassPermissions:
		return s.allowBypass()
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

// UpdateMode changes the permission mode.
func (s *Service) UpdateMode(mode options.PermissionMode) {
	s.mode = mode
}

// allowBypass returns allow result for bypass mode.
//nolint:revive,staticcheck // receiver-naming: method interface requirement
//nolint:revive // receiver-naming: Interface implementation requires receiver
func (*Service) allowBypass() (PermissionResult, error) {
	return &PermissionResultAllow{}, nil
}

// checkWithCallback uses callback if set, otherwise allows.
func (s *Service) checkWithCallback(
	ctx context.Context,
	toolName string,
	input map[string]any,
	suggestions []PermissionUpdate,
) (PermissionResult, error) {
	if s.canUseTool == nil {
		return &PermissionResultAllow{}, nil
	}

	permCtx := ToolPermissionContext{
		Suggestions: suggestions,
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

// denyUnknownMode returns deny result for unknown modes.
func (s *Service) denyUnknownMode() (PermissionResult, error) {
	return &PermissionResultDeny{
		Message: fmt.Sprintf(
			"unknown permission mode: %s",
			s.mode,
		),
		Interrupt: false,
	}, nil
}
