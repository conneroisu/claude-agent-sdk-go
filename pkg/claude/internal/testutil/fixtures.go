// Package testutil provides test utilities and fixtures.
package testutil

// AssistantMessageJSON is a test fixture for an assistant message.
var AssistantMessageJSON = map[string]any{
	"type": "assistant",
	"message": map[string]any{
		"model": "claude-sonnet-4",
		"content": []any{
			map[string]any{"type": "text", "text": "Hello"},
		},
	},
}

// UserMessageJSON is a test fixture for a user message.
var UserMessageJSON = map[string]any{
	"type": "user",
	"message": map[string]any{
		"content": []any{
			map[string]any{"type": "text", "text": "Hello Claude"},
		},
	},
}

// ResultMessageSuccessJSON is a test fixture for a success result message.
var ResultMessageSuccessJSON = map[string]any{
	"type":        "result",
	"subtype":     "success",
	"duration_ms": 1234,
	"num_turns":   1,
	"session_id":  "test-session",
}

// ResultMessageErrorJSON is a test fixture for an error result message
var ResultMessageErrorJSON = map[string]any{
	"type":        "result",
	"subtype":     "error_during_execution",
	"duration_ms": 100,
	"num_turns":   0,
	"session_id":  "test-session",
	"error": map[string]any{
		"message": "Test error message",
		"type":    "api_error",
	},
}

// SystemMessageJSON is a test fixture for a system message
var SystemMessageJSON = map[string]any{
	"type":    "system",
	"subtype": "session_start",
	"system": map[string]any{
		"session_id": "test-session",
	},
}

// ToolUseBlockJSON is a test fixture for a tool use content block
var ToolUseBlockJSON = map[string]any{
	"type": "tool_use",
	"id":   "tool-123",
	"name": "bash",
	"input": map[string]any{
		"command": "echo hello",
	},
}

// ToolResultBlockJSON is a test fixture for a tool result content block
var ToolResultBlockJSON = map[string]any{
	"type":        "tool_result",
	"tool_use_id": "tool-123",
	"content":     "hello\n",
}

// ThinkingBlockJSON is a test fixture for a thinking content block
var ThinkingBlockJSON = map[string]any{
	"type":    "thinking",
	"thinking": "Let me analyze this...",
}
