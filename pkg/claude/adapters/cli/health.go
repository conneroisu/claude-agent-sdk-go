package cli

// Close terminates the CLI subprocess and releases resources.
func (a *Adapter) Close() error {
	if a.stdin != nil {
		_ = a.stdin.pipe.Close()
	}

	if a.cmd != nil && a.cmd.Process != nil {
		return a.cmd.Wait()
	}

	return nil
}

// IsReady returns true if the transport is connected.
func (a *Adapter) IsReady() bool {
	return a.cmd != nil && a.cmd.Process != nil
}
