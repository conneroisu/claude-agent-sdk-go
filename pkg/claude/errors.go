package claude

import (
	"errors"
	"fmt"
)

var (
	ErrNotConnected   = errors.New("claude: not connected")
	ErrCLINotFound    = errors.New("claude: CLI not found")
	ErrCLIConnection  = errors.New("claude: connection failed")
	ErrProcessFailed  = errors.New("claude: process failed")
	ErrJSONDecode     = errors.New("claude: JSON decode failed")
	ErrMessageParse   = errors.New("claude: message parse failed")
	ErrControlTimeout = errors.New("claude: control request timeout")
	ErrInvalidInput   = errors.New("claude: invalid input")
)

// CLINotFoundError indicates the Claude CLI binary was not found
type CLINotFoundError struct {
	Path string
}

func (e *CLINotFoundError) Error() string {
	return fmt.Sprintf("Claude Code not found: %s", e.Path)
}

// ProcessError indicates the CLI process failed
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

// JSONDecodeError indicates JSON decoding failed
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
