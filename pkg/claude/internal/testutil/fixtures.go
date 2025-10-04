package testutil

// Common test data fixtures.
var (
	// AssistantMessageJSON is a sample assistant message.
	AssistantMessageJSON = map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model": "claude-sonnet-4",
			"content": []any{
				map[string]any{
					"type": "text",
					"text": "Hello, world!",
				},
			},
		},
	}

	// UserMessageJSON is a sample user message.
	UserMessageJSON = map[string]any{
		"type": "user",
		"message": map[string]any{
			"content": "Hello",
		},
	}

	// ResultMessageJSON is a sample result message.
	ResultMessageJSON = map[string]any{
		"type":        "result",
		"subtype":     "success",
		"duration_ms": 1234,
		"num_turns":   1,
		"session_id":  "test-session",
	}

	// StreamEventJSON is a sample stream event.
	StreamEventJSON = map[string]any{
		"type":       "stream_event",
		"uuid":       "test-uuid",
		"session_id": "test-session",
		"event": map[string]any{
			"type": "chunk",
		},
	}
)
