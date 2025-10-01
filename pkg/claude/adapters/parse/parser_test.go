package parse

import (
	"testing"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

func TestNewAdapter(t *testing.T) {
	adapter := NewAdapter()
	if adapter == nil {
		t.Fatal("Expected adapter to be created")
	}
}

func TestParse(t *testing.T) {
	adapter := NewAdapter()

	t.Run("parses user message with string content", func(t *testing.T) {
		data := map[string]any{
			"type":    "user",
			"content": "Hello",
		}

		msg, err := adapter.Parse(data)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		userMsg, ok := msg.(*messages.UserMessage)
		if !ok {
			t.Fatal("Expected UserMessage")
		}

		content, ok := userMsg.Content.(messages.StringContent)
		if !ok {
			t.Fatal("Expected StringContent")
		}
		if string(content) != "Hello" {
			t.Errorf("Expected content 'Hello', got '%s'", content)
		}
	})

	t.Run("parses assistant message with content blocks", func(t *testing.T) {
		data := map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "Hello there!",
					},
					map[string]any{
						"type":      "thinking",
						"thinking":  "Let me consider...",
						"signature": "sig_123",
					},
				},
				"model": "claude-3-5-sonnet",
			},
		}

		msg, err := adapter.Parse(data)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		assistantMsg, ok := msg.(*messages.AssistantMessage)
		if !ok {
			t.Fatal("Expected AssistantMessage")
		}

		if len(assistantMsg.Content) != 2 {
			t.Fatalf("Expected 2 content blocks, got %d", len(assistantMsg.Content))
		}

		textBlock, ok := assistantMsg.Content[0].(messages.TextBlock)
		if !ok {
			t.Fatal("Expected first block to be TextBlock")
		}
		if textBlock.Text != "Hello there!" {
			t.Errorf("Expected text 'Hello there!', got '%s'", textBlock.Text)
		}

		thinkingBlock, ok := assistantMsg.Content[1].(messages.ThinkingBlock)
		if !ok {
			t.Fatal("Expected second block to be ThinkingBlock")
		}
		if thinkingBlock.Thinking != "Let me consider..." {
			t.Errorf("Expected thinking text, got '%s'", thinkingBlock.Thinking)
		}
	})

	t.Run("parses tool use block", func(t *testing.T) {
		data := map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"content": []any{
					map[string]any{
						"type": "tool_use",
						"id":   "tool_123",
						"name": "calculator",
						"input": map[string]any{
							"expression": "2+2",
						},
					},
				},
				"model": "claude-3-5-sonnet",
			},
		}

		msg, err := adapter.Parse(data)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		assistantMsg, ok := msg.(*messages.AssistantMessage)
		if !ok {
			t.Fatal("Expected AssistantMessage")
		}

		toolBlock, ok := assistantMsg.Content[0].(messages.ToolUseBlock)
		if !ok {
			t.Fatal("Expected ToolUseBlock")
		}

		if toolBlock.ID != "tool_123" {
			t.Errorf("Expected ID 'tool_123', got '%s'", toolBlock.ID)
		}
		if toolBlock.Name != "calculator" {
			t.Errorf("Expected name 'calculator', got '%s'", toolBlock.Name)
		}
		if expr, ok := toolBlock.Input["expression"]; !ok || expr != "2+2" {
			t.Error("Expected input to contain expression='2+2'")
		}
	})

	t.Run("parses tool result block", func(t *testing.T) {
		isError := false
		data := map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"content": []any{
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": "tool_123",
						"content":     "Result: 4",
						"is_error":    isError,
					},
				},
				"model": "claude-3-5-sonnet",
			},
		}

		msg, err := adapter.Parse(data)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		assistantMsg, ok := msg.(*messages.AssistantMessage)
		if !ok {
			t.Fatal("Expected AssistantMessage")
		}

		resultBlock, ok := assistantMsg.Content[0].(messages.ToolResultBlock)
		if !ok {
			t.Fatal("Expected ToolResultBlock")
		}

		if resultBlock.ToolUseID != "tool_123" {
			t.Errorf("Expected tool_use_id 'tool_123', got '%s'", resultBlock.ToolUseID)
		}
		if resultBlock.Content != "Result: 4" {
			t.Errorf("Expected content 'Result: 4', got '%v'", resultBlock.Content)
		}
	})

	t.Run("parses system message", func(t *testing.T) {
		data := map[string]any{
			"type":    "system",
			"subtype": "notification",
			"data": map[string]any{
				"message": "System ready",
			},
		}

		msg, err := adapter.Parse(data)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		sysMsg, ok := msg.(*messages.SystemMessage)
		if !ok {
			t.Fatal("Expected SystemMessage")
		}

		if sysMsg.Subtype != "notification" {
			t.Errorf("Expected subtype 'notification', got '%s'", sysMsg.Subtype)
		}
		if val, ok := sysMsg.Data["message"]; !ok || val != "System ready" {
			t.Error("Expected data to contain message='System ready'")
		}
	})

	t.Run("parses result message", func(t *testing.T) {
		data := map[string]any{
			"type":            "result",
			"subtype":         "complete",
			"duration_ms":     1500.0,
			"duration_api_ms": 1200.0,
			"is_error":        false,
			"num_turns":       3.0,
			"session_id":      "sess_123",
			"total_cost_usd":  0.0015,
			"result":          "Success",
			"usage": map[string]any{
				"input_tokens":  100.0,
				"output_tokens": 50.0,
			},
		}

		msg, err := adapter.Parse(data)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		resultMsg, ok := msg.(*messages.ResultMessage)
		if !ok {
			t.Fatal("Expected ResultMessage")
		}

		if resultMsg.DurationMs != 1500 {
			t.Errorf("Expected duration_ms 1500, got %d", resultMsg.DurationMs)
		}
		if resultMsg.NumTurns != 3 {
			t.Errorf("Expected num_turns 3, got %d", resultMsg.NumTurns)
		}
		if resultMsg.SessionID != "sess_123" {
			t.Errorf("Expected session_id 'sess_123', got '%s'", resultMsg.SessionID)
		}
		if resultMsg.TotalCostUSD == nil || *resultMsg.TotalCostUSD != 0.0015 {
			t.Error("Expected total_cost_usd to be 0.0015")
		}
		if resultMsg.Result == nil || *resultMsg.Result != "Success" {
			t.Error("Expected result to be 'Success'")
		}
	})

	t.Run("parses stream event", func(t *testing.T) {
		data := map[string]any{
			"type":       "stream_event",
			"uuid":       "uuid_123",
			"session_id": "sess_123",
			"event": map[string]any{
				"type": "message_start",
			},
		}

		msg, err := adapter.Parse(data)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		streamEvent, ok := msg.(*messages.StreamEvent)
		if !ok {
			t.Fatal("Expected StreamEvent")
		}

		if streamEvent.UUID != "uuid_123" {
			t.Errorf("Expected uuid 'uuid_123', got '%s'", streamEvent.UUID)
		}
		if streamEvent.SessionID != "sess_123" {
			t.Errorf("Expected session_id 'sess_123', got '%s'", streamEvent.SessionID)
		}
	})

	t.Run("returns error for missing type", func(t *testing.T) {
		data := map[string]any{
			"content": "Hello",
		}

		_, err := adapter.Parse(data)
		if err == nil {
			t.Error("Expected error for missing type")
		}
	})

	t.Run("returns error for unknown type", func(t *testing.T) {
		data := map[string]any{
			"type": "unknown",
		}

		_, err := adapter.Parse(data)
		if err == nil {
			t.Error("Expected error for unknown type")
		}
	})

	t.Run("parses message with parent_tool_use_id", func(t *testing.T) {
		data := map[string]any{
			"type":              "user",
			"content":           "Hello",
			"parent_tool_use_id": "tool_parent_123",
		}

		msg, err := adapter.Parse(data)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		userMsg, ok := msg.(*messages.UserMessage)
		if !ok {
			t.Fatal("Expected UserMessage")
		}

		if userMsg.ParentToolUseID == nil {
			t.Fatal("Expected parent_tool_use_id to be set")
		}
		if *userMsg.ParentToolUseID != "tool_parent_123" {
			t.Errorf("Expected parent_tool_use_id 'tool_parent_123', got '%s'", *userMsg.ParentToolUseID)
		}
	})

	t.Run("handles tool result with error", func(t *testing.T) {
		isError := true
		data := map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"content": []any{
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": "tool_123",
						"content":     "Error occurred",
						"is_error":    isError,
					},
				},
				"model": "claude-3-5-sonnet",
			},
		}

		msg, err := adapter.Parse(data)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		assistantMsg, ok := msg.(*messages.AssistantMessage)
		if !ok {
			t.Fatal("Expected AssistantMessage")
		}

		resultBlock, ok := assistantMsg.Content[0].(messages.ToolResultBlock)
		if !ok {
			t.Fatal("Expected ToolResultBlock")
		}

		if resultBlock.IsError == nil || !*resultBlock.IsError {
			t.Error("Expected is_error to be true")
		}
	})
}

func TestParseContentBlock(t *testing.T) {
	adapter := NewAdapter()

	t.Run("returns error for invalid content block", func(t *testing.T) {
		_, err := adapter.parseContentBlock("not a map")
		if err == nil {
			t.Error("Expected error for invalid content block")
		}
	})

	t.Run("returns error for unknown content block type", func(t *testing.T) {
		block := map[string]any{
			"type": "unknown_type",
		}

		_, err := adapter.parseContentBlock(block)
		if err == nil {
			t.Error("Expected error for unknown content block type")
		}
	})
}
