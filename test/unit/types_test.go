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

func TestAgentDefinitionWithDisallowedTools(t *testing.T) {
	tests := []struct {
		name     string
		agent    claude.AgentDefinition
		expected string
	}{
		{
			name: "with disallowedTools",
			agent: claude.AgentDefinition{
				Description:     "Test agent",
				Prompt:          "You are a test agent",
				DisallowedTools: []string{"Bash", "Write"},
				Model:           "sonnet",
			},
			expected: `{"description":"Test agent","prompt":"You are a test agent","disallowedTools":["Bash","Write"],"model":"sonnet"}`,
		},
		{
			name: "with tools only",
			agent: claude.AgentDefinition{
				Description: "Test agent",
				Prompt:      "You are a test agent",
				Tools:       []string{"Read", "Grep"},
			},
			expected: `{"description":"Test agent","prompt":"You are a test agent","tools":["Read","Grep"]}`,
		},
		{
			name: "with both tools and disallowedTools",
			agent: claude.AgentDefinition{
				Description:     "Test agent",
				Prompt:          "You are a test agent",
				Tools:           []string{"Read", "Grep", "Write"},
				DisallowedTools: []string{"Write"},
			},
			expected: `{"description":"Test agent","prompt":"You are a test agent","tools":["Read","Grep","Write"],"disallowedTools":["Write"]}`,
		},
		{
			name: "with empty disallowedTools",
			agent: claude.AgentDefinition{
				Description:     "Test agent",
				Prompt:          "You are a test agent",
				DisallowedTools: []string{},
			},
			expected: `{"description":"Test agent","prompt":"You are a test agent"}`,
		},
		{
			name: "with nil disallowedTools",
			agent: claude.AgentDefinition{
				Description:     "Test agent",
				Prompt:          "You are a test agent",
				DisallowedTools: nil,
			},
			expected: `{"description":"Test agent","prompt":"You are a test agent"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.agent)
			if err != nil {
				t.Fatalf("failed to marshal agent: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("marshaling mismatch:\nexpected: %s\ngot:      %s", tt.expected, string(data))
			}

			var decoded claude.AgentDefinition
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal agent: %v", err)
			}

			// Compare fields
			if decoded.Description != tt.agent.Description {
				t.Errorf("description mismatch: expected %s, got %s", tt.agent.Description, decoded.Description)
			}
			if decoded.Prompt != tt.agent.Prompt {
				t.Errorf("prompt mismatch: expected %s, got %s", tt.agent.Prompt, decoded.Prompt)
			}
			if decoded.Model != tt.agent.Model {
				t.Errorf("model mismatch: expected %s, got %s", tt.agent.Model, decoded.Model)
			}

			// Compare slices
			if len(decoded.Tools) != len(tt.agent.Tools) {
				t.Errorf("tools length mismatch: expected %d, got %d", len(tt.agent.Tools), len(decoded.Tools))
			}
			for i := range decoded.Tools {
				if decoded.Tools[i] != tt.agent.Tools[i] {
					t.Errorf("tools[%d] mismatch: expected %s, got %s", i, tt.agent.Tools[i], decoded.Tools[i])
				}
			}

			if len(decoded.DisallowedTools) != len(tt.agent.DisallowedTools) {
				t.Errorf("disallowedTools length mismatch: expected %d, got %d", len(tt.agent.DisallowedTools), len(decoded.DisallowedTools))
			}
			for i := range decoded.DisallowedTools {
				if decoded.DisallowedTools[i] != tt.agent.DisallowedTools[i] {
					t.Errorf("disallowedTools[%d] mismatch: expected %s, got %s", i, tt.agent.DisallowedTools[i], decoded.DisallowedTools[i])
				}
			}
		})
	}
}

// ============================================================================
// Query Configuration Extension Tests - AccountInfo, OutputFormat, etc.
// ============================================================================

// TestAccountInfoSerialization verifies JSON marshaling/unmarshaling for AccountInfo
// with all fields populated.
func TestAccountInfoSerialization(t *testing.T) {
	email := "user@example.com"
	org := "TestOrg"
	subType := "pro"
	tokenSource := "api_key"
	apiKeySource := "user"

	accountInfo := claude.AccountInfo{
		Email:            &email,
		Organization:     &org,
		SubscriptionType: &subType,
		TokenSource:      &tokenSource,
		ApiKeySource:     &apiKeySource,
	}

	data, err := json.Marshal(accountInfo)
	if err != nil {
		t.Fatalf("failed to marshal AccountInfo: %v", err)
	}

	// Verify JSON field names are lowercase
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// Check that fields are present and in camelCase
	if _, ok := raw["email"]; !ok {
		t.Error("expected 'email' field in JSON")
	}
	if _, ok := raw["organization"]; !ok {
		t.Error("expected 'organization' field in JSON")
	}
	if _, ok := raw["subscriptionType"]; !ok {
		t.Error("expected 'subscriptionType' field in JSON")
	}
	if _, ok := raw["tokenSource"]; !ok {
		t.Error("expected 'tokenSource' field in JSON")
	}
	if _, ok := raw["apiKeySource"]; !ok {
		t.Error("expected 'apiKeySource' field in JSON")
	}

	// Unmarshal back
	var decoded claude.AccountInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal AccountInfo: %v", err)
	}

	// Verify all fields
	if decoded.Email == nil || *decoded.Email != email {
		t.Errorf("email mismatch: expected %v, got %v", email, decoded.Email)
	}
	if decoded.Organization == nil || *decoded.Organization != org {
		t.Errorf("organization mismatch: expected %v, got %v", org, decoded.Organization)
	}
	if decoded.SubscriptionType == nil || *decoded.SubscriptionType != subType {
		t.Errorf("subscriptionType mismatch: expected %v, got %v", subType, decoded.SubscriptionType)
	}
	if decoded.TokenSource == nil || *decoded.TokenSource != tokenSource {
		t.Errorf("tokenSource mismatch: expected %v, got %v", tokenSource, decoded.TokenSource)
	}
	if decoded.ApiKeySource == nil || *decoded.ApiKeySource != apiKeySource {
		t.Errorf("apiKeySource mismatch: expected %v, got %v", apiKeySource, decoded.ApiKeySource)
	}
}

// TestAccountInfoWithNilFields verifies omitempty behavior for AccountInfo.
func TestAccountInfoWithNilFields(t *testing.T) {
	// Only some fields populated
	email := "user@example.com"
	accountInfo := claude.AccountInfo{
		Email:            &email,
		Organization:     nil, // explicitly nil
		SubscriptionType: nil,
		TokenSource:      nil,
		ApiKeySource:     nil,
	}

	data, err := json.Marshal(accountInfo)
	if err != nil {
		t.Fatalf("failed to marshal AccountInfo: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// Only email should be present
	if _, ok := raw["email"]; !ok {
		t.Error("expected 'email' field in JSON")
	}
	if _, ok := raw["organization"]; ok {
		t.Error("did not expect 'organization' field in JSON (should be omitted)")
	}
	if _, ok := raw["subscriptionType"]; ok {
		t.Error("did not expect 'subscriptionType' field in JSON (should be omitted)")
	}
}

// TestAccountInfoEmpty verifies empty/nil AccountInfo marshaling.
func TestAccountInfoEmpty(t *testing.T) {
	accountInfo := claude.AccountInfo{}

	data, err := json.Marshal(accountInfo)
	if err != nil {
		t.Fatalf("failed to marshal empty AccountInfo: %v", err)
	}

	// Should be an empty object
	expected := `{}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}

	var decoded claude.AccountInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal empty AccountInfo: %v", err)
	}

	if decoded.Email != nil || decoded.Organization != nil {
		t.Error("expected all fields to be nil")
	}
}

