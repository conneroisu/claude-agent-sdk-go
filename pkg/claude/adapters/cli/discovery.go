package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// findCLI locates the Claude CLI binary.
// First checks PATH, then falls back to common installation locations.
// Returns the absolute path to the binary or an error if not found.
//nolint:revive,staticcheck // receiver-naming: Method interface requirement
func (*Adapter) findCLI() (string, error) {
	// Try PATH first (most common case).
	if path, err := exec.LookPath("claude"); err == nil {
		return path, nil
	}

	// Check common installation locations.
	homeDir, _ := os.UserHomeDir()
	locations := []string{
		// npm global install location.
		filepath.Join(homeDir, ".npm-global", "bin", "claude"),
		// System-wide install location.
		"/usr/local/bin/claude",
		// User-local install location.
		filepath.Join(homeDir, ".local", "bin", "claude"),
		// Project node_modules.
		filepath.Join(homeDir, "node_modules", ".bin", "claude"),
		// Yarn global install location.
		filepath.Join(homeDir, ".yarn", "bin", "claude"),
	}

	// Try each location in order.
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}
//nolint:revive // use-errors-new: formatted message provides context

	// CLI not found anywhere.
	return "", fmt.Errorf(
		"claude CLI not found in PATH or common locations",
	)
}
