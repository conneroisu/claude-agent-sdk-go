package transport

import (
	"bufio"
	"context"
	"fmt"
	"io"
)

// Transport handles communication with Claude Code process.
type Transport interface {
	// Read reads a message from the transport
	Read(ctx context.Context) ([]byte, error)

	// Write writes a message to the transport
	Write(ctx context.Context, data []byte) error

	// Close closes the transport
	Close() error
}

// StdioTransport implements Transport using stdio.
type StdioTransport struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	reader *bufio.Reader
}

// NewStdioTransport creates a new stdio transport.
func NewStdioTransport(
	stdin io.WriteCloser,
	stdout, stderr io.ReadCloser,
) *StdioTransport {
	return &StdioTransport{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		reader: bufio.NewReader(stdout),
	}
}

// Read reads a line-delimited JSON message from stdout.
func (t *StdioTransport) Read(ctx context.Context) ([]byte, error) {
	// Create a channel to receive the result
	type result struct {
		data []byte
		err  error
	}
	resultChan := make(chan result, 1)

	go func() {
		line, err := t.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				resultChan <- result{nil, err}

				return
			}
			resultChan <- result{
				nil,
				fmt.Errorf(errWrapFormat, ErrReadFailed, err),
			}

			return
		}
		resultChan <- result{line, nil}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultChan:
		return res.data, res.err
	}
}

// Write writes a line-delimited JSON message to stdin.
func (t *StdioTransport) Write(ctx context.Context, data []byte) error {
	// Create a channel to signal completion
	errChan := make(chan error, 1)

	go func() {
		// Add newline delimiter.
		message := append([]byte(nil), data...)
		message = append(message, '\n')
		_, err := t.stdin.Write(message)
		if err != nil {
			errChan <- fmt.Errorf(errWrapFormat, ErrWriteFailed, err)

			return
		}
		errChan <- nil
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// Close closes all streams.
func (t *StdioTransport) Close() error {
	err := t.stdin.Close()
	if err != nil {
		return err
	}
	err = t.stdout.Close()
	if err != nil {
		return err
	}
	err = t.stderr.Close()
	if err != nil {
		return err
	}

	return nil
}
