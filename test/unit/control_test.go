package unit

import (
	"encoding/json"
	"testing"

	claudeagent "github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
	"github.com/google/uuid"
)

// Test SDKControlRequest marshaling and unmarshaling with interrupt request.
func TestSDKControlInterruptRequestSerialization(t *testing.T) {
	msg := claudeagent.SDKControlRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		RequestID: "req-123",
		Request:   claudeagent.SDKControlInterruptRequest{},
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal interrupt request: %v", err)
	}

	t.Logf("Interrupt request JSON:\n%s", string(data))

	// Unmarshal back
	var decoded claudeagent.SDKControlRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal interrupt request: %v", err)
	}

	// Verify request ID
	if decoded.RequestID != "req-123" {
		t.Errorf("expected request ID 'req-123', got %s", decoded.RequestID)
	}

	// Verify type
	if decoded.Type() != "control_request" {
		t.Errorf("expected type 'control_request', got %s", decoded.Type())
	}

	// Verify request variant
	interruptReq, ok := decoded.Request.(claudeagent.SDKControlInterruptRequest)
	if !ok {
		t.Fatalf("expected SDKControlInterruptRequest, got %T", decoded.Request)
	}

	if interruptReq.Subtype() != "interrupt" {
		t.Errorf("expected subtype 'interrupt', got %s", interruptReq.Subtype())
	}
}

// Test SDKControlInitializeRequest marshaling and unmarshaling.
func TestSDKControlInitializeRequestSerialization(t *testing.T) {
	hooks := map[string]claudeagent.JSONValue{
		"pre_tool_use":  json.RawMessage(`{"callback_id":"hook1"}`),
		"post_tool_use": json.RawMessage(`{"callback_id":"hook2"}`),
	}

	msg := claudeagent.SDKControlRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		RequestID: "req-init-1",
		Request: claudeagent.SDKControlInitializeRequest{
			Hooks: hooks,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal initialize request: %v", err)
	}

	t.Logf("Initialize request JSON:\n%s", string(data))

	// Unmarshal back
	var decoded claudeagent.SDKControlRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal initialize request: %v", err)
	}

	// Verify request variant
	initReq, ok := decoded.Request.(claudeagent.SDKControlInitializeRequest)
	if !ok {
		t.Fatalf("expected SDKControlInitializeRequest, got %T", decoded.Request)
	}

	if initReq.Subtype() != "initialize" {
		t.Errorf("expected subtype 'initialize', got %s", initReq.Subtype())
	}

	if len(initReq.Hooks) != 2 {
		t.Errorf("expected 2 hooks, got %d", len(initReq.Hooks))
	}
}

// Test SDKControlInitializeRequest with no hooks.
func TestSDKControlInitializeRequestNoHooks(t *testing.T) {
	msg := claudeagent.SDKControlRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		RequestID: "req-init-2",
		Request: claudeagent.SDKControlInitializeRequest{
			Hooks: nil,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal initialize request: %v", err)
	}

	// Unmarshal back
	var decoded claudeagent.SDKControlRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal initialize request: %v", err)
	}

	initReq, ok := decoded.Request.(claudeagent.SDKControlInitializeRequest)
	if !ok {
		t.Fatalf("expected SDKControlInitializeRequest, got %T", decoded.Request)
	}

	if len(initReq.Hooks) != 0 {
		t.Errorf("expected no hooks, got %d", len(initReq.Hooks))
	}
}

// Test SDKControlSetPermissionModeRequest marshaling and unmarshaling.
func TestSDKControlSetPermissionModeRequestSerialization(t *testing.T) {
	msg := claudeagent.SDKControlRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		RequestID: "req-perm-1",
		Request: claudeagent.SDKControlSetPermissionModeRequest{
			Mode: "acceptEdits",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal set permission mode request: %v", err)
	}

	t.Logf("Set permission mode request JSON:\n%s", string(data))

	// Unmarshal back
	var decoded claudeagent.SDKControlRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal set permission mode request: %v", err)
	}

	// Verify request variant
	modeReq, ok := decoded.Request.(claudeagent.SDKControlSetPermissionModeRequest)
	if !ok {
		t.Fatalf("expected SDKControlSetPermissionModeRequest, got %T", decoded.Request)
	}

	if modeReq.Subtype() != "set_permission_mode" {
		t.Errorf("expected subtype 'set_permission_mode', got %s", modeReq.Subtype())
	}

	if modeReq.Mode != "acceptEdits" {
		t.Errorf("expected mode 'acceptEdits', got %s", modeReq.Mode)
	}
}

