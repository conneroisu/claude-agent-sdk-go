package unit

import (
	"encoding/json"
	"testing"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
)

// ============================================================================
// Hook Events Tests
// ============================================================================

// TestHookEventConstants verifies hook event constants have correct string values.
func TestHookEventConstants(t *testing.T) {
	tests := []struct {
		name     string
		event    claude.HookEvent
		expected string
	}{
		{"PreToolUse", claude.HookEventPreToolUse, "PreToolUse"},
		{"PostToolUse", claude.HookEventPostToolUse, "PostToolUse"},
		{"Notification", claude.HookEventNotification, "Notification"},
		{"UserPromptSubmit", claude.HookEventUserPromptSubmit, "UserPromptSubmit"},
		{"SessionStart", claude.HookEventSessionStart, "SessionStart"},
		{"SessionEnd", claude.HookEventSessionEnd, "SessionEnd"},
		{"Stop", claude.HookEventStop, "Stop"},
		{"SubagentStop", claude.HookEventSubagentStop, "SubagentStop"},
		{"PreCompact", claude.HookEventPreCompact, "PreCompact"},
		{"PermissionRequest", claude.HookEventPermissionRequest, "PermissionRequest"},
		{"SubagentStart", claude.HookEventSubagentStart, "SubagentStart"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.event) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.event))
			}
		})
	}
}

// TestHookEventsSlice verifies that all hook events are in the HookEvents slice.
func TestHookEventsSlice(t *testing.T) {
	expectedEvents := []claude.HookEvent{
		claude.HookEventPreToolUse,
		claude.HookEventPostToolUse,
		claude.HookEventNotification,
		claude.HookEventUserPromptSubmit,
		claude.HookEventSessionStart,
		claude.HookEventSessionEnd,
		claude.HookEventStop,
		claude.HookEventSubagentStop,
		claude.HookEventPreCompact,
		claude.HookEventPermissionRequest,
		claude.HookEventSubagentStart,
	}

	if len(claude.HookEvents) != len(expectedEvents) {
		t.Errorf("expected %d events, got %d", len(expectedEvents), len(claude.HookEvents))
	}

	// Verify each expected event is in the slice
	eventMap := make(map[claude.HookEvent]bool)
	for _, event := range claude.HookEvents {
		eventMap[event] = true
	}

	for _, expected := range expectedEvents {
		if !eventMap[expected] {
			t.Errorf("expected event %s not found in HookEvents slice", expected)
		}
	}
}

// ============================================================================
// PermissionRequestHookInput Tests
// ============================================================================

// TestPermissionRequestHookInputMarshaling verifies JSON marshaling with all fields.
func TestPermissionRequestHookInputMarshaling(t *testing.T) {
	input := claude.PermissionRequestHookInput{
		BaseHookInput: claude.BaseHookInput{
			SessionIDField:      "session-123",
			TranscriptPathField: "/path/to/transcript",
			CwdField:            "/home/user",
		},
		HookEventName: claude.HookEventPermissionRequest,
		ToolName:      "Write",
		ToolInput:     json.RawMessage(`{"file_path":"/test/file.txt","content":"test"}`),
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal PermissionRequestHookInput: %v", err)
	}

	// Verify JSON uses snake_case for input fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["session_id"]; !ok {
		t.Error("expected 'session_id' field in JSON")
	}
	if _, ok := raw["transcript_path"]; !ok {
		t.Error("expected 'transcript_path' field in JSON")
	}
	if _, ok := raw["cwd"]; !ok {
		t.Error("expected 'cwd' field in JSON")
	}
	if _, ok := raw["hook_event_name"]; !ok {
		t.Error("expected 'hook_event_name' field in JSON")
	}
	if _, ok := raw["tool_name"]; !ok {
		t.Error("expected 'tool_name' field in JSON")
	}
	if _, ok := raw["tool_input"]; !ok {
		t.Error("expected 'tool_input' field in JSON")
	}
}

