package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// inputWriter wraps the stdin pipe for writing data to the CLI process.
type inputWriter struct {
	pipe io.WriteCloser
}

// Write sends data to the CLI stdin.
// Returns an error if the adapter is not connected.
func (a *Adapter) Write(_ context.Context, data string) error {
	if a.stdin == nil {
		return errors.New("not connected")
	}

	// Write raw bytes to stdin pipe
	_, err := a.stdin.pipe.Write([]byte(data))

	return err
}

// ReadMessages streams messages from CLI stdout.
// Returns two channels: one for messages and one for errors.
// Message channel closes when CLI exits or context is cancelled.
func (a *Adapter) ReadMessages(
	ctx context.Context,
) (<-chan map[string]any, <-chan error) {
	msgCh := make(chan map[string]any)
	errCh := make(chan error, 1)

	// Start background goroutine to read and parse messages
	go a.readLoop(ctx, msgCh, errCh)

	return msgCh, errCh
}

// readLoop continuously reads and parses JSON messages from stdout.
// It runs in a goroutine and closes channels when done.
func (a *Adapter) readLoop(
	ctx context.Context,
	msgCh chan<- map[string]any,
	errCh chan<- error,
) {
	defer close(msgCh)
	defer close(errCh)

	// Create scanner with large buffer for JSON messages
	scanner := bufio.NewScanner(a.stdout.pipe)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	// Read line-by-line from stdout
	for scanner.Scan() {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()

			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse JSON message
		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			errCh <- fmt.Errorf("parse JSON: %w", err)

			return
		}

		// Send parsed message to channel
		msgCh <- msg
	}

	// Report scanner errors
	if err := scanner.Err(); err != nil {
		errCh <- err
	}
}

// outputReader wraps the stdout pipe for reading CLI output.
type outputReader struct {
	pipe io.ReadCloser
}

// errorReader wraps the stderr pipe for reading CLI errors.
type errorReader struct {
	pipe io.ReadCloser
}

// EndInput closes stdin to signal no more input.
// This is typically called after sending all user messages.
func (a *Adapter) EndInput() error {
	if a.stdin != nil {
		return a.stdin.pipe.Close()
	}

	return nil
}
