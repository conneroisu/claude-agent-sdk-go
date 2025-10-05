package jsonrpc_test

import (
	"context"
	"testing"

	"github.com/conneroisu/claude/pkg/claude/adapters/jsonrpc"
	"github.com/conneroisu/claude/pkg/claude/internal/testutil"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

func TestNewAdapter(t *testing.T) {
	transport := &testutil.MockTransport{}
	adapter := jsonrpc.NewAdapter(transport)

	if adapter == nil {
		t.Fatal("NewAdapter() returned nil")
	}
}

func TestHandleControlRequest(t *testing.T) {
	tests := []struct {
		name    string
		request map[string]any
		deps    ports.ControlDependencies
		wantErr bool
	}{
		{
			name: "hook_callback request",
			request: map[string]any{
				"method": "hook_callback",
				"params": map[string]any{
					"callback_id": "test_hook",
					"input": map[string]any{
						"data": "test",
					},
				},
			},
			deps: ports.ControlDependencies{
				Hooks: map[string]ports.HookCallback{
					"test_hook": func(input map[string]any) (map[string]any, error) {
						return map[string]any{"result": "ok"}, nil
					},
				},
			},
		},
		{
			name: "check_tool_permission request",
			request: map[string]any{
				"method": "check_tool_permission",
				"params": map[string]any{
					"tool_name": "bash",
					"input": map[string]any{
						"command": "ls",
					},
				},
			},
			deps: ports.ControlDependencies{
				Perms: &testutil.MockPermissionsService{
					CanUseToolFunc: func(
						ctx context.Context,
						toolName string,
						input map[string]any,
					) (bool, string, error) {
						return true, "", nil
					},
				},
			},
		},
		{
			name: "mcp_message request",
			request: map[string]any{
				"method": "mcp_message",
				"params": map[string]any{
					"server_name": "test_server",
					"message":     []byte(`{"jsonrpc":"2.0","method":"test"}`),
				},
			},
			deps: ports.ControlDependencies{
				MCPServers: map[string]ports.MCPServer{
					"test_server": &testutil.MockMCPServer{
						NameFunc: func() string { return "test_server" },
						HandleMessageFunc: func(ctx context.Context, msg []byte) ([]byte, error) {
							return []byte(`{"result":"ok"}`), nil
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &testutil.MockTransport{}
			adapter := jsonrpc.NewAdapter(transport)

			resp, err := adapter.HandleControlRequest(
				context.Background(),
				tt.request,
				tt.deps,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleControlRequest() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr && resp == nil {
				t.Error("HandleControlRequest() returned nil response")
			}
		})
	}
}
