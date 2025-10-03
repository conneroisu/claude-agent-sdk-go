package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/permissions"
)

// handleCanUseTool handles can_use_tool control requests.
// It checks if a tool can be used by calling the permissions service.
func (a *Adapter) handleCanUseTool(
	ctx context.Context,
	request map[string]any,
	perms *permissions.Service,
) (map[string]any, error) {
	// Unmarshal request into typed struct.
	var req CanUseToolRequest
	if err := unmarshalRequest(request, &req); err != nil {
		return nil, fmt.Errorf("unmarshal can_use_tool request: %w", err)
	}

	// Check if permissions service is available.
	if perms == nil {
		return nil, errors.New(
			"permissions callback not provided",
		)
	}

	// Check permission for the tool use.
	result, err := perms.CheckToolUse(
		ctx,
		req.ToolName,
		req.Input,
		req.PermissionSuggestions,
	)
	if err != nil {
		return nil, err
	}

	return a.formatPermissionResult(result), nil
}

// unmarshalRequest unmarshals a map[string]any into a typed struct.
func unmarshalRequest(request map[string]any, target any) error {
	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal request: %w", err)
	}

	return nil
}

// formatPermissionResult converts PermissionResult to response map.
// It handles both allow and deny results with appropriate formatting.
//nolint:revive // unused-receiver: method signature required
func (a *Adapter) formatPermissionResult(
	result permissions.PermissionResult,
) map[string]any {
	switch r := result.(type) {
	case *permissions.PermissionResultAllow:
		// Allow result with optional updated input/permissions.
		response := map[string]any{"allow": true}
		if r.UpdatedInput != nil {
			response["input"] = r.UpdatedInput
		}
		if len(r.UpdatedPermissions) > 0 {
			response["updated_permissions"] = r.UpdatedPermissions
		}

		return response

	case *permissions.PermissionResultDeny:
		// Deny result with reason message.
		return map[string]any{
			"allow":  false,
			"reason": r.Message,
		}

	default:
		// Unknown result type defaults to deny.
		return map[string]any{"allow": false}
	}
}

