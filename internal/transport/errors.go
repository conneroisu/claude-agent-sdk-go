// Package transport provides process management and communication
// for Claude Code subprocesses.
package transport

import "errors"

var (
	// ErrClaudeExecutableNotFound is returned when the `claude` executable
	// cannot be found in PATH and no explicit path was provided.
	ErrClaudeExecutableNotFound = errors.New(
		"claude executable not found in PATH and no " +
			"PathToClaudeCodeExecutable provided",
	)

	// ErrConfigRequired is returned when process config is nil.
	ErrConfigRequired = errors.New("process config is required")

	// ErrStdinPipe is returned when stdin pipe creation fails.
	ErrStdinPipe = errors.New("failed to create stdin pipe")

	// ErrStdoutPipe is returned when stdout pipe creation fails.
	ErrStdoutPipe = errors.New("failed to create stdout pipe")

	// ErrStderrPipe is returned when stderr pipe creation fails.
	ErrStderrPipe = errors.New("failed to create stderr pipe")

	// ErrProcessStart is returned when process fails to start.
	ErrProcessStart = errors.New("failed to start process")

	// ErrTransportClose is returned when transport close fails.
	ErrTransportClose = errors.New("failed to close transport")

	// ErrProcessKill is returned when process kill fails.
	ErrProcessKill = errors.New("failed to kill process")

	// ErrReadFailed is returned when reading from stdout fails.
	ErrReadFailed = errors.New("failed to read from stdout")

	// ErrWriteFailed is returned when writing to stdin fails.
	ErrWriteFailed = errors.New("failed to write to stdin")
)
