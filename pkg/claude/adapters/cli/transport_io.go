// Package cli provides a CLI adapter for the Claude transport interface.
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Write implements ports.Transport.
func (a *Adapter) Write(_ctx context.Context, data string) error {
	a.mu.RLock()
	shouldClose := a.closeStdinAfterWrite
	a.mu.RUnlock()

	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.ready {
		return errors.New("transport not ready")
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
		_ = a.stdin.Close() // Ignore close errors in one-shot mode
	}

	return nil
}

// ReadMessages implements ports.Transport.
func (a *Adapter) ReadMessages(
	ctx context.Context,
) (<-chan map[string]any, <-chan error) {
	msgCh := make(chan map[string]any, defaultMessageChannelBuffer)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		scanner := a.setupScanner()
		a.processLines(ctx, scanner, msgCh, errCh)
		a.handleScanCompletion(scanner, errCh)
	}()

	return msgCh, errCh
}

// setupScanner creates and configures a scanner for stdout reading.
// Configures buffer to handle large Claude responses up to maxBufferSize.
func (a *Adapter) setupScanner() *bufio.Scanner {
	scanner := bufio.NewScanner(a.stdout)
	scanBuf := make([]byte, 64*1024)
	scanner.Buffer(scanBuf, a.maxBufferSize)

	return scanner
}

// processLines reads and processes lines from scanner until completion.
// Accumulates partial JSON and sends complete messages to msgCh.
func (a *Adapter) processLines(
	ctx context.Context,
	scanner *bufio.Scanner,
	msgCh chan map[string]any,
	errCh chan error,
) {
	buffer := ""
	for scanner.Scan() {
		if ctx.Err() != nil {
			errCh <- ctx.Err()

			return
		}

		line := scanner.Text()
		buffer += line

		if len(buffer) > a.maxBufferSize {
			errCh <- fmt.Errorf(
				"message buffer exceeded %d bytes",
				a.maxBufferSize,
			)

			return
		}

		buffer = tryParseAndSend(buffer, msgCh)
	}
}

// tryParseAndSend attempts to parse buffer as JSON and send to channel.
// Returns empty string if successful, original buffer if parsing fails.
func tryParseAndSend(
	buffer string,
	msgCh chan map[string]any,
) string {
	var msg map[string]any
	if err := json.Unmarshal([]byte(buffer), &msg); err == nil {
		msgCh <- msg

		return ""
	}

	return buffer
}

// handleScanCompletion checks for scanner errors and process exit status.
// Sends any errors to errCh before goroutine completes.
func (a *Adapter) handleScanCompletion(
	scanner *bufio.Scanner,
	errCh chan error,
) {
	if err := scanner.Err(); err != nil {
		errCh <- err
	}

	if a.cmd == nil {
		return
	}
	if err := a.cmd.Wait(); err != nil {
		errCh <- fmt.Errorf("process exited with error: %w", err)
	}
}

// EndInput implements ports.Transport.
func (a *Adapter) EndInput() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.stdin != nil {
		return a.stdin.Close()
	}

	return nil
}
