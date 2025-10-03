package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Write sends data through the transport.
// For non-streaming mode, closes stdin after first write.
// This method is safe to call from multiple goroutines.
func (a *Adapter) Write(
	_ context.Context,
	data string,
) error {
	// Check close flag with read lock.
	a.mu.RLock()
	shouldClose := a.closeStdinAfterWrite
	a.mu.RUnlock()

	// Write with full lock.
	a.mu.Lock()
	defer a.mu.Unlock()

	// Verify transport is ready.
	if !a.ready {
		return errors.New("transport not ready")
	}

	// Verify process hasn't exited.
	if a.exitErr != nil {
		return fmt.Errorf("transport has exited: %w", a.exitErr)
	}

	// Write data to stdin.
	_, err := a.stdin.Write([]byte(data))
	if err != nil {
		return err
	}

	// Close stdin for one-shot queries.
	if shouldClose {
		a.closeStdinAfterWrite = false
		_ = a.stdin.Close()
	}

	return nil
}

// ReadMessages returns channels for receiving messages and errors.
// Reads JSON-RPC messages from stdout, accumulating partial lines.
// Messages are parsed and sent to msgCh, errors to errCh.
//nolint:revive // function-length,add-constant,early-return: justified for message reading
func (a *Adapter) ReadMessages(
	ctx context.Context,
) (<-chan map[string]any, <-chan error) {
	msgCh := make(chan map[string]any, 10)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		// Set up scanner with configurable buffer size.
		scanner := bufio.NewScanner(a.stdout)

		scanBuf := make([]byte, 64*1024)
		scanner.Buffer(scanBuf, a.maxBufferSize)

		// Accumulate partial lines until valid JSON.
		buffer := ""

		for scanner.Scan() {
			// Check for context cancellation.
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()

				return
			default:
			}

			// Append scanned line to buffer.
			line := scanner.Text()
			buffer += line

			// Prevent unbounded buffer growth.
			if len(buffer) > a.maxBufferSize {
				errCh <- fmt.Errorf(
					"message buffer exceeded %d bytes",
					a.maxBufferSize,
				)

				return
			}

			// Try to parse as JSON.
			var msg map[string]any
			if err := json.Unmarshal(
				[]byte(buffer),
				&msg,
			); err == nil {
				// Valid JSON, send and reset buffer.
				buffer = ""
				msgCh <- msg
			}
		}

		// Handle scanner errors.
		if err := scanner.Err(); err != nil {
			errCh <- err
		}

		// Wait for process to exit.
		if a.cmd != nil {
			if err := a.cmd.Wait(); err != nil {
				errCh <- fmt.Errorf(
					"process exited with error: %w",
					err,
				)
			}
		}
	}()

	return msgCh, errCh
}

// handleStderr processes stderr output.
// Runs in a separate goroutine and invokes callback for each line.
func (a *Adapter) handleStderr() {
	// Read stderr line by line.
	scanner := bufio.NewScanner(a.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		// Invoke callback if configured.
		if a.options.StderrCallback != nil {
			a.options.StderrCallback(line)
		}
	}
}
