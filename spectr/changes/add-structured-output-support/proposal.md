# Add Structured Output Support

## Why

The TypeScript SDK supports structured outputs through JSON Schema specification and tracking of structured output in results. The Go SDK currently lacks support for:

- Returning structured outputs in result messages
- Tracking structured output retries as error subtypes
- Properly representing structured output in the SDK result interface

This prevents Go users from leveraging the powerful structured output feature that's available in the TypeScript SDK.

## What Changes

- Add `StructuredOutput` field to `SDKResultMessage` type to capture structured output responses
- Add `Errors` field to `SDKResultMessage` type for error result subtypes
- Add `error_max_budget_usd` error subtype for budget exceeded scenarios
- Add `error_max_structured_output_retries` error subtype for structured output retry limit exceeded

## Impact

- **Affected specs**:
  - `result-message` (enhanced capability)
  - `sdk-result-types` (new capability with error subtypes)
- **Affected code**:
  - `pkg/claude/messages.go` - SDKResultMessage struct
- **Breaking changes**: None - these are additive fields
- **TypeScript SDK parity**: Achieves parity with TypeScript SDK result message structure
- **Dependencies**: Depends on Proposal 1 (OutputFormat types should be available for context)

