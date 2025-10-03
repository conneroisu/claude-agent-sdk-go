// Package cli implements the CLI subprocess transport adapter.
//
// This adapter implements the Transport port using subprocess
// communication with the Claude Code CLI.
package cli

import (
	"io"
	"os/exec"
	"sync"

	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.Transport using CLI subprocess.
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
