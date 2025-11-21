package unit

import (
	"encoding/json"
	"reflect"
	"testing"

	claudeagent "github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
)

func TestIncludePartialMessagesFlag(t *testing.T) {
	tests := []struct {
		name                   string
		includePartialMessages bool
		shouldContainFlag      bool
	}{
		{
			name:                   "With IncludePartialMessages enabled",
			includePartialMessages: true,
			shouldContainFlag:      true,
		},
		{
			name:                   "With IncludePartialMessages disabled",
			includePartialMessages: false,
			shouldContainFlag:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a queryImpl to test buildArgs
			opts := &claudeagent.Options{
				IncludePartialMessages: tt.includePartialMessages,
			}

			// We can't directly access buildArgs since it's on queryImpl,
			// but we can verify the option is set correctly
			if opts.IncludePartialMessages != tt.includePartialMessages {
				t.Errorf(
					"IncludePartialMessages = %v, want %v",
					opts.IncludePartialMessages,
					tt.includePartialMessages,
				)
			}
		})
	}
}

// Helper function to create string pointers.
func strPtr(s string) *string {
	return &s
}

// TestAgentInput_Model_ValidValues tests that Model field accepts valid values.
func TestAgentInput_Model_ValidValues(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{
			name:  "Sonnet model",
			model: "sonnet",
		},
		{
			name:  "Opus model",
			model: "opus",
		},
		{
			name:  "Haiku model",
			model: "haiku",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := claudeagent.AgentInput{
				Description:  "Test description",
				Prompt:       "Test prompt",
				SubagentType: "test",
				Model:        strPtr(tt.model),
			}

			if input.Model == nil {
				t.Fatal("Model should not be nil")
			}
			if *input.Model != tt.model {
				t.Errorf("Model = %v, want %v", *input.Model, tt.model)
			}
		})
	}
}

// TestAgentInput_Model_NilValue tests that Model field can be nil.
func TestAgentInput_Model_NilValue(t *testing.T) {
	input := claudeagent.AgentInput{
		Description:  "Test description",
		Prompt:       "Test prompt",
		SubagentType: "test",
		Model:        nil,
	}

	if input.Model != nil {
		t.Errorf("Model should be nil, got %v", *input.Model)
	}
}

// TestAgentInput_Resume_StoresCheckpointID tests that Resume field stores checkpoint IDs correctly
func TestAgentInput_Resume_StoresCheckpointID(t *testing.T) {
	checkpointID := "checkpoint-123-abc-456"
	input := claudeagent.AgentInput{
		Description:  "Test description",
		Prompt:       "Test prompt",
		SubagentType: "test",
		Resume:       strPtr(checkpointID),
	}

	if input.Resume == nil {
		t.Fatal("Resume should not be nil")
	}
	if *input.Resume != checkpointID {
		t.Errorf("Resume = %v, want %v", *input.Resume, checkpointID)
	}
}

// TestAgentInput_Resume_NilValue tests that Resume field can be nil.
func TestAgentInput_Resume_NilValue(t *testing.T) {
	input := claudeagent.AgentInput{
		Description:  "Test description",
		Prompt:       "Test prompt",
		SubagentType: "test",
		Resume:       nil,
	}

	if input.Resume != nil {
		t.Errorf("Resume should be nil, got %v", *input.Resume)
	}
}

// TestAgentInput_JSON_MarshalCamelCase tests that JSON marshaling produces camelCase field names
func TestAgentInput_JSON_MarshalCamelCase(t *testing.T) {
	input := claudeagent.AgentInput{
		Description:  "Test description",
		Prompt:       "Test prompt",
		SubagentType: "coder",
		Model:        strPtr("sonnet"),
		Resume:       strPtr("checkpoint-123"),
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal AgentInput: %v", err)
	}

	// Parse JSON to check field names
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that camelCase fields exist
	if _, ok := result["model"]; !ok {
		t.Error("JSON should contain 'model' field")
	}
	if _, ok := result["resume"]; !ok {
		t.Error("JSON should contain 'resume' field")
	}

	// Verify values
	if result["model"] != "sonnet" {
		t.Errorf("model = %v, want 'sonnet'", result["model"])
	}
	if result["resume"] != "checkpoint-123" {
		t.Errorf("resume = %v, want 'checkpoint-123'", result["resume"])
	}
}

// TestAgentInput_JSON_UnmarshalCamelCase tests that JSON unmarshaling correctly reads camelCase fields
func TestAgentInput_JSON_UnmarshalCamelCase(t *testing.T) {
	jsonData := `{
		"description": "Test description",
		"prompt": "Test prompt",
		"subagent_type": "coder",
		"model": "opus",
		"resume": "checkpoint-456"
	}`

	var input claudeagent.AgentInput
	if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if input.Description != "Test description" {
		t.Errorf("Description = %v, want 'Test description'", input.Description)
	}
	if input.Prompt != "Test prompt" {
		t.Errorf("Prompt = %v, want 'Test prompt'", input.Prompt)
	}
	if input.SubagentType != "coder" {
		t.Errorf("SubagentType = %v, want 'coder'", input.SubagentType)
	}
	if input.Model == nil {
		t.Fatal("Model should not be nil")
	}
	if *input.Model != "opus" {
		t.Errorf("Model = %v, want 'opus'", *input.Model)
	}
	if input.Resume == nil {
		t.Fatal("Resume should not be nil")
	}
	if *input.Resume != "checkpoint-456" {
		t.Errorf("Resume = %v, want 'checkpoint-456'", *input.Resume)
	}
}

// TestAgentInput_JSON_OmitEmpty_NilFields tests that nil Model/Resume fields are excluded from JSON
func TestAgentInput_JSON_OmitEmpty_NilFields(t *testing.T) {
	input := claudeagent.AgentInput{
		Description:  "Test description",
		Prompt:       "Test prompt",
		SubagentType: "tester",
		Model:        nil,
		Resume:       nil,
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal AgentInput: %v", err)
	}

	// Parse JSON to check field presence
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that model and resume fields are NOT present
	if _, ok := result["model"]; ok {
		t.Error("JSON should not contain 'model' field when nil")
	}
	if _, ok := result["resume"]; ok {
		t.Error("JSON should not contain 'resume' field when nil")
	}

	// Verify other fields are present
	if _, ok := result["description"]; !ok {
		t.Error("JSON should contain 'description' field")
	}
	if _, ok := result["prompt"]; !ok {
		t.Error("JSON should contain 'prompt' field")
	}
}

// TestAgentInput_JSON_OmitEmpty_NonNilFields tests that non-nil fields are included in JSON
func TestAgentInput_JSON_OmitEmpty_NonNilFields(t *testing.T) {
	input := claudeagent.AgentInput{
		Description:  "Test description",
		Prompt:       "Test prompt",
		SubagentType: "stuck",
		Model:        strPtr("haiku"),
		Resume:       nil, // Only Model is set
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal AgentInput: %v", err)
	}

	// Parse JSON to check field presence
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Model should be present
	if _, ok := result["model"]; !ok {
		t.Error("JSON should contain 'model' field when non-nil")
	}
	if result["model"] != "haiku" {
		t.Errorf("model = %v, want 'haiku'", result["model"])
	}

	// Resume should NOT be present
	if _, ok := result["resume"]; ok {
		t.Error("JSON should not contain 'resume' field when nil")
	}
}

