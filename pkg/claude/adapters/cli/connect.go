package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Connect establishes connection to the Claude CLI process.
// This method discovers the CLI binary, builds the command,
// and starts the subprocess with appropriate environment.
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.ready {
		return nil
	}

	// Find CLI
	cliPath, err := a.findCLI()
	if err != nil {
		return fmt.Errorf("CLI discovery failed: %w", err)
	}
	a.cliPath = cliPath

	// Build command
	cmdArgs, err := a.BuildCommand()
	if err != nil {
		return fmt.Errorf("command construction failed: %w", err)
	}

	// Set up environment
	env := a.buildEnvironment()

	// Create command
	a.cmd = exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	a.cmd.Env = env
	if a.options.Cwd != nil {
		a.cmd.Dir = *a.options.Cwd
	}

	// Set up pipes
	if err := a.setupPipes(); err != nil {
		return err
	}

	// Start process
	if err := a.cmd.Start(); err != nil {
		return fmt.Errorf("process start failed: %w", err)
	}

	// Start stderr handler if callback is set
	if a.options.StderrCallback != nil {
		go a.handleStderr()
	}

	a.ready = true
	return nil
}

func (a *Adapter) buildEnvironment() []string {
	env := os.Environ()
	env = append(env, "CLAUDE_CODE_ENTRYPOINT=sdk-go")
	for k, v := range a.options.Env {
		env = append(env, k+"="+v)
	}
	return env
}

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
