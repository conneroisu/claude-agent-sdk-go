// Package testutil provides test utilities and mocks.
package testutil

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// MockTransport implements ports.Transport for testing.
type MockTransport struct {
	ConnectFunc      func(context.Context) error
	WriteFunc        func(context.Context, string) error
	ReadMessagesFunc func(context.Context) (<-chan map[string]any, <-chan error)
	EndInputFunc     func() error
	CloseFunc        func() error
	IsReadyFunc      func() bool
}

// Connect calls the mock function.
func (m *MockTransport) Connect(ctx context.Context) error {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx)
	}

	return nil
}

// Write calls the mock function.
func (m *MockTransport) Write(ctx context.Context, data string) error {
	if m.WriteFunc != nil {
		return m.WriteFunc(ctx, data)
	}

	return nil
}

// ReadMessages calls the mock function.
func (m *MockTransport) ReadMessages(
	ctx context.Context,
) (<-chan map[string]any, <-chan error) {
	if m.ReadMessagesFunc != nil {
		return m.ReadMessagesFunc(ctx)
	}

	msgCh := make(chan map[string]any)
	errCh := make(chan error)
	close(msgCh)
	close(errCh)

	return msgCh, errCh
}

// EndInput calls the mock function.
func (m *MockTransport) EndInput() error {
	if m.EndInputFunc != nil {
		return m.EndInputFunc()
	}

	return nil
}

// Close calls the mock function.
func (m *MockTransport) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}

	return nil
}

// IsReady calls the mock function.
func (m *MockTransport) IsReady() bool {
	if m.IsReadyFunc != nil {
		return m.IsReadyFunc()
	}

	return true
}

// Verify interface compliance.
var _ ports.Transport = (*MockTransport)(nil)

// MockProtocol implements ports.ProtocolHandler for testing.
type MockProtocol struct {
	InitializeFunc     func(context.Context, any) (map[string]any, error)
	SendControlReqFunc func(
		context.Context,
		map[string]any,
	) (map[string]any, error)
	HandleControlReqFunc func(
		context.Context,
		map[string]any,
		ports.ControlDependencies,
	) (map[string]any, error)
	StartMessageRouterFunc func(
		context.Context,
		chan<- map[string]any,
		chan<- error,
		ports.ControlDependencies,
	) error
}

// Initialize calls the mock function.
func (m *MockProtocol) Initialize(
	ctx context.Context,
	config any,
) (map[string]any, error) {
	if m.InitializeFunc != nil {
		return m.InitializeFunc(ctx, config)
	}

	return nil, nil
}

// SendControlRequest calls the mock function.
func (m *MockProtocol) SendControlRequest(
	ctx context.Context,
	req map[string]any,
) (map[string]any, error) {
	if m.SendControlReqFunc != nil {
		return m.SendControlReqFunc(ctx, req)
	}

	return nil, nil
}

// HandleControlRequest calls the mock function.
func (m *MockProtocol) HandleControlRequest(
	ctx context.Context,
	req map[string]any,
	deps ports.ControlDependencies,
) (map[string]any, error) {
	if m.HandleControlReqFunc != nil {
		return m.HandleControlReqFunc(ctx, req, deps)
	}

	return nil, nil
}

// StartMessageRouter calls the mock function.
func (m *MockProtocol) StartMessageRouter(
	ctx context.Context,
	msgCh chan<- map[string]any,
	errCh chan<- error,
	deps ports.ControlDependencies,
) error {
	if m.StartMessageRouterFunc != nil {
		return m.StartMessageRouterFunc(ctx, msgCh, errCh, deps)
	}

	return nil
}

// Verify interface compliance.
var _ ports.ProtocolHandler = (*MockProtocol)(nil)

// MockParser implements ports.MessageParser for testing.
type MockParser struct {
	ParseFunc func(map[string]any) (messages.Message, error)
}

// Parse calls the mock function.
func (m *MockParser) Parse(
	data map[string]any,
) (messages.Message, error) {
	if m.ParseFunc != nil {
		return m.ParseFunc(data)
	}

	return nil, nil
}

// Verify interface compliance.
var _ ports.MessageParser = (*MockParser)(nil)
