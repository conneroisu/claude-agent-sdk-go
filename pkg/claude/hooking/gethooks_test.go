package hooking_test

import (
	"testing"

	"github.com/conneroisu/claude/pkg/claude/hooking"
)

// TestGetHooks verifies that the hooking service correctly retrieves
// and returns the hooks that have been registered with it.
func TestGetHooks(t *testing.T) {
	tests := []struct {
		name      string
		hooks     map[hooking.HookEvent][]hooking.HookMatcher
		wantCount int
	}{
		{
			name: "single hook",
			hooks: map[hooking.HookEvent][]hooking.HookMatcher{
				hooking.HookEventPreToolUse: {
					{
						Pattern: "*",
						Callback: func(
							_ map[string]any,
							_ *string,
							_ hooking.HookContext,
						) (map[string]any, error) {
							return map[string]any{}, nil
						},
					},
				},
			},
			wantCount: 1,
		},
		{
			name: "multiple hooks",
			hooks: map[hooking.HookEvent][]hooking.HookMatcher{
				hooking.HookEventPreToolUse: {
					{
						Pattern: "*",
						Callback: func(
							_ map[string]any,
							_ *string,
							_ hooking.HookContext,
						) (map[string]any, error) {
							return nil, nil
						},
					},
				},
				hooking.HookEventPostToolUse: {
					{
						Pattern: "*",
						Callback: func(
							_ map[string]any,
							_ *string,
							_ hooking.HookContext,
						) (map[string]any, error) {
							return nil, nil
						},
					},
				},
			},
			wantCount: 2,
		},
		{
			name:      "nil hooks",
			hooks:     nil,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := hooking.NewService(tt.hooks)
			result := svc.GetHooks()

			if tt.wantCount == 0 {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}

				return
			}

			if len(result) != tt.wantCount {
				t.Errorf(
					"hook count = %d, want %d",
					len(result),
					tt.wantCount,
				)
			}
		})
	}
}
