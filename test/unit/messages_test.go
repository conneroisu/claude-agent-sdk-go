package unit

import (
	"encoding/json"
	"testing"

	claudeagent "github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
	"github.com/google/uuid"
)

func TestTextContentBlockSerialization(t *testing.T) {
	block := claudeagent.TextContentBlock{
		Type: "text",
		Text: "Hello, world!",
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("failed to marshal text content block: %v", err)
	}

	var decoded claudeagent.TextContentBlock
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal text content block: %v", err)
	}

	if block.Type != decoded.Type || block.Text != decoded.Text {
		t.Errorf(
			"text content block mismatch: expected %+v, got %+v",
			block,
			decoded,
		)
	}
}

func TestToolUseContentBlockSerialization(t *testing.T) {
	block := claudeagent.ToolUseContentBlock{
		Type:  "tool_use",
		ID:    "tool-123",
		Name:  "Read",
		Input: json.RawMessage(`{"file_path":"/path/to/file"}`),
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("failed to marshal tool use content block: %v", err)
	}

	var decoded claudeagent.ToolUseContentBlock
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal tool use content block: %v", err)
	}

	if block.Type != decoded.Type ||
		block.ID != decoded.ID ||
		block.Name != decoded.Name {
		t.Errorf(
			"tool use content block mismatch: expected %+v, got %+v",
			block,
			decoded,
		)
	}
}

func TestToolResultContentText(t *testing.T) {
	text := "Success!"
	content := claudeagent.ToolResultContent{
		Text: &text,
	}

	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("failed to marshal tool result content: %v", err)
	}

	// Should be marshaled as a plain string
	var decoded string
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal as string: %v", err)
	}

	if decoded != text {
		t.Errorf("expected %s, got %s", text, decoded)
	}
}

func TestSDKUserMessageType(t *testing.T) {
	msg := claudeagent.SDKUserMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		Message: claudeagent.APIUserMessage{
			Role: "user",
			Content: []claudeagent.ContentBlock{
				claudeagent.TextContentBlock{
					Type: "text",
					Text: "Hello",
				},
			},
		},
	}

	if msg.Type() != "user" {
		t.Errorf("expected type 'user', got %s", msg.Type())
	}

	if msg.SessionID() != "test-session" {
		t.Errorf("expected session ID 'test-session', got %s", msg.SessionID())
	}
}

func TestSDKAssistantMessageType(t *testing.T) {
	msg := claudeagent.SDKAssistantMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		Message: claudeagent.APIAssistantMessage{
			ID:   "msg-123",
			Type: "message",
			Role: "assistant",
			Content: []claudeagent.ContentBlock{
				claudeagent.TextBlock{
					Type: "text",
					Text: "Hello back!",
				},
			},
			Model: "claude-sonnet-4-5",
			Usage: claudeagent.Usage{
				InputTokens:  10,
				OutputTokens: 5,
			},
		},
	}

	if msg.Type() != "assistant" {
		t.Errorf("expected type 'assistant', got %s", msg.Type())
	}

	if msg.Message.Model != "claude-sonnet-4-5" {
		t.Errorf("expected model 'claude-sonnet-4-5', got %s", msg.Message.Model)
	}
}

func TestSDKResultMessageType(t *testing.T) {
	result := "Task completed successfully"
	msg := claudeagent.SDKResultMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		Subtype:      claudeagent.ResultSubtypeSuccess,
		DurationMS:   5000,
		NumTurns:     3,
		TotalCostUSD: 0.0015,
		Result:       &result,
	}

	if msg.Type() != "result" {
		t.Errorf("expected type 'result', got %s", msg.Type())
	}

	if msg.Subtype != claudeagent.ResultSubtypeSuccess {
		t.Errorf("expected subtype 'success', got %s", msg.Subtype)
	}
}

func TestThinkingBlock(t *testing.T) {
	block := claudeagent.ThinkingBlock{
		Type:     "thinking",
		Thinking: "Let me think about this...",
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("failed to marshal thinking block: %v", err)
	}

	var decoded claudeagent.ThinkingBlock
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal thinking block: %v", err)
	}

	if block.Type != decoded.Type ||
		block.Thinking != decoded.Thinking {
		t.Errorf("thinking block mismatch: expected %+v, got %+v", block, decoded)
	}
}

func TestSDKPermissionDenial(t *testing.T) {
	denial := claudeagent.SDKPermissionDenial{
		ToolName:  "Write",
		ToolUseID: "tool-456",
		ToolInput: map[string]claudeagent.JSONValue{
			"file_path": json.RawMessage(`"/etc/passwd"`),
			"content":   json.RawMessage(`"malicious"`),
		},
	}

	data, err := json.Marshal(denial)
	if err != nil {
		t.Fatalf("failed to marshal permission denial: %v", err)
	}

	var decoded claudeagent.SDKPermissionDenial
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal permission denial: %v", err)
	}

	if denial.ToolName != decoded.ToolName || denial.ToolUseID != decoded.ToolUseID {
		t.Errorf("permission denial mismatch: expected %+v, got %+v", denial, decoded)
	}
}

