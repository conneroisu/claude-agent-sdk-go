# Add Message Type Extensions

## Why

The TypeScript SDK has added new message types for enhanced progress tracking, authentication status monitoring, system status reporting, hook response feedback, and message replay functionality. The Go SDK currently lacks these message types:

- `SDKToolProgressMessage` - For real-time tool execution progress
- `SDKAuthStatusMessage` - For authentication status updates
- `SDKStatusMessage` - For system status notifications
- `SDKHookResponseMessage` - For hook execution feedback
- `SDKUserMessageReplay` - For replayed user messages in context

These message types are essential for providing visibility into the agent's internal operations and improving user experience through detailed progress and status feedback.

## What Changes

- Add `SDKToolProgressMessage` type with tool progress tracking fields
- Add `SDKAuthStatusMessage` type with authentication status information
- Add `SDKStatusMessage` type with system status (e.g., 'compacting')
- Add `SDKStatus` type enum ('compacting' | null)
- Add `SDKHookResponseMessage` type with hook execution details
- Add `SDKUserMessageReplay` type extending SDKUserMessageContent
- Update `SDKMessage` union type to include all new message types

## Impact

- **Affected specs**:
  - `sdk-messages` (enhanced capability)
  - `sdk-status-types` (new capability)
- **Affected code**:
  - `pkg/claude/messages.go` - New message types and SDKMessage union
  - `pkg/claude/types.go` - SDKStatus type
- **Breaking changes**: None - new message types in union are additive
- **TypeScript SDK parity**: Achieves parity with TypeScript SDK message types
- **Dependencies**: None - independent of other proposals

