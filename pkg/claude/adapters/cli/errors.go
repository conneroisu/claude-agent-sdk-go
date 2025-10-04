package cli

import "errors"

var (
	// ErrNotReady indicates the adapter is not ready for operations.
	ErrNotReady = errors.New("transport not ready")

	// ErrProcessExited indicates the CLI process has terminated.
	ErrProcessExited = errors.New("transport has exited")

	// ErrCLINotFound indicates the Claude CLI binary was not found.
	ErrCLINotFound = errors.New("claude CLI not found")

	// ErrBufferExceeded indicates a message exceeded max buffer size.
	ErrBufferExceeded = errors.New("message buffer exceeded limit")
)
