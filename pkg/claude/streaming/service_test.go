package streaming_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/conneroisu/claude/pkg/claude/internal/testutil"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/ports"
	"github.com/conneroisu/claude/pkg/claude/streaming"
)

func TestConnect(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() *testutil.MockTransport
		wantErr   bool
	}{
		{
			name: "successful connect",
			setupMock: func() *testutil.MockTransport {
				return &testutil.MockTransport{}
			},
		},
		{
			name: "transport error",
			setupMock: func() *testutil.MockTransport {
				return &testutil.MockTransport{
					ConnectFunc: func(ctx context.Context) error {
						return errors.New("connect failed")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := tt.setupMock()
			protocol := &testutil.MockProtocolHandler{}
			svc := streaming.NewService(
				transport,
				protocol,
				&testutil.MockMessageParser{},
				nil,
				nil,
				nil,
			)

			err := svc.Connect(context.Background(), nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSendMessage(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() *testutil.MockTransport
		wantErr   bool
	}{
		{
			name: "successful send",
			setupMock: func() *testutil.MockTransport {
				return &testutil.MockTransport{
					IsReadyFunc: func() bool { return true },
				}
			},
		},
		{
			name: "not connected",
			setupMock: func() *testutil.MockTransport {
				return &testutil.MockTransport{
					IsReadyFunc: func() bool { return false },
				}
			},
			wantErr: true,
		},
		{
			name: "write error",
			setupMock: func() *testutil.MockTransport {
				return &testutil.MockTransport{
					IsReadyFunc: func() bool { return true },
					WriteFunc: func(ctx context.Context, data string) error {
						return errors.New("write failed")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := tt.setupMock()
			protocol := &testutil.MockProtocolHandler{
				StartMessageRouterFunc: func(
					ctx context.Context,
					msgCh chan<- map[string]any,
					errCh chan<- error,
					deps ports.ControlDependencies,
				) error {
					return nil
				},
			}
			svc := streaming.NewService(
				transport,
				protocol,
				&testutil.MockMessageParser{},
				nil,
				nil,
				nil,
			)

			if !tt.wantErr {
				_ = svc.Connect(context.Background(), nil)
			}

			err := svc.SendMessage(context.Background(), "test message")

			if (err != nil) != tt.wantErr {
				t.Errorf("SendMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReceiveMessages(t *testing.T) {
	transport := &testutil.MockTransport{}
	protocol := &testutil.MockProtocolHandler{
		StartMessageRouterFunc: func(
			ctx context.Context,
			msgCh chan<- map[string]any,
			errCh chan<- error,
			deps ports.ControlDependencies,
		) error {
			go func() {
				msgCh <- testutil.AssistantMessageJSON
				close(msgCh)
				close(errCh)
			}()

			return nil
		},
	}
	parser := &testutil.MockMessageParser{
		ParseFunc: func(raw map[string]any) (messages.Message, error) {
			return &messages.AssistantMessage{
				Content: []messages.ContentBlock{
					&messages.TextBlock{Text: "response"},
				},
			}, nil
		},
	}

	svc := streaming.NewService(transport, protocol, parser, nil, nil, nil)
	_ = svc.Connect(context.Background(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	msgCh, errCh := svc.ReceiveMessages(ctx)

	var gotMsg bool
	select {
	case msg := <-msgCh:
		if msg != nil {
			gotMsg = true
		}
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-ctx.Done():
	}

	if !gotMsg {
		t.Error("expected to receive message")
	}
}

func TestClose(t *testing.T) {
	transport := &testutil.MockTransport{
		CloseFunc: func() error {
			return nil
		},
	}
	protocol := &testutil.MockProtocolHandler{}
	svc := streaming.NewService(transport, protocol, &testutil.MockMessageParser{}, nil, nil, nil)

	_ = svc.Connect(context.Background(), nil)

	if err := svc.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	err := svc.SendMessage(context.Background(), "test")
	if err == nil {
		t.Error("SendMessage() should fail after Close()")
	}
}
