package permissions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/conneroisu/claude/pkg/claude/permissions"
)

func TestCheckToolUse(t *testing.T) {
	tests := []struct {
		name       string
		config     *permissions.PermissionsConfig
		toolName   string
		input      map[string]any
		wantAllow  bool
		wantReason string
		wantErr    bool
	}{
		{
			name: "bypass mode allows all",
			config: &permissions.PermissionsConfig{
				Mode: permissions.PermissionModeBypass,
			},
			toolName:  "bash",
			wantAllow: true,
		},
		{
			name: "nil callback allows all",
			config: &permissions.PermissionsConfig{
				Mode: permissions.PermissionModeDefault,
			},
			toolName:  "bash",
			wantAllow: true,
		},
		{
			name: "callback allows",
			config: &permissions.PermissionsConfig{
				Mode: permissions.PermissionModeDefault,
				CanUseTool: func(
					ctx context.Context,
					toolName string,
					input map[string]any,
					permCtx permissions.ToolPermissionContext,
				) (permissions.PermissionResult, error) {
					return permissions.PermissionResultAllow{}, nil
				},
			},
			toolName:  "bash",
			wantAllow: true,
		},
		{
			name: "callback denies",
			config: &permissions.PermissionsConfig{
				Mode: permissions.PermissionModeDefault,
				CanUseTool: func(
					ctx context.Context,
					toolName string,
					input map[string]any,
					permCtx permissions.ToolPermissionContext,
				) (permissions.PermissionResult, error) {
					return permissions.PermissionResultDeny{
						Message: "not allowed",
					}, nil
				},
			},
			toolName:   "bash",
			wantAllow:  false,
			wantReason: "not allowed",
		},
		{
			name: "callback returns error",
			config: &permissions.PermissionsConfig{
				Mode: permissions.PermissionModeDefault,
				CanUseTool: func(
					ctx context.Context,
					toolName string,
					input map[string]any,
					permCtx permissions.ToolPermissionContext,
				) (permissions.PermissionResult, error) {
					return nil, errors.New("check failed")
				},
			},
			toolName: "bash",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := permissions.NewService(tt.config)

			result, err := svc.CheckToolUse(
				context.Background(),
				tt.toolName,
				tt.input,
				permissions.ToolPermissionContext{},
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckToolUse() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if err != nil {
				return
			}

			switch r := result.(type) {
			case permissions.PermissionResultAllow:
				if !tt.wantAllow {
					t.Error("expected deny, got allow")
				}
			case permissions.PermissionResultDeny:
				if tt.wantAllow {
					t.Error("expected allow, got deny")
				}
				if r.Message != tt.wantReason {
					t.Errorf("deny message = %s, want %s", r.Message, tt.wantReason)
				}
			}
		})
	}
}

func TestUpdateMode(t *testing.T) {
	svc := permissions.NewService(&permissions.PermissionsConfig{
		Mode: permissions.PermissionModeDefault,
	})

	svc.UpdateMode(permissions.PermissionModeBypass)

	allowed, _, err := svc.CanUseTool(
		context.Background(),
		"bash",
		map[string]any{},
	)

	if err != nil {
		t.Fatalf("CanUseTool() error = %v", err)
	}

	if !allowed {
		t.Error("expected allowed in bypass mode")
	}
}

func TestCanUseTool(t *testing.T) {
	tests := []struct {
		name      string
		config    *permissions.PermissionsConfig
		toolName  string
		wantAllow bool
	}{
		{
			name: "allow result",
			config: &permissions.PermissionsConfig{
				CanUseTool: func(
					ctx context.Context,
					toolName string,
					input map[string]any,
					permCtx permissions.ToolPermissionContext,
				) (permissions.PermissionResult, error) {
					return permissions.PermissionResultAllow{}, nil
				},
			},
			toolName:  "bash",
			wantAllow: true,
		},
		{
			name: "deny result",
			config: &permissions.PermissionsConfig{
				CanUseTool: func(
					ctx context.Context,
					toolName string,
					input map[string]any,
					permCtx permissions.ToolPermissionContext,
				) (permissions.PermissionResult, error) {
					return permissions.PermissionResultDeny{Message: "denied"}, nil
				},
			},
			toolName:  "bash",
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := permissions.NewService(tt.config)

			allowed, reason, err := svc.CanUseTool(
				context.Background(),
				tt.toolName,
				map[string]any{},
			)

			if err != nil {
				t.Fatalf("CanUseTool() error = %v", err)
			}

			if allowed != tt.wantAllow {
				t.Errorf("allowed = %v, want %v", allowed, tt.wantAllow)
			}

			if !tt.wantAllow && reason == "" {
				t.Error("expected deny reason")
			}
		})
	}
}

func TestNilConfig(t *testing.T) {
	svc := permissions.NewService(nil)

	allowed, _, err := svc.CanUseTool(
		context.Background(),
		"bash",
		map[string]any{},
	)

	if err != nil {
		t.Fatalf("CanUseTool() error = %v", err)
	}

	if !allowed {
		t.Error("expected allow with nil config")
	}
}
