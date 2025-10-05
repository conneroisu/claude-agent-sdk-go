package cli

import (
	"fmt"
	"os/exec"
)

// findCLI locates the Claude CLI binary.
// It first checks if a custom path was provided in options,
// then falls back to searching the system PATH.
func (a *Adapter) findCLI() (string, error) {
	// Use custom CLI path if provided in options
	if a.opts.CLIPath != nil && *a.opts.CLIPath != "" {
		return *a.opts.CLIPath, nil
	}

	// Search for 'claude' binary in system PATH
	path, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude CLI not found in PATH: %w", err)
	}

	return path, nil
}
