package unit

import (
"encoding/json"
"testing"

claudeagent "github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
"github.com/google/uuid"
)

// TestSDKControlResponseTypeFieldInJSON verifies that the TypeField is included
// in JSON serialization as "type": "control_response"
func TestSDKControlResponseTypeFieldInJSON(t *testing.T) {
testCases := []struct {
name     string
response claudeagent.ControlResponseVariant
}{
{
name: "Success response",
response: claudeagent.ControlSuccessResponse{
SubtypeField:   "success",
RequestIDField: "req-success-1",
Response: map[string]claudeagent.JSONValue{
"result": json.RawMessage(`"completed"`),
},
},
},
{
name: "Error response",
response: claudeagent.ControlErrorResponse{
SubtypeField:   "error",
RequestIDField: "req-error-1",
Error:          "Something went wrong",
},
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

// Marshal to JSON
data, err := json.Marshal(msg)
if err != nil {
t.Fatalf("failed to marshal: %v", err)
}

// Parse as raw JSON to verify structure
var rawJSON map[string]interface{}
if err := json.Unmarshal(data, &rawJSON); err != nil {
t.Fatalf("failed to unmarshal as map: %v", err)
}

// Verify type field exists
typeField, ok := rawJSON["type"]
if !ok {
t.Error("'type' field is missing from JSON")
} else if typeField != "control_response" {
t.Errorf("expected type='control_response', got type=%v", typeField)
}

// Verify uuid field exists
if _, ok := rawJSON["uuid"]; !ok {
t.Error("'uuid' field is missing from JSON")
}

// Verify session_id field exists
if _, ok := rawJSON["session_id"]; !ok {
t.Error("'session_id' field is missing from JSON")
}

// Verify response field exists
if _, ok := rawJSON["response"]; !ok {
t.Error("'response' field is missing from JSON")
}

// Unmarshal back and verify TypeField is populated
var decoded claudeagent.SDKControlResponse
if err := json.Unmarshal(data, &decoded); err != nil {
t.Fatalf("failed to unmarshal back: %v", err)
}

if decoded.TypeField != "control_response" {
t.Errorf("expected TypeField='control_response', got TypeField=%q", decoded.TypeField)
}

if decoded.Type() != "control_response" {
t.Errorf("expected Type()='control_response', got Type()=%q", decoded.Type())
}
})
}
}