// TestPermissionRequestHookInputUnmarshaling verifies JSON unmarshaling.
func TestPermissionRequestHookInputUnmarshaling(t *testing.T) {
	jsonData := `{
		"session_id": "session-456",
		"transcript_path": "/path/to/transcript",
		"cwd": "/home/test",
		"hook_event_name": "PermissionRequest",
		"tool_name": "Bash",
		"tool_input": {"command":"ls -la"}
	}`

	var input claude.PermissionRequestHookInput
	if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
		t.Fatalf("failed to unmarshal PermissionRequestHookInput: %v", err)
	}

	if input.SessionID() != "session-456" {
		t.Errorf("expected session_id 'session-456', got %s", input.SessionID())
	}
	if input.TranscriptPath() != "/path/to/transcript" {
		t.Errorf("expected transcript_path '/path/to/transcript', got %s", input.TranscriptPath())
	}
	if input.Cwd() != "/home/test" {
		t.Errorf("expected cwd '/home/test', got %s", input.Cwd())
	}
	if input.ToolName != "Bash" {
		t.Errorf("expected tool_name 'Bash', got %s", input.ToolName)
	}
}

// TestPermissionRequestHookInputEventName verifies EventName() returns correct value.
func TestPermissionRequestHookInputEventName(t *testing.T) {
	input := claude.PermissionRequestHookInput{}
	if input.EventName() != claude.HookEventPermissionRequest {
		t.Errorf("expected EventName() to return %s, got %s", claude.HookEventPermissionRequest, input.EventName())
	}
}

// TestPermissionRequestHookInputDecodeHookInput verifies DecodeHookInput correctly decodes.
func TestPermissionRequestHookInputDecodeHookInput(t *testing.T) {
	jsonData := []byte(`{
		"session_id": "session-789",
		"transcript_path": "/path/to/transcript",
		"cwd": "/home/user",
		"hook_event_name": "PermissionRequest",
		"tool_name": "Read",
		"tool_input": {"file_path":"/test.txt"}
	}`)

	decoded, err := claude.DecodeHookInput(jsonData)
	if err != nil {
		t.Fatalf("failed to decode hook input: %v", err)
	}

	input, ok := decoded.(claude.PermissionRequestHookInput)
	if !ok {
		t.Fatalf("expected PermissionRequestHookInput, got %T", decoded)
	}

	if input.ToolName != "Read" {
		t.Errorf("expected tool_name 'Read', got %s", input.ToolName)
	}
	if input.EventName() != claude.HookEventPermissionRequest {
		t.Errorf("expected EventName() %s, got %s", claude.HookEventPermissionRequest, input.EventName())
	}
}

// ============================================================================
// SubagentStartHookInput Tests
// ============================================================================

// TestSubagentStartHookInputMarshaling verifies JSON marshaling with all fields.
func TestSubagentStartHookInputMarshaling(t *testing.T) {
	input := claude.SubagentStartHookInput{
		BaseHookInput: claude.BaseHookInput{
			SessionIDField:      "session-abc",
			TranscriptPathField: "/path/to/transcript",
			CwdField:            "/home/user",
		},
		HookEventName: claude.HookEventSubagentStart,
		AgentID:       "agent-123",
		AgentType:     "coder",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal SubagentStartHookInput: %v", err)
	}

	// Verify JSON uses snake_case
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["agent_id"]; !ok {
		t.Error("expected 'agent_id' field in JSON")
	}
	if _, ok := raw["agent_type"]; !ok {
		t.Error("expected 'agent_type' field in JSON")
	}
}

