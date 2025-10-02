// Package jsonrpc provides control request handlers for tool use permissions
// and hook callbacks. These handlers validate and process requests from the
// Claude agent runtime.
package jsonrpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// handleCanUseTool handles can_use_tool control requests
func handleCanUseTool(
	ctx context.Context,
	request map[string]any,
	perms ports.PermissionChecker,
) (map[string]any, error) {
	if perms == nil {
		return nil, errors.New("permissions callback not provided")
	}

	toolName, input := extractToolUseParams(request)
	suggestions := extractPermissionSuggestions(request)

	result, err := perms.CheckToolUse(ctx, toolName, input, suggestions)
	if err != nil {
		return nil, err
	}

	return buildPermissionResponse(result)
}

// extractToolUseParams extracts tool name and input from the request
func extractToolUseParams(request map[string]any) (string, map[string]any) {
	toolName, _ := request[msgFieldToolName].(string)
	input, _ := request[msgFieldInput].(map[string]any)

	return toolName, input
}

// extractPermissionSuggestions parses permission suggestions from the request
func extractPermissionSuggestions(request map[string]any) []any {
	suggestions, ok := request[msgFieldPermissionSuggests].([]any)
	if !ok {
		return nil
	}

	return suggestions
}

// buildPermissionResponse converts PermissionResult to response format
func buildPermissionResponse(
	result any,
) (map[string]any, error) {
	switch r := result.(type) {
	case *permissions.PermissionResultAllow:
		return buildAllowResponse(r), nil
	case *permissions.PermissionResultDeny:
		return buildDenyResponse(r), nil
	default:
		return nil, errors.New("unknown permission result type")
	}
}

// buildAllowResponse creates a response for allowed permissions
func buildAllowResponse(r *permissions.PermissionResultAllow) map[string]any {
	response := map[string]any{msgFieldAllow: true}

	if r.UpdatedInput != nil {
		response[msgFieldInput] = r.UpdatedInput
	}

	if len(r.UpdatedPermissions) > 0 {
		response[msgFieldUpdatedPerms] = r.UpdatedPermissions
	}

	return response
}

// buildDenyResponse creates a response for denied permissions
func buildDenyResponse(r *permissions.PermissionResultDeny) map[string]any {
	return map[string]any{
		msgFieldAllow:  false,
		msgFieldReason: r.Message,
	}
}

// handleHookCallback handles hook_callback control requests
func handleHookCallback(
	ctx context.Context,
	request map[string]any,
	hooks map[string]ports.HookCallback,
) (map[string]any, error) {
	callbackID, _ := request[msgFieldCallbackID].(string)
	input, _ := request[msgFieldInput].(map[string]any)
	toolUseID, _ := request[msgFieldToolUseID].(*string)

	callback, exists := hooks[callbackID]
	if !exists {
		return nil, fmt.Errorf("no hook callback found for ID: %s", callbackID)
	}

	// Execute callback with context for cancellation support
	result, err := callback(input, toolUseID, ctx)
	if err != nil {
		return nil, err
	}

	return result, nil
}