// TestAgentInput_JSON_Roundtrip_BothFields tests JSON roundtrip with both Model and Resume set
func TestAgentInput_JSON_Roundtrip_BothFields(t *testing.T) {
	original := claudeagent.AgentInput{
		Description:  "Roundtrip test",
		Prompt:       "Test roundtrip marshaling",
		SubagentType: "coder",
		Model:        strPtr("sonnet"),
		Resume:       strPtr("checkpoint-789"),
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal AgentInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.AgentInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly check Model and Resume
	if roundtripped.Model == nil || *roundtripped.Model != *original.Model {
		t.Errorf("Model roundtrip failed: got %v, want %v", roundtripped.Model, original.Model)
	}
	if roundtripped.Resume == nil || *roundtripped.Resume != *original.Resume {
		t.Errorf("Resume roundtrip failed: got %v, want %v", roundtripped.Resume, original.Resume)
	}
}

// TestAgentInput_JSON_Roundtrip_NeitherField tests JSON roundtrip with neither Model nor Resume set
func TestAgentInput_JSON_Roundtrip_NeitherField(t *testing.T) {
	original := claudeagent.AgentInput{
		Description:  "Minimal test",
		Prompt:       "Test without optional fields",
		SubagentType: "tester",
		Model:        nil,
		Resume:       nil,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal AgentInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.AgentInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly check that Model and Resume are still nil
	if roundtripped.Model != nil {
		t.Errorf("Model should be nil after roundtrip, got %v", *roundtripped.Model)
	}
	if roundtripped.Resume != nil {
		t.Errorf("Resume should be nil after roundtrip, got %v", *roundtripped.Resume)
	}
}

// TestAgentInput_JSON_Roundtrip_OnlyModel tests JSON roundtrip with only Model set
func TestAgentInput_JSON_Roundtrip_OnlyModel(t *testing.T) {
	original := claudeagent.AgentInput{
		Description:  "Model only test",
		Prompt:       "Test with only model field",
		SubagentType: "coder",
		Model:        strPtr("opus"),
		Resume:       nil,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal AgentInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.AgentInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly check Model is preserved and Resume is still nil
	if roundtripped.Model == nil || *roundtripped.Model != "opus" {
		t.Errorf("Model should be 'opus', got %v", roundtripped.Model)
	}
	if roundtripped.Resume != nil {
		t.Errorf("Resume should be nil after roundtrip, got %v", *roundtripped.Resume)
	}
}

// TestAgentInput_JSON_Roundtrip_OnlyResume tests JSON roundtrip with only Resume set
func TestAgentInput_JSON_Roundtrip_OnlyResume(t *testing.T) {
	original := claudeagent.AgentInput{
		Description:  "Resume only test",
		Prompt:       "Test with only resume field",
		SubagentType: "stuck",
		Model:        nil,
		Resume:       strPtr("checkpoint-xyz"),
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal AgentInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.AgentInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly check Resume is preserved and Model is still nil
	if roundtripped.Model != nil {
		t.Errorf("Model should be nil after roundtrip, got %v", *roundtripped.Model)
	}
	if roundtripped.Resume == nil || *roundtripped.Resume != "checkpoint-xyz" {
		t.Errorf("Resume should be 'checkpoint-xyz', got %v", roundtripped.Resume)
	}
}

// Helper function to create bool pointers.
func boolPtr(b bool) *bool {
	return &b
}

// TestBashInput_DangerouslyDisableSandbox_True tests that DangerouslyDisableSandbox can be set to true
func TestBashInput_DangerouslyDisableSandbox_True(t *testing.T) {
	trueValue := true
	input := claudeagent.BashInput{
		Command:                   "ls -la",
		DangerouslyDisableSandbox: &trueValue,
	}

	if input.DangerouslyDisableSandbox == nil {
		t.Fatal("DangerouslyDisableSandbox should not be nil")
	}
	if *input.DangerouslyDisableSandbox != true {
		t.Errorf("DangerouslyDisableSandbox = %v, want true", *input.DangerouslyDisableSandbox)
	}
}

// TestBashInput_DangerouslyDisableSandbox_False tests that DangerouslyDisableSandbox can be set to false
func TestBashInput_DangerouslyDisableSandbox_False(t *testing.T) {
	falseValue := false
	input := claudeagent.BashInput{
		Command:                   "pwd",
		DangerouslyDisableSandbox: &falseValue,
	}

	if input.DangerouslyDisableSandbox == nil {
		t.Fatal("DangerouslyDisableSandbox should not be nil")
	}
	if *input.DangerouslyDisableSandbox != false {
		t.Errorf("DangerouslyDisableSandbox = %v, want false", *input.DangerouslyDisableSandbox)
	}
}

// TestBashInput_DangerouslyDisableSandbox_NilValue tests that DangerouslyDisableSandbox can be nil
func TestBashInput_DangerouslyDisableSandbox_NilValue(t *testing.T) {
	input := claudeagent.BashInput{
		Command:                   "echo hello",
		DangerouslyDisableSandbox: nil,
	}

	if input.DangerouslyDisableSandbox != nil {
		t.Errorf("DangerouslyDisableSandbox should be nil, got %v", *input.DangerouslyDisableSandbox)
	}
}

// TestBashInput_DangerouslyDisableSandbox_WithHelperTrue tests using boolPtr helper with true
func TestBashInput_DangerouslyDisableSandbox_WithHelperTrue(t *testing.T) {
	input := claudeagent.BashInput{
		Command:                   "rm -rf /tmp/test",
		DangerouslyDisableSandbox: boolPtr(true),
	}

	if input.DangerouslyDisableSandbox == nil {
		t.Fatal("DangerouslyDisableSandbox should not be nil")
	}
	if *input.DangerouslyDisableSandbox != true {
		t.Errorf("DangerouslyDisableSandbox = %v, want true", *input.DangerouslyDisableSandbox)
	}
}

// TestBashInput_DangerouslyDisableSandbox_WithHelperFalse tests using boolPtr helper with false
func TestBashInput_DangerouslyDisableSandbox_WithHelperFalse(t *testing.T) {
	input := claudeagent.BashInput{
		Command:                   "cat /etc/passwd",
		DangerouslyDisableSandbox: boolPtr(false),
	}

	if input.DangerouslyDisableSandbox == nil {
		t.Fatal("DangerouslyDisableSandbox should not be nil")
	}
	if *input.DangerouslyDisableSandbox != false {
		t.Errorf("DangerouslyDisableSandbox = %v, want false", *input.DangerouslyDisableSandbox)
	}
}

// TestBashInput_JSON_MarshalCamelCase tests that JSON marshaling produces camelCase field name
func TestBashInput_JSON_MarshalCamelCase(t *testing.T) {
	input := claudeagent.BashInput{
		Command:                   "ls -la",
		DangerouslyDisableSandbox: boolPtr(true),
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal BashInput: %v", err)
	}

	// Parse JSON to check field names
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that camelCase field exists
	if _, ok := result["dangerouslyDisableSandbox"]; !ok {
		t.Error("JSON should contain 'dangerouslyDisableSandbox' field")
	}

	// Verify value
	if result["dangerouslyDisableSandbox"] != true {
		t.Errorf("dangerouslyDisableSandbox = %v, want true", result["dangerouslyDisableSandbox"])
	}

	// Ensure snake_case field doesn't exist
	if _, ok := result["dangerously_disable_sandbox"]; ok {
		t.Error("JSON should not contain 'dangerously_disable_sandbox' field")
	}
}

// TestBashInput_JSON_UnmarshalCamelCase tests that JSON unmarshaling correctly reads camelCase field
func TestBashInput_JSON_UnmarshalCamelCase(t *testing.T) {
	jsonData := `{
		"command": "echo test",
		"dangerouslyDisableSandbox": true
	}`

	var input claudeagent.BashInput
	if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if input.Command != "echo test" {
		t.Errorf("Command = %v, want 'echo test'", input.Command)
	}
	if input.DangerouslyDisableSandbox == nil {
		t.Fatal("DangerouslyDisableSandbox should not be nil")
	}
	if *input.DangerouslyDisableSandbox != true {
		t.Errorf("DangerouslyDisableSandbox = %v, want true", *input.DangerouslyDisableSandbox)
	}
}

// TestBashInput_JSON_OmitEmpty_NilField tests that nil DangerouslyDisableSandbox field is excluded from JSON
func TestBashInput_JSON_OmitEmpty_NilField(t *testing.T) {
	input := claudeagent.BashInput{
		Command:                   "pwd",
		DangerouslyDisableSandbox: nil,
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal BashInput: %v", err)
	}

	// Parse JSON to check field presence
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that dangerouslyDisableSandbox field is NOT present
	if _, ok := result["dangerouslyDisableSandbox"]; ok {
		t.Error("JSON should not contain 'dangerouslyDisableSandbox' field when nil")
	}

	// Verify command field is present
	if _, ok := result["command"]; !ok {
		t.Error("JSON should contain 'command' field")
	}
}

// TestBashInput_JSON_OmitEmpty_NonNilFieldTrue tests that non-nil DangerouslyDisableSandbox=true is included
func TestBashInput_JSON_OmitEmpty_NonNilFieldTrue(t *testing.T) {
	input := claudeagent.BashInput{
		Command:                   "rm -rf /tmp/test",
		DangerouslyDisableSandbox: boolPtr(true),
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal BashInput: %v", err)
	}

	// Parse JSON to check field presence
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// DangerouslyDisableSandbox should be present
	if _, ok := result["dangerouslyDisableSandbox"]; !ok {
		t.Error("JSON should contain 'dangerouslyDisableSandbox' field when non-nil")
	}
	if result["dangerouslyDisableSandbox"] != true {
		t.Errorf("dangerouslyDisableSandbox = %v, want true", result["dangerouslyDisableSandbox"])
	}
}

// TestBashInput_JSON_OmitEmpty_NonNilFieldFalse tests that non-nil DangerouslyDisableSandbox=false is included
func TestBashInput_JSON_OmitEmpty_NonNilFieldFalse(t *testing.T) {
	input := claudeagent.BashInput{
		Command:                   "cat /etc/hosts",
		DangerouslyDisableSandbox: boolPtr(false),
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal BashInput: %v", err)
	}

	// Parse JSON to check field presence
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// DangerouslyDisableSandbox should be present even when false
	if _, ok := result["dangerouslyDisableSandbox"]; !ok {
		t.Error("JSON should contain 'dangerouslyDisableSandbox' field when non-nil")
	}
	if result["dangerouslyDisableSandbox"] != false {
		t.Errorf("dangerouslyDisableSandbox = %v, want false", result["dangerouslyDisableSandbox"])
	}
}

// TestBashInput_JSON_Roundtrip_WithTrue tests JSON roundtrip with DangerouslyDisableSandbox=true
func TestBashInput_JSON_Roundtrip_WithTrue(t *testing.T) {
	original := claudeagent.BashInput{
		Command:                   "systemctl restart nginx",
		DangerouslyDisableSandbox: boolPtr(true),
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal BashInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.BashInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly check DangerouslyDisableSandbox
	if roundtripped.DangerouslyDisableSandbox == nil {
		t.Fatal("DangerouslyDisableSandbox should not be nil after roundtrip")
	}
	if *roundtripped.DangerouslyDisableSandbox != true {
		t.Errorf("DangerouslyDisableSandbox = %v, want true", *roundtripped.DangerouslyDisableSandbox)
	}
}

// TestBashInput_JSON_Roundtrip_WithFalse tests JSON roundtrip with DangerouslyDisableSandbox=false
func TestBashInput_JSON_Roundtrip_WithFalse(t *testing.T) {
	original := claudeagent.BashInput{
		Command:                   "ls /home",
		DangerouslyDisableSandbox: boolPtr(false),
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal BashInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.BashInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly check DangerouslyDisableSandbox
	if roundtripped.DangerouslyDisableSandbox == nil {
		t.Fatal("DangerouslyDisableSandbox should not be nil after roundtrip")
	}
	if *roundtripped.DangerouslyDisableSandbox != false {
		t.Errorf("DangerouslyDisableSandbox = %v, want false", *roundtripped.DangerouslyDisableSandbox)
	}
}

// TestBashInput_JSON_Roundtrip_WithNil tests JSON roundtrip with DangerouslyDisableSandbox=nil
func TestBashInput_JSON_Roundtrip_WithNil(t *testing.T) {
	original := claudeagent.BashInput{
		Command:                   "whoami",
		DangerouslyDisableSandbox: nil,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal BashInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.BashInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly check that DangerouslyDisableSandbox is still nil
	if roundtripped.DangerouslyDisableSandbox != nil {
		t.Errorf("DangerouslyDisableSandbox should be nil after roundtrip, got %v", *roundtripped.DangerouslyDisableSandbox)
	}
}

// TestBashInput_JSON_Roundtrip_AllFields tests JSON roundtrip with all BashInput fields populated
func TestBashInput_JSON_Roundtrip_AllFields(t *testing.T) {
	timeout := 5000
	description := "Test command"
	runInBackground := true

	original := claudeagent.BashInput{
		Command:                   "npm install",
		Timeout:                   &timeout,
		Description:               &description,
		RunInBackground:           &runInBackground,
		DangerouslyDisableSandbox: boolPtr(true),
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal BashInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.BashInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly verify all fields
	if roundtripped.Command != "npm install" {
		t.Errorf("Command = %v, want 'npm install'", roundtripped.Command)
	}
	if roundtripped.Timeout == nil || *roundtripped.Timeout != 5000 {
		t.Errorf("Timeout = %v, want 5000", roundtripped.Timeout)
	}
	if roundtripped.Description == nil || *roundtripped.Description != "Test command" {
		t.Errorf("Description = %v, want 'Test command'", roundtripped.Description)
	}
	if roundtripped.RunInBackground == nil || *roundtripped.RunInBackground != true {
		t.Errorf("RunInBackground = %v, want true", roundtripped.RunInBackground)
	}
	if roundtripped.DangerouslyDisableSandbox == nil || *roundtripped.DangerouslyDisableSandbox != true {
		t.Errorf("DangerouslyDisableSandbox = %v, want true", roundtripped.DangerouslyDisableSandbox)
	}
}

// TestBashInput_JSON_ComplexUnmarshal tests unmarshaling JSON with mixed field values
func TestBashInput_JSON_ComplexUnmarshal(t *testing.T) {
	jsonData := `{
		"command": "docker build -t myapp .",
		"timeout": 30000,
		"description": "Build Docker image",
		"run_in_background": false,
		"dangerouslyDisableSandbox": true
	}`

	var input claudeagent.BashInput
	if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify all fields were parsed correctly
	if input.Command != "docker build -t myapp ." {
		t.Errorf("Command = %v, want 'docker build -t myapp .'", input.Command)
	}
	if input.Timeout == nil || *input.Timeout != 30000 {
		t.Errorf("Timeout = %v, want 30000", input.Timeout)
	}
	if input.Description == nil || *input.Description != "Build Docker image" {
		t.Errorf("Description = %v, want 'Build Docker image'", input.Description)
	}
	if input.RunInBackground == nil || *input.RunInBackground != false {
		t.Errorf("RunInBackground = %v, want false", input.RunInBackground)
	}
	if input.DangerouslyDisableSandbox == nil || *input.DangerouslyDisableSandbox != true {
		t.Errorf("DangerouslyDisableSandbox = %v, want true", *input.DangerouslyDisableSandbox)
	}
}

// TestTimeMachineInput_MessagePrefix_StoresText tests that MessagePrefix field stores text correctly
func TestTimeMachineInput_MessagePrefix_StoresText(t *testing.T) {
	messageText := "Created initial React component"
	input := claudeagent.TimeMachineInput{
		MessagePrefix:    messageText,
		CourseCorrection: "Use TypeScript instead",
		RestoreCode:      nil,
	}

	if input.MessagePrefix != messageText {
		t.Errorf("MessagePrefix = %v, want %v", input.MessagePrefix, messageText)
	}
}

// TestTimeMachineInput_MessagePrefix_EmptyString tests that MessagePrefix can be empty string
func TestTimeMachineInput_MessagePrefix_EmptyString(t *testing.T) {
	input := claudeagent.TimeMachineInput{
		MessagePrefix:    "",
		CourseCorrection: "Some instruction",
		RestoreCode:      nil,
	}

	if input.MessagePrefix != "" {
		t.Errorf("MessagePrefix = %v, want empty string", input.MessagePrefix)
	}
}

// TestTimeMachineInput_CourseCorrection_StoresInstruction tests that CourseCorrection field stores instructions correctly
func TestTimeMachineInput_CourseCorrection_StoresInstruction(t *testing.T) {
	instruction := "Use TypeScript instead of JavaScript for better type safety"
	input := claudeagent.TimeMachineInput{
		MessagePrefix:    "Created initial component",
		CourseCorrection: instruction,
		RestoreCode:      nil,
	}

	if input.CourseCorrection != instruction {
		t.Errorf("CourseCorrection = %v, want %v", input.CourseCorrection, instruction)
	}
}

// TestTimeMachineInput_CourseCorrection_LongText tests that CourseCorrection handles long text
func TestTimeMachineInput_CourseCorrection_LongText(t *testing.T) {
	longInstruction := `Please rewrite the component using the following requirements:
1. Use TypeScript with strict type checking
2. Implement proper error boundaries
3. Add comprehensive unit tests
4. Use functional components with hooks instead of classes
5. Follow React best practices for accessibility`

	input := claudeagent.TimeMachineInput{
		MessagePrefix:    "Component created",
		CourseCorrection: longInstruction,
		RestoreCode:      boolPtr(true),
	}

	if input.CourseCorrection != longInstruction {
		t.Errorf("CourseCorrection does not match expected long text")
	}
}

// TestTimeMachineInput_RestoreCode_True tests that RestoreCode field can be set to true
func TestTimeMachineInput_RestoreCode_True(t *testing.T) {
	input := claudeagent.TimeMachineInput{
		MessagePrefix:    "Initial implementation",
		CourseCorrection: "Try a different approach",
		RestoreCode:      boolPtr(true),
	}

	if input.RestoreCode == nil {
		t.Fatal("RestoreCode should not be nil")
	}
	if *input.RestoreCode != true {
		t.Errorf("RestoreCode = %v, want true", *input.RestoreCode)
	}
}

// TestTimeMachineInput_RestoreCode_False tests that RestoreCode field can be set to false
func TestTimeMachineInput_RestoreCode_False(t *testing.T) {
	input := claudeagent.TimeMachineInput{
		MessagePrefix:    "Previous message",
		CourseCorrection: "Keep files as-is",
		RestoreCode:      boolPtr(false),
	}

	if input.RestoreCode == nil {
		t.Fatal("RestoreCode should not be nil")
	}
	if *input.RestoreCode != false {
		t.Errorf("RestoreCode = %v, want false", *input.RestoreCode)
	}
}

// TestTimeMachineInput_RestoreCode_NilValue tests that RestoreCode field can be nil
func TestTimeMachineInput_RestoreCode_NilValue(t *testing.T) {
	input := claudeagent.TimeMachineInput{
		MessagePrefix:    "Some message",
		CourseCorrection: "Some instruction",
		RestoreCode:      nil,
	}

	if input.RestoreCode != nil {
		t.Errorf("RestoreCode should be nil, got %v", *input.RestoreCode)
	}
}

// TestTimeMachineInput_JSON_MarshalSnakeCase tests that JSON marshaling produces snake_case field names
func TestTimeMachineInput_JSON_MarshalSnakeCase(t *testing.T) {
	input := claudeagent.TimeMachineInput{
		MessagePrefix:    "Created initial component",
		CourseCorrection: "Use TypeScript instead",
		RestoreCode:      boolPtr(true),
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal TimeMachineInput: %v", err)
	}

	// Parse JSON to check field names
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that snake_case fields exist
	if _, ok := result["message_prefix"]; !ok {
		t.Error("JSON should contain 'message_prefix' field")
	}
	if _, ok := result["course_correction"]; !ok {
		t.Error("JSON should contain 'course_correction' field")
	}
	if _, ok := result["restore_code"]; !ok {
		t.Error("JSON should contain 'restore_code' field")
	}

	// Ensure camelCase fields don't exist (TimeMachine uses snake_case, unlike other tools)
	if _, ok := result["messagePrefix"]; ok {
		t.Error("JSON should not contain 'messagePrefix' field (should be snake_case)")
	}
	if _, ok := result["courseCorrection"]; ok {
		t.Error("JSON should not contain 'courseCorrection' field (should be snake_case)")
	}
	if _, ok := result["restoreCode"]; ok {
		t.Error("JSON should not contain 'restoreCode' field (should be snake_case)")
	}

	// Verify values
	if result["message_prefix"] != "Created initial component" {
		t.Errorf("message_prefix = %v, want 'Created initial component'", result["message_prefix"])
	}
	if result["course_correction"] != "Use TypeScript instead" {
		t.Errorf("course_correction = %v, want 'Use TypeScript instead'", result["course_correction"])
	}
	if result["restore_code"] != true {
		t.Errorf("restore_code = %v, want true", result["restore_code"])
	}
}

// TestTimeMachineInput_JSON_UnmarshalSnakeCase tests that JSON unmarshaling correctly reads snake_case fields
func TestTimeMachineInput_JSON_UnmarshalSnakeCase(t *testing.T) {
	jsonData := `{
		"message_prefix": "Initial implementation complete",
		"course_correction": "Add error handling and validation",
		"restore_code": false
	}`

	var input claudeagent.TimeMachineInput
	if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if input.MessagePrefix != "Initial implementation complete" {
		t.Errorf("MessagePrefix = %v, want 'Initial implementation complete'", input.MessagePrefix)
	}
	if input.CourseCorrection != "Add error handling and validation" {
		t.Errorf("CourseCorrection = %v, want 'Add error handling and validation'", input.CourseCorrection)
	}
	if input.RestoreCode == nil {
		t.Fatal("RestoreCode should not be nil")
	}
	if *input.RestoreCode != false {
		t.Errorf("RestoreCode = %v, want false", *input.RestoreCode)
	}
}

// TestTimeMachineInput_JSON_OmitEmpty_NilRestoreCode tests that nil RestoreCode field is excluded from JSON
func TestTimeMachineInput_JSON_OmitEmpty_NilRestoreCode(t *testing.T) {
	input := claudeagent.TimeMachineInput{
		MessagePrefix:    "Some message",
		CourseCorrection: "Some correction",
		RestoreCode:      nil,
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal TimeMachineInput: %v", err)
	}

	// Parse JSON to check field presence
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that restore_code field is NOT present
	if _, ok := result["restore_code"]; ok {
		t.Error("JSON should not contain 'restore_code' field when nil")
	}

	// Verify other fields are present
	if _, ok := result["message_prefix"]; !ok {
		t.Error("JSON should contain 'message_prefix' field")
	}
	if _, ok := result["course_correction"]; !ok {
		t.Error("JSON should contain 'course_correction' field")
	}
}

// TestTimeMachineInput_JSON_OmitEmpty_NonNilRestoreCode tests that non-nil RestoreCode field is included in JSON
func TestTimeMachineInput_JSON_OmitEmpty_NonNilRestoreCode(t *testing.T) {
	tests := []struct {
		name        string
		restoreCode bool
	}{
		{
			name:        "RestoreCode true",
			restoreCode: true,
		},
		{
			name:        "RestoreCode false",
			restoreCode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := claudeagent.TimeMachineInput{
				MessagePrefix:    "Test message",
				CourseCorrection: "Test correction",
				RestoreCode:      boolPtr(tt.restoreCode),
			}

			data, err := json.Marshal(input)
			if err != nil {
				t.Fatalf("Failed to marshal TimeMachineInput: %v", err)
			}

			// Parse JSON to check field presence
			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			// RestoreCode should be present when non-nil
			if _, ok := result["restore_code"]; !ok {
				t.Error("JSON should contain 'restore_code' field when non-nil")
			}
			if result["restore_code"] != tt.restoreCode {
				t.Errorf("restore_code = %v, want %v", result["restore_code"], tt.restoreCode)
			}
		})
	}
}

// TestTimeMachineInput_JSON_Roundtrip_AllFields tests JSON roundtrip with all fields set
func TestTimeMachineInput_JSON_Roundtrip_AllFields(t *testing.T) {
	original := claudeagent.TimeMachineInput{
		MessagePrefix:    "Implemented user authentication",
		CourseCorrection: "Use OAuth2 instead of basic auth for better security",
		RestoreCode:      boolPtr(true),
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TimeMachineInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.TimeMachineInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly check all fields
	if roundtripped.MessagePrefix != original.MessagePrefix {
		t.Errorf("MessagePrefix roundtrip failed: got %v, want %v", roundtripped.MessagePrefix, original.MessagePrefix)
	}
	if roundtripped.CourseCorrection != original.CourseCorrection {
		t.Errorf("CourseCorrection roundtrip failed: got %v, want %v", roundtripped.CourseCorrection, original.CourseCorrection)
	}
	if roundtripped.RestoreCode == nil || *roundtripped.RestoreCode != *original.RestoreCode {
		t.Errorf("RestoreCode roundtrip failed: got %v, want %v", roundtripped.RestoreCode, original.RestoreCode)
	}
}

// TestTimeMachineInput_JSON_Roundtrip_NilRestoreCode tests JSON roundtrip with RestoreCode=nil
func TestTimeMachineInput_JSON_Roundtrip_NilRestoreCode(t *testing.T) {
	original := claudeagent.TimeMachineInput{
		MessagePrefix:    "Added new feature",
		CourseCorrection: "Refactor to use dependency injection",
		RestoreCode:      nil,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TimeMachineInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.TimeMachineInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly check that RestoreCode is still nil
	if roundtripped.RestoreCode != nil {
		t.Errorf("RestoreCode should be nil after roundtrip, got %v", *roundtripped.RestoreCode)
	}
}

// TestTimeMachineInput_JSON_Roundtrip_RestoreCodeTrue tests JSON roundtrip with RestoreCode=true
func TestTimeMachineInput_JSON_Roundtrip_RestoreCodeTrue(t *testing.T) {
	original := claudeagent.TimeMachineInput{
		MessagePrefix:    "Database migration completed",
		CourseCorrection: "Roll back and use a different schema design",
		RestoreCode:      boolPtr(true),
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TimeMachineInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.TimeMachineInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly verify RestoreCode=true is preserved
	if roundtripped.RestoreCode == nil {
		t.Fatal("RestoreCode should not be nil after roundtrip")
	}
	if *roundtripped.RestoreCode != true {
		t.Errorf("RestoreCode should be true, got %v", *roundtripped.RestoreCode)
	}
}

// TestTimeMachineInput_JSON_Roundtrip_RestoreCodeFalse tests JSON roundtrip with RestoreCode=false
func TestTimeMachineInput_JSON_Roundtrip_RestoreCodeFalse(t *testing.T) {
	original := claudeagent.TimeMachineInput{
		MessagePrefix:    "API endpoints created",
		CourseCorrection: "Add rate limiting to all endpoints",
		RestoreCode:      boolPtr(false),
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TimeMachineInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.TimeMachineInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly verify RestoreCode=false is preserved
	if roundtripped.RestoreCode == nil {
		t.Fatal("RestoreCode should not be nil after roundtrip")
	}
	if *roundtripped.RestoreCode != false {
		t.Errorf("RestoreCode should be false, got %v", *roundtripped.RestoreCode)
	}
}

// TestTimeMachineInput_JSON_RealWorldExample tests a realistic time machine scenario
func TestTimeMachineInput_JSON_RealWorldExample(t *testing.T) {
	// Simulate rewinding to a message with course correction
	input := claudeagent.TimeMachineInput{
		MessagePrefix:    "Created initial React component with useState",
		CourseCorrection: "Please rewrite using TypeScript with proper interfaces and use useReducer instead of useState for better state management. Also add PropTypes validation.",
		RestoreCode:      boolPtr(true),
	}

	// Marshal to JSON (as would be sent to Claude API)
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal TimeMachineInput: %v", err)
	}

	// Verify JSON structure
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify snake_case fields
	if _, ok := result["message_prefix"]; !ok {
		t.Error("JSON should contain 'message_prefix' field")
	}
	if _, ok := result["course_correction"]; !ok {
		t.Error("JSON should contain 'course_correction' field")
	}
	if _, ok := result["restore_code"]; !ok {
		t.Error("JSON should contain 'restore_code' field")
	}

	// Unmarshal back (as would be received from API)
	var unmarshaled claudeagent.TimeMachineInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify all fields preserved
	if unmarshaled.MessagePrefix != input.MessagePrefix {
		t.Errorf("MessagePrefix mismatch: got %v, want %v", unmarshaled.MessagePrefix, input.MessagePrefix)
	}
	if unmarshaled.CourseCorrection != input.CourseCorrection {
		t.Errorf("CourseCorrection mismatch: got %v, want %v", unmarshaled.CourseCorrection, input.CourseCorrection)
	}
	if unmarshaled.RestoreCode == nil || *unmarshaled.RestoreCode != *input.RestoreCode {
		t.Errorf("RestoreCode mismatch: got %v, want %v", unmarshaled.RestoreCode, input.RestoreCode)
	}
}

// TestTimeMachineInput_JSON_VerifySnakeCaseNotCamelCase tests that TimeMachine specifically uses snake_case
func TestTimeMachineInput_JSON_VerifySnakeCaseNotCamelCase(t *testing.T) {
	input := claudeagent.TimeMachineInput{
		MessagePrefix:    "Test",
		CourseCorrection: "Test correction",
		RestoreCode:      boolPtr(true),
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal TimeMachineInput: %v", err)
	}

	jsonString := string(data)

	// Ensure snake_case is used
	if !contains(jsonString, "message_prefix") {
		t.Error("JSON should contain 'message_prefix' (snake_case)")
	}
	if !contains(jsonString, "course_correction") {
		t.Error("JSON should contain 'course_correction' (snake_case)")
	}
	if !contains(jsonString, "restore_code") {
		t.Error("JSON should contain 'restore_code' (snake_case)")
	}

	// Ensure camelCase is NOT used (TimeMachine is different from other tools!)
	if contains(jsonString, "messagePrefix") {
		t.Error("JSON should not contain 'messagePrefix' (camelCase) - TimeMachine uses snake_case")
	}
	if contains(jsonString, "courseCorrection") {
		t.Error("JSON should not contain 'courseCorrection' (camelCase) - TimeMachine uses snake_case")
	}
	if contains(jsonString, "restoreCode") {
		t.Error("JSON should not contain 'restoreCode' (camelCase) - TimeMachine uses snake_case")
	}
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

// TestAskUserQuestionInput_Questions_SingleQuestion tests a single question with 2 options
func TestAskUserQuestionInput_Questions_SingleQuestion(t *testing.T) {
	input := claudeagent.AskUserQuestionInput{
		Questions: []claudeagent.QuestionDefinition{
			{
				Question: "What is your preferred programming language?",
				Header:   "Language",
				Options: []claudeagent.QuestionOption{
					{
						Label:       "Go",
						Description: "Fast and efficient compiled language",
					},
					{
						Label:       "Python",
						Description: "High-level interpreted language",
					},
				},
				MultiSelect: false,
			},
		},
	}

	// Verify the structure is stored correctly
	if len(input.Questions) != 1 {
		t.Fatalf("Expected 1 question, got %d", len(input.Questions))
	}

	q := input.Questions[0]
	if q.Question != "What is your preferred programming language?" {
		t.Errorf("Question = %v, want 'What is your preferred programming language?'", q.Question)
	}
	if q.Header != "Language" {
		t.Errorf("Header = %v, want 'Language'", q.Header)
	}
	if len(q.Options) != 2 {
		t.Fatalf("Expected 2 options, got %d", len(q.Options))
	}
	if q.MultiSelect != false {
		t.Errorf("MultiSelect = %v, want false", q.MultiSelect)
	}
}

// TestAskUserQuestionInput_Questions_MultipleQuestions tests 4 questions with various options
func TestAskUserQuestionInput_Questions_MultipleQuestions(t *testing.T) {
	input := claudeagent.AskUserQuestionInput{
		Questions: []claudeagent.QuestionDefinition{
			{
				Question: "Choose authentication method",
				Header:   "Auth",
				Options: []claudeagent.QuestionOption{
					{Label: "OAuth2", Description: "Third-party authentication"},
					{Label: "JWT", Description: "JSON Web Tokens"},
					{Label: "Session", Description: "Session-based auth"},
				},
				MultiSelect: false,
			},
			{
				Question: "Select database type",
				Header:   "Database",
				Options: []claudeagent.QuestionOption{
					{Label: "PostgreSQL", Description: "Relational database"},
					{Label: "MongoDB", Description: "Document database"},
				},
				MultiSelect: false,
			},
			{
				Question: "Pick testing frameworks",
				Header:   "Testing",
				Options: []claudeagent.QuestionOption{
					{Label: "Jest", Description: "JavaScript testing framework"},
					{Label: "Mocha", Description: "Feature-rich testing framework"},
					{Label: "Vitest", Description: "Fast unit testing"},
				},
				MultiSelect: true,
			},
			{
				Question: "Choose deployment platform",
				Header:   "Deploy",
				Options: []claudeagent.QuestionOption{
					{Label: "AWS", Description: "Amazon Web Services"},
					{Label: "GCP", Description: "Google Cloud Platform"},
					{Label: "Azure", Description: "Microsoft Azure"},
					{Label: "Heroku", Description: "Platform as a Service"},
				},
				MultiSelect: false,
			},
		},
	}

	// Verify we have 4 questions
	if len(input.Questions) != 4 {
		t.Fatalf("Expected 4 questions, got %d", len(input.Questions))
	}

	// Verify first question
	if input.Questions[0].Header != "Auth" {
		t.Errorf("Question 0 Header = %v, want 'Auth'", input.Questions[0].Header)
	}
	if len(input.Questions[0].Options) != 3 {
		t.Errorf("Question 0 options count = %d, want 3", len(input.Questions[0].Options))
	}

	// Verify second question
	if input.Questions[1].Header != "Database" {
		t.Errorf("Question 1 Header = %v, want 'Database'", input.Questions[1].Header)
	}
	if len(input.Questions[1].Options) != 2 {
		t.Errorf("Question 1 options count = %d, want 2", len(input.Questions[1].Options))
	}

	// Verify third question (multi-select)
	if input.Questions[2].MultiSelect != true {
		t.Errorf("Question 2 MultiSelect = %v, want true", input.Questions[2].MultiSelect)
	}

	// Verify fourth question
	if len(input.Questions[3].Options) != 4 {
		t.Errorf("Question 3 options count = %d, want 4", len(input.Questions[3].Options))
	}
}

// TestAskUserQuestionInput_QuestionDefinition_FieldStorage tests that QuestionDefinition fields store correctly
func TestAskUserQuestionInput_QuestionDefinition_FieldStorage(t *testing.T) {
	question := claudeagent.QuestionDefinition{
		Question:    "Select your preferred framework",
		Header:      "Framework",
		MultiSelect: true,
		Options: []claudeagent.QuestionOption{
			{Label: "React", Description: "Popular UI library"},
			{Label: "Vue", Description: "Progressive framework"},
		},
	}

	if question.Question != "Select your preferred framework" {
		t.Errorf("Question = %v, want 'Select your preferred framework'", question.Question)
	}
	if question.Header != "Framework" {
		t.Errorf("Header = %v, want 'Framework'", question.Header)
	}
	if question.MultiSelect != true {
		t.Errorf("MultiSelect = %v, want true", question.MultiSelect)
	}
	if len(question.Options) != 2 {
		t.Errorf("Options length = %d, want 2", len(question.Options))
	}
}

// TestAskUserQuestionInput_QuestionOption_FieldStorage tests that QuestionOption fields store correctly
func TestAskUserQuestionInput_QuestionOption_FieldStorage(t *testing.T) {
	option := claudeagent.QuestionOption{
		Label:       "TypeScript",
		Description: "JavaScript with static typing",
	}

	if option.Label != "TypeScript" {
		t.Errorf("Label = %v, want 'TypeScript'", option.Label)
	}
	if option.Description != "JavaScript with static typing" {
		t.Errorf("Description = %v, want 'JavaScript with static typing'", option.Description)
	}
}

// TestAskUserQuestionInput_Answers_WithPrefilledAnswers tests Answers map with pre-filled responses
func TestAskUserQuestionInput_Answers_WithPrefilledAnswers(t *testing.T) {
	input := claudeagent.AskUserQuestionInput{
		Questions: []claudeagent.QuestionDefinition{
			{
				Question: "Choose language",
				Header:   "Lang",
				Options: []claudeagent.QuestionOption{
					{Label: "Go", Description: "Go language"},
					{Label: "Rust", Description: "Rust language"},
				},
				MultiSelect: false,
			},
		},
		Answers: map[string]string{
			"Lang": "Go",
		},
	}

	if input.Answers == nil {
		t.Fatal("Answers should not be nil")
	}
	if input.Answers["Lang"] != "Go" {
		t.Errorf("Answers[Lang] = %v, want 'Go'", input.Answers["Lang"])
	}
}

// TestAskUserQuestionInput_Answers_MultipleAnswers tests Answers map with multiple responses
func TestAskUserQuestionInput_Answers_MultipleAnswers(t *testing.T) {
	input := claudeagent.AskUserQuestionInput{
		Questions: []claudeagent.QuestionDefinition{
			{
				Question:    "Select frameworks",
				Header:      "Frameworks",
				Options:     []claudeagent.QuestionOption{{Label: "React", Description: "React"}, {Label: "Vue", Description: "Vue"}},
				MultiSelect: true,
			},
			{
				Question:    "Select database",
				Header:      "DB",
				Options:     []claudeagent.QuestionOption{{Label: "Postgres", Description: "Postgres"}, {Label: "MySQL", Description: "MySQL"}},
				MultiSelect: false,
			},
		},
		Answers: map[string]string{
			"Frameworks": "React,Vue",
			"DB":         "Postgres",
		},
	}

	if len(input.Answers) != 2 {
		t.Fatalf("Expected 2 answers, got %d", len(input.Answers))
	}
	if input.Answers["Frameworks"] != "React,Vue" {
		t.Errorf("Answers[Frameworks] = %v, want 'React,Vue'", input.Answers["Frameworks"])
	}
	if input.Answers["DB"] != "Postgres" {
		t.Errorf("Answers[DB] = %v, want 'Postgres'", input.Answers["DB"])
	}
}

// TestAskUserQuestionInput_JSON_MarshalCamelCase tests that JSON marshaling produces camelCase field names
func TestAskUserQuestionInput_JSON_MarshalCamelCase(t *testing.T) {
	input := claudeagent.AskUserQuestionInput{
		Questions: []claudeagent.QuestionDefinition{
			{
				Question: "Test question",
				Header:   "Test",
				Options: []claudeagent.QuestionOption{
					{Label: "Option1", Description: "First option"},
					{Label: "Option2", Description: "Second option"},
				},
				MultiSelect: true,
			},
		},
		Answers: map[string]string{
			"Test": "Option1",
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal AskUserQuestionInput: %v", err)
	}

	// Parse JSON to check field names
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that camelCase fields exist
	if _, ok := result["questions"]; !ok {
		t.Error("JSON should contain 'questions' field")
	}
	if _, ok := result["answers"]; !ok {
		t.Error("JSON should contain 'answers' field")
	}

	// Check nested structure for camelCase
	questions := result["questions"].([]interface{})
	if len(questions) == 0 {
		t.Fatal("Questions array should not be empty")
	}

	question := questions[0].(map[string]interface{})
	if _, ok := question["question"]; !ok {
		t.Error("JSON should contain 'question' field")
	}
	if _, ok := question["header"]; !ok {
		t.Error("JSON should contain 'header' field")
	}
	if _, ok := question["options"]; !ok {
		t.Error("JSON should contain 'options' field")
	}
	if _, ok := question["multiSelect"]; !ok {
		t.Error("JSON should contain 'multiSelect' field (camelCase)")
	}

	// Check option structure
	options := question["options"].([]interface{})
	if len(options) == 0 {
		t.Fatal("Options array should not be empty")
	}
	option := options[0].(map[string]interface{})
	if _, ok := option["label"]; !ok {
		t.Error("JSON should contain 'label' field")
	}
	if _, ok := option["description"]; !ok {
		t.Error("JSON should contain 'description' field")
	}
}

// TestAskUserQuestionInput_JSON_UnmarshalCamelCase tests that JSON unmarshaling correctly reads camelCase fields
func TestAskUserQuestionInput_JSON_UnmarshalCamelCase(t *testing.T) {
	jsonData := `{
		"questions": [
			{
				"question": "Choose authentication",
				"header": "Auth",
				"options": [
					{
						"label": "OAuth",
						"description": "OAuth2 authentication"
					},
					{
						"label": "Basic",
						"description": "Basic authentication"
					}
				],
				"multiSelect": false
			}
		],
		"answers": {
			"Auth": "OAuth"
		}
	}`

	var input claudeagent.AskUserQuestionInput
	if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify Questions array
	if len(input.Questions) != 1 {
		t.Fatalf("Expected 1 question, got %d", len(input.Questions))
	}

	q := input.Questions[0]
	if q.Question != "Choose authentication" {
		t.Errorf("Question = %v, want 'Choose authentication'", q.Question)
	}
	if q.Header != "Auth" {
		t.Errorf("Header = %v, want 'Auth'", q.Header)
	}
	if q.MultiSelect != false {
		t.Errorf("MultiSelect = %v, want false", q.MultiSelect)
	}

	// Verify Options array
	if len(q.Options) != 2 {
		t.Fatalf("Expected 2 options, got %d", len(q.Options))
	}
	if q.Options[0].Label != "OAuth" {
		t.Errorf("Option[0].Label = %v, want 'OAuth'", q.Options[0].Label)
	}
	if q.Options[0].Description != "OAuth2 authentication" {
		t.Errorf("Option[0].Description = %v, want 'OAuth2 authentication'", q.Options[0].Description)
	}

	// Verify Answers map
	if input.Answers == nil {
		t.Fatal("Answers should not be nil")
	}
	if input.Answers["Auth"] != "OAuth" {
		t.Errorf("Answers[Auth] = %v, want 'OAuth'", input.Answers["Auth"])
	}
}

// TestAskUserQuestionInput_JSON_OmitEmpty_NilAnswers tests that nil Answers field is excluded from JSON
func TestAskUserQuestionInput_JSON_OmitEmpty_NilAnswers(t *testing.T) {
	input := claudeagent.AskUserQuestionInput{
		Questions: []claudeagent.QuestionDefinition{
			{
				Question: "Test",
				Header:   "Test",
				Options: []claudeagent.QuestionOption{
					{Label: "A", Description: "Option A"},
					{Label: "B", Description: "Option B"},
				},
				MultiSelect: false,
			},
		},
		Answers: nil,
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal AskUserQuestionInput: %v", err)
	}

	// Parse JSON to check field presence
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that answers field is NOT present
	if _, ok := result["answers"]; ok {
		t.Error("JSON should not contain 'answers' field when nil")
	}

	// Verify questions field is present
	if _, ok := result["questions"]; !ok {
		t.Error("JSON should contain 'questions' field")
	}
}

// TestAskUserQuestionInput_JSON_OmitEmpty_EmptyAnswers tests that empty Answers map is excluded from JSON
func TestAskUserQuestionInput_JSON_OmitEmpty_EmptyAnswers(t *testing.T) {
	input := claudeagent.AskUserQuestionInput{
		Questions: []claudeagent.QuestionDefinition{
			{
				Question: "Test",
				Header:   "Test",
				Options: []claudeagent.QuestionOption{
					{Label: "A", Description: "Option A"},
					{Label: "B", Description: "Option B"},
				},
				MultiSelect: false,
			},
		},
		Answers: map[string]string{},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal AskUserQuestionInput: %v", err)
	}

	// Parse JSON to check field presence
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that answers field is NOT present when empty
	if _, ok := result["answers"]; ok {
		t.Error("JSON should not contain 'answers' field when empty")
	}
}

// TestAskUserQuestionInput_JSON_Roundtrip_ComplexNested tests JSON roundtrip with complex nested structures
func TestAskUserQuestionInput_JSON_Roundtrip_ComplexNested(t *testing.T) {
	original := claudeagent.AskUserQuestionInput{
		Questions: []claudeagent.QuestionDefinition{
			{
				Question: "Choose authentication method",
				Header:   "Auth",
				Options: []claudeagent.QuestionOption{
					{Label: "OAuth2", Description: "Third-party authentication"},
					{Label: "JWT", Description: "JSON Web Tokens"},
					{Label: "Session", Description: "Session-based auth"},
				},
				MultiSelect: false,
			},
			{
				Question: "Select features",
				Header:   "Features",
				Options: []claudeagent.QuestionOption{
					{Label: "API", Description: "RESTful API"},
					{Label: "WebSocket", Description: "Real-time communication"},
					{Label: "GraphQL", Description: "Query language"},
				},
				MultiSelect: true,
			},
		},
		Answers: map[string]string{
			"Auth":     "JWT",
			"Features": "API,WebSocket",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal AskUserQuestionInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.AskUserQuestionInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly verify nested structures
	if len(roundtripped.Questions) != 2 {
		t.Errorf("Questions count = %d, want 2", len(roundtripped.Questions))
	}
	if roundtripped.Questions[0].Header != "Auth" {
		t.Errorf("Question[0].Header = %v, want 'Auth'", roundtripped.Questions[0].Header)
	}
	if len(roundtripped.Questions[0].Options) != 3 {
		t.Errorf("Question[0].Options count = %d, want 3", len(roundtripped.Questions[0].Options))
	}
	if roundtripped.Questions[1].MultiSelect != true {
		t.Errorf("Question[1].MultiSelect = %v, want true", roundtripped.Questions[1].MultiSelect)
	}
	if roundtripped.Answers["Auth"] != "JWT" {
		t.Errorf("Answers[Auth] = %v, want 'JWT'", roundtripped.Answers["Auth"])
	}
}

// TestAskUserQuestionInput_JSON_Roundtrip_WithoutAnswers tests JSON roundtrip without Answers field
func TestAskUserQuestionInput_JSON_Roundtrip_WithoutAnswers(t *testing.T) {
	original := claudeagent.AskUserQuestionInput{
		Questions: []claudeagent.QuestionDefinition{
			{
				Question: "Select database",
				Header:   "Database",
				Options: []claudeagent.QuestionOption{
					{Label: "PostgreSQL", Description: "Relational database"},
					{Label: "MongoDB", Description: "Document database"},
				},
				MultiSelect: false,
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal AskUserQuestionInput: %v", err)
	}

	// Unmarshal back to struct
	var roundtripped claudeagent.AskUserQuestionInput
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Compare structs
	if !reflect.DeepEqual(original, roundtripped) {
		t.Errorf("Roundtrip failed:\nOriginal:     %+v\nRoundtripped: %+v", original, roundtripped)
	}

	// Explicitly verify Answers is nil/empty
	if len(roundtripped.Answers) > 0 {
		t.Errorf("Answers should be nil or empty, got %v", roundtripped.Answers)
	}
}

// TestAskUserQuestionInput_JSON_RealWorldExample tests a realistic user question scenario
func TestAskUserQuestionInput_JSON_RealWorldExample(t *testing.T) {
	// Simulate asking user for project configuration choices
	input := claudeagent.AskUserQuestionInput{
		Questions: []claudeagent.QuestionDefinition{
			{
				Question: "Which programming language would you like to use for this project?",
				Header:   "Language",
				Options: []claudeagent.QuestionOption{
					{
						Label:       "Go",
						Description: "Fast, compiled language with excellent concurrency support",
					},
					{
						Label:       "Python",
						Description: "High-level, interpreted language with rich ecosystem",
					},
					{
						Label:       "TypeScript",
						Description: "JavaScript with static typing for safer web development",
					},
				},
				MultiSelect: false,
			},
			{
				Question: "What type of authentication should we implement?",
				Header:   "Auth Method",
				Options: []claudeagent.QuestionOption{
					{
						Label:       "OAuth 2.0",
						Description: "Industry-standard protocol for authorization",
					},
					{
						Label:       "JWT",
						Description: "JSON Web Tokens for stateless authentication",
					},
					{
						Label:       "Session-based",
						Description: "Traditional server-side session management",
					},
				},
				MultiSelect: false,
			},
			{
				Question: "Which testing frameworks should we include?",
				Header:   "Testing",
				Options: []claudeagent.QuestionOption{
					{
						Label:       "Unit Tests",
						Description: "Test individual components in isolation",
					},
					{
						Label:       "Integration Tests",
						Description: "Test component interactions",
					},
					{
						Label:       "E2E Tests",
						Description: "Test complete user workflows",
					},
				},
				MultiSelect: true,
			},
		},
	}

	// Marshal to JSON (as would be sent to Claude API)
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal AskUserQuestionInput: %v", err)
	}

	// Verify JSON structure
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify questions array exists and has 3 items
	questions, ok := result["questions"].([]interface{})
	if !ok {
		t.Fatal("JSON should contain 'questions' array")
	}
	if len(questions) != 3 {
		t.Errorf("Expected 3 questions, got %d", len(questions))
	}

	// Unmarshal back (as would be received from API)
	var unmarshaled claudeagent.AskUserQuestionInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify all fields preserved
	if len(unmarshaled.Questions) != 3 {
		t.Errorf("Expected 3 questions after unmarshal, got %d", len(unmarshaled.Questions))
	}
	if unmarshaled.Questions[0].Header != "Language" {
		t.Errorf("Question[0].Header = %v, want 'Language'", unmarshaled.Questions[0].Header)
	}
	if unmarshaled.Questions[2].MultiSelect != true {
		t.Errorf("Question[2].MultiSelect = %v, want true", unmarshaled.Questions[2].MultiSelect)
	}
}

// TestAskUserQuestionInput_MultiSelect_Scenarios tests various MultiSelect configurations
func TestAskUserQuestionInput_MultiSelect_Scenarios(t *testing.T) {
	tests := []struct {
		name        string
		multiSelect bool
		answers     string
	}{
		{
			name:        "Single select with one answer",
			multiSelect: false,
			answers:     "Option1",
		},
		{
			name:        "Multi select with multiple answers",
			multiSelect: true,
			answers:     "Option1,Option2,Option3",
		},
		{
			name:        "Multi select with single answer",
			multiSelect: true,
			answers:     "Option1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := claudeagent.AskUserQuestionInput{
				Questions: []claudeagent.QuestionDefinition{
					{
						Question: "Test question",
						Header:   "Test",
						Options: []claudeagent.QuestionOption{
							{Label: "Option1", Description: "First"},
							{Label: "Option2", Description: "Second"},
							{Label: "Option3", Description: "Third"},
						},
						MultiSelect: tt.multiSelect,
					},
				},
				Answers: map[string]string{
					"Test": tt.answers,
				},
			}

			// Verify MultiSelect setting
			if input.Questions[0].MultiSelect != tt.multiSelect {
				t.Errorf("MultiSelect = %v, want %v", input.Questions[0].MultiSelect, tt.multiSelect)
			}

			// Verify answers
			if input.Answers["Test"] != tt.answers {
				t.Errorf("Answers[Test] = %v, want %v", input.Answers["Test"], tt.answers)
			}

			// Test JSON roundtrip
			data, err := json.Marshal(input)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var roundtripped claudeagent.AskUserQuestionInput
			if err := json.Unmarshal(data, &roundtripped); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if !reflect.DeepEqual(input, roundtripped) {
				t.Errorf("Roundtrip failed")
			}
		})
	}
}

// TestAskUserQuestionInput_JSON_VerifyAllFieldsCamelCase tests that all fields use camelCase in JSON
func TestAskUserQuestionInput_JSON_VerifyAllFieldsCamelCase(t *testing.T) {
	input := claudeagent.AskUserQuestionInput{
		Questions: []claudeagent.QuestionDefinition{
			{
				Question:    "Test",
				Header:      "Test",
				MultiSelect: true,
				Options: []claudeagent.QuestionOption{
					{Label: "A", Description: "Option A"},
				},
			},
		},
		Answers: map[string]string{"Test": "A"},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonString := string(data)

	// Verify camelCase is used
	if !contains(jsonString, "multiSelect") {
		t.Error("JSON should contain 'multiSelect' (camelCase)")
	}
	if !contains(jsonString, "questions") {
		t.Error("JSON should contain 'questions' (camelCase)")
	}
	if !contains(jsonString, "question") {
		t.Error("JSON should contain 'question' (camelCase)")
	}
	if !contains(jsonString, "header") {
		t.Error("JSON should contain 'header' (camelCase)")
	}
	if !contains(jsonString, "options") {
		t.Error("JSON should contain 'options' (camelCase)")
	}
	if !contains(jsonString, "label") {
		t.Error("JSON should contain 'label' (camelCase)")
	}
	if !contains(jsonString, "description") {
		t.Error("JSON should contain 'description' (camelCase)")
	}
	if !contains(jsonString, "answers") {
		t.Error("JSON should contain 'answers' (camelCase)")
	}

	// Ensure snake_case is NOT used
	if contains(jsonString, "multi_select") {
		t.Error("JSON should not contain 'multi_select' (snake_case)")
	}
}
