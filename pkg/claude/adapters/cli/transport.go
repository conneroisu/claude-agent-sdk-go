// Package cli provides Claude CLI subprocess transport adapter.
package cli

import (
	"os/exec"

	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.Transport for Claude CLI subprocess.
type Adapter struct {
	opts    *options.AgentOptions
	cliPath string
	cmd     *exec.Cmd
	stdin   *inputWriter
	stdout  *outputReader
	stderr  *errorReader
}

// NewAdapter creates a new CLI transport adapter.
func NewAdapter(opts *options.AgentOptions) *Adapter {
	finalOpts := opts
	if finalOpts == nil {
		finalOpts = &options.AgentOptions{}
	}

	return &Adapter{
		opts: finalOpts,
	}
}

// Compile-time interface verification.
var _ ports.Transport = (*Adapter)(nil)