// TestSubagentStartHookInputUnmarshaling verifies JSON unmarshaling.
func TestSubagentStartHookInputUnmarshaling(t *testing.T) {
	jsonData := `{
		"session_id": "session-xyz",
		"transcript_path": "/path/to/transcript",
		"cwd": "/home/test",
		"hook_event_name": "SubagentStart",
		"agent_id": "agent-456",
		"agent_type": "tester"
	}`

	var input claude.SubagentStartHookInput
	if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
		t.Fatalf("failed to unmarshal SubagentStartHookInput: %v", err)
	}

	if input.AgentID != "agent-456" {
		t.Errorf("expected agent_id 'agent-456', got %s", input.AgentID)
	}
	if input.AgentType != "tester" {
		t.Errorf("expected agent_type 'tester', got %s", input.AgentType)
	}
}

// TestSubagentStartHookInputEventName verifies EventName() returns correct value.
func TestSubagentStartHookInputEventName(t *testing.T) {
	input := claude.SubagentStartHookInput{}
	if input.EventName() != claude.HookEventSubagentStart {
		t.Errorf("expected EventName() to return %s, got %s", claude.HookEventSubagentStart, input.EventName())
	}
}

// TestSubagentStartHookInputDecodeHookInput verifies DecodeHookInput correctly decodes.
func TestSubagentStartHookInputDecodeHookInput(t *testing.T) {
	jsonData := []byte(`{
		"session_id": "session-decode",
		"transcript_path": "/path/to/transcript",
		"cwd": "/home/user",
		"hook_event_name": "SubagentStart",
		"agent_id": "agent-decode",
		"agent_type": "stuck"
	}`)

	decoded, err := claude.DecodeHookInput(jsonData)
	if err != nil {
		t.Fatalf("failed to decode hook input: %v", err)
	}

	input, ok := decoded.(claude.SubagentStartHookInput)
	if !ok {
		t.Fatalf("expected SubagentStartHookInput, got %T", decoded)
	}

	if input.AgentID != "agent-decode" {
		t.Errorf("expected agent_id 'agent-decode', got %s", input.AgentID)
	}
	if input.EventName() != claude.HookEventSubagentStart {
		t.Errorf("expected EventName() %s, got %s", claude.HookEventSubagentStart, input.EventName())
	}
}

// ============================================================================
// PermissionRequestHookOutput Tests
// ============================================================================

// TestPermissionRequestHookOutputAllowWithUpdatedInput verifies marshaling Allow decision with updatedInput.
func TestPermissionRequestHookOutputAllowWithUpdatedInput(t *testing.T) {
	updatedInput := map[string]interface{}{
		"file_path": "/modified/path.txt",
		"content":   "modified content",
	}

	output := claude.PermissionRequestHookOutput{
		HookEventName: claude.HookEventPermissionRequest,
		Decision: claude.PermissionRequestAllow{
			Behavior:     "allow",
			UpdatedInput: &updatedInput,
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal PermissionRequestHookOutput: %v", err)
	}

	// Verify JSON uses camelCase for output fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["hookEventName"]; !ok {
		t.Error("expected 'hookEventName' field in JSON")
	}

	decision, ok := raw["decision"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'decision' to be an object")
	}

	if decision["behavior"] != "allow" {
		t.Errorf("expected behavior 'allow', got %v", decision["behavior"])
	}

	if _, ok := decision["updatedInput"]; !ok {
		t.Error("expected 'updatedInput' field in decision")
	}
}