// Test all permission modes.
func TestSDKControlSetPermissionModeRequestAllModes(t *testing.T) {
	modes := []string{
		string(claudeagent.PermissionModeDefault),
		string(claudeagent.PermissionModeAcceptEdits),
		string(claudeagent.PermissionModeBypassPermissions),
		string(claudeagent.PermissionModePlan),
	}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			msg := claudeagent.SDKControlRequest{
				BaseMessage: claudeagent.BaseMessage{
					UUIDField:      uuid.New(),
					SessionIDField: "test-session",
				},
				RequestID: "req-perm-mode",
				Request: claudeagent.SDKControlSetPermissionModeRequest{
					Mode: mode,
				},
			}

			data, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

			var decoded claudeagent.SDKControlRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal request: %v", err)
			}

			modeReq, ok := decoded.Request.(claudeagent.SDKControlSetPermissionModeRequest)
			if !ok {
				t.Fatalf("expected SDKControlSetPermissionModeRequest, got %T", decoded.Request)
			}

			if modeReq.Mode != mode {
				t.Errorf("expected mode '%s', got '%s'", mode, modeReq.Mode)
			}
		})
	}
}

// Test SDKControlMcpMessageRequest marshaling and unmarshaling.
func TestSDKControlMcpMessageRequestSerialization(t *testing.T) {
	msg := claudeagent.SDKControlRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		RequestID: "req-mcp-1",
		Request: claudeagent.SDKControlMcpMessageRequest{
			ServerName: "test-server",
			Message:    json.RawMessage(`{"method":"tools/list"}`),
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal mcp message request: %v", err)
	}

	t.Logf("MCP message request JSON:\n%s", string(data))

	// Unmarshal back
	var decoded claudeagent.SDKControlRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal mcp message request: %v", err)
	}

	// Verify request variant
	mcpReq, ok := decoded.Request.(claudeagent.SDKControlMcpMessageRequest)
	if !ok {
		t.Fatalf("expected SDKControlMcpMessageRequest, got %T", decoded.Request)
	}

	if mcpReq.Subtype() != "mcp_message" {
		t.Errorf("expected subtype 'mcp_message', got %s", mcpReq.Subtype())
	}

	if mcpReq.ServerName != "test-server" {
		t.Errorf("expected server name 'test-server', got %s", mcpReq.ServerName)
	}

	// Verify message content
	var msgContent map[string]interface{}
	if err := json.Unmarshal(mcpReq.Message, &msgContent); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if msgContent["method"] != "tools/list" {
		t.Errorf("expected method 'tools/list', got %v", msgContent["method"])
	}
}

