package parse_test

import (
	"testing"

	"github.com/conneroisu/claude/pkg/claude/adapters/parse"
	"github.com/conneroisu/claude/pkg/claude/messages"
)

func TestParseAssistant(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		wantErr bool
		check   func(*testing.T, *messages.AssistantMessage)
	}{
		{
			name: "with model and text",
			input: map[string]any{
				"type": "assistant",
				"message": map[string]any{
					"model": "claude-sonnet-4",
					"content": []any{
						map[string]any{"type": "text", "text": "Hello"},
					},
				},
			},
			check: func(t *testing.T, msg *messages.AssistantMessage) {
				if msg.Model == nil || *msg.Model != "claude-sonnet-4" {
					t.Errorf("model = %v, want claude-sonnet-4", msg.Model)
				}
				if len(msg.Content) != 1 {
					t.Errorf("content length = %d, want 1", len(msg.Content))
				}
			},
		},
		{
			name: "with stop reason",
			input: map[string]any{
				"type": "assistant",
				"message": map[string]any{
					"stop_reason": "end_turn",
					"content": []any{
						map[string]any{"type": "text", "text": "Done"},
					},
				},
			},
			check: func(t *testing.T, msg *messages.AssistantMessage) {
				// Note: stop_reason parsing not implemented yet
				// This test verifies the message parses without error
				if len(msg.Content) == 0 {
					t.Error("expected non-empty content")
				}
			},
		},
		{
			name: "missing message field",
			input: map[string]any{
				"type": "assistant",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parse.NewAdapter()
			msg, err := p.Parse(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if err != nil {
				return
			}

			assistantMsg, ok := msg.(*messages.AssistantMessage)
			if !ok {
				t.Fatalf("expected *AssistantMessage, got %T", msg)
			}

			if tt.check != nil {
				tt.check(t, assistantMsg)
			}
		})
	}
}

func TestParseToolUse(t *testing.T) {
	input := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"content": []any{
				map[string]any{
					"type": "tool_use",
					"id":   "tool_123",
					"name": "bash",
					"input": map[string]any{
						"command": "echo hello",
					},
				},
			},
		},
	}

	p := parse.NewAdapter()
	msg, err := p.Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	assistantMsg := msg.(*messages.AssistantMessage)
	if len(assistantMsg.Content) != 1 {
		t.Fatalf("content length = %d, want 1", len(assistantMsg.Content))
	}

	toolUse, ok := assistantMsg.Content[0].(*messages.ToolUseBlock)
	if !ok {
		t.Fatalf("expected *ToolUseBlock, got %T", assistantMsg.Content[0])
	}

	if toolUse.ID != "tool_123" {
		t.Errorf("ID = %s, want tool_123", toolUse.ID)
	}
	if toolUse.Name != "bash" {
		t.Errorf("Name = %s, want bash", toolUse.Name)
	}
	if toolUse.Input == nil {
		t.Error("Input is nil")
	}
}

func TestParseToolResult(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]any
		check func(*testing.T, *messages.ToolResultBlock)
	}{
		{
			name: "string content",
			input: map[string]any{
				"type": "assistant",
				"message": map[string]any{
					"content": []any{
						map[string]any{
							"type":        "tool_result",
							"tool_use_id": "tool_123",
							"content":     "result text",
						},
					},
				},
			},
			check: func(t *testing.T, tr *messages.ToolResultBlock) {
				if tr.ToolUseID != "tool_123" {
					t.Errorf("ToolUseID = %s, want tool_123", tr.ToolUseID)
				}
				if str, ok := tr.Content.(messages.ToolResultString); !ok {
					t.Errorf("Content type = %T, want ToolResultString", tr.Content)
				} else if string(str) != "result text" {
					t.Errorf("Content = %s, want 'result text'", str)
				}
			},
		},
		{
			name: "with error",
			input: map[string]any{
				"type": "assistant",
				"message": map[string]any{
					"content": []any{
						map[string]any{
							"type":        "tool_result",
							"tool_use_id": "tool_456",
							"is_error":    true,
							"content":     "error occurred",
						},
					},
				},
			},
			check: func(t *testing.T, tr *messages.ToolResultBlock) {
				if !tr.IsError {
					t.Error("IsError = false, want true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parse.NewAdapter()
			msg, err := p.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			assistantMsg := msg.(*messages.AssistantMessage)
			toolResult, ok := assistantMsg.Content[0].(*messages.ToolResultBlock)
			if !ok {
				t.Fatalf("expected *ToolResultBlock, got %T", assistantMsg.Content[0])
			}

			if tt.check != nil {
				tt.check(t, toolResult)
			}
		})
	}
}
