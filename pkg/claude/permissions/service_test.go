package permissions_test

import (
	"context"
	"testing"

	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
)

func TestService_CheckToolUse_BypassPermissions(t *testing.T) {
	config := &permissions.PermissionsConfig{
		Mode: options.PermissionModeBypassPermissions,
	}
	service := permissions.NewService(config)

	result, err := service.CheckToolUse(
		context.Background(),
		"Bash",
		map[string]any{"command": "ls"},
		nil,
	)

	if err != nil {
		t.Fatalf("CheckToolUse() error = %v", err)
	}

	allowResult, ok := result.(*permissions.PermissionResultAllow)
	if !ok {
		t.Fatalf("CheckToolUse() got type %T, want *permissions.PermissionResultAllow", result)
	}

	if allowResult.UpdatedInput != nil {
		t.Error("CheckToolUse() unexpected updated input")
	}
}

func TestService_CheckToolUse_WithCallback(t *testing.T) {
	callbackInvoked := false
	config := &permissions.PermissionsConfig{
		Mode: options.PermissionModeDefault,
		CanUseTool: func(ctx context.Context, toolName string, input map[string]any, permCtx permissions.ToolPermissionContext) (permissions.PermissionResult, error) {
			callbackInvoked = true
			if toolName == "Bash" {
				return &permissions.PermissionResultDeny{
					Message: "Bash not allowed in tests",
				}, nil
			}

			return &permissions.PermissionResultAllow{}, nil
		},
	}
	service := permissions.NewService(config)

	result, err := service.CheckToolUse(
		context.Background(),
		"Bash",
		map[string]any{"command": "ls"},
		nil,
	)

	if err != nil {
		t.Fatalf("CheckToolUse() error = %v", err)
	}

	if !callbackInvoked {
		t.Error("CheckToolUse() callback was not invoked")
	}

	if _, ok := result.(*permissions.PermissionResultDeny); !ok {
		t.Fatalf("CheckToolUse() got type %T, want *permissions.PermissionResultDeny", result)
	}
}

func TestService_CheckToolUse_CallbackWithModifiedInput(t *testing.T) {
	config := &permissions.PermissionsConfig{
		Mode: options.PermissionModeDefault,
		CanUseTool: func(ctx context.Context, toolName string, input map[string]any, permCtx permissions.ToolPermissionContext) (permissions.PermissionResult, error) {
			if command, ok := input["command"].(string); ok {
				input["command"] = command + " --safe-mode"
			}

			return &permissions.PermissionResultAllow{
				UpdatedInput: input,
			}, nil
		},
	}
	service := permissions.NewService(config)

	result, err := service.CheckToolUse(
		context.Background(),
		"Bash",
		map[string]any{"command": "ls"},
		nil,
	)

	if err != nil {
		t.Fatalf("CheckToolUse() error = %v", err)
	}

	allowResult, ok := result.(*permissions.PermissionResultAllow)
	if !ok {
		t.Fatalf("CheckToolUse() got type %T, want *permissions.PermissionResultAllow", result)
	}

	if allowResult.UpdatedInput == nil {
		t.Fatal("CheckToolUse() expected updated input, got nil")
	}

	modifiedCommand, ok := allowResult.UpdatedInput["command"].(string)
	if !ok || modifiedCommand != "ls --safe-mode" {
		t.Errorf("CheckToolUse() got command %q, want %q", modifiedCommand, "ls --safe-mode")
	}
}

func TestService_CheckToolUse_DefaultAllowsWithoutCallback(t *testing.T) {
	config := &permissions.PermissionsConfig{
		Mode: options.PermissionModeDefault,
		// No callback provided
	}
	service := permissions.NewService(config)

	result, err := service.CheckToolUse(
		context.Background(),
		"Bash",
		map[string]any{"command": "ls"},
		nil,
	)

	if err != nil {
		t.Fatalf("CheckToolUse() error = %v", err)
	}

	// Should allow when no callback is provided (CLI handles prompting)
	if _, ok := result.(*permissions.PermissionResultAllow); !ok {
		t.Fatalf("CheckToolUse() got type %T, want *permissions.PermissionResultAllow", result)
	}
}

func TestService_UpdateMode(t *testing.T) {
	config := &permissions.PermissionsConfig{
		Mode: options.PermissionModeDefault,
		CanUseTool: func(ctx context.Context, toolName string, input map[string]any, permCtx permissions.ToolPermissionContext) (permissions.PermissionResult, error) {
			return &permissions.PermissionResultDeny{Message: "denied"}, nil
		},
	}
	service := permissions.NewService(config)

	// Initially deny via callback
	result, _ := service.CheckToolUse(context.Background(), "Bash", map[string]any{}, nil)
	if _, ok := result.(*permissions.PermissionResultDeny); !ok {
		t.Error("Expected deny via callback initially")
	}

	// Update to bypass mode
	service.UpdateMode(options.PermissionModeBypassPermissions)

	// Now bypass permissions (allow without callback)
	result, _ = service.CheckToolUse(context.Background(), "Bash", map[string]any{}, nil)
	if _, ok := result.(*permissions.PermissionResultAllow); !ok {
		t.Error("Expected allow after updating to bypass mode")
	}
}

func TestService_NilConfig(t *testing.T) {
	service := permissions.NewService(nil)

	result, err := service.CheckToolUse(
		context.Background(),
		"Bash",
		map[string]any{},
		nil,
	)

	if err != nil {
		t.Fatalf("CheckToolUse() error = %v", err)
	}

	// Nil config defaults to Ask mode, which is not handled in switch and returns deny
	if _, ok := result.(*permissions.PermissionResultDeny); !ok {
		t.Fatalf("CheckToolUse() got type %T, want *permissions.PermissionResultDeny", result)
	}
}
