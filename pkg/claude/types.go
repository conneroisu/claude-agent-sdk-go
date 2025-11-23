// Package claude provides core SDK type definitions for Claude agent
// interactions.
//
// This file contains fundamental types including configuration (API keys,
// system prompts, permissions), MCP server configurations (stdio, SSE, HTTP,
// SDK), permission management, usage tracking, and callback interfaces. The
// high number of public structs is intentional to support the comprehensive
// configuration and extension capabilities of the SDK.
package claude

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// UUID type alias for consistency.
type UUID = uuid.UUID

// JSONValue preserves raw JSON for caller-controlled decoding.
type JSONValue = json.RawMessage

// APIKeySource represents the source of the API key.
type APIKeySource string

const (
	APIKeySourceUser      APIKeySource = "user"
	APIKeySourceProject   APIKeySource = "project"
	APIKeySourceOrg       APIKeySource = "org"
	APIKeySourceTemporary APIKeySource = "temporary"
)

// ConfigScope represents configuration scope.
type ConfigScope string

const (
	ConfigScopeLocal   ConfigScope = "local"
	ConfigScopeUser    ConfigScope = "user"
	ConfigScopeProject ConfigScope = "project"
)

// SystemPromptConfig captures the union of system prompt options:
//   - nil (vanilla Claude prompt)
//   - literal string prompts
//   - preset prompt with optional append text
type SystemPromptConfig interface {
	isSystemPromptConfig()
}

// SystemPromptPreset selects a preset system prompt and optional appended text.
type SystemPromptPreset struct {
	Type   string  `json:"type"`             // always "preset"
	Preset string  `json:"preset"`           // e.g. "claude_code"
	Append *string `json:"append,omitempty"` // optional additional instructions
}

func (SystemPromptPreset) isSystemPromptConfig() {}

// SystemPromptLiteral wraps a free-form system prompt string.
type SystemPromptLiteral string

func (SystemPromptLiteral) isSystemPromptConfig() {}

// SystemPromptUnset explicitly opts into vanilla behavior.
type SystemPromptUnset struct{}

func (SystemPromptUnset) isSystemPromptConfig() {}

// PermissionMode defines how permissions are handled.
type PermissionMode string

const (
	PermissionModeDefault           PermissionMode = "default"
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
	PermissionModePlan              PermissionMode = "plan"
)

// PermissionBehavior defines permission decisions.
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

// Usage represents token usage statistics.
// Note: Using int for token counts. TypeScript uses 'number' which maps to
// float64, but token counts are always integers. int is sufficient for token
// counts up to 2^31-1 (~2.1 billion tokens), which exceeds current model
// context windows from Anthropic.
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// ModelUsage represents detailed model usage.
type ModelUsage struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
	WebSearchRequests        int     `json:"webSearchRequests"`
	CostUSD                  float64 `json:"costUSD"`
	ContextWindow            int     `json:"contextWindow"`
}

// McpServerConfig represents different MCP server configurations.
type McpServerConfig interface {
	mcpServerConfig()
}

