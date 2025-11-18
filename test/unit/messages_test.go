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

// ============================================================================
// Extension Message Type Tests (5 new message types)
// ============================================================================

// TestSDKToolProgressMessageJSON verifies full JSON marshaling/unmarshaling
// for SDKToolProgressMessage with all fields populated.
func TestSDKToolProgressMessageJSON(t *testing.T) {
	parentToolUseID := "parent-tool-456"
	original := claudeagent.SDKToolProgressMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session-123",
		},
		TypeField:          "tool_progress",
		ToolUseID:          "tool-789",
		ToolName:           "Write",
		ParentToolUseID:    &parentToolUseID,
		ElapsedTimeSeconds: 3.14159,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal SDKToolProgressMessage: %v", err)
	}

	// Unmarshal back
	var unmarshaled claudeagent.SDKToolProgressMessage
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal SDKToolProgressMessage: %v", err)
	}

	// Verify all fields match
	if unmarshaled.UUIDField != original.UUIDField {
		t.Errorf("UUID mismatch: expected %v, got %v", original.UUIDField, unmarshaled.UUIDField)
	}
	if unmarshaled.SessionIDField != original.SessionIDField {
		t.Errorf("SessionID mismatch: expected %s, got %s", original.SessionIDField, unmarshaled.SessionIDField)
	}
	if unmarshaled.TypeField != original.TypeField {
		t.Errorf("TypeField mismatch: expected %s, got %s", original.TypeField, unmarshaled.TypeField)
	}
	if unmarshaled.ToolUseID != original.ToolUseID {
		t.Errorf("ToolUseID mismatch: expected %s, got %s", original.ToolUseID, unmarshaled.ToolUseID)
	}
	if unmarshaled.ToolName != original.ToolName {
		t.Errorf("ToolName mismatch: expected %s, got %s", original.ToolName, unmarshaled.ToolName)
	}
	if unmarshaled.ParentToolUseID == nil || *unmarshaled.ParentToolUseID != *original.ParentToolUseID {
		t.Errorf("ParentToolUseID mismatch: expected %v, got %v", original.ParentToolUseID, unmarshaled.ParentToolUseID)
	}
	if unmarshaled.ElapsedTimeSeconds != original.ElapsedTimeSeconds {
		t.Errorf("ElapsedTimeSeconds mismatch: expected %f, got %f", original.ElapsedTimeSeconds, unmarshaled.ElapsedTimeSeconds)
	}
}

// TestSDKToolProgressMessageNilParent verifies nil parent_tool_use_id handling.
func TestSDKToolProgressMessageNilParent(t *testing.T) {
	original := claudeagent.SDKToolProgressMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		TypeField:          "tool_progress",
		ToolUseID:          "tool-123",
		ToolName:           "Read",
		ParentToolUseID:    nil, // Explicitly nil
		ElapsedTimeSeconds: 1.5,
	}

	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled claudeagent.SDKToolProgressMessage
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.ParentToolUseID != nil {
		t.Errorf("expected nil ParentToolUseID, got %v", unmarshaled.ParentToolUseID)
	}
}

// TestSDKToolProgressMessageType verifies Type() method.
func TestSDKToolProgressMessageType(t *testing.T) {
	msg := claudeagent.SDKToolProgressMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		TypeField: "tool_progress",
		ToolUseID: "tool-123",
		ToolName:  "Bash",
	}

	if msg.Type() != "tool_progress" {
		t.Errorf("expected type 'tool_progress', got %s", msg.Type())
	}
}

// TestSDKToolProgressMessageInterface verifies SDKMessage interface implementation.
func TestSDKToolProgressMessageInterface(t *testing.T) {
	testUUID := uuid.New()
	msg := claudeagent.SDKToolProgressMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      testUUID,
			SessionIDField: "test-session-456",
		},
		ToolUseID: "tool-789",
		ToolName:  "Write",
	}

	if msg.UUID() != testUUID {
		t.Errorf("UUID() mismatch: expected %v, got %v", testUUID, msg.UUID())
	}
	if msg.SessionID() != "test-session-456" {
		t.Errorf("SessionID() mismatch: expected test-session-456, got %s", msg.SessionID())
	}
}

