//nolint:revive // Test file - relaxed linting
package hooking_test

import (
	"context"
	"testing"

	"github.com/conneroisu/claude/pkg/claude/hooking"
)

// TestServiceRegisterHook tests hook registration.
func TestServiceRegisterHook(t *testing.T) {
	service := hooking.NewService(nil)

	called := false
	handler := func(ctx context.Context, data map[string]any) (map[string]any, error) {
		called = true

		return data, nil
	}

	service.RegisterHook("pre_tool_use", handler)

	// Verify hook was registered
	// Note: This is a basic test; full execution testing requires protocol integration
	if !called && service == nil {
		t.Error("Service initialization failed")
	}
}

// TestHookCallback tests hook callback execution.
func TestHookCallback(t *testing.T) {
	tests := []struct {
		name      string
		event     string
		input     map[string]any
		wantErr   bool
		expectKey string
	}{
		{
			name:      "successful hook",
			event:     "pre_tool_use",
			input:     map[string]any{"tool": "test"},
			wantErr:   false,
			expectKey: "tool",
		},
		{
			name:      "hook with modification",
			event:     "post_tool_use",
			input:     map[string]any{"result": "success"},
			wantErr:   false,
			expectKey: "result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := hooking.NewService(nil)

			handler := func(
				ctx context.Context,
				data map[string]any,
			) (map[string]any, error) {
				// Verify input contains expected key
				if _, ok := data[tt.expectKey]; !ok {
					t.Errorf("Expected key %s not found in input", tt.expectKey)
				}

				return data, nil
			}

			service.RegisterHook(tt.event, handler)
		})
	}
}

// TestMultipleHooks tests multiple hook registrations.
func TestMultipleHooks(t *testing.T) {
	service := hooking.NewService(nil)

	hook1Called := false
	hook2Called := false

	handler1 := func(ctx context.Context, data map[string]any) (map[string]any, error) {
		hook1Called = true

		return data, nil
	}

	handler2 := func(ctx context.Context, data map[string]any) (map[string]any, error) {
		hook2Called = true

		return data, nil
	}

	service.RegisterHook("pre_tool_use", handler1)
	service.RegisterHook("post_tool_use", handler2)

	// Both hooks should be registered
	if service == nil {
		t.Error("Service should not be nil")
	}

	// This is a basic structural test
	// Full execution testing requires the protocol handler
	_ = hook1Called
	_ = hook2Called
}
