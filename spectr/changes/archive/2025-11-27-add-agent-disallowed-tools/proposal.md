# Add DisallowedTools Support to AgentDefinition

## Why

The Claude Code CLI recently added support for the `disallowedTools` field in custom agent definitions, allowing explicit tool blocking at the agent level. The Go SDK's `AgentDefinition` type currently lacks this field, creating parity issues with the TypeScript SDK and preventing Go users from fully configuring custom agents to exclude specific tools.

## What Changes

- Add `DisallowedTools []string` field to the `AgentDefinition` struct in `pkg/claude/options.go`
- Add JSON struct tag `json:"disallowedTools,omitempty"` to match TypeScript SDK serialization
- Update documentation to explain the relationship between `Tools` and `DisallowedTools` fields
- Ensure the field is properly marshaled when passing agent definitions to the CLI

## Impact

- **Affected specs**: `custom-agents` (new capability)
- **Affected code**:
  - `pkg/claude/options.go` - AgentDefinition struct
  - Documentation/examples referencing custom agents
- **Breaking changes**: None - this is an additive change
- **TypeScript SDK parity**: Achieves full parity with TypeScript SDK's AgentDefinition type
