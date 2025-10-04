package cli

import "errors"

var (
	// ErrNotConnected indicates the transport is not connected.
	ErrNotConnected = errors.New("transport not connected")

	// ErrAlreadyConnected indicates connection already established.
	ErrAlreadyConnected = errors.New("already connected")

	// ErrCommandFailed indicates CLI command execution failed.
	ErrCommandFailed = errors.New("CLI command failed")

	// ErrProcessDied indicates the CLI process terminated unexpectedly.
	ErrProcessDied = errors.New("CLI process died unexpectedly")

	// ErrStdinClosed indicates stdin was closed before writing.
	ErrStdinClosed = errors.New("stdin already closed")

	// ErrReadTimeout indicates a read operation timed out.
	ErrReadTimeout = errors.New("read operation timed out")

	// ErrBufferOverflow indicates buffer size exceeded.
	ErrBufferOverflow = errors.New("buffer overflow")
)
