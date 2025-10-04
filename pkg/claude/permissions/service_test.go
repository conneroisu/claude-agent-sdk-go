//nolint:revive // Test file - relaxed linting
package permissions_test

import (
	"context"
	"testing"

	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
)

// TestServiceCheckToolUse tests permission checking.
func TestServiceCheckToolUse(t *testing.T) {
	tests := []struct {
		name       string
		mode       options.PermissionMode
		toolName   string
		expectType string
	}{
		{
			name:       "bypass mode permits all",
			mode:       options.PermissionModeBypassPermissions,
			toolName:   "test_tool",
			expectType: "allow",
		},
		{
			name:       "default mode checks",
			mode:       options.PermissionModeDefault,
			toolName:   "test_tool",
			expectType: "allow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := tt.mode
			service := permissions.NewService(&permissions.Config{
				Mode: mode,
			})

			result, err := service.CheckToolUse(
				context.Background(),
				tt.toolName,
				map[string]any{},
				[]permissions.PermissionUpdate{},
			)

			if err != nil {
				t.Fatalf("CheckToolUse() error = %v", err)
			}

			switch tt.expectType {
			case "allow":
				if _, ok := result.(*permissions.PermissionResultAllow); !ok {
					t.Errorf("Expected PermissionResultAllow, got %T", result)
				}
			case "deny":
				if _, ok := result.(*permissions.PermissionResultDeny); !ok {
					t.Errorf("Expected PermissionResultDeny, got %T", result)
				}
			}
		})
	}
}

// TestServiceWithCallback tests custom callback execution.
func TestServiceWithCallback(t *testing.T) {
	callbackCalled := false
	mode := options.PermissionModeAsk

	service := permissions.NewService(&permissions.Config{
		Mode: mode,
	})

	callback := func(
		ctx context.Context,
		toolName string,
		input map[string]any,
		permCtx permissions.ToolPermissionContext,
	) (permissions.PermissionResult, error) {
		callbackCalled = true

		return &permissions.PermissionResultAllow{}, nil
	}

	service.SetCallback(callback)

	result, err := service.CheckToolUse(
		context.Background(),
		"custom_tool",
		map[string]any{},
		[]permissions.PermissionUpdate{},
	)

	if err != nil {
		t.Fatalf("CheckToolUse() error = %v", err)
	}

	if !callbackCalled {
		t.Error("Custom callback was not called")
	}

	if _, ok := result.(*permissions.PermissionResultAllow); !ok {
		t.Errorf("Expected PermissionResultAllow, got %T", result)
	}
}

// TestPermissionModes tests all permission modes.
func TestPermissionModes(t *testing.T) {
	modes := []options.PermissionMode{
		options.PermissionModeBypassPermissions,
		options.PermissionModeDefault,
		options.PermissionModeAsk,
	}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			service := permissions.NewService(&permissions.Config{
				Mode: mode,
			})

			_, err := service.CheckToolUse(
				context.Background(),
				"test_tool",
				map[string]any{},
				[]permissions.PermissionUpdate{},
			)

			if err != nil {
				t.Errorf("Mode %s failed: %v", mode, err)
			}
		})
	}
}
