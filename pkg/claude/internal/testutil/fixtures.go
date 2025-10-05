package testutil

// Common test fixtures for message parsing and testing.
var (
	// AssistantMessageJSON represents a basic assistant message.
	AssistantMessageJSON = map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model": "claude-sonnet-4",
			"content": []any{
				map[string]any{"type": "text", "text": "Hello"},
			},
		},
	}

	// ThinkingMessageJSON represents an assistant message with thinking.
	ThinkingMessageJSON = map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model": "claude-sonnet-4",
			"content": []any{
				map[string]any{"type": "thinking", "thinking": "Let me think..."},
				map[string]any{"type": "text", "text": "The answer is 4"},
			},
		},
	}

	// ToolUseMessageJSON represents an assistant message with tool use.
	ToolUseMessageJSON = map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model": "claude-sonnet-4",
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

	// ToolResultMessageJSON represents an assistant message with tool result.
	ToolResultMessageJSON = map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model": "claude-sonnet-4",
			"content": []any{
				map[string]any{
					"type":        "tool_result",
					"tool_use_id": "tool_123",
					"content":     "hello",
				},
			},
		},
	}

	// ResultMessageJSON represents a successful result message.
	ResultMessageJSON = map[string]any{
		"type": "result",
		"result": map[string]any{
			"session_id": "test-session",
		},
	}

	// ErrorResultMessageJSON represents an error result message.
	ErrorResultMessageJSON = map[string]any{
		"type":      "result",
		"subtype":   "error",
		"message":   "Something went wrong",
		"num_turns": 1,
	}

	// UserMessageJSON represents a user message.
	UserMessageJSON = map[string]any{
		"type": "user",
		"message": map[string]any{
			"content": "Hello Claude",
		},
	}

	// SystemMessageJSON represents a system message.
	SystemMessageJSON = map[string]any{
		"type": "system",
		"message": map[string]any{
			"content": "You are a helpful assistant",
		},
	}

	// StreamEventJSON represents a stream event message.
	StreamEventJSON = map[string]any{
		"type":       "stream_event",
		"event_type": "content_block_start",
		"data": map[string]any{
			"index": 0,
		},
	}
)

// StringPtr returns a pointer to a string value.
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to an int value.
func IntPtr(i int) *int {
	return &i
}