// TestPermissionRequestHookOutputAllowWithoutUpdatedInput verifies marshaling Allow decision without updatedInput.
func TestPermissionRequestHookOutputAllowWithoutUpdatedInput(t *testing.T) {
	output := claude.PermissionRequestHookOutput{
		HookEventName: claude.HookEventPermissionRequest,
		Decision: claude.PermissionRequestAllow{
			Behavior:     "allow",
			UpdatedInput: nil,
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal PermissionRequestHookOutput: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	decision, ok := raw["decision"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'decision' to be an object")
	}

	if decision["behavior"] != "allow" {
		t.Errorf("expected behavior 'allow', got %v", decision["behavior"])
	}

	// updatedInput should be omitted when nil
	if _, ok := decision["updatedInput"]; ok {
		t.Error("expected 'updatedInput' to be omitted when nil")
	}
}

// TestPermissionRequestHookOutputDenyWithMessage verifies marshaling Deny decision with message.
func TestPermissionRequestHookOutputDenyWithMessage(t *testing.T) {
	message := "Permission denied for security reasons"
	output := claude.PermissionRequestHookOutput{
		HookEventName: claude.HookEventPermissionRequest,
		Decision: claude.PermissionRequestDeny{
			Behavior: "deny",
			Message:  &message,
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal PermissionRequestHookOutput: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	decision, ok := raw["decision"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'decision' to be an object")
	}

	if decision["behavior"] != "deny" {
		t.Errorf("expected behavior 'deny', got %v", decision["behavior"])
	}

	if decision["message"] != message {
		t.Errorf("expected message '%s', got %v", message, decision["message"])
	}
}

// TestPermissionRequestHookOutputDenyWithInterrupt verifies marshaling Deny decision with interrupt.
func TestPermissionRequestHookOutputDenyWithInterrupt(t *testing.T) {
	message := "Critical security violation"
	interrupt := true
	output := claude.PermissionRequestHookOutput{
		HookEventName: claude.HookEventPermissionRequest,
		Decision: claude.PermissionRequestDeny{
			Behavior:  "deny",
			Message:   &message,
			Interrupt: &interrupt,
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal PermissionRequestHookOutput: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	decision, ok := raw["decision"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'decision' to be an object")
	}

	if decision["interrupt"] != true {
		t.Errorf("expected interrupt true, got %v", decision["interrupt"])
	}
}

// TestPermissionRequestHookOutputEventName verifies EventName() returns correct value.
func TestPermissionRequestHookOutputEventName(t *testing.T) {
	output := claude.PermissionRequestHookOutput{
		HookEventName: claude.HookEventPermissionRequest,
		Decision: claude.PermissionRequestAllow{
			Behavior: "allow",
		},
	}

	if output.EventName() != claude.HookEventPermissionRequest {
		t.Errorf("expected EventName() to return %s, got %s", claude.HookEventPermissionRequest, output.EventName())
	}
}

// ============================================================================
// SubagentStartHookOutput Tests
// ============================================================================

// TestSubagentStartHookOutputWithAdditionalContext verifies marshaling with additionalContext.
func TestSubagentStartHookOutputWithAdditionalContext(t *testing.T) {
	context := "Additional instructions for the subagent"
	output := claude.SubagentStartHookOutput{
		HookEventName:     claude.HookEventSubagentStart,
		AdditionalContext: &context,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal SubagentStartHookOutput: %v", err)
	}

	// Verify JSON uses camelCase
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["hookEventName"]; !ok {
		t.Error("expected 'hookEventName' field in JSON")
	}

	if raw["additionalContext"] != context {
		t.Errorf("expected additionalContext '%s', got %v", context, raw["additionalContext"])
	}
}

// TestSubagentStartHookOutputWithoutAdditionalContext verifies marshaling without additionalContext.
func TestSubagentStartHookOutputWithoutAdditionalContext(t *testing.T) {
	output := claude.SubagentStartHookOutput{
		HookEventName:     claude.HookEventSubagentStart,
		AdditionalContext: nil,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal SubagentStartHookOutput: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// additionalContext should be omitted when nil
	if _, ok := raw["additionalContext"]; ok {
		t.Error("expected 'additionalContext' to be omitted when nil")
	}
}

// TestSubagentStartHookOutputEventName verifies EventName() returns correct value.
func TestSubagentStartHookOutputEventName(t *testing.T) {
	output := claude.SubagentStartHookOutput{
		HookEventName: claude.HookEventSubagentStart,
	}

	if output.EventName() != claude.HookEventSubagentStart {
		t.Errorf("expected EventName() to return %s, got %s", claude.HookEventSubagentStart, output.EventName())
	}
}

// ============================================================================
// Enhanced Hook Inputs Tests
// ============================================================================

// TestPreToolUseHookInputWithToolUseID verifies PreToolUseHookInput has ToolUseID field.
func TestPreToolUseHookInputWithToolUseID(t *testing.T) {
	input := claude.PreToolUseHookInput{
		BaseHookInput: claude.BaseHookInput{
			SessionIDField:      "session-123",
			TranscriptPathField: "/path/to/transcript",
			CwdField:            "/home/user",
		},
		HookEventName: claude.HookEventPreToolUse,
		ToolName:      "Bash",
		ToolInput:     json.RawMessage(`{"command":"echo test"}`),
		ToolUseID:     "tool-use-123",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal PreToolUseHookInput: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["tool_use_id"]; !ok {
		t.Error("expected 'tool_use_id' field in JSON")
	}

	if raw["tool_use_id"] != "tool-use-123" {
		t.Errorf("expected tool_use_id 'tool-use-123', got %v", raw["tool_use_id"])
	}

	// Unmarshal and verify
	var decoded claude.PreToolUseHookInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal PreToolUseHookInput: %v", err)
	}

	if decoded.ToolUseID != "tool-use-123" {
		t.Errorf("expected ToolUseID 'tool-use-123', got %s", decoded.ToolUseID)
	}
}

// TestPostToolUseHookInputWithToolUseID verifies PostToolUseHookInput has ToolUseID field.
func TestPostToolUseHookInputWithToolUseID(t *testing.T) {
	input := claude.PostToolUseHookInput{
		BaseHookInput: claude.BaseHookInput{
			SessionIDField:      "session-456",
			TranscriptPathField: "/path/to/transcript",
			CwdField:            "/home/user",
		},
		HookEventName: claude.HookEventPostToolUse,
		ToolName:      "Read",
		ToolInput:     json.RawMessage(`{"file_path":"/test.txt"}`),
		ToolResponse:  json.RawMessage(`{"content":"file contents"}`),
		ToolUseID:     "tool-use-456",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal PostToolUseHookInput: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["tool_use_id"]; !ok {
		t.Error("expected 'tool_use_id' field in JSON")
	}

	if raw["tool_use_id"] != "tool-use-456" {
		t.Errorf("expected tool_use_id 'tool-use-456', got %v", raw["tool_use_id"])
	}

	// Unmarshal and verify
	var decoded claude.PostToolUseHookInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal PostToolUseHookInput: %v", err)
	}

	if decoded.ToolUseID != "tool-use-456" {
		t.Errorf("expected ToolUseID 'tool-use-456', got %s", decoded.ToolUseID)
	}
}

// TestNotificationHookInputWithNotificationType verifies NotificationHookInput has NotificationType field.
func TestNotificationHookInputWithNotificationType(t *testing.T) {
	input := claude.NotificationHookInput{
		BaseHookInput: claude.BaseHookInput{
			SessionIDField:      "session-789",
			TranscriptPathField: "/path/to/transcript",
			CwdField:            "/home/user",
		},
		HookEventName:    claude.HookEventNotification,
		Message:          "Test notification",
		NotificationType: "info",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal NotificationHookInput: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["notification_type"]; !ok {
		t.Error("expected 'notification_type' field in JSON")
	}

	if raw["notification_type"] != "info" {
		t.Errorf("expected notification_type 'info', got %v", raw["notification_type"])
	}

	// Unmarshal and verify
	var decoded claude.NotificationHookInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal NotificationHookInput: %v", err)
	}

	if decoded.NotificationType != "info" {
		t.Errorf("expected NotificationType 'info', got %s", decoded.NotificationType)
	}
}

// TestSubagentStopHookInputWithAgentFields verifies SubagentStopHookInput has AgentID and AgentTranscriptPath fields.
func TestSubagentStopHookInputWithAgentFields(t *testing.T) {
	input := claude.SubagentStopHookInput{
		BaseHookInput: claude.BaseHookInput{
			SessionIDField:      "session-stop",
			TranscriptPathField: "/path/to/transcript",
			CwdField:            "/home/user",
		},
		HookEventName:        claude.HookEventSubagentStop,
		StopHookActive:       true,
		AgentID:              "agent-stop-123",
		AgentTranscriptPath:  "/path/to/agent/transcript",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal SubagentStopHookInput: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["agent_id"]; !ok {
		t.Error("expected 'agent_id' field in JSON")
	}

	if _, ok := raw["agent_transcript_path"]; !ok {
		t.Error("expected 'agent_transcript_path' field in JSON")
	}

	if raw["agent_id"] != "agent-stop-123" {
		t.Errorf("expected agent_id 'agent-stop-123', got %v", raw["agent_id"])
	}

	if raw["agent_transcript_path"] != "/path/to/agent/transcript" {
		t.Errorf("expected agent_transcript_path '/path/to/agent/transcript', got %v", raw["agent_transcript_path"])
	}

	// Unmarshal and verify
	var decoded claude.SubagentStopHookInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SubagentStopHookInput: %v", err)
	}

	if decoded.AgentID != "agent-stop-123" {
		t.Errorf("expected AgentID 'agent-stop-123', got %s", decoded.AgentID)
	}

	if decoded.AgentTranscriptPath != "/path/to/agent/transcript" {
		t.Errorf("expected AgentTranscriptPath '/path/to/agent/transcript', got %s", decoded.AgentTranscriptPath)
	}
}

// ============================================================================
// Enhanced Hook Outputs Tests
// ============================================================================

// TestPreToolUseHookOutputWithUpdatedInput verifies PreToolUseHookOutput has UpdatedInput field.
func TestPreToolUseHookOutputWithUpdatedInput(t *testing.T) {
	updatedInput := map[string]interface{}{
		"command": "echo modified",
	}

	output := claude.PreToolUseHookOutput{
		HookEventName: claude.HookEventPreToolUse,
		UpdatedInput:  &updatedInput,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal PreToolUseHookOutput: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["updatedInput"]; !ok {
		t.Error("expected 'updatedInput' field in JSON")
	}

	// Unmarshal and verify
	var decoded claude.PreToolUseHookOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal PreToolUseHookOutput: %v", err)
	}

	if decoded.UpdatedInput == nil {
		t.Fatal("expected UpdatedInput to be non-nil")
	}

	if (*decoded.UpdatedInput)["command"] != "echo modified" {
		t.Errorf("expected UpdatedInput command 'echo modified', got %v", (*decoded.UpdatedInput)["command"])
	}
}

// TestPostToolUseHookOutputWithUpdatedMCPToolOutput verifies PostToolUseHookOutput has UpdatedMCPToolOutput field.
func TestPostToolUseHookOutputWithUpdatedMCPToolOutput(t *testing.T) {
	updatedOutput := map[string]interface{}{
		"content": "modified output",
	}

	output := claude.PostToolUseHookOutput{
		HookEventName:        claude.HookEventPostToolUse,
		UpdatedMCPToolOutput: updatedOutput,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal PostToolUseHookOutput: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["updatedMCPToolOutput"]; !ok {
		t.Error("expected 'updatedMCPToolOutput' field in JSON")
	}

	// Unmarshal and verify
	var decoded claude.PostToolUseHookOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal PostToolUseHookOutput: %v", err)
	}

	if decoded.UpdatedMCPToolOutput == nil {
		t.Fatal("expected UpdatedMCPToolOutput to be non-nil")
	}

	outputMap, ok := decoded.UpdatedMCPToolOutput.(map[string]interface{})
	if !ok {
		t.Fatalf("expected UpdatedMCPToolOutput to be map[string]interface{}, got %T", decoded.UpdatedMCPToolOutput)
	}

	if outputMap["content"] != "modified output" {
		t.Errorf("expected UpdatedMCPToolOutput content 'modified output', got %v", outputMap["content"])
	}
}

// ============================================================================
// Permission System Enhancement Tests
// ============================================================================

// TestPermissionAllowWithToolUseID verifies PermissionAllow has ToolUseID field.
func TestPermissionAllowWithToolUseID(t *testing.T) {
	toolUseID := "tool-use-perm-123"
	allow := claude.PermissionAllow{
		Behavior:     claude.PermissionBehaviorAllow,
		ToolUseID:    &toolUseID,
		UpdatedInput: map[string]claude.JSONValue{},
	}

	data, err := json.Marshal(allow)
	if err != nil {
		t.Fatalf("failed to marshal PermissionAllow: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["toolUseID"]; !ok {
		t.Error("expected 'toolUseID' field in JSON")
	}

	if raw["toolUseID"] != toolUseID {
		t.Errorf("expected toolUseID '%s', got %v", toolUseID, raw["toolUseID"])
	}

	// Unmarshal and verify
	var decoded claude.PermissionAllow
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal PermissionAllow: %v", err)
	}

	if decoded.ToolUseID == nil || *decoded.ToolUseID != toolUseID {
		t.Errorf("expected ToolUseID '%s', got %v", toolUseID, decoded.ToolUseID)
	}
}

// TestPermissionDenyWithToolUseID verifies PermissionDeny has ToolUseID field.
func TestPermissionDenyWithToolUseID(t *testing.T) {
	toolUseID := "tool-use-perm-456"
	deny := claude.PermissionDeny{
		Behavior:  claude.PermissionBehaviorDeny,
		ToolUseID: &toolUseID,
		Message:   "Access denied",
		Interrupt: false,
	}

	data, err := json.Marshal(deny)
	if err != nil {
		t.Fatalf("failed to marshal PermissionDeny: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["toolUseID"]; !ok {
		t.Error("expected 'toolUseID' field in JSON")
	}

	if raw["toolUseID"] != toolUseID {
		t.Errorf("expected toolUseID '%s', got %v", toolUseID, raw["toolUseID"])
	}

	// Unmarshal and verify
	var decoded claude.PermissionDeny
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal PermissionDeny: %v", err)
	}

	if decoded.ToolUseID == nil || *decoded.ToolUseID != toolUseID {
		t.Errorf("expected ToolUseID '%s', got %v", toolUseID, decoded.ToolUseID)
	}
}

// ============================================================================
// HookCallbackMatcher Timeout Tests
// ============================================================================

// TestHookCallbackMatcherWithTimeout verifies HookCallbackMatcher marshals Timeout field.
func TestHookCallbackMatcherWithTimeout(t *testing.T) {
	timeout := 5000
	matcher := "*.txt"
	hookMatcher := claude.HookCallbackMatcher{
		Matcher: &matcher,
		Timeout: &timeout,
	}

	data, err := json.Marshal(hookMatcher)
	if err != nil {
		t.Fatalf("failed to marshal HookCallbackMatcher: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["timeout"]; !ok {
		t.Error("expected 'timeout' field in JSON")
	}

	if raw["timeout"] != float64(5000) {
		t.Errorf("expected timeout 5000, got %v", raw["timeout"])
	}

	// Unmarshal and verify
	var decoded claude.HookCallbackMatcher
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal HookCallbackMatcher: %v", err)
	}

	if decoded.Timeout == nil || *decoded.Timeout != timeout {
		t.Errorf("expected Timeout %d, got %v", timeout, decoded.Timeout)
	}
}

// TestHookCallbackMatcherWithoutTimeout verifies HookCallbackMatcher omits Timeout when nil.
func TestHookCallbackMatcherWithoutTimeout(t *testing.T) {
	matcher := "*.go"
	hookMatcher := claude.HookCallbackMatcher{
		Matcher: &matcher,
		Timeout: nil,
	}

	data, err := json.Marshal(hookMatcher)
	if err != nil {
		t.Fatalf("failed to marshal HookCallbackMatcher: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["timeout"]; ok {
		t.Error("expected 'timeout' to be omitted when nil")
	}
}

// ============================================================================
// JSON Round-Trip Tests
// ============================================================================

// TestPermissionRequestHookInputRoundTrip verifies JSON round-trip for PermissionRequestHookInput.
func TestPermissionRequestHookInputRoundTrip(t *testing.T) {
	original := claude.PermissionRequestHookInput{
		BaseHookInput: claude.BaseHookInput{
			SessionIDField:      "session-roundtrip",
			TranscriptPathField: "/path/to/transcript",
			CwdField:            "/home/user",
		},
		HookEventName: claude.HookEventPermissionRequest,
		ToolName:      "Write",
		ToolInput:     json.RawMessage(`{"file_path":"/test.txt","content":"data"}`),
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded claude.PermissionRequestHookInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify all fields match
	if decoded.SessionID() != original.SessionID() {
		t.Errorf("SessionID mismatch: expected %s, got %s", original.SessionID(), decoded.SessionID())
	}
	if decoded.ToolName != original.ToolName {
		t.Errorf("ToolName mismatch: expected %s, got %s", original.ToolName, decoded.ToolName)
	}
	if decoded.EventName() != original.EventName() {
		t.Errorf("EventName mismatch: expected %s, got %s", original.EventName(), decoded.EventName())
	}
}

// TestSubagentStartHookInputRoundTrip verifies JSON round-trip for SubagentStartHookInput.
func TestSubagentStartHookInputRoundTrip(t *testing.T) {
	original := claude.SubagentStartHookInput{
		BaseHookInput: claude.BaseHookInput{
			SessionIDField:      "session-roundtrip-2",
			TranscriptPathField: "/path/to/transcript",
			CwdField:            "/home/user",
		},
		HookEventName: claude.HookEventSubagentStart,
		AgentID:       "agent-roundtrip",
		AgentType:     "coder",
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded claude.SubagentStartHookInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify all fields match
	if decoded.SessionID() != original.SessionID() {
		t.Errorf("SessionID mismatch: expected %s, got %s", original.SessionID(), decoded.SessionID())
	}
	if decoded.AgentID != original.AgentID {
		t.Errorf("AgentID mismatch: expected %s, got %s", original.AgentID, decoded.AgentID)
	}
	if decoded.AgentType != original.AgentType {
		t.Errorf("AgentType mismatch: expected %s, got %s", original.AgentType, decoded.AgentType)
	}
	if decoded.EventName() != original.EventName() {
		t.Errorf("EventName mismatch: expected %s, got %s", original.EventName(), decoded.EventName())
	}
}

// TestPermissionRequestHookOutputMarshaling verifies JSON marshaling for PermissionRequestHookOutput.
// Note: Full round-trip testing would require custom unmarshaling logic for the Decision interface,
// which is beyond the scope of this test. We verify that marshaling produces valid JSON.
func TestPermissionRequestHookOutputMarshaling(t *testing.T) {
	updatedInput := map[string]interface{}{
		"modified": "value",
	}

	original := claude.PermissionRequestHookOutput{
		HookEventName: claude.HookEventPermissionRequest,
		Decision: claude.PermissionRequestAllow{
			Behavior:     "allow",
			UpdatedInput: &updatedInput,
		},
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify the JSON structure
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// Verify hookEventName is present
	if raw["hookEventName"] != string(claude.HookEventPermissionRequest) {
		t.Errorf("expected hookEventName %s, got %v", claude.HookEventPermissionRequest, raw["hookEventName"])
	}

	// Verify decision is present and has the correct structure
	decision, ok := raw["decision"].(map[string]interface{})
	if !ok {
		t.Fatal("expected decision to be an object")
	}

	if decision["behavior"] != "allow" {
		t.Errorf("expected behavior 'allow', got %v", decision["behavior"])
	}

	if _, ok := decision["updatedInput"]; !ok {
		t.Error("expected updatedInput in decision")
	}
}
