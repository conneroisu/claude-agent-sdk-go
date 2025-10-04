//nolint:revive // Test file allows higher complexity
package streaming_test

import (
	"context"
	"testing"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/internal/testutil"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
	"github.com/conneroisu/claude/pkg/claude/streaming"
)

// TestConnect tests connection establishment.
func TestConnect(t *testing.T) {
	transport := &testutil.MockTransport{
		ConnectFunc: func(_ context.Context) error {
			return nil
		},
	}

	protocol := &testutil.MockProtocol{}
	parser := &testutil.MockParser{}

	svc := streaming.NewService(&streaming.Config{
		Transport:   transport,
		Protocol:    protocol,
		Parser:      parser,
		Hooks:       hooking.NewService(nil),
		Permissions: permissions.New(nil),
		MCPServers:  make(map[string]ports.MCPServer),
	})

	if err := svc.Connect(context.Background(), nil); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
}

// TestSendMessage tests message sending.
func TestSendMessage(t *testing.T) {
	writeCalled := false
	transport := &testutil.MockTransport{
		ConnectFunc: func(_ context.Context) error {
			return nil
		},
		WriteFunc: func(_ context.Context, _ string) error {
			writeCalled = true

			return nil
		},
	}

	protocol := &testutil.MockProtocol{}
	parser := &testutil.MockParser{}

	svc := streaming.NewService(&streaming.Config{
		Transport:   transport,
		Protocol:    protocol,
		Parser:      parser,
		Hooks:       hooking.NewService(nil),
		Permissions: permissions.New(nil),
		MCPServers:  make(map[string]ports.MCPServer),
	})

	if err := svc.Connect(context.Background(), nil); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if err := svc.SendMessage(context.Background(), "test"); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if !writeCalled {
		t.Error("Write was not called")
	}
}

// TestReceiveMessages tests message reception.
func TestReceiveMessages(t *testing.T) {
	transport := &testutil.MockTransport{
		ConnectFunc: func(_ context.Context) error {
			return nil
		},
	}

	protocol := &testutil.MockProtocol{
		StartMessageRouterFunc: func(
			_ context.Context,
			msgCh chan<- map[string]any,
			errCh chan<- error,
			_ ports.ControlDependencies,
		) error {
			go func() {
				msgCh <- testutil.AssistantMessageJSON
				close(msgCh)
				close(errCh)
			}()

			return nil
		},
	}

	parser := &testutil.MockParser{
		ParseFunc: func(_ map[string]any) (messages.Message, error) {
			return &messages.AssistantMessage{
				Model: "claude-sonnet-4",
			}, nil
		},
	}

	svc := streaming.NewService(&streaming.Config{
		Transport:   transport,
		Protocol:    protocol,
		Parser:      parser,
		Hooks:       hooking.NewService(nil),
		Permissions: permissions.New(nil),
		MCPServers:  make(map[string]ports.MCPServer),
	})

	if err := svc.Connect(context.Background(), nil); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	msgCh, errCh := svc.ReceiveMessages(context.Background())

	select {
	case msg := <-msgCh:
		if msg == nil {
			t.Fatal("expected message, got nil")
		}
	case err := <-errCh:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}
