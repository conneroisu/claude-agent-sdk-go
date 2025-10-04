package cli

import (
	"context"
	"fmt"
	"os/exec"
)

// Connect establishes connection to Claude CLI subprocess.
// This method discovers the CLI binary, builds the command,
// and starts the subprocess with proper I/O pipes.
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.ready {
		return ErrAlreadyConnected
	}

	// Discover CLI binary
	if a.cliPath == "" {
		path, err := a.findCLI()
		if err != nil {
			return fmt.Errorf("CLI discovery failed: %w", err)
		}
		a.cliPath = path
	}

	// Build command arguments
	cmdArgs, err := a.BuildCommand()
	if err != nil {
		return fmt.Errorf("command build failed: %w", err)
	}

	// Create command with context
	a.cmd = exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)

	// Setup I/O pipes
	if err := a.setupPipes(); err != nil {
		return fmt.Errorf("pipe setup failed: %w", err)
	}

	// Start process
	if err := a.cmd.Start(); err != nil {
		return fmt.Errorf("process start failed: %w", err)
	}

	a.ready = true

	// Monitor process in background
	go a.monitorProcess()

	return nil
}

// setupPipes creates stdin, stdout, stderr pipes for the subprocess.
func (a *Adapter) setupPipes() error {
	stdin, err := a.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe failed: %w", err)
	}
	a.stdin = stdin

	stdout, err := a.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe failed: %w", err)
	}
	a.stdout = stdout

	stderr, err := a.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe failed: %w", err)
	}
	a.stderr = stderr

	return nil
}
