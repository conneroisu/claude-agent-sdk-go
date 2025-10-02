// Package hooking provides a hook system for intercepting and responding to
// events in the Claude Agent SDK.
package hooking

import "context"

// Service manages hook execution.
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

	return s.executeMatcherHooks(ctx, matchers, input, toolUseID)
}

// Register adds a new hook.
func (s *Service) Register(event HookEvent, matcher HookMatcher) {
	if s.hooks == nil {
		s.hooks = make(map[HookEvent][]HookMatcher)
	}
	s.hooks[event] = append(s.hooks[event], matcher)
}
