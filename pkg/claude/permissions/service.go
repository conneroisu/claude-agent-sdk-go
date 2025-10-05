// Package permissions provides tool permission checking services.
package permissions

import (
	"context"
	"sync"
)

// Service manages tool permission checks and mode updates.
type Service struct {
	mu         sync.RWMutex
	mode       PermissionMode
	canUseTool CanUseToolFunc
}

// NewService creates a new permissions service.
func NewService(cfg *PermissionsConfig) *Service {
	if cfg == nil {
		return &Service{
			mode: PermissionModeDefault,
		}
	}

	return &Service{
		mode:       cfg.Mode,
		canUseTool: cfg.CanUseTool,
	}
}

// CheckToolUse checks if a tool can be used based on the current mode
// and user callback.
func (s *Service) CheckToolUse(
	ctx context.Context,
	toolName string,
	input map[string]any,
	permCtx ToolPermissionContext,
) (PermissionResult, error) {
	s.mu.RLock()
	mode := s.mode
	callback := s.canUseTool
	s.mu.RUnlock()

	if mode == PermissionModeBypass {
		return PermissionResultAllow{}, nil
	}

	if callback == nil {
		return PermissionResultAllow{}, nil
	}

	return callback(ctx, toolName, input, permCtx)
}

// UpdateMode changes the current permission mode.
func (s *Service) UpdateMode(mode PermissionMode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = mode
}

// CanUseTool implements ports.PermissionsService.
// This is a simplified adapter that wraps CheckToolUse for the protocol
// layer.
func (s *Service) CanUseTool(
	ctx context.Context,
	toolName string,
	input map[string]any,
) (allowed bool, reason string, err error) {
	permCtx := ToolPermissionContext{}
	result, checkErr := s.CheckToolUse(ctx, toolName, input, permCtx)
	if checkErr != nil {
		return false, "", checkErr
	}

	switch r := result.(type) {
	case PermissionResultAllow:
		return true, "", nil
	case PermissionResultDeny:
		return false, r.Message, nil
	default:
		return false, "unknown result type", nil
	}
}