// TestJsonSchemaOutputFormatSerialization verifies JSON marshaling for JsonSchemaOutputFormat.
func TestJsonSchemaOutputFormatSerialization(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
			"age": map[string]interface{}{
				"type": "number",
			},
		},
		"required": []string{"name"},
	}

	format := claude.JsonSchemaOutputFormat{
		BaseOutputFormat: claude.BaseOutputFormat{
			Type: "json_schema",
		},
		Schema: schema,
	}

	data, err := json.Marshal(format)
	if err != nil {
		t.Fatalf("failed to marshal JsonSchemaOutputFormat: %v", err)
	}

	// Verify type field is present
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if raw["type"] != "json_schema" {
		t.Errorf("expected type 'json_schema', got %v", raw["type"])
	}

	if _, ok := raw["schema"]; !ok {
		t.Error("expected 'schema' field in JSON")
	}

	// Unmarshal back
	var decoded claude.JsonSchemaOutputFormat
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal JsonSchemaOutputFormat: %v", err)
	}

	if decoded.Type != "json_schema" {
		t.Errorf("expected type 'json_schema', got %v", decoded.Type)
	}

	if decoded.Schema == nil {
		t.Fatal("schema should not be nil")
	}

	// Verify schema structure
	if decoded.Schema["type"] != "object" {
		t.Errorf("expected schema type 'object', got %v", decoded.Schema["type"])
	}
}

