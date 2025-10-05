// Package testutil provides test utilities and mocks for testing.
// It includes mock implementations of core ports for unit testing.
package testutil

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// MockTransport implements ports.Transport for testing.
// Each method can be overridden by setting the corresponding Func field.
// If a Func is nil, the default behavior returns nil or empty channels.
type MockTransport struct {
	ConnectFunc      func(context.Context) error
	WriteFunc        func(context.Context, string) error
	ReadMessagesFunc func(context.Context) (<-chan map[string]any, <-chan error)
	EndInputFunc     func() error
	CloseFunc        func() error
	IsReadyFunc      func() bool
}

// Connect establishes a connection to the transport.
func (m *MockTransport) Connect(ctx context.Context) error {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx)
	}

	return nil
}

// Write sends data through the transport.
func (m *MockTransport) Write(ctx context.Context, data string) error {
	if m.WriteFunc != nil {
		return m.WriteFunc(ctx, data)
	}

	return nil
}

// ReadMessages returns channels for receiving messages and errors.
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

// EndInput signals end of input to the transport.
func (m *MockTransport) EndInput() error {
	if m.EndInputFunc != nil {
		return m.EndInputFunc()
	}

	return nil
}

// Close closes the transport connection.
func (m *MockTransport) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}

	return nil
}

// IsReady checks if the transport is ready for communication.
func (m *MockTransport) IsReady() bool {
	if m.IsReadyFunc != nil {
		return m.IsReadyFunc()
	}

	return true
}

var _ ports.Transport = (*MockTransport)(nil)

// MockProtocolHandler implements ports.ProtocolHandler for testing.
// Provides mock implementations for protocol message handling.
type MockProtocolHandler struct {
	InitializeFunc           InitFunc
	SendControlRequestFunc   SendCtrlReqFunc
	HandleControlRequestFunc HandleCtrlReqFunc
	StartMessageRouterFunc   StartRouterFunc
}

type (
	InitFunc func(
		context.Context,
		map[string]any,
	) (map[string]any, error)
	SendCtrlReqFunc func(
		context.Context,
		map[string]any,
	) (map[string]any, error)
	HandleCtrlReqFunc func(
		context.Context,
		map[string]any,
		ports.ControlDependencies,
	) (map[string]any, error)
	StartRouterFunc func(
		context.Context,
		chan<- map[string]any,
		chan<- error,
		ports.ControlDependencies,
	) error
)

func (m *MockProtocolHandler) Initialize(
	ctx context.Context,
	cfg map[string]any,
) (map[string]any, error) {
	if m.InitializeFunc != nil {
		return m.InitializeFunc(ctx, cfg)
	}

	return map[string]any{"status": "ok"}, nil
}

func (m *MockProtocolHandler) SendControlRequest(
	ctx context.Context,
	req map[string]any,
) (map[string]any, error) {
	if m.SendControlRequestFunc != nil {
		return m.SendControlRequestFunc(ctx, req)
	}

	return make(map[string]any), nil
}

func (m *MockProtocolHandler) HandleControlRequest(
	ctx context.Context,
	req map[string]any,
	deps ports.ControlDependencies,
) (map[string]any, error) {
	if m.HandleControlRequestFunc != nil {
		return m.HandleControlRequestFunc(ctx, req, deps)
	}

	return make(map[string]any), nil
}

func (m *MockProtocolHandler) StartMessageRouter(
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

var _ ports.ProtocolHandler = (*MockProtocolHandler)(nil)

// MockMessageParser implements ports.MessageParser for testing.
type MockMessageParser struct {
	ParseFunc func(map[string]any) (messages.Message, error)
}

// Parse converts raw message data into a typed Message.
func (m *MockMessageParser) Parse(
	raw map[string]any,
) (messages.Message, error) {
	if m.ParseFunc != nil {
		return m.ParseFunc(raw)
	}

	return &messages.UnknownMessage{RawData: raw}, nil
}

var _ ports.MessageParser = (*MockMessageParser)(nil)
