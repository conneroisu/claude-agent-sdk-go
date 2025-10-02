// Package cli provides a CLI adapter for the Claude transport interface.
package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// findCLI locates the Claude CLI binary
func findCLI() (string, error) {
	// Check PATH first
	if path, err := exec.LookPath(claudeBinaryName); err == nil {
		return path, nil
	}

	// Check common installation locations
	homeDir, _ := os.UserHomeDir()
	locations := []string{
		filepath.Join(homeDir, ".npm-global", "bin", claudeBinaryName),
		"/usr/local/bin/" + claudeBinaryName,
		filepath.Join(homeDir, ".local", "bin", claudeBinaryName),
		filepath.Join(homeDir, "node_modules", ".bin", claudeBinaryName),
		filepath.Join(homeDir, ".yarn", "bin", claudeBinaryName),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	return "", errors.New(
		"claude CLI not found in PATH or common locations",
	)
}

// Connect implements ports.Transport
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.ready {
		return nil
	}

	if err := a.setupCLIAndCommand(ctx); err != nil {
		return err
	}

	if err := a.setupPipes(); err != nil {
		return err
	}

	if err := a.startProcess(); err != nil {
		return err
	}

	a.configureStdinMode()
	a.ready = true

	return nil
}

// setupCLIAndCommand locates Claude CLI and builds exec.Cmd with
// context. CLAUDE_CODE_ENTRYPOINT identifies process as originating
// from Go SDK.
func (a *Adapter) setupCLIAndCommand(ctx context.Context) error {
	cliPath, err := findCLI()
	if err != nil {
		return fmt.Errorf("CLI discovery failed: %w", err)
	}
	a.cliPath = cliPath
	cmdArgs, err := a.BuildCommand()
	if err != nil {
		return fmt.Errorf("command construction failed: %w", err)
	}

	env := os.Environ()
	env = append(env, "CLAUDE_CODE_ENTRYPOINT=sdk-go")
	for k, v := range a.options.Env {
		env = append(env, k+"="+v)
	}

	a.cmd = exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	a.cmd.Env = env
	if a.options.Cwd != nil {
		a.cmd.Dir = *a.options.Cwd
	}

	return nil
}

// setupPipes establishes bidirectional communication channels.
// All three pipes must succeed for proper CLI interaction.
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

// startProcess launches the CLI subprocess and begins stderr monitoring.
// The stderr goroutine runs independently to avoid blocking.
func (a *Adapter) startProcess() error {
	if err := a.cmd.Start(); err != nil {
		return fmt.Errorf("process start failed: %w", err)
	}

	if a.options.StderrCallback != nil {
		go a.handleStderr()
	}

	return nil
}

// configureStdinMode determines stdin handling strategy.
// Non-streaming mode closes stdin after first write to signal EOF to CLI.
func (a *Adapter) configureStdinMode() {
	if !a.options.IsStreaming {
		a.closeStdinAfterWrite = true
	}
}

// handleStderr continuously reads stderr and invokes callback.
// Runs in separate goroutine to prevent blocking main communication.
func (a *Adapter) handleStderr() {
	scanner := bufio.NewScanner(a.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if a.options.StderrCallback != nil {
			a.options.StderrCallback(line)
		}
	}
}
