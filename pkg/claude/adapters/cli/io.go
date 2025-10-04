package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// Write sends data to the CLI subprocess stdin.
// Data should be newline-terminated JSON messages.
func (a *Adapter) Write(_ context.Context, data string) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.ready {
		return ErrNotConnected
	}

	if a.stdin == nil {
		return ErrStdinClosed
	}

	// Write data with newline
	_, err := io.WriteString(a.stdin, data+"\n")
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	// Close stdin if configured to do so
	if a.closeStdinAfterWrite {
		_ = a.stdin.Close()
		a.stdin = nil
	}

	return nil
}

// EndInput closes stdin to signal no more input.
// This is used in one-shot query mode.
func (a *Adapter) EndInput() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.stdin != nil {
		err := a.stdin.Close()
		a.stdin = nil

		return err
	}

	return nil
}

// ReadMessages reads JSON messages from stdout.
// Returns channels for messages and errors.
// Messages are raw map[string]any for flexibility.
func (a *Adapter) ReadMessages(
	ctx context.Context,
) (<-chan map[string]any, <-chan error) {
	msgCh := make(chan map[string]any, 10) //nolint:revive // Buffer size appropriate
	errCh := make(chan error, 1)

	go a.readLoop(ctx, msgCh, errCh)

	return msgCh, errCh
}

// readLoop implements the message reading logic.
// This runs in a goroutine and sends to channels.
func (a *Adapter) readLoop(
	ctx context.Context,
	msgCh chan<- map[string]any,
	errCh chan<- error,
) {
	defer close(msgCh)
	defer close(errCh)

	a.mu.RLock()
	stdout := a.stdout
	a.mu.RUnlock()

	if stdout == nil {
		errCh <- ErrNotConnected

		return
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), a.maxBufferSize) //nolint:revive // Standard buffer

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()

			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal(line, &msg); err != nil {
			// Skip invalid JSON lines (e.g., debug output)
			continue
		}

		select {
		case msgCh <- msg:
		case <-ctx.Done():
			errCh <- ctx.Err()

			return
		}
	}

	if err := scanner.Err(); err != nil {
		errCh <- fmt.Errorf("scan error: %w", err)
	}
}
