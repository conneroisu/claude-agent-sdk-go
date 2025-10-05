package querying_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/conneroisu/claude/pkg/claude/internal/testutil"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/ports"
	"github.com/conneroisu/claude/pkg/claude/querying"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() (*testutil.MockTransport, *testutil.MockProtocolHandler, *testutil.MockMessageParser)
		wantErr   bool
		wantMsgs  int
	}{
		{
			name: "successful query",
			setupMock: func() (*testutil.MockTransport, *testutil.MockProtocolHandler, *testutil.MockMessageParser) {
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

				return transport, protocol, parser
			},
			wantMsgs: 1,
		},
		{
			name: "transport connect error",
			setupMock: func() (*testutil.MockTransport, *testutil.MockProtocolHandler, *testutil.MockMessageParser) {
				transport := &testutil.MockTransport{
					ConnectFunc: func(ctx context.Context) error {
						return errors.New("connect failed")
					},
				}
				return transport, &testutil.MockProtocolHandler{}, &testutil.MockMessageParser{}
			},
			wantErr: true,
		},
		{
			name: "parser error",
			setupMock: func() (*testutil.MockTransport, *testutil.MockProtocolHandler, *testutil.MockMessageParser) {
				transport := &testutil.MockTransport{}
				protocol := &testutil.MockProtocolHandler{
					StartMessageRouterFunc: func(
						ctx context.Context,
						msgCh chan<- map[string]any,
						errCh chan<- error,
						deps ports.ControlDependencies,
					) error {
						go func() {
							msgCh <- map[string]any{"type": "test"}
							close(msgCh)
							close(errCh)
						}()

						return nil
					},
				}
				parser := &testutil.MockMessageParser{
					ParseFunc: func(raw map[string]any) (messages.Message, error) {
						return nil, errors.New("parse failed")
					},
				}

				return transport, protocol, parser
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, protocol, parser := tt.setupMock()
			svc := querying.NewService(
				transport,
				protocol,
				parser,
				nil,
				nil,
				nil,
			)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			msgCh, errCh := svc.Execute(ctx, "test prompt")

			var gotMsgs int
			var gotErr error

		loop:
			for {
				select {
				case msg, ok := <-msgCh:
					if !ok {
						break loop
					}
					if msg != nil {
						gotMsgs++
					}
				case err, ok := <-errCh:
					if ok && err != nil {
						gotErr = err
					}
				case <-ctx.Done():
					break loop
				}
			}

			if (gotErr != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", gotErr, tt.wantErr)
			}

			if !tt.wantErr && gotMsgs != tt.wantMsgs {
				t.Errorf("got %d messages, want %d", gotMsgs, tt.wantMsgs)
			}
		})
	}
}

func TestExecuteContextCancellation(t *testing.T) {
	transport := &testutil.MockTransport{}
	protocol := &testutil.MockProtocolHandler{
		StartMessageRouterFunc: func(
			ctx context.Context,
			msgCh chan<- map[string]any,
			errCh chan<- error,
			deps ports.ControlDependencies,
		) error {
			<-ctx.Done()
			close(msgCh)
			close(errCh)

			return nil
		},
	}

	svc := querying.NewService(transport, protocol, &testutil.MockMessageParser{}, nil, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	msgCh, errCh := svc.Execute(ctx, "test")

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected error from canceled context")
	}

	select {
	case _, ok := <-msgCh:
		if ok {
			t.Error("message channel should be closed")
		}
	case <-time.After(100 * time.Millisecond):
	}
}
