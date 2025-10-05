package hooking_test

import (
	"testing"

	"github.com/conneroisu/claude/pkg/claude/hooking"
)

// TestGetCallbacks verifies that the hooking service correctly returns
// the callback functions that have been registered with it.
func TestGetCallbacks(t *testing.T) {
	callback := func(
		_ map[string]any,
		_ *string,
		_ hooking.HookContext,
	) (map[string]any, error) {
		return nil, nil
	}

	hooks := map[hooking.HookEvent][]hooking.HookMatcher{
		hooking.HookEventPreToolUse: {
			{Pattern: "*", Callback: callback},
		},
	}

	svc := hooking.NewService(hooks)
	_ = svc.GetHooks()

	callbacks := svc.GetCallbacks()

	if len(callbacks) == 0 {
		t.Error("expected callbacks, got none")
	}
}
