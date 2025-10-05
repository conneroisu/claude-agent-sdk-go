package hooking

import (
	"context"
	"fmt"
	"runtime/debug"
)

// hookResult holds the output and error from a hook callback execution.
type hookResult struct {
	output map[string]any
	err    error
}

// executeCallback runs a hook callback with panic recovery and context
// cancellation. It executes the callback in a goroutine to support timeouts
// and captures panics. Returns the callback output or an error if execution
// fails or context is cancelled.
//
//nolint:revive // 5 params needed for full hook execution context
func (*Service) executeCallback(
	ctx context.Context,
	callback HookCallback,
	input map[string]any,
	toolUseID *string,
	hookCtx HookContext,
) (map[string]any, error) {
	resultCh := make(chan hookResult, 1)

	// Execute callback in goroutine with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Capture stack trace for debugging
				stack := debug.Stack()
				resultCh <- hookResult{
					err: fmt.Errorf(
						"hook panicked: %v\nstack: %s",
						r,
						string(stack),
					),
				}
			}
		}()

		// Run the callback
		output, err := callback(input, toolUseID, hookCtx)
		resultCh <- hookResult{output: output, err: err}
	}()

	// Wait for result or context cancellation
	select {
	case result := <-resultCh:
		return result.output, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// mergeResults merges source map into destination map.
// Existing keys in dest are overwritten by values from src.
func (*Service) mergeResults(dest, src map[string]any) {
	for k, v := range src {
		dest[k] = v
	}
}