// TestSDKAuthStatusMessageJSON verifies full JSON marshaling/unmarshaling
// for SDKAuthStatusMessage with all fields populated.
func TestSDKAuthStatusMessageJSON(t *testing.T) {
	errorMsg := "authentication failed"
	original := claudeagent.SDKAuthStatusMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "auth-session-123",
		},
		TypeField:        "auth_status",
		IsAuthenticating: true,
		Output:           []string{"line1", "line2", "line3"},
		Error:            &errorMsg,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal SDKAuthStatusMessage: %v", err)
	}

	// Unmarshal back
	var unmarshaled claudeagent.SDKAuthStatusMessage
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal SDKAuthStatusMessage: %v", err)
	}

	// Verify all fields match
	if unmarshaled.UUIDField != original.UUIDField {
		t.Errorf("UUID mismatch: expected %v, got %v", original.UUIDField, unmarshaled.UUIDField)
	}
	if unmarshaled.SessionIDField != original.SessionIDField {
		t.Errorf("SessionID mismatch: expected %s, got %s", original.SessionIDField, unmarshaled.SessionIDField)
	}
	if unmarshaled.IsAuthenticating != original.IsAuthenticating {
		t.Errorf("IsAuthenticating mismatch: expected %v, got %v", original.IsAuthenticating, unmarshaled.IsAuthenticating)
	}
	if len(unmarshaled.Output) != len(original.Output) {
		t.Errorf("Output length mismatch: expected %d, got %d", len(original.Output), len(unmarshaled.Output))
	}
	for i, line := range original.Output {
		if unmarshaled.Output[i] != line {
			t.Errorf("Output[%d] mismatch: expected %s, got %s", i, line, unmarshaled.Output[i])
		}
	}
	if unmarshaled.Error == nil || *unmarshaled.Error != *original.Error {
		t.Errorf("Error mismatch: expected %v, got %v", original.Error, unmarshaled.Error)
	}
}

// TestSDKAuthStatusMessageNilError verifies nil error field handling.
func TestSDKAuthStatusMessageNilError(t *testing.T) {
	original := claudeagent.SDKAuthStatusMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "auth-session",
		},
		TypeField:        "auth_status",
		IsAuthenticating: false,
		Output:           []string{"success"},
		Error:            nil,
	}

	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled claudeagent.SDKAuthStatusMessage
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Error != nil {
		t.Errorf("expected nil Error, got %v", unmarshaled.Error)
	}
}

// TestSDKAuthStatusMessageType verifies Type() method.
func TestSDKAuthStatusMessageType(t *testing.T) {
	msg := claudeagent.SDKAuthStatusMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		TypeField:        "auth_status",
		IsAuthenticating: true,
		Output:           []string{},
	}

	if msg.Type() != "auth_status" {
		t.Errorf("expected type 'auth_status', got %s", msg.Type())
	}
}

// TestSDKAuthStatusMessageBoolValues verifies IsAuthenticating bool handling.
func TestSDKAuthStatusMessageBoolValues(t *testing.T) {
	tests := []struct {
		name             string
		isAuthenticating bool
	}{
		{"authenticating_true", true},
		{"authenticating_false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := claudeagent.SDKAuthStatusMessage{
				BaseMessage: claudeagent.BaseMessage{
					UUIDField:      uuid.New(),
					SessionIDField: "test-session",
				},
				IsAuthenticating: tt.isAuthenticating,
				Output:           []string{},
			}

			jsonBytes, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var unmarshaled claudeagent.SDKAuthStatusMessage
			if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if unmarshaled.IsAuthenticating != tt.isAuthenticating {
				t.Errorf("IsAuthenticating mismatch: expected %v, got %v", tt.isAuthenticating, unmarshaled.IsAuthenticating)
			}
		})
	}
}

// TestSDKStatusMessageJSON verifies full JSON marshaling/unmarshaling
// for SDKStatusMessage.
func TestSDKStatusMessageJSON(t *testing.T) {
	original := claudeagent.SDKStatusMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "status-session-123",
		},
		TypeField:    "system",
		SubtypeField: "status",
		Status:       claudeagent.SDKStatusCompacting,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal SDKStatusMessage: %v", err)
	}

	// Unmarshal back
	var unmarshaled claudeagent.SDKStatusMessage
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal SDKStatusMessage: %v", err)
	}

	// Verify all fields match
	if unmarshaled.UUIDField != original.UUIDField {
		t.Errorf("UUID mismatch: expected %v, got %v", original.UUIDField, unmarshaled.UUIDField)
	}
	if unmarshaled.SessionIDField != original.SessionIDField {
		t.Errorf("SessionID mismatch: expected %s, got %s", original.SessionIDField, unmarshaled.SessionIDField)
	}
	if unmarshaled.TypeField != original.TypeField {
		t.Errorf("TypeField mismatch: expected %s, got %s", original.TypeField, unmarshaled.TypeField)
	}
	if unmarshaled.SubtypeField != original.SubtypeField {
		t.Errorf("SubtypeField mismatch: expected %s, got %s", original.SubtypeField, unmarshaled.SubtypeField)
	}
	if unmarshaled.Status != original.Status {
		t.Errorf("Status mismatch: expected %s, got %s", original.Status, unmarshaled.Status)
	}
}