// Test SDKControlResponse with success response.
func TestSDKControlSuccessResponseSerialization(t *testing.T) {
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
				"status":  json.RawMessage(`"ok"`),
				"version": json.RawMessage(`"1.0.0"`),
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal success response: %v", err)
	}

	t.Logf("Success response JSON:\n%s", string(data))

	// Verify the type field is present
	var rawMsg map[string]interface{}
	if err := json.Unmarshal(data, &rawMsg); err != nil {
		t.Fatalf("failed to unmarshal as map: %v", err)
	}

	typeField, ok := rawMsg["type"]
	if !ok {
		t.Error("type field is missing from JSON")
	} else if typeField != "control_response" {
		t.Errorf("expected type 'control_response', got %v", typeField)
	}

	// Unmarshal back
	var decoded claudeagent.SDKControlResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal success response: %v", err)
	}

	// Verify type
	if decoded.Type() != "control_response" {
		t.Errorf("expected type 'control_response', got %s", decoded.Type())
	}

	// Verify TypeField is set
	if decoded.TypeField != "control_response" {
		t.Errorf("expected TypeField 'control_response', got %s", decoded.TypeField)
	}

	// Verify response variant
	successResp, ok := decoded.Response.(claudeagent.ControlSuccessResponse)
	if !ok {
		t.Fatalf("expected ControlSuccessResponse, got %T", decoded.Response)
	}

	if successResp.Subtype() != "success" {
		t.Errorf("expected subtype 'success', got %s", successResp.Subtype())
	}

	if successResp.RequestID() != "req-123" {
		t.Errorf("expected request ID 'req-123', got %s", successResp.RequestID())
	}

	if len(successResp.Response) != 2 {
		t.Errorf("expected 2 response fields, got %d", len(successResp.Response))
	}
}

// Test SDKControlResponse with error response.
func TestSDKControlErrorResponseSerialization(t *testing.T) {
	msg := claudeagent.SDKControlResponse{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		TypeField: "control_response",
		Response: claudeagent.ControlErrorResponse{
			SubtypeField:   "error",
			RequestIDField: "req-456",
			Error:          "Permission denied: cannot access file",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal error response: %v", err)
	}

	t.Logf("Error response JSON:\n%s", string(data))

	// Verify the type field is present
	var rawMsg map[string]interface{}
	if err := json.Unmarshal(data, &rawMsg); err != nil {
		t.Fatalf("failed to unmarshal as map: %v", err)
	}

	typeField, ok := rawMsg["type"]
	if !ok {
		t.Error("type field is missing from JSON")
	} else if typeField != "control_response" {
		t.Errorf("expected type 'control_response', got %v", typeField)
	}

	// Unmarshal back
	var decoded claudeagent.SDKControlResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	// Verify TypeField is set
	if decoded.TypeField != "control_response" {
		t.Errorf("expected TypeField 'control_response', got %s", decoded.TypeField)
	}

	// Verify response variant
	errorResp, ok := decoded.Response.(claudeagent.ControlErrorResponse)
	if !ok {
		t.Fatalf("expected ControlErrorResponse, got %T", decoded.Response)
	}

	if errorResp.Subtype() != "error" {
		t.Errorf("expected subtype 'error', got %s", errorResp.Subtype())
	}

	if errorResp.RequestID() != "req-456" {
		t.Errorf("expected request ID 'req-456', got %s", errorResp.RequestID())
	}

	if errorResp.Error != "Permission denied: cannot access file" {
		t.Errorf("unexpected error message: %s", errorResp.Error)
	}
}

// Test SDKControlPermissionRequest marshaling and unmarshaling.
func TestSDKControlPermissionRequestSerialization(t *testing.T) {
	msg := claudeagent.SDKControlPermissionRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		RequestIDField: "req-perm-check-1",
		SubtypeField:   "can_use_tool",
		ToolName:       "Write",
		Input: map[string]claudeagent.JSONValue{
			"file_path": json.RawMessage(`"/home/user/file.txt"`),
			"content":   json.RawMessage(`"Hello, world!"`),
		},
		PermissionSuggestions: []claudeagent.JSONValue{
			json.RawMessage(
				`{"type":"addRules","rules":[{"toolName":"Write"}],"behavior":"allow"}`,
			),
		},
		BlockedPath: stringPtr("/home/user"),
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal permission request: %v", err)
	}

	t.Logf("Permission request JSON:\n%s", string(data))

	// Unmarshal back
	var decoded claudeagent.SDKControlPermissionRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal permission request: %v", err)
	}

	// Verify fields
	if decoded.Type() != "control_request" {
		t.Errorf("expected type 'control_request', got %s", decoded.Type())
	}

	if decoded.Subtype() != "can_use_tool" {
		t.Errorf("expected subtype 'can_use_tool', got %s", decoded.Subtype())
	}

	if decoded.RequestID() != "req-perm-check-1" {
		t.Errorf("expected request ID 'req-perm-check-1', got %s", decoded.RequestID())
	}

	if decoded.ToolName != "Write" {
		t.Errorf("expected tool name 'Write', got %s", decoded.ToolName)
	}

	if len(decoded.Input) != 2 {
		t.Errorf("expected 2 input fields, got %d", len(decoded.Input))
	}

	if len(decoded.PermissionSuggestions) != 1 {
		t.Errorf("expected 1 permission suggestion, got %d", len(decoded.PermissionSuggestions))
	}

	if decoded.BlockedPath == nil {
		t.Error("expected blocked path to be set")
	} else if *decoded.BlockedPath != "/home/user" {
		t.Errorf("expected blocked path '/home/user', got %s", *decoded.BlockedPath)
	}
}