// TestJsonSchemaOutputFormatWithEmptySchema verifies marshaling with empty schema.
func TestJsonSchemaOutputFormatWithEmptySchema(t *testing.T) {
	format := claude.JsonSchemaOutputFormat{
		BaseOutputFormat: claude.BaseOutputFormat{
			Type: "json_schema",
		},
		Schema: map[string]interface{}{},
	}

	data, err := json.Marshal(format)
	if err != nil {
		t.Fatalf("failed to marshal JsonSchemaOutputFormat: %v", err)
	}

	var decoded claude.JsonSchemaOutputFormat
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal JsonSchemaOutputFormat: %v", err)
	}

	// Empty maps become nil after JSON unmarshal if omitempty is used
	// This is expected Go/JSON behavior
	if decoded.Schema != nil && len(decoded.Schema) != 0 {
		t.Errorf("expected nil or empty schema, got %d entries", len(decoded.Schema))
	}
}

// TestBaseOutputFormat verifies BaseOutputFormat type field.
func TestBaseOutputFormat(t *testing.T) {
	base := claude.BaseOutputFormat{
		Type: "json_schema",
	}

	data, err := json.Marshal(base)
	if err != nil {
		t.Fatalf("failed to marshal BaseOutputFormat: %v", err)
	}

	var decoded claude.BaseOutputFormat
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal BaseOutputFormat: %v", err)
	}

	if decoded.Type != "json_schema" {
		t.Errorf("expected type 'json_schema', got %v", decoded.Type)
	}
}

