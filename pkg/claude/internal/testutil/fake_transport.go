package testutil

import (
	"context"
	"sync"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// FakeTransport simulates CLI transport behavior for hermetic testing.
// It queues responses and tracks write history without spawning processes.
type FakeTransport struct {
	mu            sync.Mutex
	responses     []map[string]any
	isConnected   bool
	writeHistory  []string
	simulateError error
}

// NewFakeTransport creates a new fake transport for testing.
func NewFakeTransport() *FakeTransport {
	return &FakeTransport{
		responses:    make([]map[string]any, 0),
		writeHistory: make([]string, 0),
	}
}

// QueueResponse adds a response to be returned by ReadMessages.
func (f *FakeTransport) QueueResponse(msg map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.responses = append(f.responses, msg)
}

// SimulateError causes the next operation to fail.
func (f *FakeTransport) SimulateError(err error) {
	f.simulateError = err
}

// Connect simulates connecting to the CLI.
func (f *FakeTransport) Connect(ctx context.Context) error {
	if f.simulateError != nil {
		return f.simulateError
	}
	f.isConnected = true

	return nil
}

// Write simulates writing to CLI stdin and tracks history.
func (f *FakeTransport) Write(ctx context.Context, data string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.simulateError != nil {
		return f.simulateError
	}
	f.writeHistory = append(f.writeHistory, data)

	return nil
}

// ReadMessages returns queued responses as a channel.
func (f *FakeTransport) ReadMessages(
	ctx context.Context,
) (<-chan map[string]any, <-chan error) {
	msgCh := make(chan map[string]any, len(f.responses))
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		if f.simulateError != nil {
			errCh <- f.simulateError

			return
		}

		f.mu.Lock()
		defer f.mu.Unlock()

		for _, msg := range f.responses {
			select {
			case msgCh <- msg:
			case <-ctx.Done():
				errCh <- ctx.Err()

				return
			}
		}
	}()

	return msgCh, errCh
}

// EndInput simulates ending input to the CLI.
func (f *FakeTransport) EndInput() error {
	return nil
}

// Close simulates closing the connection.
func (f *FakeTransport) Close() error {
	f.isConnected = false

	return nil
}

// IsReady returns whether the transport is connected.
func (f *FakeTransport) IsReady() bool {
	return f.isConnected
}

// GetWriteHistory returns all data written to the transport.
func (f *FakeTransport) GetWriteHistory() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	history := make([]string, len(f.writeHistory))
	copy(history, f.writeHistory)

	return history
}

// Compile-time interface check.
var _ ports.Transport = (*FakeTransport)(nil)
