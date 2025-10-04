package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
)

// Write sends data to the CLI process via stdin.
// If closeStdinAfterWrite is set, stdin will be closed after writing.
func (a *Adapter) Write(_ context.Context, data string) error {
	a.mu.RLock()
	shouldClose := a.closeStdinAfterWrite
	a.mu.RUnlock()

	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.ready {
		return fmt.Errorf("transport not ready")
	}

	if a.exitErr != nil {
		return fmt.Errorf("transport has exited: %w", a.exitErr)
	}

	_, err := a.stdin.Write([]byte(data))
	if err != nil {
		return err
	}

	// Close stdin after write for one-shot queries
	if shouldClose {
		a.closeStdinAfterWrite = false
		_ = a.stdin.Close()
	}

	return nil
}

// ReadMessages continuously reads JSON messages from stdout.
// Returns channels for messages and errors.
func (a *Adapter) ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error) {
	msgCh := make(chan map[string]any, 10)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		scanner := bufio.NewScanner(a.stdout)
		scanBuf := make([]byte, 64*1024)
		scanner.Buffer(scanBuf, a.maxBufferSize)

		buffer := ""
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			line := scanner.Text()
			buffer += line

			if len(buffer) > a.maxBufferSize {
				errCh <- fmt.Errorf("message buffer exceeded %d bytes", a.maxBufferSize)
				return
			}

			var msg map[string]any
			if err := json.Unmarshal([]byte(buffer), &msg); err == nil {
				buffer = ""
				msgCh <- msg
			}
		}

		if err := scanner.Err(); err != nil {
			errCh <- err
		}

		if a.cmd != nil {
			if err := a.cmd.Wait(); err != nil {
				errCh <- fmt.Errorf("process exited with error: %w", err)
			}
		}
	}()

	return msgCh, errCh
}

// EndInput closes stdin to signal no more input will be sent.
func (a *Adapter) EndInput() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.stdin != nil {
		return a.stdin.Close()
	}
	return nil
}
