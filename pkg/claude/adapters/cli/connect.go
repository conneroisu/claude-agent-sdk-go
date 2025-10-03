package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Connect establishes the CLI subprocess connection.
// This method locates the CLI binary, constructs the command,
// sets up pipes, and starts the subprocess.
//nolint:revive // function-length: complexity justified for connection setup
//nolint:revive // function-length: Complex initialization requires 26 statements
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Skip if already connected.
	if a.ready {
		return nil
	}

	// Locate the Claude CLI binary on the system.
	cliPath, err := a.findCLI()
	if err != nil {
		return fmt.Errorf("CLI discovery failed: %w", err)
	}
	a.cliPath = cliPath

	// Build command-line arguments from options.
	cmdArgs, err := a.BuildCommand()
	if err != nil {
		return fmt.Errorf(
			"command construction failed: %w",
			err,
		)
	}

	// Prepare environment variables for the subprocess.
	env := a.buildEnvironment()

	// Create the command with context for cancellation support.
	a.cmd = exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	a.cmd.Env = env

	// Set working directory if specified.
	if a.options.Cwd != nil {
		a.cmd.Dir = *a.options.Cwd
	}

	// Set up stdin/stdout/stderr pipes for communication.
	if err := a.setupPipes(); err != nil {
		return err
	}

	// Start the subprocess.
	if err := a.cmd.Start(); err != nil {
		return fmt.Errorf("process start failed: %w", err)
	}

	// Start stderr handler if callback provided.
	if a.options.StderrCallback != nil {
		go a.handleStderr()
	}

	// For non-streaming mode, close stdin after first write.
	if !a.options.IsStreaming {
		a.closeStdinAfterWrite = true
	}

	// Mark adapter as ready for communication.
	a.ready = true

	return nil
}

// buildEnvironment constructs environment variables.
// Includes parent process env plus SDK-specific variables.
func (a *Adapter) buildEnvironment() []string {
	// Start with current process environment.
	env := os.Environ()
	// Add SDK identifier for CLI to detect SDK usage.
	env = append(env, "CLAUDE_CODE_ENTRYPOINT=sdk-go")

	// Add user-specified environment variables.
	for k, v := range a.options.Env {
		env = append(env, k+"="+v)
	}

	return env
}

// setupPipes creates stdin/stdout/stderr pipes.
// All three pipes are required for communication with the CLI.
func (a *Adapter) setupPipes() error {
	// Create stdin pipe for sending messages.
	stdin, err := a.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe failed: %w", err)
	}
	a.stdin = stdin

	// Create stdout pipe for receiving messages.
	stdout, err := a.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe failed: %w", err)
	}
	a.stdout = stdout

	// Create stderr pipe for diagnostics.
	stderr, err := a.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe failed: %w", err)
	}
	a.stderr = stderr

	return nil
}
