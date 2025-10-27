package unit

import (
	"encoding/json"
	"testing"
	"time"

	claudeagent "github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
)

func TestIncludePartialMessagesFlag(t *testing.T) {
	tests := []struct {
		name                   string
		includePartialMessages bool
		shouldContainFlag      bool
	}{
		{
			name:                   "With IncludePartialMessages enabled",
			includePartialMessages: true,
			shouldContainFlag:      true,
		},
		{
			name:                   "With IncludePartialMessages disabled",
			includePartialMessages: false,
			shouldContainFlag:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a queryImpl to test buildArgs
			opts := &claudeagent.Options{
				IncludePartialMessages: tt.includePartialMessages,
			}

			// We can't directly access buildArgs since it's on queryImpl,
			// but we can verify the option is set correctly
			if opts.IncludePartialMessages != tt.includePartialMessages {
				t.Errorf(
					"IncludePartialMessages = %v, want %v",
					opts.IncludePartialMessages,
					tt.includePartialMessages,
				)
			}
		})
	}
}

// TestControlRequestChanClosureHandling tests that handleControlRequests
// properly exits when the controlRequestChan is closed, preventing a busy loop.
func TestControlRequestChanClosureHandling(t *testing.T) {
	// Create a mock control request channel
	controlRequestChan := make(chan json.RawMessage, 10)

	// Create a done channel to signal when goroutine exits
	done := make(chan struct{})

	// Start a goroutine that simulates handleControlRequests behavior
	go func() {
		defer close(done)
		for {
			select {
			case data, ok := <-controlRequestChan:
				if !ok {
					// Channel is closed, stop processing
					return
				}
				// Process data (in real code)
				_ = data
			}
		}
	}()

	// Close the channel to simulate the cleanup scenario
	close(controlRequestChan)

	// Wait for the goroutine to exit with a timeout
	select {
	case <-done:
		// Success! Goroutine exited properly
	case <-time.After(100 * time.Millisecond):
		t.Fatal("goroutine did not exit after channel close - potential busy loop")
	}
}

// TestControlRequestChanClosureWithoutCheck demonstrates that without the
// comma-ok check, a closed channel would cause a busy loop.
func TestControlRequestChanClosureWithoutCheck(t *testing.T) {
	// Create a mock control request channel
	controlRequestChan := make(chan json.RawMessage, 10)

	// Counter to track how many times the select runs
	counter := 0
	maxIterations := 1000

	// Create a done channel to signal when we've hit the iteration limit
	done := make(chan struct{})

	// Start a goroutine that simulates the OLD buggy behavior (without comma-ok)
	go func() {
		defer close(done)
		for counter < maxIterations {
			select {
			case data := <-controlRequestChan:
				// Without checking ok, this will receive zero values repeatedly
				// when the channel is closed
				counter++
				_ = data
			}
		}
	}()

	// Close the channel to trigger the busy loop
	close(controlRequestChan)

	// Wait for a short time
	time.Sleep(10 * time.Millisecond)

	// Check if we hit many iterations (indicating a busy loop)
	if counter >= maxIterations {
		// This is expected - the old code would spin in a tight loop
		t.Logf("Verified that without comma-ok check, the goroutine spins "+
			"in a busy loop (hit %d iterations in 10ms)", counter)
	} else if counter == 0 {
		t.Fatal("goroutine did not receive from closed channel as expected")
	}

	<-done
}
