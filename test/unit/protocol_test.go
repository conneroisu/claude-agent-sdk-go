package unit

import (
	"encoding/json"
	"testing"

	claudeagent "github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
	"github.com/google/uuid"
)

// Expected format from test:
//
//nolint:revive
//	```json
//	{"type":"user","message":{"role":"user","content":[{"type":"text","text":"What is 2+2?"}]}}
//	```

func TestUserMessageFormat(t *testing.T) {
	msg := claudeagent.SDKUserMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		TypeField: "user",
		Message: claudeagent.APIUserMessage{
			Role: "user",
			Content: []claudeagent.ContentBlock{
				claudeagent.TextContentBlock{
					Type: "text",
					Text: "What is 2+2?",
				},
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	t.Logf("User message JSON:\n%s", string(data))

	var envelope map[string]any
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Check it has the type field
	if typ, ok := envelope["type"]; !ok || typ != "user" {
		t.Errorf("expected type field to be 'user', got %v", typ)
	}

	// Check it has the message field
	if _, ok := envelope["message"]; !ok {
		t.Error("expected message field")
	}
}
