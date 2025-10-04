package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// findCLI locates the Claude CLI binary.
// Checks PATH first, then common installation locations.
//
//nolint:revive // Receiver required for method interface consistency
func (a *Adapter) findCLI() (string, error) {
	_ = a
	// Check PATH first
	if path, err := exec.LookPath("claude"); err == nil {
		return path, nil
	}

	// Check common installation locations
	homeDir, _ := os.UserHomeDir()
	locations := []string{
		filepath.Join(homeDir, ".npm-global", "bin", "claude"),
		"/usr/local/bin/claude",
		filepath.Join(homeDir, ".local", "bin", "claude"),
		filepath.Join(homeDir, "node_modules", ".bin", "claude"),
		filepath.Join(homeDir, ".yarn", "bin", "claude"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	return "", fmt.Errorf(
		"claude CLI not found in PATH or common locations",
	)
}
