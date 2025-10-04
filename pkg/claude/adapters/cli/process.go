package cli

import (
	"fmt"
)

// monitorProcess watches the CLI process and captures exit status.
// This runs in a goroutine started by Connect.
func (a *Adapter) monitorProcess() {
	if a.cmd == nil || a.cmd.Process == nil {
		return
	}

	err := a.cmd.Wait()

	a.mu.Lock()
	defer a.mu.Unlock()

	a.ready = false
	if err != nil {
		a.exitErr = fmt.Errorf("%w: %v", ErrProcessDied, err)
	}
}

// IsReady returns whether the transport is ready for I/O.
func (a *Adapter) IsReady() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.ready
}

// Close terminates the CLI process and cleans up resources.
// This method ensures graceful shutdown of the subprocess.
func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.ready {
		return nil
	}

	a.ready = false

	// Close stdin to signal end of input
	if a.stdin != nil {
		_ = a.stdin.Close()
	}

	// Kill process if still running
	if a.cmd != nil && a.cmd.Process != nil {
		_ = a.cmd.Process.Kill()
	}

	return nil
}
