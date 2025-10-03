package parse_test

import (
	"testing"

	"github.com/conneroisu/claude/pkg/claude/adapters/parse"
	"github.com/conneroisu/claude/pkg/claude/internal/testutil"
	"github.com/conneroisu/claude/pkg/claude/messages"
)

func TestAdapter_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		wantType string
		wantErr bool
	}{
		{
			name:     "parse assistant message",
			input:    testutil.AssistantMessageJSON,
			wantType: "assistant",
			wantErr:  false,
		},
		{
			name:     "parse user message",
			input:    testutil.UserMessageJSON,
			wantType: "user",
			wantErr:  false,
		},
		{
			name:     "parse success result",
			input:    testutil.ResultMessageSuccessJSON,
			wantType: "result_success",
			wantErr:  false,
		},
		{
			name:     "parse error result",
			input:    testutil.ResultMessageErrorJSON,
			wantType: "result_error",
			wantErr:  false,
		},
		{
			name:     "parse system message",
			input:    testutil.SystemMessageJSON,
			wantType: "system",
			wantErr:  false,
		},
		{
			name: "unknown message type",
			input: map[string]any{
				"type": "unknown_type",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := parse.NewAdapter()
			got, err := parser.Parse(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if got == nil {
				t.Error("Parse() returned nil message")
				return
			}

			// Type-specific checks
			switch tt.wantType {
			case "assistant":
				if _, ok := got.(*messages.AssistantMessage); !ok {
					t.Errorf("Parse() got type %T, want *messages.AssistantMessage", got)
				}
			case "user":
				if _, ok := got.(*messages.UserMessage); !ok {
					t.Errorf("Parse() got type %T, want *messages.UserMessage", got)
				}
			case "result_success":
				if _, ok := got.(*messages.ResultMessageSuccess); !ok {
					t.Errorf("Parse() got type %T, want *messages.ResultMessageSuccess", got)
				}
			case "result_error":
				if _, ok := got.(*messages.ResultMessageError); !ok {
					t.Errorf("Parse() got type %T, want *messages.ResultMessageError", got)
				}
			case "system":
				if _, ok := got.(*messages.SystemMessage); !ok {
					t.Errorf("Parse() got type %T, want *messages.SystemMessage", got)
				}
			}
		})
	}
}

func TestAdapter_ParseContentBlocks(t *testing.T) {
	tests := []struct {
		name     string
		blocks   []any
		wantLen  int
		wantType string
	}{
		{
			name: "text block",
			blocks: []any{
				map[string]any{"type": "text", "text": "Hello"},
			},
			wantLen:  1,
			wantType: "text",
		},
		{
			name: "thinking block",
			blocks: []any{
				testutil.ThinkingBlockJSON,
			},
			wantLen:  1,
			wantType: "thinking",
		},
		{
			name: "tool use block",
			blocks: []any{
				testutil.ToolUseBlockJSON,
			},
			wantLen:  1,
			wantType: "tool_use",
		},
		{
			name: "tool result block",
			blocks: []any{
				testutil.ToolResultBlockJSON,
			},
			wantLen:  1,
			wantType: "tool_result",
		},
		{
			name: "mixed blocks",
			blocks: []any{
				map[string]any{"type": "text", "text": "First"},
				testutil.ThinkingBlockJSON,
				testutil.ToolUseBlockJSON,
			},
			wantLen:  3,
			wantType: "mixed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := parse.NewAdapter()
			input := map[string]any{
				"type": "assistant",
				"message": map[string]any{
					"model":   "claude-sonnet-4",
					"content": tt.blocks,
				},
			}

			got, err := parser.Parse(input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			assistantMsg, ok := got.(*messages.AssistantMessage)
			if !ok {
				t.Fatalf("Parse() got type %T, want *messages.AssistantMessage", got)
			}

			if len(assistantMsg.Content) != tt.wantLen {
				t.Errorf("Parse() got %d content blocks, want %d", len(assistantMsg.Content), tt.wantLen)
			}
		})
	}
}
