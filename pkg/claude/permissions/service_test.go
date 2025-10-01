package permissions

import (
	"context"
	"errors"
	"testing"

	"github.com/conneroisu/claude/pkg/claude/options"
)

func TestNewService(t *testing.T) {
	t.Run("creates service with config", func(t *testing.T) {
		config := &PermissionsConfig{
			Mode: options.PermissionModeDefault,
		}

		svc := NewService(config)
		if svc == nil {
			t.Fatal("Expected service to be created")
		}
		if svc.mode != options.PermissionModeDefault {
			t.Error("Expected mode to be set from config")
		}
	})

	t.Run("creates service with nil config", func(t *testing.T) {
		svc := NewService(nil)
		if svc == nil {
			t.Fatal("Expected service to be created")
		}
		if svc.mode != options.PermissionModeDefault {
			t.Error("Expected default mode for nil config")
		}
	})
}

func TestCheckToolUse(t *testing.T) {
	ctx := context.Background()

	t.Run("bypass permissions mode always allows", func(t *testing.T) {
		config := &PermissionsConfig{
			Mode: options.PermissionModeBypassPermissions,
		}
		svc := NewService(config)

		result, err := svc.CheckToolUse(ctx, "any_tool", map[string]any{})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !result.IsAllowed() {
			t.Error("Expected tool use to be allowed in bypass mode")
		}
	})

	t.Run("default mode with no callback allows", func(t *testing.T) {
		config := &PermissionsConfig{
			Mode: options.PermissionModeDefault,
		}
		svc := NewService(config)

		result, err := svc.CheckToolUse(ctx, "test_tool", map[string]any{})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !result.IsAllowed() {
			t.Error("Expected tool use to be allowed with no callback")
		}
	})

	t.Run("callback can allow tool use", func(t *testing.T) {
		config := &PermissionsConfig{
			Mode: options.PermissionModeDefault,
			CanUseTool: func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error) {
				return &PermissionResultAllow{}, nil
			},
		}
		svc := NewService(config)

		result, err := svc.CheckToolUse(ctx, "test_tool", map[string]any{})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !result.IsAllowed() {
			t.Error("Expected tool use to be allowed by callback")
		}
	})

	t.Run("callback can deny tool use", func(t *testing.T) {
		config := &PermissionsConfig{
			Mode: options.PermissionModeDefault,
			CanUseTool: func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error) {
				return &PermissionResultDeny{
					Message: "Tool not allowed",
				}, nil
			},
		}
		svc := NewService(config)

		result, err := svc.CheckToolUse(ctx, "test_tool", map[string]any{})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result.IsAllowed() {
			t.Error("Expected tool use to be denied by callback")
		}
		if result.GetDenyMessage() != "Tool not allowed" {
			t.Errorf("Expected deny message, got %s", result.GetDenyMessage())
		}
	})

	t.Run("callback can return updated input", func(t *testing.T) {
		config := &PermissionsConfig{
			Mode: options.PermissionModeDefault,
			CanUseTool: func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error) {
				return &PermissionResultAllow{
					UpdatedInput: map[string]any{
						"modified": true,
					},
				}, nil
			},
		}
		svc := NewService(config)

		result, err := svc.CheckToolUse(ctx, "test_tool", map[string]any{})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !result.IsAllowed() {
			t.Error("Expected tool use to be allowed")
		}

		updated := result.GetUpdatedInput()
		if updated == nil {
			t.Fatal("Expected updated input")
		}
		if val, ok := updated["modified"]; !ok || val != true {
			t.Error("Expected updated input to contain modified=true")
		}
	})

	t.Run("callback error is propagated", func(t *testing.T) {
		expectedErr := errors.New("callback failed")
		config := &PermissionsConfig{
			Mode: options.PermissionModeDefault,
			CanUseTool: func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error) {
				return nil, expectedErr
			},
		}
		svc := NewService(config)

		_, err := svc.CheckToolUse(ctx, "test_tool", map[string]any{})
		if err == nil {
			t.Fatal("Expected error from callback")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected error to be wrapped, got %v", err)
		}
	})

	t.Run("callback receives tool name and input", func(t *testing.T) {
		var receivedToolName string
		var receivedInput map[string]any

		config := &PermissionsConfig{
			Mode: options.PermissionModeDefault,
			CanUseTool: func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error) {
				receivedToolName = toolName
				receivedInput = input

				return &PermissionResultAllow{}, nil
			},
		}
		svc := NewService(config)

		testInput := map[string]any{"key": "value"}
		_, err := svc.CheckToolUse(ctx, "my_tool", testInput)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if receivedToolName != "my_tool" {
			t.Errorf("Expected tool name 'my_tool', got '%s'", receivedToolName)
		}
		if val, ok := receivedInput["key"]; !ok || val != "value" {
			t.Error("Expected input to be passed to callback")
		}
	})

	t.Run("plan mode respects callback", func(t *testing.T) {
		config := &PermissionsConfig{
			Mode: options.PermissionModePlan,
			CanUseTool: func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error) {
				return &PermissionResultDeny{Message: "Denied in plan mode"}, nil
			},
		}
		svc := NewService(config)

		result, err := svc.CheckToolUse(ctx, "test_tool", map[string]any{})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result.IsAllowed() {
			t.Error("Expected tool use to be denied")
		}
	})

	t.Run("accept edits mode respects callback", func(t *testing.T) {
		config := &PermissionsConfig{
			Mode: options.PermissionModeAcceptEdits,
			CanUseTool: func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error) {
				return &PermissionResultAllow{}, nil
			},
		}
		svc := NewService(config)

		result, err := svc.CheckToolUse(ctx, "test_tool", map[string]any{})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !result.IsAllowed() {
			t.Error("Expected tool use to be allowed")
		}
	})
}