// TestSDKStatusMessageType verifies Type() method returns "system".
func TestSDKStatusMessageType(t *testing.T) {
	msg := claudeagent.SDKStatusMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		TypeField:    "system",
		SubtypeField: "status",
		Status:       claudeagent.SDKStatusCompacting,
	}

	if msg.Type() != "system" {
		t.Errorf("expected type 'system', got %s", msg.Type())
	}
}

// TestSDKStatusMessageSubtype verifies Subtype() method returns "status".
func TestSDKStatusMessageSubtype(t *testing.T) {
	msg := claudeagent.SDKStatusMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		TypeField:    "system",
		SubtypeField: "status",
		Status:       claudeagent.SDKStatusCompacting,
	}

	if msg.Subtype() != "status" {
		t.Errorf("expected subtype 'status', got %s", msg.Subtype())
	}
}

// TestSDKStatusMessageSDKStatusEnum verifies SDKStatus enum value.
func TestSDKStatusMessageSDKStatusEnum(t *testing.T) {
	msg := claudeagent.SDKStatusMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		TypeField:    "system",
		SubtypeField: "status",
		Status:       claudeagent.SDKStatusCompacting,
	}

	if msg.Status != "compacting" {
		t.Errorf("expected status 'compacting', got %s", msg.Status)
	}
}

// TestSDKHookResponseMessageJSON verifies full JSON marshaling/unmarshaling
// for SDKHookResponseMessage with all fields populated.
func TestSDKHookResponseMessageJSON(t *testing.T) {
	exitCode := 0
	original := claudeagent.SDKHookResponseMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "hook-session-123",
		},
		TypeField:    "system",
		SubtypeField: "hook_response",
		HookName:     "pre-commit",
		HookEvent:    "before_tool_use",
		Stdout:       "Hook executed successfully",
		Stderr:       "Warning: deprecated function",
		ExitCode:     &exitCode,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal SDKHookResponseMessage: %v", err)
	}

	// Unmarshal back
	var unmarshaled claudeagent.SDKHookResponseMessage
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal SDKHookResponseMessage: %v", err)
	}

	// Verify all fields match
	if unmarshaled.UUIDField != original.UUIDField {
		t.Errorf("UUID mismatch: expected %v, got %v", original.UUIDField, unmarshaled.UUIDField)
	}
	if unmarshaled.SessionIDField != original.SessionIDField {
		t.Errorf("SessionID mismatch: expected %s, got %s", original.SessionIDField, unmarshaled.SessionIDField)
	}
	if unmarshaled.TypeField != original.TypeField {
		t.Errorf("TypeField mismatch: expected %s, got %s", original.TypeField, unmarshaled.TypeField)
	}
	if unmarshaled.SubtypeField != original.SubtypeField {
		t.Errorf("SubtypeField mismatch: expected %s, got %s", original.SubtypeField, unmarshaled.SubtypeField)
	}
	if unmarshaled.HookName != original.HookName {
		t.Errorf("HookName mismatch: expected %s, got %s", original.HookName, unmarshaled.HookName)
	}
	if unmarshaled.HookEvent != original.HookEvent {
		t.Errorf("HookEvent mismatch: expected %s, got %s", original.HookEvent, unmarshaled.HookEvent)
	}
	if unmarshaled.Stdout != original.Stdout {
		t.Errorf("Stdout mismatch: expected %s, got %s", original.Stdout, unmarshaled.Stdout)
	}
	if unmarshaled.Stderr != original.Stderr {
		t.Errorf("Stderr mismatch: expected %s, got %s", original.Stderr, unmarshaled.Stderr)
	}
	if unmarshaled.ExitCode == nil || *unmarshaled.ExitCode != *original.ExitCode {
		t.Errorf("ExitCode mismatch: expected %v, got %v", original.ExitCode, unmarshaled.ExitCode)
	}
}

// TestSDKHookResponseMessageNilExitCode verifies nil exit_code handling.
func TestSDKHookResponseMessageNilExitCode(t *testing.T) {
	original := claudeagent.SDKHookResponseMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "hook-session",
		},
		TypeField:    "system",
		SubtypeField: "hook_response",
		HookName:     "pre-commit",
		HookEvent:    "before_query",
		Stdout:       "output",
		Stderr:       "",
		ExitCode:     nil,
	}

	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled claudeagent.SDKHookResponseMessage
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.ExitCode != nil {
		t.Errorf("expected nil ExitCode, got %v", unmarshaled.ExitCode)
	}
}

// TestSDKHookResponseMessageType verifies Type() method returns "system".
func TestSDKHookResponseMessageType(t *testing.T) {
	msg := claudeagent.SDKHookResponseMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		TypeField:    "system",
		SubtypeField: "hook_response",
		HookName:     "test-hook",
		HookEvent:    "test-event",
	}

	if msg.Type() != "system" {
		t.Errorf("expected type 'system', got %s", msg.Type())
	}
}

