package hooking

import (
	"context"
	"fmt"
)

// Service manages hook execution and lifecycle.
// It coordinates hook registration, pattern matching, and callback execution
// for all hook events throughout the agent's operation.
type Service struct {
	hooks map[HookEvent][]HookMatcher
}

// NewService creates a new hooking service with provided configuration.
// The hooks map defines which callbacks execute for each event type.
func NewService(hooks map[HookEvent][]HookMatcher) *Service {
	return &Service{
		hooks: hooks,
	}
}

// GetHooks returns the hook configuration.
// Returns nil if the service is nil, allowing safe access.
func (s *Service) GetHooks() map[HookEvent][]HookMatcher {
	if s == nil {
		return nil
	}
	return s.hooks
}

// Execute runs hooks for a given event.
// It finds matching hooks, executes them in order, and aggregates results.
// If any hook returns decision="block", execution stops immediately.
// Returns nil result if no hooks match or service is nil.
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

// executeMatchers runs all matching hook callbacks and aggregates results.
func (s *Service) executeMatchers(
	ctx context.Context,
	matchers []HookMatcher,
	input map[string]any,
	toolUseID *string,
) (map[string]any, error) {
	aggregatedResult := map[string]any{}
	hookCtx := HookContext{
		Signal: ctx,
	}

	for _, matcher := range matchers {
		if !s.matchesPattern(matcher.Matcher, input) {
			continue
		}

		blockResult, blocked, err := s.executeCallbacks(
			ctx,
			matcher.Hooks,
			input,
			toolUseID,
			hookCtx,
			aggregatedResult,
		)
		if err != nil {
			return nil, err
		}
		if blocked {
			return blockResult, nil
		}
	}

	return aggregatedResult, nil
}

// executeCallbacks runs all callbacks for a matcher.
func (s *Service) executeCallbacks(
	ctx context.Context,
	callbacks []HookCallback,
	input map[string]any,
	toolUseID *string,
	hookCtx HookContext,
	aggregatedResult map[string]any,
) (map[string]any, bool, error) {
	for _, callback := range callbacks {
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		default:
		}

		result, err := callback(input, toolUseID, hookCtx)
		if err != nil {
			return nil, false, fmt.Errorf("hook execution failed: %w", err)
		}

		if result == nil {
			continue
		}

		if decision, ok := result["decision"].(string); ok {
			if decision == "block" {
				return result, true, nil
			}
		}

		for k, v := range result {
			aggregatedResult[k] = v
		}
	}

	return nil, false, nil
}

// Register adds a new hook to the service.
// It initializes the hooks map if needed.
func (s *Service) Register(event HookEvent, matcher HookMatcher) {
	if s.hooks == nil {
		s.hooks = make(map[HookEvent][]HookMatcher)
	}
	s.hooks[event] = append(s.hooks[event], matcher)
}

// matchesPattern checks if a hook matcher pattern applies to the given input.
// Empty string and "*" match all events.
// For tool-related events, it matches against the tool_name field.
func (*Service) matchesPattern(pattern string, input map[string]any) bool {
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