// McpStdioServerConfig represents stdio-based MCP server.
type McpStdioServerConfig struct {
	Type    *string           `json:"type,omitempty"` // Optional: "stdio" - matches TypeScript type?: 'stdio'
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

func (McpStdioServerConfig) mcpServerConfig() {}

// McpSSEServerConfig represents SSE-based MCP server.
type McpSSEServerConfig struct {
	Type    string            `json:"type"` // "sse"
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (McpSSEServerConfig) mcpServerConfig() {}

// McpHTTPServerConfig represents HTTP-based MCP server.
type McpHTTPServerConfig struct {
	Type    string            `json:"type"` // "http"
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (McpHTTPServerConfig) mcpServerConfig() {}

// McpSdkServerConfig represents SDK-based MCP server.
type McpSdkServerConfig struct {
	Type     string    `json:"type"` // "sdk"
	Name     string    `json:"name"`
	Instance McpServer `json:"-"`
}

func (McpSdkServerConfig) mcpServerConfig() {}

// PermissionRuleValue represents a permission rule.
type PermissionRuleValue struct {
	ToolName    string  `json:"toolName"`
	RuleContent *string `json:"ruleContent,omitempty"`
}

// PermissionUpdateDestination specifies where to save permission updates.
type PermissionUpdateDestination string

const (
	PermissionDestinationUserSettings    PermissionUpdateDestination = "userSettings"
	PermissionDestinationProjectSettings PermissionUpdateDestination = "projectSettings"
	PermissionDestinationLocalSettings   PermissionUpdateDestination = "localSettings"
	PermissionDestinationSession         PermissionUpdateDestination = "session"
)

// PermissionUpdate represents different types of permission updates.
type PermissionUpdate interface {
	permissionUpdate()
}

// AddRulesUpdate adds permission rules.
type AddRulesUpdate struct {
	Type        string                      `json:"type"` // "addRules"
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
}

func (AddRulesUpdate) permissionUpdate() {}

// ReplaceRulesUpdate replaces permission rules atomically.
type ReplaceRulesUpdate struct {
	Type        string                      `json:"type"` // "replaceRules"
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
}

func (ReplaceRulesUpdate) permissionUpdate() {}

// RemoveRulesUpdate removes permission rules.
type RemoveRulesUpdate struct {
	Type        string                      `json:"type"` // "removeRules"
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
}

func (RemoveRulesUpdate) permissionUpdate() {}

// AddDirectoriesUpdate adds directories to permissions.
type AddDirectoriesUpdate struct {
	Type        string                      `json:"type"` // "addDirectories"
	Directories []string                    `json:"directories"`
	Destination PermissionUpdateDestination `json:"destination"`
}

func (AddDirectoriesUpdate) permissionUpdate() {}

// RemoveDirectoriesUpdate removes directories from permissions.
type RemoveDirectoriesUpdate struct {
	Type        string                      `json:"type"` // "removeDirectories"
	Directories []string                    `json:"directories"`
	Destination PermissionUpdateDestination `json:"destination"`
}

func (RemoveDirectoriesUpdate) permissionUpdate() {}

// SetModeUpdate changes permission mode.
type SetModeUpdate struct {
	Type        string                      `json:"type"` // "setMode"
	Mode        PermissionMode              `json:"mode"`
	Destination PermissionUpdateDestination `json:"destination"`
}

func (SetModeUpdate) permissionUpdate() {}

// PermissionResult represents the result of a permission check.
//
// Design Note: The TypeScript SDK uses a discriminated union based on the
// 'behavior' field (behavior: "allow" | "deny").
//
// Go doesn't have native discriminated unions, so we model
// this as an interface with two implementations: PermissionAllow and
// PermissionDeny.
//
// Alternative approaches considered:
//  1. Single struct with Behavior enum field (simpler but less type-safe)
//  2. Interface with implementations (chosen - provides compile-time type
//     safety)
//
// The interface approach is more idiomatic Go and provides better type safety
// at the cost of slightly more verbose code. Users can use type assertions or
// type switches to handle the different result types:
//
//	switch result := permResult.(type) {
//	case *PermissionAllow:
//	    // handle allow case
//	case *PermissionDeny:
//	    // handle deny case
//	}
type PermissionResult interface {
	permissionResult()
}

// PermissionAllow represents an allowed permission result.
type PermissionAllow struct {
	Behavior           PermissionBehavior   `json:"behavior"` // "allow"
	ToolUseID          *string              `json:"toolUseID,omitempty"`
	UpdatedInput       map[string]JSONValue `json:"updatedInput"`
	UpdatedPermissions []PermissionUpdate   `json:"updatedPermissions,omitempty"`
}

func (PermissionAllow) permissionResult() {}

// PermissionDeny represents a denied permission result.
type PermissionDeny struct {
	Behavior  PermissionBehavior `json:"behavior"` // "deny"
	ToolUseID *string            `json:"toolUseID,omitempty"`
	Message   string             `json:"message"`
	Interrupt bool               `json:"interrupt,omitempty"`
}

func (PermissionDeny) permissionResult() {}

// CanUseToolFunc is a function that checks if a tool can be used.
// Parameters:
//   - ctx: The context for the permission check
//   - toolName: The name of the tool being checked
//   - input: The input parameters for the tool
//   - suggestions: Suggested permission updates
//   - toolUseID: Unique identifier for this specific tool use
//   - agentID: ID of the agent making the request (optional)
//   - blockedPath: Path that was blocked (optional)
//   - decisionReason: Reason for the permission decision (optional)
type CanUseToolFunc func(
	ctx context.Context,
	toolName string,
	input map[string]JSONValue,
	suggestions []PermissionUpdate,
	toolUseID string,
	agentID *string,
	blockedPath *string,
	decisionReason *string,
) (PermissionResult, error)

// SDKStatus represents system status values for the Claude SDK.
// This type is used to communicate system state changes such as message compaction.
type SDKStatus string

const (
	// SDKStatusCompacting indicates the system is performing message compaction.
	SDKStatusCompacting SDKStatus = "compacting"
)

// AccountInfo contains information about the account associated with the API key.
// This information is returned by query operations and provides details about the
// authenticated user or organization.
type AccountInfo struct {
	Email            *string `json:"email,omitempty"`
	Organization     *string `json:"organization,omitempty"`
	SubscriptionType *string `json:"subscriptionType,omitempty"`
	TokenSource      *string `json:"tokenSource,omitempty"`
	ApiKeySource     *string `json:"apiKeySource,omitempty"`
}

// OutputFormatType represents the type of output format requested.
// Currently supports structured output formats like JSON schema.
type OutputFormatType string

// BaseOutputFormat provides the common type field for all output format variants.
// This is extended by specific output format types like JsonSchemaOutputFormat.
type BaseOutputFormat struct {
	Type OutputFormatType `json:"type"`
}

// JsonSchemaOutputFormat specifies a JSON schema constraint for model output.
// The Schema field defines the expected structure of the model's response using
// JSON Schema format. This enables structured output generation where the model's
// response conforms to a predefined schema.
type JsonSchemaOutputFormat struct {
	BaseOutputFormat
	Schema map[string]interface{} `json:"schema,omitempty"`
}

// OutputFormat is the primary type for specifying output format constraints.
// Currently aliased to JsonSchemaOutputFormat for JSON schema-based structured output.
type OutputFormat = JsonSchemaOutputFormat

// SdkPluginConfig describes the configuration for an SDK plugin.
// Plugins extend SDK functionality through external modules loaded at runtime.
// The Type field specifies the plugin type, and Path indicates the location
// of the plugin module.
type SdkPluginConfig struct {
	Type string `json:"type"`
	Path string `json:"path"`
}
