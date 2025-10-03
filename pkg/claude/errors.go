// Package claude provides the public API for the Claude Agent SDK.
//
// This package acts as a facade over the domain services, hiding
// the complexity of ports and adapters from SDK users.
package claude

import (
	"errors"
	"fmt"
)

var (
	// ErrNotConnected indicates the client is not connected.
	ErrNotConnected = errors.New("claude: not connected")

	// ErrCLINotFound indicates the Claude CLI was not found.
	ErrCLINotFound = errors.New("claude: CLI not found")

	// ErrCLIConnection indicates connection to CLI failed.
	ErrCLIConnection = errors.New("claude: connection failed")

	// ErrProcessFailed indicates the CLI process failed.
	ErrProcessFailed = errors.New("claude: process failed")

	// ErrJSONDecode indicates JSON decoding failed.
	ErrJSONDecode = errors.New("claude: JSON decode failed")

	// ErrMessageParse indicates message parsing failed.
	ErrMessageParse = errors.New("claude: message parse failed")

	// ErrControlTimeout indicates a control request timeout.
	ErrControlTimeout = errors.New(
		"claude: control request timeout",
	)

	// ErrInvalidInput indicates invalid input was provided.
	ErrInvalidInput = errors.New("claude: invalid input")
)

// CLINotFoundError indicates the Claude CLI was not found.
type CLINotFoundError struct {
	Path string
}

// Error implements the error interface.
func (e *CLINotFoundError) Error() string {
	return fmt.Sprintf("Claude Code not found: %s", e.Path)
}

// ProcessError indicates a CLI process error.
type ProcessError struct {
	ExitCode int
	Stderr   string
}

// Error implements the error interface.
func (e *ProcessError) Error() string {
	return fmt.Sprintf(
		"process failed with exit code %d: %s",
		e.ExitCode,
		e.Stderr,
	)
}

// JSONDecodeError indicates a JSON decoding error.
type JSONDecodeError struct {
	Line string
	Err  error
}

// Error implements the error interface.
func (e *JSONDecodeError) Error() string {
	return fmt.Sprintf("failed to decode JSON: %v", e.Err)
}

// Unwrap returns the underlying error.
func (e *JSONDecodeError) Unwrap() error {
	return e.Err
}
