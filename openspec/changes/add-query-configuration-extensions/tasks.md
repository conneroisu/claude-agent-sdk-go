# Implementation Tasks

## 1. Type Definitions
- [x] 1.1 Add `AccountInfo` type to `pkg/claude/types.go` with fields: email, organization, subscriptionType, tokenSource, apiKeySource (all optional strings)
- [x] 1.2 Add `OutputFormatType` type alias (`type OutputFormatType string`)
- [x] 1.3 Add `BaseOutputFormat` struct with `Type` field (OutputFormatType)
- [x] 1.4 Add `JsonSchemaOutputFormat` struct extending BaseOutputFormat with `Schema` field (map[string]interface{})
- [x] 1.5 Add `OutputFormat` type alias (`type OutputFormat = JsonSchemaOutputFormat`)
- [x] 1.6 Add `SdkPluginConfig` struct with `Type` field (string, value: "local") and `Path` field (string)
- [x] 1.7 Add godoc comments for all new types explaining their purpose and usage

## 2. Options Struct Extensions
- [x] 2.1 Add `MaxBudgetUsd` field to `ClientOptions` struct with type float64 and JSON tag `json:"maxBudgetUsd,omitempty"`
- [x] 2.2 Add `OutputFormat` field to `ClientOptions` struct with JSON tag `json:"outputFormat,omitempty"`
- [x] 2.3 Add `AllowDangerouslySkipPermissions` field to `ClientOptions` struct with type bool and JSON tag `json:"allowDangerouslySkipPermissions,omitempty"`
- [x] 2.4 Add `Plugins` field to `ClientOptions` struct ([]SdkPluginConfig) with JSON tag `json:"plugins,omitempty"`
- [x] 2.5 Verify field ordering matches TypeScript SDK for consistency
- [x] 2.6 Add godoc comments for each new field explaining purpose and constraints

## 3. Query Interface Methods
- [x] 3.1 Add `SetMaxThinkingTokens(maxThinkingTokens *int) error` method to Query interface
- [x] 3.2 Add `AccountInfo(ctx context.Context) (*AccountInfo, error)` method to Query interface
- [x] 3.3 Implement SetMaxThinkingTokens in Query implementation (likely in query.go or new query_config.go)
- [x] 3.4 Implement AccountInfo in Query implementation
- [x] 3.5 Add godoc comments explaining parameter semantics and error conditions

## 4. Documentation & Comments
- [x] 4.1 Document MaxBudgetUsd: behavior when exceeded, precision, units (USD)
- [x] 4.2 Document OutputFormat: usage for structured outputs, schema validation
- [x] 4.3 Document AllowDangerouslySkipPermissions: security implications and use cases
- [x] 4.4 Document Plugins: how to configure plugins, local plugin support
- [x] 4.5 Document AgentDefinition.Model type constraint (values: 'sonnet', 'opus', 'haiku', 'inherit')
- [x] 4.6 Document AccountInfo fields and when each is populated

## 5. Testing
- [x] 5.1 Write unit tests for AccountInfo type marshaling/unmarshaling
- [x] 5.2 Write unit tests for OutputFormat types and validation
- [x] 5.3 Write unit tests for SdkPluginConfig type
- [x] 5.4 Write unit tests for ClientOptions marshaling with new fields
- [x] 5.5 Add integration test for SetMaxThinkingTokens method
- [x] 5.6 Add integration test for AccountInfo method
- [x] 5.7 Test edge cases: nil budgets, empty plugins, invalid schema formats

## 6. Validation & Verification
- [x] 6.1 Run `go test ./...` to verify all tests pass
- [x] 6.2 Run `golangci-lint run` to verify code quality
- [x] 6.3 Cross-reference Go types with TypeScript SDK (sdk.d.ts) for complete parity
- [x] 6.4 Verify JSON serialization matches TypeScript SDK format
- [x] 6.5 Manual testing: create client with new options and verify they serialize correctly
- [x] 6.6 Verify type constraints and documentation are clear

