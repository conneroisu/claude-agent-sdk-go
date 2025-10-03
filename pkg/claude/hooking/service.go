// Hook service implementation.
package hooking

import (
	"context"
	"fmt"
)

// Service manages hook execution.
//
// This is a domain service - pure business logic for hook management.
type Service struct {
	hooks map[HookEvent][]HookMatcher
}

// NewService creates a new hook service.
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
//
// Executes all matching hooks in order and aggregates results.
// Returns immediately if any hook returns decision="block".
func (s *Service) Execute(
	ctx context.Context,
	event HookEvent,
	input map[string]any,
	toolUseID *string,
) (map[string]any, error) {
	if s == nil || s.hooks == nil {
		return nil, nil
	}

	// Find matching hooks for event
	matchers, exists := s.hooks[event]
	if !exists || len(matchers) == 0 {
		return nil, nil
	}

	// Execute hooks and aggregate results
	return s.executeMatchers(ctx, matchers, input, toolUseID)
}

// executeMatchers runs all matching hooks and aggregates results.
func (s *Service) executeMatchers(
	ctx context.Context,
	matchers []HookMatcher,
	input map[string]any,
	toolUseID *string,
) (map[string]any, error) {
	aggregatedResult := map[string]any{}
	hookCtx := HookContext{Signal: ctx}

	for _, matcher := range matchers {
		if !s.matchesPattern(matcher.Matcher, input) {
			continue
		}

		result, err := s.executeCallbacks(
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
		if result != nil {
			return result, nil // Blocking decision
		}
	}

	return aggregatedResult, nil
}

// executeCallbacks runs hook callbacks and handles blocking.
func (s *Service) executeCallbacks(
	ctx context.Context,
	callbacks []HookCallback,
	input map[string]any,
	toolUseID *string,
	hookCtx HookContext,
	aggregated map[string]any,
) (map[string]any, error) {
	for _, callback := range callbacks {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Execute callback
		result, err := callback(input, toolUseID, hookCtx)
		if err != nil {
			return nil, fmt.Errorf("hook execution failed: %w", err)
		}
		if result == nil {
			continue
		}

		// Check for blocking decision
		if decision, ok := result["decision"].(string); ok {
			if decision == "block" {
				return result, nil
			}
		}

		// Aggregate results
		for k, v := range result {
			aggregated[k] = v
		}
	}

	return nil, nil // No blocking decision
}

// Register adds a new hook.
func (s *Service) Register(event HookEvent, matcher HookMatcher) {
	if s.hooks == nil {
		s.hooks = make(map[HookEvent][]HookMatcher)
	}
	s.hooks[event] = append(s.hooks[event], matcher)
}

// matchesPattern checks if a hook matcher pattern applies to input.
func (s *Service) matchesPattern(
	pattern string,
	input map[string]any,
) bool {
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
