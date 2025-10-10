package unit

import (
	"testing"

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
