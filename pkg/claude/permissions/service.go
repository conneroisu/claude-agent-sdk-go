// Package permissions handles tool permission checks and updates.
package permissions

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// CanUseToolFunc is a callback for permission checks.
// input is intentionally map[string]any as tool inputs vary by tool.
type CanUseToolFunc func(
	ctx context.Context,
	toolName string,
	input map[string]any,
	permCtx ToolPermissionContext,
) (PermissionResult, error)

// Config holds permission service configuration.
type Config struct {
	// Mode sets the permission handling mode
	Mode options.PermissionMode

	// CanUseTool is the permission check callback
	CanUseTool CanUseToolFunc
}

// Service manages tool permissions.
type Service struct {
	mode       options.PermissionMode
	canUseTool CanUseToolFunc
}

// New creates a new permissions service with default mode.
func New(mode *options.PermissionMode) *Service {
	effectiveMode := mode
	if effectiveMode == nil {
		defaultMode := options.PermissionModeAsk
		effectiveMode = &defaultMode
	}

	return &Service{
		mode: *effectiveMode,
	}
}

// NewService creates a new permissions service.
func NewService(config *Config) *Service {
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
// suggestions parameter comes from control protocol permission_suggestions.
func (s *Service) CheckToolUse(
	ctx context.Context,
	toolName string,
	input map[string]any,
	suggestions []PermissionUpdate,
) (PermissionResult, error) {
	switch s.mode {
	case options.PermissionModeBypassPermissions:
		return &PermissionResultAllow{}, nil

	case options.PermissionModeDefault,
		options.PermissionModeAcceptEdits,
		options.PermissionModePlan,
		options.PermissionModeAsk:
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

// SetCallback sets the permission callback function.
func (s *Service) SetCallback(callback CanUseToolFunc) {
	s.canUseTool = callback
}

// checkWithCallback calls the permission callback or uses default.
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
			return nil, fmt.Errorf(
				"permission callback failed: %w",
				err,
			)
		}

		return result, nil
	}

	return &PermissionResultAllow{}, nil
}

// denyUnknownMode denies access for unknown permission modes.
func (s *Service) denyUnknownMode() (PermissionResult, error) {
	return &PermissionResultDeny{
		Message:   fmt.Sprintf("unknown permission mode: %s", s.mode),
		Interrupt: false,
	}, nil
}
