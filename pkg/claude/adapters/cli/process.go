package cli

import "bufio"

// handleStderr continuously reads stderr and calls the callback.
// This runs in a goroutine started by Connect.
func (a *Adapter) handleStderr() {
	scanner := bufio.NewScanner(a.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if a.options.StderrCallback != nil {
			a.options.StderrCallback(line)
		}
	}
}
