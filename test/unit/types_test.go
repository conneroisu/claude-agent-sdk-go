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