// TestSDKHookResponseMessageSubtype verifies Subtype() method.
func TestSDKHookResponseMessageSubtype(t *testing.T) {
	msg := claudeagent.SDKHookResponseMessage{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		TypeField:    "system",
		SubtypeField: "hook_response",
		HookName:     "test-hook",
		HookEvent:    "test-event",
	}

	if msg.Subtype() != "hook_response" {
		t.Errorf("expected subtype 'hook_response', got %s", msg.Subtype())
	}
}

// TestSDKUserMessageReplayJSON verifies full JSON marshaling/unmarshaling
// for SDKUserMessageReplay with all fields populated.
func TestSDKUserMessageReplayJSON(t *testing.T) {
	parentToolUseID := "parent-tool-123"
	original := claudeagent.SDKUserMessageReplay{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "replay-session-123",
		},
		TypeField: "user",
		Message: claudeagent.APIUserMessage{
			Role: "user",
			Content: []claudeagent.ContentBlock{
				claudeagent.TextContentBlock{
					Type: "text",
					Text: "This is a replayed message",
				},
			},
		},
		ParentToolUseID: &parentToolUseID,
		IsSynthetic:     true,
		IsReplay:        true,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal SDKUserMessageReplay: %v", err)
	}

	// Unmarshal back
	var unmarshaled claudeagent.SDKUserMessageReplay
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal SDKUserMessageReplay: %v", err)
	}

	// Verify all fields match
	if unmarshaled.UUIDField != original.UUIDField {
		t.Errorf("UUID mismatch: expected %v, got %v", original.UUIDField, unmarshaled.UUIDField)
	}
	if unmarshaled.SessionIDField != original.SessionIDField {
		t.Errorf("SessionID mismatch: expected %s, got %s", original.SessionIDField, unmarshaled.SessionIDField)
	}
	if unmarshaled.TypeField != original.TypeField {
		t.Errorf("TypeField mismatch: expected %s, got %s", original.TypeField, unmarshaled.TypeField)
	}
	if unmarshaled.ParentToolUseID == nil || *unmarshaled.ParentToolUseID != *original.ParentToolUseID {
		t.Errorf("ParentToolUseID mismatch: expected %v, got %v", original.ParentToolUseID, unmarshaled.ParentToolUseID)
	}
	if unmarshaled.IsSynthetic != original.IsSynthetic {
		t.Errorf("IsSynthetic mismatch: expected %v, got %v", original.IsSynthetic, unmarshaled.IsSynthetic)
	}
	if unmarshaled.IsReplay != original.IsReplay {
		t.Errorf("IsReplay mismatch: expected %v, got %v", original.IsReplay, unmarshaled.IsReplay)
	}
	if unmarshaled.Message.Role != original.Message.Role {
		t.Errorf("Message.Role mismatch: expected %s, got %s", original.Message.Role, unmarshaled.Message.Role)
	}
}

// TestSDKUserMessageReplayIsReplayAlwaysTrue verifies IsReplay is always true.
func TestSDKUserMessageReplayIsReplayAlwaysTrue(t *testing.T) {
	msg := claudeagent.SDKUserMessageReplay{
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
					Text: "test",
				},
			},
		},
		IsReplay: true,
	}

	if !msg.IsReplay {
		t.Errorf("IsReplay should always be true, got %v", msg.IsReplay)
	}
}

// TestSDKUserMessageReplayType verifies Type() method returns "user".
func TestSDKUserMessageReplayType(t *testing.T) {
	msg := claudeagent.SDKUserMessageReplay{
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
					Text: "test",
				},
			},
		},
		IsReplay: true,
	}

	if msg.Type() != "user" {
		t.Errorf("expected type 'user', got %s", msg.Type())
	}
}

// TestSDKUserMessageReplayInterface verifies SDKMessage interface implementation.
func TestSDKUserMessageReplayInterface(t *testing.T) {
	testUUID := uuid.New()
	msg := claudeagent.SDKUserMessageReplay{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      testUUID,
			SessionIDField: "replay-session-789",
		},
		Message: claudeagent.APIUserMessage{
			Role: "user",
			Content: []claudeagent.ContentBlock{
				claudeagent.TextContentBlock{
					Type: "text",
					Text: "test",
				},
			},
		},
		IsReplay: true,
	}

	if msg.UUID() != testUUID {
		t.Errorf("UUID() mismatch: expected %v, got %v", testUUID, msg.UUID())
	}
	if msg.SessionID() != "replay-session-789" {
		t.Errorf("SessionID() mismatch: expected replay-session-789, got %s", msg.SessionID())
	}
}
