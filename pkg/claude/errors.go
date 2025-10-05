// Package claude provides the main entry point for the Claude Agent SDK.
package claude

import (
	"errors"
	"fmt"
)

// Sentinel errors for programmatic error handling.
// Use errors.Is() to check for these error types.
var (
	// ErrNotConnected indicates the transport is not connected.
	ErrNotConnected = errors.New("claude: not connected")

	// ErrCLINotFound indicates the Claude Code executable was not found.
	ErrCLINotFound = errors.New("claude: CLI not found")

	// ErrCLIConnection indicates CLI connection failed.
	ErrCLIConnection = errors.New("claude: CLI connection failed")

	// ErrProcessFailed indicates the CLI process exited with an error.
	ErrProcessFailed = errors.New("claude: process failed")

	// ErrJSONDecode indicates invalid JSON from CLI.
	ErrJSONDecode = errors.New("claude: JSON decode error")

	// ErrMessageParse indicates message parsing failed.
	ErrMessageParse = errors.New("claude: message parse error")

	// ErrControlTimeout indicates control request timed out (60s).
	ErrControlTimeout = errors.New("claude: control request timeout")

	// ErrInvalidInput indicates user provided invalid input.
	ErrInvalidInput = errors.New("claude: invalid input")
)

// CLINotFoundError provides context about where we searched for the CLI.
// Use errors.As() to extract this type and access the Path field.
type CLINotFoundError struct {
	// Path is the location(s) where we searched for the CLI
	Path string
}

func (e *CLINotFoundError) Error() string {
	return fmt.Sprintf("Claude Code not found in: %s", e.Path)
}

func (*CLINotFoundError) Unwrap() error {
	return ErrCLINotFound
}

// ProcessError captures CLI execution failures.
// Use errors.As() to extract this type and access exit code and stderr.
type ProcessError struct {
	// ExitCode is the process exit code
	ExitCode int

	// Stderr contains the process stderr output
	Stderr string
}

func (e *ProcessError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf(
			"process failed with exit code %d: %s",
			e.ExitCode,
			e.Stderr,
		)
	}

	return fmt.Sprintf("process failed with exit code %d", e.ExitCode)
}

func (*ProcessError) Unwrap() error {
	return ErrProcessFailed
}

// JSONDecodeError captures JSON decoding failures.
// Use errors.As() to extract this type and access the failed line and cause.
type JSONDecodeError struct {
	// Line is the JSON line that failed to decode
	Line string

	// Cause is the underlying error
	Cause error
}

func (e *JSONDecodeError) Error() string {
	return fmt.Sprintf("failed to decode JSON: %v (line: %s)", e.Cause, e.Line)
}

func (*JSONDecodeError) Unwrap() error {
	return ErrJSONDecode
}

// MessageParseError captures message parsing failures.
// Use errors.As() to extract this type and access the message type.
type MessageParseError struct {
	// MessageType is the type of message that failed to parse
	MessageType string

	// Cause is the underlying error
	Cause error
}

func (e *MessageParseError) Error() string {
	return fmt.Sprintf(
		"failed to parse %s message: %v",
		e.MessageType,
		e.Cause,
	)
}

func (*MessageParseError) Unwrap() error {
	return ErrMessageParse
}

// InvalidInputError captures user input validation failures.
// Use errors.As() to extract this type and access the field and reason.
type InvalidInputError struct {
	// Field is the input field that is invalid
	Field string

	// Reason describes why the input is invalid
	Reason string
}

func (e *InvalidInputError) Error() string {
	return fmt.Sprintf("invalid input for %s: %s", e.Field, e.Reason)
}

func (*InvalidInputError) Unwrap() error {
	return ErrInvalidInput
}
