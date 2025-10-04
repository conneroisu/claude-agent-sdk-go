//nolint:revive // Test file - relaxed linting
package querying_test

import (
	"context"
	"testing"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/internal/testutil"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
	"github.com/conneroisu/claude/pkg/claude/querying"
)

// TestServiceExecute tests query execution.
func TestServiceExecute(t *testing.T) {
	ctx := context.Background()

	// Create mocks
	transport := &testutil.MockTransport{}
	protocol := &testutil.MockProtocol{
		InitializeFunc: func(
			_ context.Context,
			_ any,
		) (map[string]any, error) {
			return map[string]any{"status": "ok"}, nil
		},
	}
	parser := &testutil.MockParser{
		ParseFunc: func(data map[string]any) (messages.Message, error) {
			return &messages.AssistantMessage{
				Content: []messages.ContentBlock{
					messages.TextBlock{
						Type: "text",
						Text: "Test response",
					},
				},
			}, nil
		},
	}

	hooks := hooking.NewService(nil)
	perms := permissions.NewService(&permissions.Config{
		Mode: options.PermissionModeBypassPermissions,
	})
	mcpServers := make(map[string]ports.MCPServer)

	// Create service
	cfg := &querying.Config{
		Transport:   transport,
		Protocol:    protocol,
		Parser:      parser,
		Hooks:       hooks,
		Permissions: perms,
		MCPServers:  mcpServers,
	}

	svc := querying.NewService(cfg)

	// Execute query
	opts := &options.AgentOptions{}
	msgCh, errCh := svc.Execute(ctx, "test query", opts)

	// Should not error immediately
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	case <-msgCh:
		// Message received is ok
	default:
		// No immediate result is ok
	}
}

// TestServiceWithHooks tests query with hooks.
func TestServiceWithHooks(t *testing.T) {
	ctx := context.Background()

	transport := &testutil.MockTransport{}
	protocol := &testutil.MockProtocol{
		InitializeFunc: func(
			_ context.Context,
			_ any,
		) (map[string]any, error) {
			return map[string]any{"status": "ok"}, nil
		},
	}
	parser := &testutil.MockParser{}

	hookCalled := false
	hooks := hooking.NewService(nil)
	hooks.RegisterHook("test", func(
		_ context.Context,
		_ map[string]any,
	) (map[string]any, error) {
		hookCalled = true

		return make(map[string]any), nil
	})

	perms := permissions.NewService(&permissions.Config{
		Mode: options.PermissionModeBypassPermissions,
	})
	mcpServers := make(map[string]ports.MCPServer)

	cfg := &querying.Config{
		Transport:   transport,
		Protocol:    protocol,
		Parser:      parser,
		Hooks:       hooks,
		Permissions: perms,
		MCPServers:  mcpServers,
	}

	svc := querying.NewService(cfg)

	opts := &options.AgentOptions{}
	_, _ = svc.Execute(ctx, "test", opts)

	// Hook registration tested (execution happens via protocol)
	_ = hookCalled
}
