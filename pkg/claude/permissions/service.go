package permissions

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// Service manages tool permissions.
// It coordinates permission checks, mode management, and callback execution
// to control which tools can be used during agent operation.
type Service struct {
	mode       options.PermissionMode
	canUseTool CanUseToolFunc
}

// NewService creates a new permissions service with the provided configuration.
// If config is nil, it defaults to PermissionModeAsk.
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
// suggestions parameter comes from the control protocol's
// permission_suggestions field.
// Returns either PermissionResultAllow or PermissionResultDeny.
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
		return s.checkWithCallback(ctx, toolName, input, suggestions)

	default:
		return s.handleUnknownMode()
	}
}

// checkWithCallback invokes the user's permission callback if set.
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
			return nil, fmt.Errorf("permission callback failed: %w", err)
		}
		return result, nil
	}

	return &PermissionResultAllow{}, nil
}

// handleUnknownMode returns a denial for unknown permission modes.
func (s *Service) handleUnknownMode() (PermissionResult, error) {
	return &PermissionResultDeny{
		Message:   fmt.Sprintf("unknown permission mode: %s", s.mode),
		Interrupt: false,
	}, nil
}

// UpdateMode changes the permission mode.
// This allows dynamic permission behavior changes during agent operation.
func (s *Service) UpdateMode(mode options.PermissionMode) {
	s.mode = mode
}

// CanUseTool implements the ports.PermissionService interface.
// It adapts the protocol's permission request format to the
// service's CheckToolUse method.
func (s *Service) CanUseTool(
	ctx context.Context,
	req map[string]any,
) (map[string]any, error) {
	toolName, _ := req["tool_name"].(string)
	input, _ := req["input"].(map[string]any)
	suggestions := extractSuggestions(req)

	result, err := s.CheckToolUse(ctx, toolName, input, suggestions)
	if err != nil {
		return nil, err
	}

	return serializeResult(result), nil
}

// extractSuggestions extracts permission suggestions from the request.
func extractSuggestions(req map[string]any) []PermissionUpdate {
	suggestionsRaw, ok := req["permission_suggestions"].([]any)
	if !ok {
		return nil
	}

	suggestions := make([]PermissionUpdate, 0, len(suggestionsRaw))
	for _, s := range suggestionsRaw {
		if suggestionMap, ok := s.(map[string]any); ok {
			suggestions = append(suggestions, deserializeUpdate(suggestionMap))
		}
	}

	return suggestions
}

// deserializeUpdate converts a map to PermissionUpdate.
func deserializeUpdate(_ map[string]any) PermissionUpdate {
	return PermissionUpdate{}
}

// serializeResult converts a PermissionResult to a map for the protocol.
func serializeResult(result PermissionResult) map[string]any {
	switch r := result.(type) {
	case *PermissionResultAllow:
		resp := map[string]any{"allow": true}
		if r.UpdatedInput != nil {
			resp["updated_input"] = r.UpdatedInput
		}
		if r.UpdatedPermissions != nil {
			resp["updated_permissions"] = r.UpdatedPermissions
		}

		return resp
	case *PermissionResultDeny:
		resp := map[string]any{"allow": false, "message": r.Message}
		if r.Interrupt {
			resp["interrupt"] = true
		}

		return resp
	}

	return map[string]any{"allow": true}
}
