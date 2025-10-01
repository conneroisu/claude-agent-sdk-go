package hooking

import (
	"context"
	"fmt"
)

// HookEvent represents different hook trigger points
type HookEvent string

const (
	HookEventPreToolUse       HookEvent = "PreToolUse"
	HookEventPostToolUse      HookEvent = "PostToolUse"
	HookEventUserPromptSubmit HookEvent = "UserPromptSubmit"
	HookEventStop             HookEvent = "Stop"
	HookEventSubagentStop     HookEvent = "SubagentStop"
	HookEventPreCompact       HookEvent = "PreCompact"
)

// HookContext provides context for hook execution
type HookContext struct {
	// Future: signal support for cancellation
}

// HookCallback is a function that handles hook events
type HookCallback func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error)

// HookMatcher defines when a hook should execute
type HookMatcher struct {
	Matcher string         // Pattern to match (e.g., tool name, event type)
	Hooks   []HookCallback // Callbacks to execute
}

// Service manages hook execution
type Service struct {
	hooks map[HookEvent][]HookMatcher
}

// NewService creates a new hook service
func NewService(hooks map[HookEvent][]HookMatcher) *Service {
	return &Service{
		hooks: hooks,
	}
}

// GetHooks returns the hook configuration
func (s *Service) GetHooks() map[HookEvent][]HookMatcher {
	if s == nil {
		return nil
	}

	return s.hooks
}

// Execute runs hooks for a given event
func (s *Service) Execute(ctx context.Context, event HookEvent, input map[string]any, toolUseID *string) (map[string]any, error) {
	if s == nil || s.hooks == nil {
		return nil, nil
	}

	// Find matching hooks for event
	matchers, exists := s.hooks[event]
	if !exists || len(matchers) == 0 {
		return nil, nil
	}

	// Execute hooks in order and aggregate results
	aggregatedResult := map[string]any{}
	hookCtx := HookContext{}

	for _, matcher := range matchers {
		// Check if matcher applies to this input
		// TODO: Implement pattern matching logic based on matcher.Matcher field

		for _, callback := range matcher.Hooks {
			// Execute hook callback
			result, err := callback(input, toolUseID, hookCtx)
			if err != nil {
				return nil, fmt.Errorf("hook execution failed: %w", err)
			}

			if result == nil {
				continue
			}

			// Handle blocking decisions
			// If hook returns decision="block", stop execution immediately
			if decision, ok := result["decision"].(string); ok && decision == "block" {
				return result, nil
			}

			// Aggregate results (later hooks can override earlier ones)
			for k, v := range result {
				aggregatedResult[k] = v
			}
		}
	}

	return aggregatedResult, nil
}

// Register adds a new hook
func (s *Service) Register(event HookEvent, matcher HookMatcher) {
	if s.hooks == nil {
		s.hooks = make(map[HookEvent][]HookMatcher)
	}
	s.hooks[event] = append(s.hooks[event], matcher)
}
