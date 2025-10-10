package transport

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

const errWrapFormat = "%w: %w"

// Process represents a Claude Code subprocess.
type Process struct {
	cmd       *exec.Cmd
	transport Transport
	done      chan struct{}
	err       error
	errOnce   sync.Once
	mu        sync.Mutex
}

// ProcessConfig configures process spawning.
type ProcessConfig struct {
	Executable    string
	Args          []string
	Env           []string
	Cwd           string
	StderrHandler func(string)
}

// NewProcess spawns a new Claude Code process.
func NewProcess(
	ctx context.Context,
	config *ProcessConfig,
) (*Process, error) {
	if config == nil {
		return nil, ErrConfigRequired
	}

	executable, err := resolveExecutable(config.Executable)
	if err != nil {
		return nil, err
	}

	cmd := createCommand(ctx, executable, config)

	pipes, err := createPipes(cmd)
	if err != nil {
		return nil, err
	}

	transport := NewStdioTransport(pipes.stdin, pipes.stdout, pipes.stderr)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf(errWrapFormat, ErrProcessStart, err)
	}

	proc := &Process{
		cmd:       cmd,
		transport: transport,
		done:      make(chan struct{}),
	}

	if config.StderrHandler != nil {
		go proc.handleStderr(pipes.stderr, config.StderrHandler)
	}

	go proc.waitInternal()

	return proc, nil
}

// resolveExecutable determines the executable path to use.
func resolveExecutable(executable string) (string, error) {
	if executable != "" {
		return executable, nil
	}

	path, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf(errWrapFormat, ErrClaudeExecutableNotFound, err)
	}

	return path, nil
}

// createCommand creates and configures the exec.Cmd.
func createCommand(
	ctx context.Context,
	executable string,
	config *ProcessConfig,
) *exec.Cmd {
	cmd := exec.CommandContext(ctx, executable, config.Args...)

	if config.Cwd != "" {
		cmd.Dir = config.Cwd
	}

	if len(config.Env) > 0 {
		cmd.Env = append(os.Environ(), config.Env...)
	}

	return cmd
}

// pipeSet holds the stdin, stdout, and stderr pipes.
type pipeSet struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// createPipes creates stdin, stdout, and stderr pipes for the command.
func createPipes(cmd *exec.Cmd) (pipeSet, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return pipeSet{}, fmt.Errorf(errWrapFormat, ErrStdinPipe, err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return pipeSet{}, fmt.Errorf(errWrapFormat, ErrStdoutPipe, err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return pipeSet{}, fmt.Errorf(errWrapFormat, ErrStderrPipe, err)
	}

	return pipeSet{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}, nil
}

// handleStderr reads from stderr and calls the handler for each line.
func (*Process) handleStderr(stderr io.Reader, handler func(string)) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		handler(scanner.Text())
	}
}

// waitInternal waits for the process to complete.
func (p *Process) waitInternal() {
	err := p.cmd.Wait()
	p.errOnce.Do(func() {
		p.err = err
		close(p.done)
	})
}

// Transport returns the process transport.
func (p *Process) Transport() Transport {
	return p.transport
}

// Wait waits for the process to complete.
func (p *Process) Wait(ctx context.Context) error {
	select {
	case <-p.done:
		return p.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close kills the process and cleans up resources.
func (p *Process) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Close transport first
	if err := p.transport.Close(); err != nil {
		return fmt.Errorf(errWrapFormat, ErrTransportClose, err)
	}

	// Kill the process if it's still running
	if p.cmd.Process != nil {
		if err := p.cmd.Process.Kill(); err != nil {
			return fmt.Errorf(errWrapFormat, ErrProcessKill, err)
		}
	}

	// Wait for completion
	<-p.done

	return nil
}
