package testutil

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// MockTransport implements ports.Transport for testing
type MockTransport struct {
	ConnectFunc      func(context.Context) error
	WriteFunc        func(context.Context, string) error
	ReadMessagesFunc func(context.Context) (<-chan map[string]any, <-chan error)
	EndInputFunc     func() error
	CloseFunc        func() error
	IsReadyFunc      func() bool
}

func (m *MockTransport) Connect(ctx context.Context) error {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx)
	}
	return nil
}

func (m *MockTransport) Write(ctx context.Context, data string) error {
	if m.WriteFunc != nil {
		return m.WriteFunc(ctx, data)
	}
	return nil
}

func (m *MockTransport) ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error) {
	if m.ReadMessagesFunc != nil {
		return m.ReadMessagesFunc(ctx)
	}
	msgCh := make(chan map[string]any)
	errCh := make(chan error)
	close(msgCh)
	close(errCh)
	return msgCh, errCh
}

func (m *MockTransport) EndInput() error {
	if m.EndInputFunc != nil {
		return m.EndInputFunc()
	}
	return nil
}

func (m *MockTransport) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func (m *MockTransport) IsReady() bool {
	if m.IsReadyFunc != nil {
		return m.IsReadyFunc()
	}
	return true
}

var _ ports.Transport = (*MockTransport)(nil)

// MockProtocolHandler implements ports.ProtocolHandler for testing
type MockProtocolHandler struct {
	InitializeFunc         func(context.Context, any) (map[string]any, error)
	SendControlRequestFunc func(context.Context, map[string]any) (map[string]any, error)
	HandleControlRequestFunc func(context.Context, map[string]any, ports.PermissionService, map[string]ports.HookCallback, map[string]ports.MCPServer) (map[string]any, error)
	StartMessageRouterFunc func(context.Context, chan<- map[string]any, chan<- error, ports.PermissionService, map[string]ports.HookCallback, map[string]ports.MCPServer) error
}

func (m *MockProtocolHandler) Initialize(ctx context.Context, config any) (map[string]any, error) {
	if m.InitializeFunc != nil {
		return m.InitializeFunc(ctx, config)
	}
	return nil, nil
}

func (m *MockProtocolHandler) SendControlRequest(ctx context.Context, req map[string]any) (map[string]any, error) {
	if m.SendControlRequestFunc != nil {
		return m.SendControlRequestFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockProtocolHandler) HandleControlRequest(
	ctx context.Context,
	req map[string]any,
	perms ports.PermissionService,
	hooks map[string]ports.HookCallback,
	mcpServers map[string]ports.MCPServer,
) (map[string]any, error) {
	if m.HandleControlRequestFunc != nil {
		return m.HandleControlRequestFunc(ctx, req, perms, hooks, mcpServers)
	}
	return nil, nil
}

func (m *MockProtocolHandler) StartMessageRouter(
	ctx context.Context,
	msgCh chan<- map[string]any,
	errCh chan<- error,
	perms ports.PermissionService,
	hooks map[string]ports.HookCallback,
	mcpServers map[string]ports.MCPServer,
) error {
	if m.StartMessageRouterFunc != nil {
		return m.StartMessageRouterFunc(ctx, msgCh, errCh, perms, hooks, mcpServers)
	}
	return nil
}

var _ ports.ProtocolHandler = (*MockProtocolHandler)(nil)

// MockMessageParser implements ports.MessageParser for testing
type MockMessageParser struct {
	ParseFunc func(map[string]any) (messages.Message, error)
}

func (m *MockMessageParser) Parse(raw map[string]any) (messages.Message, error) {
	if m.ParseFunc != nil {
		return m.ParseFunc(raw)
	}
	return nil, nil
}

var _ ports.MessageParser = (*MockMessageParser)(nil)

// MockMCPServer implements ports.MCPServer for testing
type MockMCPServer struct {
	NameFunc          func() string
	HandleMessageFunc func(context.Context, []byte) ([]byte, error)
	CloseFunc         func() error
}

func (m *MockMCPServer) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock-server"
}

func (m *MockMCPServer) HandleMessage(ctx context.Context, msg []byte) ([]byte, error) {
	if m.HandleMessageFunc != nil {
		return m.HandleMessageFunc(ctx, msg)
	}
	return []byte(`{}`), nil
}

func (m *MockMCPServer) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

var _ ports.MCPServer = (*MockMCPServer)(nil)

// MockPermissionService implements ports.PermissionService for testing
type MockPermissionService struct {
	CheckToolUseFunc func(context.Context, string, map[string]any, any) (any, error)
}

func (m *MockPermissionService) CheckToolUse(
	ctx context.Context,
	toolName string,
	input map[string]any,
	suggestions any,
) (any, error) {
	if m.CheckToolUseFunc != nil {
		return m.CheckToolUseFunc(ctx, toolName, input, suggestions)
	}
	return nil, nil
}

var _ ports.PermissionService = (*MockPermissionService)(nil)
