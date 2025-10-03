package cli

// EndInput signals no more input will be sent.
// Closes stdin to trigger EOF in the CLI subprocess.
func (a *Adapter) EndInput() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Close stdin if it exists.
	if a.stdin != nil {
		return a.stdin.Close()
	}

	return nil
}

// Close terminates the transport connection.
// Forcefully kills the subprocess if still running.
func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Mark as not ready to prevent new writes.
	a.ready = false

	// Close stdin to signal end of input.
	if a.stdin != nil {
		_ = a.stdin.Close()
	}

	// Kill process and wait for cleanup.
	if a.cmd != nil && a.cmd.Process != nil {
		_ = a.cmd.Process.Kill()
		_ = a.cmd.Wait()
	}

	return nil
}

// IsReady checks if transport is ready for communication.
// Returns true after successful Connect, false after Close.
func (a *Adapter) IsReady() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.ready
}
