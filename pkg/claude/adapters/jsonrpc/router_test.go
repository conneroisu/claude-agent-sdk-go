package jsonrpc

import (
	"context"
	"testing"
	"time"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// mockTransport is a mock implementation of ports.Transport for testing
type mockTransport struct {
	messages []map[string]any
	msgCh    chan map[string]any
	errCh    chan error
}

func newMockTransport(messages []map[string]any) *mockTransport {
	return &mockTransport{
		messages: messages,
		msgCh:    make(chan map[string]any),
		errCh:    make(chan error),
	}
}

func (*mockTransport) Connect(_ context.Context) error {
	return nil
}

func (*mockTransport) Write(_ context.Context, _ string) error {
	return nil
}

func (m *mockTransport) ReadMessages(
	_ context.Context,
) (<-chan map[string]any, <-chan error) {
	go func() {
		defer close(m.msgCh)
		defer close(m.errCh)

		for _, msg := range m.messages {
			m.msgCh <- msg
		}
	}()

	return m.msgCh, m.errCh
}

func (*mockTransport) EndInput() error {
	return nil
}

func (*mockTransport) Close() error {
	return nil
}

func (m *mockTransport) IsReady() bool {
	return true
}

// TestRouterClosesChannelsOnCompletion tests that the router properly closes
// output channels when the transport finishes sending messages.
// This test reproduces the deadlock bug where channels are never closed.
func TestRouterClosesChannelsOnCompletion(t *testing.T) {
	// Create mock transport with a single message
	messages := []map[string]any{
		{
			"type": "assistant",
			"uuid": "test-123",
			"content": []map[string]any{
				{"type": "text", "text": "Hello"},
			},
		},
	}
	transport := newMockTransport(messages)

	// Create adapter and start router
	adapter := NewAdapter(transport)
	msgCh := make(chan map[string]any)
	errCh := make(chan error, 1)

	ctx := context.Background()
	deps := ports.ControlDependencies{
		Permissions: nil,
		Hooks:       nil,
		MCPServers:  nil,
	}

	err := adapter.StartMessageRouter(ctx, msgCh, errCh, deps)
	if err != nil {
		t.Fatalf("StartMessageRouter failed: %v", err)
	}

	// Try to read all messages with a timeout
	// If channels don't close, this will deadlock
	timeout := time.After(2 * time.Second)
	messageCount := 0

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				// Channel closed properly - test passes
				t.Logf("Message channel closed after %d messages", messageCount)
				return
			}
			messageCount++
			t.Logf("Received message: %v", msg)

		case err, ok := <-errCh:
			if !ok {
				// Error channel closed - this is fine
				continue
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

		case <-timeout:
			t.Fatal("DEADLOCK: Router did not close channels after transport completed")
		}
	}
}