func TestUpdateMode(t *testing.T) {
	t.Run("updates permission mode", func(t *testing.T) {
		config := &PermissionsConfig{
			Mode: options.PermissionModeDefault,
		}
		svc := NewService(config)

		svc.UpdateMode(options.PermissionModeBypassPermissions)

		if svc.mode != options.PermissionModeBypassPermissions {
			t.Error("Expected mode to be updated")
		}

		// Verify new mode is effective
		result, _ := svc.CheckToolUse(context.Background(), "test", map[string]any{})
		if !result.IsAllowed() {
			t.Error("Expected bypass mode to allow all tools")
		}
	})
}

func TestPermissionResultAllow(t *testing.T) {
	t.Run("implements PermissionResult interface", func(t *testing.T) {
		result := &PermissionResultAllow{
			UpdatedInput: map[string]any{"key": "value"},
		}

		var _ PermissionResult = result

		if !result.IsAllowed() {
			t.Error("Expected IsAllowed to return true")
		}
		if result.GetDenyMessage() != "" {
			t.Error("Expected GetDenyMessage to return empty string")
		}
		if result.GetUpdatedInput() == nil {
			t.Error("Expected GetUpdatedInput to return input")
		}
	})
}

func TestPermissionResultDeny(t *testing.T) {
	t.Run("implements PermissionResult interface", func(t *testing.T) {
		result := &PermissionResultDeny{
			Message:   "Access denied",
			Interrupt: true,
		}

		var _ PermissionResult = result

		if result.IsAllowed() {
			t.Error("Expected IsAllowed to return false")
		}
		if result.GetDenyMessage() != "Access denied" {
			t.Errorf("Expected deny message, got %s", result.GetDenyMessage())
		}
		if result.GetUpdatedInput() != nil {
			t.Error("Expected GetUpdatedInput to return nil")
		}
	})
}

func TestPermissionBehavior(t *testing.T) {
	behaviors := []PermissionBehavior{
		PermissionBehaviorAllow,
		PermissionBehaviorDeny,
		PermissionBehaviorAsk,
	}

	for _, behavior := range behaviors {
		t.Run(string(behavior), func(t *testing.T) {
			if behavior == "" {
				t.Error("Permission behavior should not be empty")
			}
		})
	}
}

func TestPermissionUpdateDestination(t *testing.T) {
	destinations := []PermissionUpdateDestination{
		PermissionDestinationUserSettings,
		PermissionDestinationProjectSettings,
		PermissionDestinationLocalSettings,
		PermissionDestinationSession,
	}

	for _, dest := range destinations {
		t.Run(string(dest), func(t *testing.T) {
			if dest == "" {
				t.Error("Permission destination should not be empty")
			}
		})
	}
}
