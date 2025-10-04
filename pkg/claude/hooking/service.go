// Package hooking provides hook execution and lifecycle management.
package hooking

import (
	"context"
	"fmt"
)

// Service manages hook execution.
type Service struct {
	hooks map[HookEvent][]HookMatcher
}

// New creates a new hooking service.
func New() *Service {
	return &Service{
		hooks: make(map[HookEvent][]HookMatcher),
	}
}

// NewService creates a new hooking service with initial hooks.
func NewService(hooks map[HookEvent][]HookMatcher) *Service {
	return &Service{
		hooks: hooks,
	}
}

// GetHooks returns the hook configuration.
func (s *Service) GetHooks() map[HookEvent][]HookMatcher {
	if s == nil {
		return nil
	}

	return s.hooks
}

// Execute runs hooks for a given event.
func (s *Service) Execute(
	ctx context.Context,
	event HookEvent,
	input map[string]any,
	toolUseID *string,
) (map[string]any, error) {
	if s == nil || s.hooks == nil {
		return nil, nil
	}

	matchers, exists := s.hooks[event]
	if !exists || len(matchers) == 0 {
		return nil, nil
	}

	return s.executeMatchers(ctx, matchers, input, toolUseID)
}

// Register adds a new hook.
func (s *Service) Register(event HookEvent, matcher HookMatcher) {
	if s.hooks == nil {
		s.hooks = make(map[HookEvent][]HookMatcher)
	}
	s.hooks[event] = append(s.hooks[event], matcher)
}

// RegisterHook adds a hook handler for a specific event.
// This is a simplified interface for common use cases.
func (s *Service) RegisterHook(
	event string,
	handler func(context.Context, map[string]any) (map[string]any, error),
) {
	hookEvent := HookEvent(event)
	matcher := HookMatcher{
		Matcher: "*",
		Hooks: []HookCallback{
			func(
				input map[string]any,
				_ *string,
				ctx HookContext,
			) (map[string]any, error) {
				return handler(ctx.Signal, input)
			},
		},
	}
	s.Register(hookEvent, matcher)
}

// executeMatchers runs matching hooks and aggregates results.
func (s *Service) executeMatchers(
	ctx context.Context,
	matchers []HookMatcher,
	input map[string]any,
	toolUseID *string,
) (map[string]any, error) {
	aggregatedResult := make(map[string]any)
	hookCtx := HookContext{Signal: ctx}

	for _, matcher := range matchers {
		if !s.matchesPattern(matcher.Matcher, input) {
			continue
		}

		result, err := s.executeCallbacks(callbackExecParams{
			ctx:       ctx,
			callbacks: matcher.Hooks,
			input:     input,
			toolUseID: toolUseID,
			hookCtx:   hookCtx,
		})
		if err != nil {
			return nil, err
		}

		if shouldStopExecution(result) {
			return result, nil
		}

		mergeResults(aggregatedResult, result)
	}

	return aggregatedResult, nil
}

// callbackExecParams contains parameters for callback execution.
// This struct reduces the argument count to comply with linting rules.
type callbackExecParams struct {
	ctx       context.Context
	callbacks []HookCallback
	input     map[string]any
	toolUseID *string
	hookCtx   HookContext
}

// executeCallbacks runs hook callbacks sequentially.
// Returns the last non-nil result or nil if all callbacks return nil.
func (s *Service) executeCallbacks(
	params callbackExecParams,
) (map[string]any, error) {
	_ = s // unused but required for method consistency
	ctx := params.ctx
	callbacks := params.callbacks
	var lastResult map[string]any

	for _, callback := range callbacks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result, err := callback(
			params.input,
			params.toolUseID,
			params.hookCtx,
		)
		if err != nil {
			return nil, fmt.Errorf("hook execution failed: %w", err)
		}

		if result == nil {
			continue
		}

		lastResult = result
		if shouldStopExecution(result) {
			return result, nil
		}
	}

	return lastResult, nil
}

// shouldStopExecution checks if a hook result requests stopping execution.
// Returns true if result contains decision="block".
func shouldStopExecution(result map[string]any) bool {
	if result == nil {
		return false
	}
	decision, ok := result["decision"].(string)

	return ok && decision == "block"
}

// mergeResults copies all key-value pairs from src to dest.
// Existing keys in dest are overwritten by values from src.
func mergeResults(dest, src map[string]any) {
	if src == nil {
		return
	}
	for k, v := range src {
		dest[k] = v
	}
}

// matchesPattern checks if the hook pattern matches the input.
// Supports exact tool name matching and wildcard "*" for all tools.
func (s *Service) matchesPattern(pattern string, input map[string]any) bool {
	_ = s // unused but required for method consistency
	if pattern == "" || pattern == "*" {
		return true
	}

	toolName, ok := input["tool_name"].(string)
	if !ok {
		return false
	}

	return pattern == toolName
}
