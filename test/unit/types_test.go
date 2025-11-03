package unit

import (
	"encoding/json"
	"testing"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
)

func TestPermissionBehavior(t *testing.T) {
	tests := []struct {
		name     string
		behavior claude.PermissionBehavior
		expected string
	}{
		{
			"Allow",
			claude.PermissionBehaviorAllow,
			"allow",
		},
		{
			"Deny",
			claude.PermissionBehaviorDeny,
			"deny",
		},
		{
			"Ask",
			claude.PermissionBehaviorAsk,
			"ask",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.behavior) != tt.expected {
				t.Errorf(
					"expected %s, got %s",
					tt.expected,
					string(tt.behavior),
				)
			}
		})
	}
}

func TestPermissionMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     claude.PermissionMode
		expected string
	}{
		{
			"Default",
			claude.PermissionModeDefault,
			"default",
		},
		{
			"AcceptEdits",
			claude.PermissionModeAcceptEdits,
			"acceptEdits",
		},
		{
			"BypassPermissions",
			claude.PermissionModeBypassPermissions,
			"bypassPermissions",
		},
		{
			"Plan",
			claude.PermissionModePlan,
			"plan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.mode) != tt.expected {
				t.Errorf(
					"expected %s, got %s",
					tt.expected,
					string(tt.mode),
				)
			}
		})
	}
}

func TestApiKeySource(t *testing.T) {
	tests := []struct {
		name     string
		source   claude.APIKeySource
		expected string
	}{
		{"User", claude.APIKeySourceUser, "user"},
		{"Project", claude.APIKeySourceProject, "project"},
		{"Org", claude.APIKeySourceOrg, "org"},
		{"Temporary", claude.APIKeySourceTemporary, "temporary"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.source) != tt.expected {
				t.Errorf(
					"expected %s, got %s",
					tt.expected,
					string(tt.source),
				)
			}
		})
	}
}

func TestUsageSerialization(t *testing.T) {
	usage := claude.Usage{
		InputTokens:              100,
		OutputTokens:             50,
		CacheReadInputTokens:     10,
		CacheCreationInputTokens: 5,
	}

	data, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("failed to marshal usage: %v", err)
	}

	var decoded claude.Usage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal usage: %v", err)
	}

	if usage != decoded {
		t.Errorf("usage mismatch: expected %+v, got %+v", usage, decoded)
	}
}

func TestModelUsageSerialization(t *testing.T) {
	usage := claude.ModelUsage{
		InputTokens:              100,
		OutputTokens:             50,
		CacheReadInputTokens:     10,
		CacheCreationInputTokens: 5,
		WebSearchRequests:        2,
		CostUSD:                  0.0015,
		ContextWindow:            200000,
	}

	data, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("failed to marshal model usage: %v", err)
	}

	// Check that webSearchRequests is in camelCase
	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	if err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	_, ok := raw["webSearchRequests"]
	if !ok {
		t.Error("expected webSearchRequests in camelCase")
	}

	var decoded claude.ModelUsage
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf(
			"failed to unmarshal model usage: %v",
			err,
		)
	}

	if usage != decoded {
		t.Errorf(
			"model usage mismatch: expected %+v, got %+v",
			usage,
			decoded,
		)
	}
}

func TestPermissionRuleValue(t *testing.T) {
	ruleContent := "test rule"
	rule := claude.PermissionRuleValue{
		ToolName:    "Read",
		RuleContent: &ruleContent,
	}

	data, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("failed to marshal permission rule: %v", err)
	}

	var decoded claude.PermissionRuleValue
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal permission rule: %v", err)
	}

	if rule.ToolName != decoded.ToolName {
		t.Errorf(
			"tool name mismatch: expected %s, got %s",
			rule.ToolName,
			decoded.ToolName,
		)
	}

	if rule.RuleContent == nil || decoded.RuleContent == nil {
		t.Fatal("rule content should not be nil")
	}

	if *rule.RuleContent != *decoded.RuleContent {
		t.Errorf(
			"rule content mismatch: expected %s, got %s",
			*rule.RuleContent,
			*decoded.RuleContent,
		)
	}
}