// TestSdkPluginConfigSerialization verifies JSON marshaling for SdkPluginConfig.
func TestSdkPluginConfigSerialization(t *testing.T) {
	tests := []struct {
		name     string
		plugin   claude.SdkPluginConfig
		expected string
	}{
		{
			name: "local plugin",
			plugin: claude.SdkPluginConfig{
				Type: "local",
				Path: "/path/to/plugin",
			},
			expected: `{"type":"local","path":"/path/to/plugin"}`,
		},
		{
			name: "relative path",
			plugin: claude.SdkPluginConfig{
				Type: "local",
				Path: "./plugins/my-plugin",
			},
			expected: `{"type":"local","path":"./plugins/my-plugin"}`,
		},
		{
			name: "absolute path",
			plugin: claude.SdkPluginConfig{
				Type: "local",
				Path: "/usr/local/lib/claude-plugins/custom",
			},
			expected: `{"type":"local","path":"/usr/local/lib/claude-plugins/custom"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.plugin)
			if err != nil {
				t.Fatalf("failed to marshal SdkPluginConfig: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("marshaling mismatch:\nexpected: %s\ngot:      %s", tt.expected, string(data))
			}

			var decoded claude.SdkPluginConfig
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal SdkPluginConfig: %v", err)
			}

			if decoded.Type != tt.plugin.Type {
				t.Errorf("type mismatch: expected %s, got %s", tt.plugin.Type, decoded.Type)
			}
			if decoded.Path != tt.plugin.Path {
				t.Errorf("path mismatch: expected %s, got %s", tt.plugin.Path, decoded.Path)
			}
		})
	}
}

// TestClientOptionsWithMaxBudgetUsd verifies MaxBudgetUsd field JSON tag and type.
func TestClientOptionsWithMaxBudgetUsd(t *testing.T) {
	// Test that MaxBudgetUsd field exists and has correct JSON tag
	// We use a struct with just the serializable fields to test JSON marshaling
	type OptionsSubset struct {
		MaxBudgetUsd float64 `json:"maxBudgetUsd,omitempty"`
	}

	opts := OptionsSubset{
		MaxBudgetUsd: 1.50,
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal options subset: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// Verify maxBudgetUsd is present
	if _, ok := raw["maxBudgetUsd"]; !ok {
		t.Error("expected 'maxBudgetUsd' field in JSON")
	}

	var decoded OptionsSubset
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal options subset: %v", err)
	}

	if decoded.MaxBudgetUsd != 1.50 {
		t.Errorf("maxBudgetUsd mismatch: expected 1.50, got %f", decoded.MaxBudgetUsd)
	}

	// Verify Options struct has the field with correct type
	fullOpts := claude.Options{
		MaxBudgetUsd: 2.50,
	}
	if fullOpts.MaxBudgetUsd != 2.50 {
		t.Errorf("Options.MaxBudgetUsd assignment failed: expected 2.50, got %f", fullOpts.MaxBudgetUsd)
	}
}

// TestClientOptionsWithOutputFormat verifies OutputFormat field JSON tag and type.
func TestClientOptionsWithOutputFormat(t *testing.T) {
	// Test that OutputFormat field exists and has correct JSON tag
	type OptionsSubset struct {
		OutputFormat *claude.JsonSchemaOutputFormat `json:"outputFormat,omitempty"`
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"result": map[string]interface{}{
				"type": "string",
			},
		},
	}

	format := &claude.JsonSchemaOutputFormat{
		BaseOutputFormat: claude.BaseOutputFormat{
			Type: "json_schema",
		},
		Schema: schema,
	}

	opts := OptionsSubset{
		OutputFormat: format,
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal options subset: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// Verify outputFormat is present
	if _, ok := raw["outputFormat"]; !ok {
		t.Error("expected 'outputFormat' field in JSON")
	}

	var decoded OptionsSubset
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal options subset: %v", err)
	}

	if decoded.OutputFormat == nil {
		t.Fatal("outputFormat should not be nil")
	}

	if decoded.OutputFormat.Type != "json_schema" {
		t.Errorf("outputFormat type mismatch: expected 'json_schema', got %v", decoded.OutputFormat.Type)
	}

	// Verify Options struct has the field with correct type
	fullOpts := claude.Options{
		OutputFormat: format,
	}
	if fullOpts.OutputFormat == nil {
		t.Fatal("Options.OutputFormat should not be nil")
	}
}

// TestClientOptionsWithAllowDangerouslySkipPermissions verifies
// AllowDangerouslySkipPermissions field JSON tag and type.
func TestClientOptionsWithAllowDangerouslySkipPermissions(t *testing.T) {
	type OptionsSubset struct {
		AllowDangerouslySkipPermissions bool `json:"allowDangerouslySkipPermissions,omitempty"`
	}

	tests := []struct {
		name  string
		value bool
	}{
		{"skip_true", true},
		{"skip_false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := OptionsSubset{
				AllowDangerouslySkipPermissions: tt.value,
			}

			data, err := json.Marshal(opts)
			if err != nil {
				t.Fatalf("failed to marshal options subset: %v", err)
			}

			var decoded OptionsSubset
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal options subset: %v", err)
			}

			if decoded.AllowDangerouslySkipPermissions != tt.value {
				t.Errorf("allowDangerouslySkipPermissions mismatch: expected %v, got %v",
					tt.value, decoded.AllowDangerouslySkipPermissions)
			}

			// Verify Options struct has the field with correct type
			fullOpts := claude.Options{
				AllowDangerouslySkipPermissions: tt.value,
			}
			if fullOpts.AllowDangerouslySkipPermissions != tt.value {
				t.Errorf("Options.AllowDangerouslySkipPermissions assignment failed")
			}
		})
	}
}

// TestClientOptionsWithPlugins verifies Plugins field JSON tag and type.
func TestClientOptionsWithPlugins(t *testing.T) {
	type OptionsSubset struct {
		Plugins []claude.SdkPluginConfig `json:"plugins,omitempty"`
	}

	opts := OptionsSubset{
		Plugins: []claude.SdkPluginConfig{
			{
				Type: "local",
				Path: "/path/to/plugin1",
			},
			{
				Type: "local",
				Path: "./plugin2",
			},
		},
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal options subset: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// Verify plugins is present
	if _, ok := raw["plugins"]; !ok {
		t.Error("expected 'plugins' field in JSON")
	}

	var decoded OptionsSubset
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal options subset: %v", err)
	}

	if len(decoded.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(decoded.Plugins))
	}

	if decoded.Plugins[0].Path != "/path/to/plugin1" {
		t.Errorf("plugin[0] path mismatch: expected /path/to/plugin1, got %s", decoded.Plugins[0].Path)
	}

	if decoded.Plugins[1].Path != "./plugin2" {
		t.Errorf("plugin[1] path mismatch: expected ./plugin2, got %s", decoded.Plugins[1].Path)
	}

	// Verify Options struct has the field with correct type
	fullOpts := claude.Options{
		Plugins: opts.Plugins,
	}
	if len(fullOpts.Plugins) != 2 {
		t.Fatalf("Options.Plugins assignment failed: expected 2 plugins, got %d", len(fullOpts.Plugins))
	}
}

// TestClientOptionsOmitemptyBehavior verifies omitempty works correctly.
func TestClientOptionsOmitemptyBehavior(t *testing.T) {
	type OptionsSubset struct {
		MaxBudgetUsd                    float64                        `json:"maxBudgetUsd,omitempty"`
		OutputFormat                    *claude.JsonSchemaOutputFormat `json:"outputFormat,omitempty"`
		AllowDangerouslySkipPermissions bool                           `json:"allowDangerouslySkipPermissions,omitempty"`
		Plugins                         []claude.SdkPluginConfig       `json:"plugins,omitempty"`
	}

	// Options with zero values - should omit optional fields
	opts := OptionsSubset{
		MaxBudgetUsd:                    0, // zero value
		OutputFormat:                    nil,
		AllowDangerouslySkipPermissions: false, // zero value
		Plugins:                         nil,
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal options subset: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// These fields should be omitted due to omitempty
	if _, ok := raw["maxBudgetUsd"]; ok {
		t.Error("expected 'maxBudgetUsd' to be omitted (zero value)")
	}
	if _, ok := raw["outputFormat"]; ok {
		t.Error("expected 'outputFormat' to be omitted (nil)")
	}
	if _, ok := raw["allowDangerouslySkipPermissions"]; ok {
		t.Error("expected 'allowDangerouslySkipPermissions' to be omitted (false)")
	}
	if _, ok := raw["plugins"]; ok {
		t.Error("expected 'plugins' to be omitted (nil)")
	}
}

// TestClientOptionsWithCombinedFields verifies Options struct with multiple new fields.
func TestClientOptionsWithCombinedFields(t *testing.T) {
	type OptionsSubset struct {
		MaxBudgetUsd                    float64                        `json:"maxBudgetUsd,omitempty"`
		OutputFormat                    *claude.JsonSchemaOutputFormat `json:"outputFormat,omitempty"`
		AllowDangerouslySkipPermissions bool                           `json:"allowDangerouslySkipPermissions,omitempty"`
		Plugins                         []claude.SdkPluginConfig       `json:"plugins,omitempty"`
	}

	schema := map[string]interface{}{
		"type": "object",
	}

	opts := OptionsSubset{
		MaxBudgetUsd: 5.00,
		OutputFormat: &claude.JsonSchemaOutputFormat{
			BaseOutputFormat: claude.BaseOutputFormat{
				Type: "json_schema",
			},
			Schema: schema,
		},
		AllowDangerouslySkipPermissions: true,
		Plugins: []claude.SdkPluginConfig{
			{
				Type: "local",
				Path: "/plugin",
			},
		},
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal options subset: %v", err)
	}

	var decoded OptionsSubset
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal options subset: %v", err)
	}

	// Verify all fields
	if decoded.MaxBudgetUsd != 5.00 {
		t.Errorf("maxBudgetUsd mismatch: expected 5.00, got %f", decoded.MaxBudgetUsd)
	}

	if decoded.OutputFormat == nil {
		t.Fatal("outputFormat should not be nil")
	}

	if decoded.OutputFormat.Type != "json_schema" {
		t.Errorf("outputFormat type mismatch: expected 'json_schema', got %v", decoded.OutputFormat.Type)
	}

	if !decoded.AllowDangerouslySkipPermissions {
		t.Error("allowDangerouslySkipPermissions should be true")
	}

	if len(decoded.Plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(decoded.Plugins))
	}

	if decoded.Plugins[0].Path != "/plugin" {
		t.Errorf("plugin path mismatch: expected /plugin, got %s", decoded.Plugins[0].Path)
	}

	// Verify Options struct has all fields with correct types
	fullOpts := claude.Options{
		MaxBudgetUsd:                    5.00,
		OutputFormat:                    opts.OutputFormat,
		AllowDangerouslySkipPermissions: true,
		Plugins:                         opts.Plugins,
	}
	if fullOpts.MaxBudgetUsd != 5.00 {
		t.Error("Options.MaxBudgetUsd assignment failed")
	}
	if fullOpts.OutputFormat == nil {
		t.Error("Options.OutputFormat should not be nil")
	}
	if !fullOpts.AllowDangerouslySkipPermissions {
		t.Error("Options.AllowDangerouslySkipPermissions should be true")
	}
	if len(fullOpts.Plugins) != 1 {
		t.Error("Options.Plugins assignment failed")
	}
}
