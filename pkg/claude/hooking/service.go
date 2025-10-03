package hooking

import (
	"context"
	"fmt"
)

// Service manages hook execution.
// This is a DOMAIN service containing only business logic.
type Service struct {
	hooks map[HookEvent][]HookMatcher
}

// NewService creates a new hooking service.
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
func (s *Service) Register(
	event HookEvent,
	matcher HookMatcher,
) {
	if s.hooks == nil {
		s.hooks = make(map[HookEvent][]HookMatcher)
	}
	s.hooks[event] = append(s.hooks[event], matcher)
}

// executeMatchers runs matching hooks and aggregates results.
func (s *Service) executeMatchers(
	ctx context.Context,
	matchers []HookMatcher,
	input map[string]any,
	toolUseID *string,
) (map[string]any, error) {
	aggregatedResult := make(map[string]any)
	hookCtx := HookContext{
		Signal: ctx,
	}

	for _, matcher := range matchers {
		if !matchesPattern(matcher.Matcher, input) {
			continue
		}

		result, err := s.executeCallbacks(
			ctx,
			matcher.Hooks,
			input,
			toolUseID,
			hookCtx,
		)
		if err != nil {
			return nil, err
		}

		if result == nil {
			continue
		}

		if isBlockingDecision(result) {
			return result, nil
		}

		aggregateResults(aggregatedResult, result)
	}

	return aggregatedResult, nil
}

// executeCallbacks runs a list of callbacks.
//nolint:revive // unused-receiver: method signature required
func (s *Service) executeCallbacks(
	ctx context.Context,
	callbacks []HookCallback,
	input map[string]any,
	toolUseID *string,
	hookCtx HookContext,
) (map[string]any, error) {
	var lastResult map[string]any

	for _, callback := range callbacks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result, err := callback(input, toolUseID, hookCtx)
		if err != nil {
			return nil, fmt.Errorf(
				"hook execution failed: %w",
				err,
			)
		}

		if result != nil {
			lastResult = result
		}
	}

	return lastResult, nil
}

// matchesPattern checks if a hook matcher pattern applies.
func matchesPattern(
	pattern string,
	input map[string]any,
) bool {
	if pattern == "" || pattern == "*" {
		return true
	}

	if toolName, ok := input["tool_name"].(string); ok {
		if pattern == toolName {
			return true
		}
	}

	return false
}

// isBlockingDecision checks if result is a blocking decision.
func isBlockingDecision(result map[string]any) bool {
	decision, ok := result["decision"].(string)

	return ok && decision == "block"
}

// aggregateResults merges result into aggregated map.
func aggregateResults(
	aggregated,
	result map[string]any,
) {
	for k, v := range result {
		aggregated[k] = v
	}
}
