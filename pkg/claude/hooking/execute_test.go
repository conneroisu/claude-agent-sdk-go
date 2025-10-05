package hooking_test

import (
	"context"
	"errors"
	"testing"

	"github.com/conneroisu/claude/pkg/claude/hooking"
)

// TestExecute verifies that the hooking service correctly executes hooks
// based on events and handles various scenarios like blocking decisions
// and error conditions.
func TestExecute(t *testing.T) {
	tests := []struct {
		name    string
		hooks   map[hooking.HookEvent][]hooking.HookMatcher
		event   hooking.HookEvent
		input   map[string]any
		wantRes map[string]any
		wantErr bool
	}{
		{
			name: "matching hook executes",
			hooks: map[hooking.HookEvent][]hooking.HookMatcher{
				hooking.HookEventPreToolUse: {
					{
						Pattern: "*",
						Callback: func(
							_ map[string]any,
							_ *string,
							_ hooking.HookContext,
						) (map[string]any, error) {
							return map[string]any{
								"executed": true,
							}, nil
						},
					},
				},
			},
			event:   hooking.HookEventPreToolUse,
			input:   map[string]any{},
			wantRes: map[string]any{"executed": true},
		},
		{
			name: "non-matching event",
			hooks: map[hooking.HookEvent][]hooking.HookMatcher{
				hooking.HookEventPreToolUse: {
					{
						Pattern: "*",
						Callback: func(
							_ map[string]any,
							_ *string,
							_ hooking.HookContext,
						) (map[string]any, error) {
							return map[string]any{
								"executed": true,
							}, nil
						},
					},
				},
			},
			event:   hooking.HookEventPostToolUse,
			input:   map[string]any{},
			wantRes: nil,
		},
		{
			name: "hook returns error",
			hooks: map[hooking.HookEvent][]hooking.HookMatcher{
				hooking.HookEventPreToolUse: {
					{
						Pattern: "*",
						Callback: func(
							_ map[string]any,
							_ *string,
							_ hooking.HookContext,
						) (map[string]any, error) {
							return nil, errors.New("hook error")
						},
					},
				},
			},
			event:   hooking.HookEventPreToolUse,
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name: "block decision stops execution",
			hooks: map[hooking.HookEvent][]hooking.HookMatcher{
				hooking.HookEventPreToolUse: {
					{
						Pattern: "*",
						Callback: func(
							_ map[string]any,
							_ *string,
							_ hooking.HookContext,
						) (map[string]any, error) {
							return map[string]any{
								"decision": "block",
							}, nil
						},
					},
				},
			},
			event:   hooking.HookEventPreToolUse,
			input:   map[string]any{},
			wantRes: map[string]any{"decision": "block"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := hooking.NewService(tt.hooks)

			result, err := svc.Execute(
				context.Background(),
				tt.event,
				tt.input,
				nil,
				hooking.HookContext{},
			)

			if (err != nil) != tt.wantErr {
				t.Errorf(
					"Execute() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)

				return
			}

			if tt.wantRes == nil && result != nil {
				t.Errorf("Execute() = %v, want nil", result)
			}

			if tt.wantRes != nil && result == nil {
				t.Error("Execute() returned nil, want result")
			}

			if tt.wantRes != nil && result != nil {
				for k, v := range tt.wantRes {
					if result[k] != v {
						t.Errorf(
							"result[%s] = %v, want %v",
							k,
							result[k],
							v,
						)
					}
				}
			}
		})
	}
}
