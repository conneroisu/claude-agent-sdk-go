// Package hooking provides a hook system for intercepting and responding to
// events in the Claude Agent SDK.
package hooking

import (
	"context"
	"fmt"
)

// executionContext holds parameters for hook execution.
type executionContext struct {
	ctx              context.Context
	input            map[string]any
	toolUseID        *string
	hookCtx          HookContext
	aggregatedResult map[string]any
}

// executeMatcherHooks executes all matching hooks and aggregates results.
func (*Service) executeMatcherHooks(
	ctx context.Context,
	matchers []HookMatcher,
	input map[string]any,
	toolUseID *string,
) (map[string]any, error) {
	execCtx := executionContext{
		ctx:              ctx,
		input:            input,
		toolUseID:        toolUseID,
		hookCtx:          HookContext{Signal: ctx},
		aggregatedResult: make(map[string]any),
	}

	for _, matcher := range matchers {
		if !matchesPattern(matcher.Matcher, input) {
			continue
		}

		result, err := executeCallbacks(matcher.Hooks, &execCtx)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil // Blocking decision
		}
	}

	return execCtx.aggregatedResult, nil
}

// executeCallbacks runs hook callbacks and handles blocking decisions.
func executeCallbacks(
	callbacks []HookCallback,
	execCtx *executionContext,
) (map[string]any, error) {
	for _, callback := range callbacks {
		if err := checkContextCancellation(execCtx.ctx); err != nil {
			return nil, err
		}

		result, err := executeCallback(callback, execCtx)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil
		}
	}

	return nil, nil
}

// checkContextCancellation checks if context is cancelled.
func checkContextCancellation(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// executeCallback runs a single hook callback.
func executeCallback(
	callback HookCallback,
	execCtx *executionContext,
) (map[string]any, error) {
	result, err := callback(
		execCtx.input,
		execCtx.toolUseID,
		execCtx.hookCtx,
	)
	if err != nil {
		return nil, fmt.Errorf("hook execution failed: %w", err)
	}
	if result == nil {
		return nil, nil
	}

	return handleHookResult(result, execCtx.aggregatedResult), nil
}

// handleHookResult processes hook result and checks for blocking decisions.
func handleHookResult(
	result, aggregatedResult map[string]any,
) map[string]any {
	if decision, ok := result["decision"].(string); ok {
		if decision == "block" {
			return result
		}
	}

	for k, v := range result {
		aggregatedResult[k] = v
	}

	return nil
}

// matchesPattern checks if a hook matcher pattern applies to the given input.
func matchesPattern(pattern string, input map[string]any) bool {
	// Empty matcher matches all events
	if pattern == "" {
		return true
	}

	// Wildcard matches all
	if pattern == "*" {
		return true
	}

	// For PreToolUse/PostToolUse hooks, match against tool_name
	if toolName, ok := input["tool_name"].(string); ok {
		// Exact match
		if pattern == toolName {
			return true
		}
	}

	// Pattern doesn't match
	return false
}
