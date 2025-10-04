// Package claude provides a Go SDK for Claude Agent.
package claude

import (
	"errors"
	"fmt"
)

// Common SDK errors.
var (
	// ErrNotConnected indicates the client is not connected to Claude CLI.
	ErrNotConnected = errors.New("claude: not connected")

	// ErrCLINotFound indicates the Claude CLI binary was not found.
	ErrCLINotFound = errors.New("claude: CLI not found")

	// ErrCLIConnection indicates a connection failure to Claude CLI.
	ErrCLIConnection = errors.New("claude: connection failed")

	// ErrProcessFailed indicates the CLI process failed.
	ErrProcessFailed = errors.New("claude: process failed")

	// ErrJSONDecode indicates a JSON decoding failure.
	ErrJSONDecode = errors.New("claude: JSON decode failed")

	// ErrMessageParse indicates a message parsing failure.
	ErrMessageParse = errors.New("claude: message parse failed")

	// ErrControlTimeout indicates a control request timeout.
	ErrControlTimeout = errors.New("claude: control request timeout")

	// ErrInvalidInput indicates invalid user input.
	ErrInvalidInput = errors.New("claude: invalid input")
)

// CLINotFoundError indicates the Claude CLI binary was not found.
//
// This error provides the path that was searched for troubleshooting.
type CLINotFoundError struct {
	// Path is the location where the CLI was expected.
	Path string
}

// Error implements the error interface.
func (e *CLINotFoundError) Error() string {
	return fmt.Sprintf("Claude Code not found: %s", e.Path)
}

// ProcessError indicates a CLI process failure.
//
// This error includes the exit code and stderr output for debugging.
type ProcessError struct {
	// ExitCode is the process exit code.
	ExitCode int

	// Stderr contains the process stderr output.
	Stderr string
}

// Error implements the error interface.
func (e *ProcessError) Error() string {
	return fmt.Sprintf(
		"process failed with exit code %d: %s",
		e.ExitCode,
		e.Stderr,
	)
}

// JSONDecodeError indicates a JSON decoding failure.
//
// This error wraps the underlying error and includes the problematic line.
type JSONDecodeError struct {
	// Line is the JSON line that failed to decode.
	Line string

	// Err is the underlying decoding error.
	Err error
}

// Error implements the error interface.
func (e *JSONDecodeError) Error() string {
	return fmt.Sprintf("failed to decode JSON: %v", e.Err)
}

// Unwrap returns the underlying error.
func (e *JSONDecodeError) Unwrap() error {
	return e.Err
}
