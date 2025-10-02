// Package cli provides a CLI adapter for the Claude transport interface.
package cli

import (
	"io"
	"os/exec"
	"sync"

	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.Transport using CLI subprocess.
// It manages the lifecycle of a Claude CLI process, including:
// - Process initialization and command building
// - Bidirectional communication via stdin/stdout
// - Thread-safe state management
// - Buffer management for large responses.
type Adapter struct {
	options              *options.AgentOptions
	cliPath              string
	cmd                  *exec.Cmd
	stdin                io.WriteCloser
	stdout               io.ReadCloser
	stderr               io.ReadCloser
	ready                bool
	exitErr              error
	closeStdinAfterWrite bool // For one-shot queries
	mu                   sync.RWMutex
	maxBufferSize        int
}

// Verify interface compliance at compile time.
var _ ports.Transport = (*Adapter)(nil)

const (
	// defaultMaxBufferSize limits memory usage for streaming responses
	// Set to 1MB to handle typical Claude responses while preventing OOM.
	defaultMaxBufferSize = 1024 * 1024
	// claudeBinaryName is the expected name of the CLI executable.
	claudeBinaryName = "claude"
	// defaultMessageChannelBuffer is the buffer size for message channel.
	defaultMessageChannelBuffer = 10
)

// NewAdapter creates a new CLI transport adapter.
func NewAdapter(opts *options.AgentOptions) *Adapter {
	maxBuf := defaultMaxBufferSize
	if opts.MaxBufferSize != nil {
		maxBuf = *opts.MaxBufferSize
	}

	return &Adapter{
		options:       opts,
		maxBufferSize: maxBuf,
	}
}

// Close implements ports.Transport.
func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.ready = false

	// Close stdin
	if a.stdin != nil {
		_ = a.stdin.Close() // Best effort close
	}

	// Terminate process
	if a.cmd != nil && a.cmd.Process != nil {
		_ = a.cmd.Process.Kill() // Best effort kill
		_ = a.cmd.Wait()         // Best effort wait
	}

	return nil
}

// IsReady implements ports.Transport.
func (a *Adapter) IsReady() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.ready
}