func TestAgentDefinitionSerialization(t *testing.T) {
	tests := []struct {
		name     string
		agent    claude.AgentDefinition
		wantJSON string
	}{
		{
			name: "Basic agent with tools allowlist",
			agent: claude.AgentDefinition{
				Description: "Test agent",
				Prompt:      "You are a test agent",
				Tools:       []string{"Read", "Write"},
				Model:       "claude-sonnet-4-5",
			},
			wantJSON: `{"description":"Test agent","prompt":"You are a test agent","tools":["Read","Write"],"model":"claude-sonnet-4-5"}`,
		},
		{
			name: "Agent with disallowedTools",
			agent: claude.AgentDefinition{
				Description:     "Test agent with disallowed tools",
				Prompt:          "You are restricted",
				DisallowedTools: []string{"Bash", "WebSearch"},
				Model:           "claude-sonnet-4-5",
			},
			wantJSON: `{"description":"Test agent with disallowed tools","prompt":"You are restricted","tools":null,"disallowedTools":["Bash","WebSearch"],"model":"claude-sonnet-4-5"}`,
		},
		{
			name: "Agent with empty disallowedTools omitted",
			agent: claude.AgentDefinition{
				Description:     "Test agent",
				Prompt:          "You are a test agent",
				Tools:           []string{"Read"},
				DisallowedTools: []string{},
			},
			wantJSON: `{"description":"Test agent","prompt":"You are a test agent","tools":["Read"]}`,
		},
		{
			name: "Agent with nil disallowedTools omitted",
			agent: claude.AgentDefinition{
				Description:     "Test agent",
				Prompt:          "You are a test agent",
				Tools:           []string{"Read"},
				DisallowedTools: nil,
			},
			wantJSON: `{"description":"Test agent","prompt":"You are a test agent","tools":["Read"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.agent)
			if err != nil {
				t.Fatalf("failed to marshal agent: %v", err)
			}

			gotJSON := string(data)
			if gotJSON != tt.wantJSON {
				t.Errorf("JSON mismatch:\nwant: %s\ngot:  %s", tt.wantJSON, gotJSON)
			}

			var decoded claude.AgentDefinition
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal agent: %v", err)
			}

			if decoded.Description != tt.agent.Description {
				t.Errorf("description mismatch: expected %s, got %s", tt.agent.Description, decoded.Description)
			}
			if decoded.Prompt != tt.agent.Prompt {
				t.Errorf("prompt mismatch: expected %s, got %s", tt.agent.Prompt, decoded.Prompt)
			}
			if decoded.Model != tt.agent.Model {
				t.Errorf("model mismatch: expected %s, got %s", tt.agent.Model, decoded.Model)
			}
		})
	}
}

func TestAgentDefinitionDisallowedToolsField(t *testing.T) {
	agent := claude.AgentDefinition{
		Description:     "Test agent",
		Prompt:          "Test prompt",
		DisallowedTools: []string{"Bash", "WebSearch", "Task"},
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("failed to marshal agent: %v", err)
	}

	// Verify disallowedTools field is in camelCase
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["disallowedTools"]; !ok {
		t.Error("expected disallowedTools field in camelCase")
	}

	// Verify round-trip
	var decoded claude.AgentDefinition
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal agent: %v", err)
	}

	if len(decoded.DisallowedTools) != 3 {
		t.Errorf("expected 3 disallowed tools, got %d", len(decoded.DisallowedTools))
	}

	expectedTools := []string{"Bash", "WebSearch", "Task"}
	for i, tool := range expectedTools {
		if decoded.DisallowedTools[i] != tool {
			t.Errorf("disallowed tool %d mismatch: expected %s, got %s", i, tool, decoded.DisallowedTools[i])
		}
	}
}
