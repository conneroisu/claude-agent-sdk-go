// Package querying provides hook callback building and registration.
package querying

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// buildHookCallbacks creates hook callbacks map from hooking service.
// It iterates through all registered hooks and converts them to the
// port-compatible callback format.
func (s *Service) buildHookCallbacks() map[string]ports.HookCallback {
	if s.hooks == nil {
		return nil
	}
	hookCallbacks := make(map[string]ports.HookCallback)
	hooks := s.hooks.GetHooks()
	for event, matchers := range hooks {
		s.registerMatcherHooks(
			hookCallbacks,
			event,
			matchers,
		)
	}

	return hookCallbacks
}

// registerMatcherHooks registers all hooks for a set of matchers.
// It creates callback wrappers that adapt the hook signature to match
// the port interface requirements.
func (*Service) registerMatcherHooks(
	callbacks map[string]ports.HookCallback,
	event hooking.HookEvent,
	matchers []hooking.HookMatcher,
) {
	for _, matcher := range matchers {
		for i, callback := range matcher.Hooks {
			callbackID := fmt.Sprintf(
				"hook_%s_%d",
				string(event),
				i,
			)
			cb := callback
			callbacks[callbackID] = func(
				input map[string]any,
				toolUseID *string,
				ctx any,
			) (map[string]any, error) {
				hookCtx := hooking.HookContext{
					Signal: context.Background(),
				}
				if hc, ok := ctx.(hooking.HookContext); ok {
					hookCtx = hc
				} else if c, ok := ctx.(context.Context); ok {
					hookCtx = hooking.HookContext{Signal: c}
				}

				return cb(input, toolUseID, hookCtx)
			}
		}
	}
}