// Test SDKControlPermissionRequest without optional fields.
func TestSDKControlPermissionRequestMinimal(t *testing.T) {
	msg := claudeagent.SDKControlPermissionRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		RequestIDField: "req-perm-check-2",
		SubtypeField:   "can_use_tool",
		ToolName:       "Read",
		Input: map[string]claudeagent.JSONValue{
			"file_path": json.RawMessage(`"/etc/passwd"`),
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal permission request: %v", err)
	}

	// Unmarshal back
	var decoded claudeagent.SDKControlPermissionRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal permission request: %v", err)
	}

	// Verify minimal fields
	if decoded.ToolName != "Read" {
		t.Errorf("expected tool name 'Read', got %s", decoded.ToolName)
	}

	if len(decoded.Input) != 1 {
		t.Errorf("expected 1 input field, got %d", len(decoded.Input))
	}

	if len(decoded.PermissionSuggestions) != 0 {
		t.Errorf("expected no permission suggestions, got %d", len(decoded.PermissionSuggestions))
	}

	if decoded.BlockedPath != nil {
		t.Errorf("expected no blocked path, got %v", *decoded.BlockedPath)
	}
}

// Test SDKHookCallbackRequest marshaling and unmarshaling.
func TestSDKHookCallbackRequestSerialization(t *testing.T) {
	toolUseID := "tool-use-123"
	msg := claudeagent.SDKHookCallbackRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		RequestIDField: "req-hook-1",
		SubtypeField:   "hook_callback",
		CallbackID:     "pre_tool_use_callback_1",
		Input: json.RawMessage(
			`{"tool_name":"Read","input":{"file_path":"/path/to/file"}}`,
		),
		ToolUseID: &toolUseID,
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal hook callback request: %v", err)
	}

	t.Logf("Hook callback request JSON:\n%s", string(data))

	// Unmarshal back
	var decoded claudeagent.SDKHookCallbackRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal hook callback request: %v", err)
	}

	// Verify fields
	if decoded.Type() != "control_request" {
		t.Errorf("expected type 'control_request', got %s", decoded.Type())
	}

	if decoded.Subtype() != "hook_callback" {
		t.Errorf("expected subtype 'hook_callback', got %s", decoded.Subtype())
	}

	if decoded.RequestID() != "req-hook-1" {
		t.Errorf("expected request ID 'req-hook-1', got %s", decoded.RequestID())
	}

	if decoded.CallbackID != "pre_tool_use_callback_1" {
		t.Errorf("expected callback ID 'pre_tool_use_callback_1', got %s", decoded.CallbackID)
	}

	if decoded.ToolUseID == nil {
		t.Error("expected tool use ID to be set")
	} else if *decoded.ToolUseID != "tool-use-123" {
		t.Errorf("expected tool use ID 'tool-use-123', got %s", *decoded.ToolUseID)
	}

	// Verify input content
	var inputContent map[string]interface{}
	if err := json.Unmarshal(decoded.Input, &inputContent); err != nil {
		t.Fatalf("failed to unmarshal input: %v", err)
	}

	if inputContent["tool_name"] != "Read" {
		t.Errorf("expected tool name 'Read', got %v", inputContent["tool_name"])
	}
}

