package jsonrpc

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
)

// handleHookCallback handles hook_callback control requests.
// It looks up the callback by ID and executes it with the provided input.
func (*Adapter) handleHookCallback(
	ctx context.Context,
	request map[string]any,
	hooks map[string]hooking.HookCallback,
) (map[string]any, error) {
	// Unmarshal request into typed struct.
	var req HookCallbackRequest
	if err := unmarshalRequest(request, &req); err != nil {
		return nil, fmt.Errorf("unmarshal hook_callback request: %w", err)
	}

	// Look up the callback by ID.
	callback, exists := hooks[req.CallbackID]
	if !exists {
		return nil, fmt.Errorf(
			"no hook callback found for ID: %s",
			req.CallbackID,
		)
	}

	// Build hook context with cancellation signal.
	hookCtx := hooking.HookContext{
		Signal: ctx,
	}

	// Execute the callback and return its result.
	result, err := callback(req.Input, req.ToolUseID, hookCtx)
	if err != nil {
		return nil, err
	}

	return result, nil
}
