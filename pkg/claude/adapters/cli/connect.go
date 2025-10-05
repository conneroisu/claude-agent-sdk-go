package cli

import (
	"context"
	"fmt"
	"os/exec"
)

// Connect starts the Claude CLI subprocess and establishes I/O pipes.
// It locates the CLI binary, builds command arguments,
// and sets up stdin/stdout/stderr pipes for communication.
func (a *Adapter) Connect(ctx context.Context) error {
	// Locate the Claude CLI binary in PATH or custom location
	cliPath, err := a.findCLI()
	if err != nil {
		return err
	}
	a.cliPath = cliPath

	// Build command-line arguments from adapter options
	args, err := a.buildCommand()
	if err != nil {
		return err
	}

	// Create subprocess with context for cancellation support
	a.cmd = exec.CommandContext(ctx, cliPath, args...)

	// Set up bidirectional I/O pipes for process communication
	if err := a.setupPipes(); err != nil {
		return fmt.Errorf("setup pipes: %w", err)
	}

	// Start the CLI process in the background
	if err := a.cmd.Start(); err != nil {
		return fmt.Errorf("start CLI: %w", err)
	}

	return nil
}

// setupPipes creates stdin, stdout, and stderr pipes for the CLI subprocess.
// These pipes enable sending commands and receiving streaming responses.
func (a *Adapter) setupPipes() error {
	// Create stdin pipe for sending user input to CLI
	stdin, err := a.cmd.StdinPipe()
	if err != nil {
		return err
	}
	a.stdin = &inputWriter{pipe: stdin}

	// Create stdout pipe for receiving JSON-streamed messages
	stdout, err := a.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	a.stdout = &outputReader{pipe: stdout}

	// Create stderr pipe for capturing error output
	stderr, err := a.cmd.StderrPipe()
	if err != nil {
		return err
	}
	a.stderr = &errorReader{pipe: stderr}

	return nil
}
