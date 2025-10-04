//nolint:revive // Test file - relaxed linting
package parse_test

import (
	"testing"

	"github.com/conneroisu/claude/pkg/claude/adapters/parse"
	"github.com/conneroisu/claude/pkg/claude/messages"
)

// TestParseUserMessage tests parsing user messages.
func TestParseUserMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		wantErr bool
	}{
		{
			name: "string content",
			input: map[string]any{
				"type": "user",
				"message": map[string]any{
					"content": "Hello",
				},
			},
			wantErr: false,
		},
		{
			name: "block content",
			input: map[string]any{
				"type": "user",
				"message": map[string]any{
					"content": []any{
						map[string]any{
							"type": "text",
							"text": "Hello",
						},
					},
				},
			},
			wantErr: false,
		},
	}

	adapter := parse.NewAdapter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := adapter.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if err == nil {
				if _, ok := msg.(*messages.UserMessage); !ok {
					t.Errorf("Expected UserMessage, got %T", msg)
				}
			}
		})
	}
}

// TestParseAssistantMessage tests parsing assistant messages.
func TestParseAssistantMessage(t *testing.T) {
	input := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model": "claude-sonnet-4",
			"content": []any{
				map[string]any{
					"type": "text",
					"text": "Hello",
				},
			},
		},
	}

	adapter := parse.NewAdapter()
	msg, err := adapter.Parse(input)

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	assistant, ok := msg.(*messages.AssistantMessage)
	if !ok {
		t.Fatalf("Expected AssistantMessage, got %T", msg)
	}

	if assistant.Model != "claude-sonnet-4" {
		t.Errorf("Expected model claude-sonnet-4, got %s", assistant.Model)
	}
}

// TestParseSystemMessage tests parsing system messages.
func TestParseSystemMessage(t *testing.T) {
	input := map[string]any{
		"type":    "system",
		"subtype": "init",
		"data": map[string]any{
			"session_id": "test-session",
		},
	}

	adapter := parse.NewAdapter()
	msg, err := adapter.Parse(input)

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	system, ok := msg.(*messages.SystemMessage)
	if !ok {
		t.Fatalf("Expected SystemMessage, got %T", msg)
	}

	if system.Subtype != "init" {
		t.Errorf("Expected subtype init, got %s", system.Subtype)
	}
}

// TestParseResultMessage tests parsing result messages.
func TestParseResultMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		subtype string
		wantErr bool
	}{
		{
			name: "success result",
			input: map[string]any{
				"type":    "result",
				"subtype": "success",
				"data":    map[string]any{},
			},
			subtype: "success",
			wantErr: false,
		},
		{
			name: "error result",
			input: map[string]any{
				"type":    "result",
				"subtype": "error_max_turns",
				"error":   "Max turns reached",
			},
			subtype: "error_max_turns",
			wantErr: false,
		},
	}

	adapter := parse.NewAdapter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := adapter.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParseStreamEvent tests parsing stream events.
func TestParseStreamEvent(t *testing.T) {
	input := map[string]any{
		"type":       "stream_event",
		"uuid":       "test-uuid",
		"session_id": "test-session",
		"event": map[string]any{
			"type": "content_block_delta",
		},
	}

	adapter := parse.NewAdapter()
	msg, err := adapter.Parse(input)

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	stream, ok := msg.(*messages.StreamEvent)
	if !ok {
		t.Fatalf("Expected StreamEvent, got %T", msg)
	}

	if stream.UUID != "test-uuid" {
		t.Errorf("Expected uuid test-uuid, got %s", stream.UUID)
	}
}
