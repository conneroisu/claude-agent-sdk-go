package parse_test

import (
	"testing"

	"github.com/conneroisu/claude/pkg/claude/adapters/parse"
	"github.com/conneroisu/claude/pkg/claude/internal/testutil"
	"github.com/conneroisu/claude/pkg/claude/messages"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		wantTyp string
		wantErr bool
	}{
		{
			name:    "assistant message",
			input:   testutil.AssistantMessageJSON,
			wantTyp: "assistant",
		},
		{
			name:    "user message",
			input:   testutil.UserMessageJSON,
			wantTyp: "user",
		},
		{
			name:    "system message",
			input:   testutil.SystemMessageJSON,
			wantTyp: "system",
		},
		{
			name:    "result message",
			input:   testutil.ResultMessageJSON,
			wantTyp: "result",
		},
		{
			name:    "stream event",
			input:   testutil.StreamEventJSON,
			wantTyp: "stream_event",
		},
		{
			name: "unknown type",
			input: map[string]any{
				"type": "unknown_type",
			},
			wantTyp: "unknown",
		},
		{
			name:    "missing type",
			input:   map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parse.NewAdapter()
			got, err := p.Parse(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if err != nil {
				return
			}

			if got == nil {
				t.Fatal("Parse() returned nil message")
			}

			switch tt.wantTyp {
			case "assistant":
				if _, ok := got.(*messages.AssistantMessage); !ok {
					t.Errorf("expected *AssistantMessage, got %T", got)
				}
			case "user":
				if _, ok := got.(*messages.UserMessage); !ok {
					t.Errorf("expected *UserMessage, got %T", got)
				}
			case "system":
				if _, ok := got.(*messages.SystemMessage); !ok {
					t.Errorf("expected *SystemMessage, got %T", got)
				}
			case "result":
				if _, ok := got.(messages.ResultMessage); !ok {
					t.Errorf("expected ResultMessage implementation, got %T", got)
				}
			case "stream_event":
				if _, ok := got.(*messages.StreamEvent); !ok {
					t.Errorf("expected *StreamEvent, got %T", got)
				}
			case "unknown":
				if _, ok := got.(*messages.UnknownMessage); !ok {
					t.Errorf("expected *UnknownMessage, got %T", got)
				}
			}
		})
	}
}

func TestParseContentBlocks(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]any
		wantCount int
		wantTypes []string
	}{
		{
			name:      "text block",
			input:     testutil.AssistantMessageJSON,
			wantCount: 1,
			wantTypes: []string{"text"},
		},
		{
			name:      "thinking and text",
			input:     testutil.ThinkingMessageJSON,
			wantCount: 2,
			wantTypes: []string{"thinking", "text"},
		},
		{
			name:      "tool use",
			input:     testutil.ToolUseMessageJSON,
			wantCount: 1,
			wantTypes: []string{"tool_use"},
		},
		{
			name:      "tool result",
			input:     testutil.ToolResultMessageJSON,
			wantCount: 1,
			wantTypes: []string{"tool_result"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parse.NewAdapter()
			msg, err := p.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			assistantMsg, ok := msg.(*messages.AssistantMessage)
			if !ok {
				t.Fatalf("expected *AssistantMessage, got %T", msg)
			}

			if len(assistantMsg.Content) != tt.wantCount {
				t.Errorf("content count = %d, want %d",
					len(assistantMsg.Content), tt.wantCount)
			}

			for i, wantType := range tt.wantTypes {
				if i >= len(assistantMsg.Content) {
					break
				}
				block := assistantMsg.Content[i]
				switch wantType {
				case "text":
					if _, ok := block.(*messages.TextBlock); !ok {
						t.Errorf("block[%d] type = %T, want *TextBlock", i, block)
					}
				case "thinking":
					if _, ok := block.(*messages.ThinkingBlock); !ok {
						t.Errorf("block[%d] type = %T, want *ThinkingBlock", i, block)
					}
				case "tool_use":
					if _, ok := block.(*messages.ToolUseBlock); !ok {
						t.Errorf("block[%d] type = %T, want *ToolUseBlock", i, block)
					}
				case "tool_result":
					if _, ok := block.(*messages.ToolResultBlock); !ok {
						t.Errorf("block[%d] type = %T, want *ToolResultBlock", i, block)
					}
				}
			}
		})
	}
}
