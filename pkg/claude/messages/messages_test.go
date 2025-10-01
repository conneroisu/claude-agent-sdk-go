package messages

import (
	"testing"
)

func TestMessageInterfaces(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
	}{
		{
			name: "UserMessage implements Message",
			msg:  &UserMessage{},
		},
		{
			name: "AssistantMessage implements Message",
			msg:  &AssistantMessage{},
		},
		{
			name: "SystemMessage implements Message",
			msg:  &SystemMessage{},
		},
		{
			name: "ResultMessage implements Message",
			msg:  &ResultMessage{},
		},
		{
			name: "StreamEvent implements Message",
			msg:  &StreamEvent{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// If this compiles, the interface is implemented
			var _ Message = tt.msg
		})
	}
}

func TestContentBlockInterfaces(t *testing.T) {
	tests := []struct {
		name  string
		block ContentBlock
	}{
		{
			name:  "TextBlock implements ContentBlock",
			block: TextBlock{},
		},
		{
			name:  "ThinkingBlock implements ContentBlock",
			block: ThinkingBlock{},
		},
		{
			name:  "ToolUseBlock implements ContentBlock",
			block: ToolUseBlock{},
		},
		{
			name:  "ToolResultBlock implements ContentBlock",
			block: ToolResultBlock{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var _ ContentBlock = tt.block
		})
	}
}

func TestMessageContentInterfaces(t *testing.T) {
	tests := []struct {
		name    string
		content MessageContent
	}{
		{
			name:    "StringContent implements MessageContent",
			content: StringContent("test"),
		},
		{
			name:    "BlockListContent implements MessageContent",
			content: BlockListContent{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var _ MessageContent = tt.content
		})
	}
}

func TestUserMessage(t *testing.T) {
	t.Run("with string content", func(t *testing.T) {
		msg := &UserMessage{
			Content: StringContent("Hello"),
		}

		if msg.Content == nil {
			t.Error("Content should not be nil")
		}
	})

	t.Run("with block list content", func(t *testing.T) {
		msg := &UserMessage{
			Content: BlockListContent{
				TextBlock{Text: "Hello"},
			},
		}

		if msg.Content == nil {
			t.Error("Content should not be nil")
		}
	})

	t.Run("with parent tool use ID", func(t *testing.T) {
		parentID := "tool_123"
		msg := &UserMessage{
			Content:         StringContent("Hello"),
			ParentToolUseID: &parentID,
		}

		if msg.ParentToolUseID == nil {
			t.Error("ParentToolUseID should not be nil")
		}
		if *msg.ParentToolUseID != "tool_123" {
			t.Errorf("Expected ParentToolUseID 'tool_123', got '%s'", *msg.ParentToolUseID)
		}
	})
}

func TestAssistantMessage(t *testing.T) {
	t.Run("with multiple content blocks", func(t *testing.T) {
		msg := &AssistantMessage{
			Content: []ContentBlock{
				TextBlock{Text: "Hello"},
				ThinkingBlock{Thinking: "Considering..."},
				ToolUseBlock{ID: "1", Name: "test", Input: map[string]any{}},
			},
			Model: "claude-3-5-sonnet",
		}

		if len(msg.Content) != 3 {
			t.Errorf("Expected 3 content blocks, got %d", len(msg.Content))
		}
		if msg.Model != "claude-3-5-sonnet" {
			t.Errorf("Expected model 'claude-3-5-sonnet', got '%s'", msg.Model)
		}
	})
}

func TestSystemMessage(t *testing.T) {
	t.Run("with data", func(t *testing.T) {
		msg := &SystemMessage{
			Subtype: "test",
			Data: map[string]any{
				"key": "value",
			},
		}

		if msg.Subtype != "test" {
			t.Errorf("Expected subtype 'test', got '%s'", msg.Subtype)
		}
		if val, ok := msg.Data["key"]; !ok || val != "value" {
			t.Error("Expected data to contain key='value'")
		}
	})
}

func TestResultMessage(t *testing.T) {
	t.Run("successful result", func(t *testing.T) {
		cost := 0.0015
		result := "Success"
		msg := &ResultMessage{
			Subtype:       "complete",
			DurationMs:    1500,
			DurationAPIMs: 1200,
			IsError:       false,
			NumTurns:      3,
			SessionID:     "sess_123",
			TotalCostUSD:  &cost,
			Result:        &result,
		}

		if msg.IsError {
			t.Error("Expected IsError to be false")
		}
		if msg.DurationMs != 1500 {
			t.Errorf("Expected DurationMs 1500, got %d", msg.DurationMs)
		}
		if msg.TotalCostUSD == nil || *msg.TotalCostUSD != 0.0015 {
			t.Error("Expected TotalCostUSD to be 0.0015")
		}
		if msg.Result == nil || *msg.Result != "Success" {
			t.Error("Expected Result to be 'Success'")
		}
	})

	t.Run("error result", func(t *testing.T) {
		msg := &ResultMessage{
			IsError: true,
		}

		if !msg.IsError {
			t.Error("Expected IsError to be true")
		}
	})
}

func TestStreamEvent(t *testing.T) {
	t.Run("basic stream event", func(t *testing.T) {
		msg := &StreamEvent{
			UUID:      "uuid_123",
			SessionID: "sess_123",
			Event: map[string]any{
				"type": "message_start",
			},
		}

		if msg.UUID != "uuid_123" {
			t.Errorf("Expected UUID 'uuid_123', got '%s'", msg.UUID)
		}
		if msg.SessionID != "sess_123" {
			t.Errorf("Expected SessionID 'sess_123', got '%s'", msg.SessionID)
		}
	})
}

func TestContentBlocks(t *testing.T) {
	t.Run("TextBlock", func(t *testing.T) {
		block := TextBlock{Text: "Hello world"}
		if block.Text != "Hello world" {
			t.Errorf("Expected text 'Hello world', got '%s'", block.Text)
		}
	})

	t.Run("ThinkingBlock", func(t *testing.T) {
		block := ThinkingBlock{
			Thinking:  "Let me think...",
			Signature: "sig_123",
		}
		if block.Thinking != "Let me think..." {
			t.Errorf("Expected thinking text, got '%s'", block.Thinking)
		}
	})

	t.Run("ToolUseBlock", func(t *testing.T) {
		block := ToolUseBlock{
			ID:   "tool_1",
			Name: "calculator",
			Input: map[string]any{
				"expression": "2+2",
			},
		}
		if block.Name != "calculator" {
			t.Errorf("Expected tool name 'calculator', got '%s'", block.Name)
		}
	})

	t.Run("ToolResultBlock with error", func(t *testing.T) {
		isError := true
		block := ToolResultBlock{
			ToolUseID: "tool_1",
			Content:   "Error occurred",
			IsError:   &isError,
		}
		if block.IsError == nil || !*block.IsError {
			t.Error("Expected IsError to be true")
		}
	})
}
