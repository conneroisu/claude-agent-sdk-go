// Package claude provides the public API for Claude Agent SDK.
//
// This file defines error types used throughout the SDK.
package claude

import (
	"errors"
	"fmt"
)

// Common errors returned by the SDK.
var (
	// ErrNotConnected indicates the client is not connected to Claude CLI.
	ErrNotConnected = errors.New("claude: not connected")

	// ErrCLINotFound indicates the Claude CLI executable was not found.
	ErrCLINotFound = errors.New("claude: CLI not found")

	// ErrCLIConnection indicates connection to the CLI failed.
	ErrCLIConnection = errors.New("claude: connection failed")

	// ErrProcessFailed indicates the CLI process failed.
	ErrProcessFailed = errors.New("claude: process failed")

	// ErrJSONDecode indicates JSON decoding failed.
	ErrJSONDecode = errors.New("claude: JSON decode failed")

	// ErrMessageParse indicates message parsing failed.
	ErrMessageParse = errors.New("claude: message parse failed")

	// ErrControlTimeout indicates a control request timed out.
	ErrControlTimeout = errors.New("claude: control request timeout")

	// ErrInvalidInput indicates invalid user input.
	ErrInvalidInput = errors.New("claude: invalid input")
)

// CLINotFoundError indicates the Claude CLI was not found at the path.
type CLINotFoundError struct {
	Path string
}

func (e *CLINotFoundError) Error() string {
	return fmt.Sprintf("Claude Code not found: %s", e.Path)
}

// ProcessError indicates the CLI process failed with an exit code.
type ProcessError struct {
	ExitCode int
	Stderr   string
}

func (e *ProcessError) Error() string {
	return fmt.Sprintf(
		"process failed with exit code %d: %s",
		e.ExitCode,
		e.Stderr,
	)
}

// JSONDecodeError indicates JSON decoding failed on a specific line.
type JSONDecodeError struct {
	Line string
	Err  error
}

func (e *JSONDecodeError) Error() string {
	return fmt.Sprintf("failed to decode JSON: %v", e.Err)
}

func (e *JSONDecodeError) Unwrap() error {
	return e.Err
}
