package cli_test

import (
	"testing"

	"github.com/conneroisu/claude/pkg/claude/adapters/cli"
	"github.com/conneroisu/claude/pkg/claude/options"
)

func TestNewAdapter(t *testing.T) {
	tests := []struct {
		name string
		opts *options.AgentOptions
	}{
		{
			name: "with options",
			opts: &options.AgentOptions{
				Model: "claude-sonnet-4",
			},
		},
		{
			name: "nil options",
			opts: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := cli.NewAdapter(tt.opts)
			if adapter == nil {
				t.Fatal("NewAdapter() returned nil")
			}
		})
	}
}
