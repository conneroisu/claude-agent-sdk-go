// Package testutil provides test utilities and mocks for testing.
package testutil

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// MockMCPServer implements ports.MCPServer for testing.
// Provides mock MCP server functionality.
type MockMCPServer struct {
	NameFunc          func() string
	HandleMessageFunc func(context.Context, []byte) ([]byte, error)
	CloseFunc         func() error
}

// Name returns the name of the MCP server.
func (m *MockMCPServer) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}

	return "mock-server"
}

// HandleMessage processes a message from the MCP server.
func (m *MockMCPServer) HandleMessage(
	ctx context.Context,
	msg []byte,
) ([]byte, error) {
	if m.HandleMessageFunc != nil {
		return m.HandleMessageFunc(ctx, msg)
	}

	return []byte(`{"result": "ok"}`), nil
}

// Close closes the MCP server.
func (m *MockMCPServer) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}

	return nil
}

var _ ports.MCPServer = (*MockMCPServer)(nil)
