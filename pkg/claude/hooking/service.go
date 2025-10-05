// Package hooking provides hook callback execution services.
package hooking

import (
	"context"
	"fmt"
	"sync"
)

// Service orchestrates hook callback execution based on event matchers.
// It manages hook registration, pattern matching, and callback execution.
type Service struct {
	mu              sync.RWMutex
	callbackCounter int
	callbacks       map[string]HookCallback // Maps callback IDs to functions
	hooks           map[HookEvent][]HookMatcher
}

// NewService creates a new hooking service.
// The hooks map defines which callbacks to execute for each event type.
func NewService(
	hooks map[HookEvent][]HookMatcher,
) *Service {
	return &Service{
		callbacks: make(map[string]HookCallback),
		hooks:     hooks,
	}
}

// GetHooks returns the hook configuration map for initialization.
// It generates unique callback IDs and builds the event mapping for the
// protocol. Returns nil if service or hooks are not configured.
func (s *Service) GetHooks() map[string]string {
	if s == nil || s.hooks == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build callback ID to event type mapping
	result := make(map[string]string)
	for event, matchers := range s.hooks {
		for _, matcher := range matchers {
			// Generate unique callback ID
			s.callbackCounter++
			callbackID := fmt.Sprintf("hook_%d", s.callbackCounter)
			s.callbacks[callbackID] = matcher.Callback
			result[callbackID] = string(event)
		}
	}

	return result
}

// GetCallbacks returns the registered callback map for protocol handlers.
// This enables the protocol layer to invoke callbacks by ID.
func (s *Service) GetCallbacks() map[string]HookCallback {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]HookCallback, len(s.callbacks))
	for id, cb := range s.callbacks {
		result[id] = cb
	}

	return result
}

// Execute runs matching hooks for an event and returns aggregated results.
//
//nolint:revive // 5 params acceptable for hook execution
func (s *Service) Execute(
	ctx context.Context,
	event HookEvent,
	input map[string]any,
	toolUseID *string,
	hookCtx HookContext,
) (map[string]any, error) {
	if s == nil {
		return nil, nil
	}

	s.mu.RLock()
	matchers, exists := s.hooks[event]
	s.mu.RUnlock()

	if !exists || len(matchers) == 0 {
		return nil, nil
	}

	result := make(map[string]any)

	for _, matcher := range matchers {
		if !s.matches(matcher.Pattern, input) {
			continue
		}

		output, err := s.executeCallback(
			ctx,
			matcher.Callback,
			input,
			toolUseID,
			hookCtx,
		)
		if err != nil {
			return nil, err
		}

		decision, ok := output["decision"].(string)
		if ok && decision == "block" {
			return output, nil
		}

		s.mergeResults(result, output)
	}

	return result, nil
}