func TestSDKControlRequest(t *testing.T) {
	msg := claudeagent.SDKControlRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		RequestID: "req-123",
		Request:   claudeagent.SDKControlInterruptRequest{},
	}

	if msg.Type() != "control_request" {
		t.Errorf("expected type 'control_request', got %s", msg.Type())
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal control request: %v", err)
	}

	var decoded claudeagent.SDKControlRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal control request: %v", err)
	}

	if decoded.RequestID != "req-123" {
		t.Errorf("expected request ID 'req-123', got %s", decoded.RequestID)
	}

	interruptReq, ok := decoded.Request.(claudeagent.SDKControlInterruptRequest)
	if !ok {
		t.Fatalf("expected SDKControlInterruptRequest, got %T", decoded.Request)
	}

	if interruptReq.Subtype() != "interrupt" {
		t.Errorf("expected subtype 'interrupt', got %s", interruptReq.Subtype())
	}
}

func TestSDKControlSetPermissionModeRequest(t *testing.T) {
	msg := claudeagent.SDKControlRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		RequestID: "req-456",
		Request: claudeagent.SDKControlSetPermissionModeRequest{
			Mode: "auto",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal set permission mode request: %v", err)
	}

	var decoded claudeagent.SDKControlRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal set permission mode request: %v", err)
	}

	modeReq, ok := decoded.Request.(claudeagent.SDKControlSetPermissionModeRequest)
	if !ok {
		t.Fatalf("expected SDKControlSetPermissionModeRequest, got %T", decoded.Request)
	}

	if modeReq.Mode != "auto" {
		t.Errorf("expected mode 'auto', got %s", modeReq.Mode)
	}
}

func TestSDKControlResponse(t *testing.T) {
	msg := claudeagent.SDKControlResponse{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		TypeField: "control_response",
		Response: claudeagent.ControlSuccessResponse{
			SubtypeField:   "success",
			RequestIDField: "req-123",
			Response: map[string]claudeagent.JSONValue{
				"status": json.RawMessage(`"ok"`),
			},
		},
	}

	if msg.Type() != "control_response" {
		t.Errorf("expected type 'control_response', got %s", msg.Type())
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal control response: %v", err)
	}

	var decoded claudeagent.SDKControlResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal control response: %v", err)
	}

	successResp, ok := decoded.Response.(claudeagent.ControlSuccessResponse)
	if !ok {
		t.Fatalf("expected ControlSuccessResponse, got %T", decoded.Response)
	}

	if successResp.RequestID() != "req-123" {
		t.Errorf("expected request ID 'req-123', got %s", successResp.RequestID())
	}

	if successResp.Subtype() != "success" {
		t.Errorf("expected subtype 'success', got %s", successResp.Subtype())
	}
}

func TestSDKControlErrorResponse(t *testing.T) {
	msg := claudeagent.SDKControlResponse{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		TypeField: "control_response",
		Response: claudeagent.ControlErrorResponse{
			SubtypeField:   "error",
			RequestIDField: "req-789",
			Error:          "Permission denied",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal control error response: %v", err)
	}

	var decoded claudeagent.SDKControlResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal control error response: %v", err)
	}

	errorResp, ok := decoded.Response.(claudeagent.ControlErrorResponse)
	if !ok {
		t.Fatalf("expected ControlErrorResponse, got %T", decoded.Response)
	}

	if errorResp.Error != "Permission denied" {
		t.Errorf("expected error 'Permission denied', got %s", errorResp.Error)
	}
}

func TestSDKControlPermissionRequest(t *testing.T) {
	msg := claudeagent.SDKControlPermissionRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		RequestIDField: "req-perm-1",
		SubtypeField:   "can_use_tool",
		ToolName:       "Write",
		Input: map[string]claudeagent.JSONValue{
			"file_path": json.RawMessage(`"/home/user/file.txt"`),
			"content":   json.RawMessage(`"Hello"`),
		},
	}

	if msg.Type() != "control_request" {
		t.Errorf("expected type 'control_request', got %s", msg.Type())
	}

	if msg.Subtype() != "can_use_tool" {
		t.Errorf("expected subtype 'can_use_tool', got %s", msg.Subtype())
	}

	if msg.RequestID() != "req-perm-1" {
		t.Errorf("expected request ID 'req-perm-1', got %s", msg.RequestID())
	}

	if msg.ToolName != "Write" {
		t.Errorf("expected tool name 'Write', got %s", msg.ToolName)
	}
}

func TestSDKHookCallbackRequest(t *testing.T) {
	msg := claudeagent.SDKHookCallbackRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		RequestIDField: "req-hook-1",
		SubtypeField:   "hook_callback",
		CallbackID:     "hook-123",
		Input:          json.RawMessage(`{"data":"test"}`),
	}

	if msg.Type() != "control_request" {
		t.Errorf("expected type 'control_request', got %s", msg.Type())
	}

	if msg.Subtype() != "hook_callback" {
		t.Errorf("expected subtype 'hook_callback', got %s", msg.Subtype())
	}

	if msg.RequestID() != "req-hook-1" {
		t.Errorf("expected request ID 'req-hook-1', got %s", msg.RequestID())
	}

	if msg.CallbackID != "hook-123" {
		t.Errorf("expected callback ID 'hook-123', got %s", msg.CallbackID)
	}
}