// Test SDKHookCallbackRequest without optional tool_use_id.
func TestSDKHookCallbackRequestNoToolUseID(t *testing.T) {
	msg := claudeagent.SDKHookCallbackRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: "test-session",
		},
		RequestIDField: "req-hook-2",
		SubtypeField:   "hook_callback",
		CallbackID:     "session_start_callback",
		Input:          json.RawMessage(`{"session_id":"test-session"}`),
		ToolUseID:      nil,
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal hook callback request: %v", err)
	}

	// Unmarshal back
	var decoded claudeagent.SDKHookCallbackRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal hook callback request: %v", err)
	}

	if decoded.ToolUseID != nil {
		t.Errorf("expected no tool use ID, got %v", *decoded.ToolUseID)
	}
}

// Test control request variant type discrimination.
func TestControlRequestVariantTypes(t *testing.T) {
	testCases := []struct {
		name            string
		request         claudeagent.ControlRequestVariant
		expectedSubtype string
	}{
		{
			name:            "Interrupt",
			request:         claudeagent.SDKControlInterruptRequest{},
			expectedSubtype: "interrupt",
		},
		{
			name:            "Initialize",
			request:         claudeagent.SDKControlInitializeRequest{},
			expectedSubtype: "initialize",
		},
		{
			name: "SetPermissionMode",
			request: claudeagent.SDKControlSetPermissionModeRequest{
				Mode: "default",
			},
			expectedSubtype: "set_permission_mode",
		},
		{
			name: "McpMessage",
			request: claudeagent.SDKControlMcpMessageRequest{
				ServerName: "test",
				Message:    json.RawMessage(`{}`),
			},
			expectedSubtype: "mcp_message",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := claudeagent.SDKControlRequest{
				BaseMessage: claudeagent.BaseMessage{
					UUIDField:      uuid.New(),
					SessionIDField: "test-session",
				},
				RequestID: "req-test",
				Request:   tc.request,
			}

			data, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

			var decoded claudeagent.SDKControlRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal request: %v", err)
			}

			if decoded.Request.Subtype() != tc.expectedSubtype {
				t.Errorf(
					"expected subtype '%s', got '%s'",
					tc.expectedSubtype,
					decoded.Request.Subtype(),
				)
			}
		})
	}
}

// Test control response variant type discrimination.
func TestControlResponseVariantTypes(t *testing.T) {
	testCases := []struct {
		name            string
		response        claudeagent.ControlResponseVariant
		expectedSubtype string
		expectedReqID   string
	}{
		{
			name: "Success",
			response: claudeagent.ControlSuccessResponse{
				SubtypeField:   "success",
				RequestIDField: "req-success",
				Response:       map[string]claudeagent.JSONValue{},
			},
			expectedSubtype: "success",
			expectedReqID:   "req-success",
		},
		{
			name: "Error",
			response: claudeagent.ControlErrorResponse{
				SubtypeField:   "error",
				RequestIDField: "req-error",
				Error:          "test error",
			},
			expectedSubtype: "error",
			expectedReqID:   "req-error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := claudeagent.SDKControlResponse{
				BaseMessage: claudeagent.BaseMessage{
					UUIDField:      uuid.New(),
					SessionIDField: "test-session",
				},
				TypeField: "control_response",
				Response:  tc.response,
			}

			data, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("failed to marshal response: %v", err)
			}

			var decoded claudeagent.SDKControlResponse
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if decoded.Response.Subtype() != tc.expectedSubtype {
				t.Errorf(
					"expected subtype '%s', got '%s'",
					tc.expectedSubtype,
					decoded.Response.Subtype(),
				)
			}

			if decoded.Response.RequestID() != tc.expectedReqID {
				t.Errorf(
					"expected request ID '%s', got '%s'",
					tc.expectedReqID,
					decoded.Response.RequestID(),
				)
			}
		})
	}
}

// Helper function to create string pointers.
func stringPtr(s string) *string {
	return &s
}
