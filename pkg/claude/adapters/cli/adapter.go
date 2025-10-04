package cli

import (
	"io"
	"os/exec"
	"sync"

	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.Transport using CLI subprocess.
// This adapter manages the lifecycle of a Claude CLI process,
// handling stdin/stdout communication and process management.
type Adapter struct {
	options              *options.AgentOptions
	cliPath              string
	cmd                  *exec.Cmd
	stdin                io.WriteCloser
	stdout               io.ReadCloser
	stderr               io.ReadCloser
	ready                bool
	exitErr              error
	closeStdinAfterWrite bool
	mu                   sync.RWMutex
	maxBufferSize        int
}

// Verify interface compliance at compile time.
var _ ports.Transport = (*Adapter)(nil)

const defaultMaxBufferSize = 1024 * 1024 // 1MB

// NewAdapter creates a new CLI transport adapter.
// The adapter must be connected via Connect() before use.
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

// IsReady returns true if the adapter is connected and ready.
func (a *Adapter) IsReady() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.ready
}

// Close terminates the CLI process and releases resources.
func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ready = false

	// Close stdin
	if a.stdin != nil {
		_ = a.stdin.Close()
	}

	// Terminate process
	if a.cmd != nil && a.cmd.Process != nil {
		_ = a.cmd.Process.Kill()
		_ = a.cmd.Wait()
	}

	return nil
}
