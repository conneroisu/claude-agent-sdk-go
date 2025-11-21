# Add Query Interface and Configuration Extensions

## Why

The TypeScript SDK has recently added critical features for cost management, account information retrieval, and advanced configuration options that are missing from the Go SDK. These features are essential for production deployments:

- **Cost Control**: `maxBudgetUsd` option and `SetMaxThinkingTokens()` method enable budget management
- **Account Information**: `accountInfo()` query method provides account context needed for proper API usage
- **Plugin System**: Plugin configuration support for extending SDK functionality
- **Structured Output**: Output format specification for controlling response structure
- **Permission Control**: Fine-grained permission configuration

The Go SDK currently lacks these capabilities, creating parity issues with the TypeScript SDK and limiting production usability.

## What Changes

- Add `SetMaxThinkingTokens(maxThinkingTokens *int)` query method to dynamically control thinking token budget
- Add `AccountInfo()` query method to retrieve account information
- Add `AccountInfo` type definition with email, organization, subscription type, and API key source fields
- Add `MaxBudgetUsd` field to ClientOptions struct for cost management
- Add `OutputFormat` field to ClientOptions struct with `JsonSchemaOutputFormat` support
- Add `AllowDangerouslySkipPermissions` field to ClientOptions struct
- Add `Plugins` field to ClientOptions struct with `SdkPluginConfig` type
- Add `SdkPluginConfig` type for plugin configuration
- Add `OutputFormat` types (`OutputFormatType`, `BaseOutputFormat`, `JsonSchemaOutputFormat`)
- Update `AgentDefinition.Model` field to use string type (already correct in Go, document type constraint: must be 'sonnet', 'opus', 'haiku', or 'inherit')

## Impact

- **Affected specs**:
  - `query-interface` (new capability)
  - `client-options` (enhanced capability)
  - `agent-definition` (enhanced capability)
- **Affected code**:
  - `pkg/claude/query.go` - Add new query methods
  - `pkg/claude/options.go` - Add new options fields
  - `pkg/claude/types.go` - Add AccountInfo, OutputFormat types
- **Breaking changes**: None - all additions are additive
- **TypeScript SDK parity**: Achieves parity with core query/config features from TypeScript SDK

