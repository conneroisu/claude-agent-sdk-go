// Package claude provides the public API for the Claude Agent SDK.
package claude

import (
	"errors"
	"fmt"
)

// Common errors returned by the SDK.
var (
	// ErrNotConnected indicates the client is not connected to Claude CLI.
	ErrNotConnected = errors.New("claude: not connected")

	// ErrCLINotFound indicates Claude CLI executable was not found.
	ErrCLINotFound = errors.New("claude: CLI not found")

	// ErrCLIConnection indicates connection to Claude CLI failed.
	ErrCLIConnection = errors.New("claude: connection failed")

	// ErrProcessFailed indicates the CLI process failed.
	ErrProcessFailed = errors.New("claude: process failed")

	// ErrJSONDecode indicates JSON decoding failed.
	ErrJSONDecode = errors.New("claude: JSON decode failed")

	// ErrMessageParse indicates message parsing failed.
	ErrMessageParse = errors.New("claude: message parse failed")

	// ErrControlTimeout indicates control request timeout.
	ErrControlTimeout = errors.New("claude: control request timeout")

	// ErrInvalidInput indicates invalid input was provided.
	ErrInvalidInput = errors.New("claude: invalid input")
)

// CLINotFoundError indicates Claude Code CLI was not found at expected path.
type CLINotFoundError struct {
	// Path is the path that was searched
	Path string
}

func (e *CLINotFoundError) Error() string {
	return fmt.Sprintf("Claude Code not found: %s", e.Path)
}

// ProcessError indicates the CLI process failed with an exit code.
type ProcessError struct {
	// ExitCode is the process exit code
	ExitCode int

	// Stderr contains error output from the process
	Stderr string
}

func (e *ProcessError) Error() string {
	return fmt.Sprintf(
		"process failed with exit code %d: %s",
		e.ExitCode,
		e.Stderr,
	)
}

// JSONDecodeError indicates JSON decoding failed.
type JSONDecodeError struct {
	// Line is the raw input line that failed to decode
	Line string

	// Err is the underlying error
	Err error
}

func (e *JSONDecodeError) Error() string {
	return fmt.Sprintf("failed to decode JSON: %v", e.Err)
}

func (e *JSONDecodeError) Unwrap() error {
	return e.Err
}
