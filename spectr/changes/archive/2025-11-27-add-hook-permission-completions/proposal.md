# Add Hook and Permission System Completions

## Why

The TypeScript SDK has extended the hook and permission system with:

- Two new hook events: `PermissionRequest` and `SubagentStart` (in addition to existing `SubagentStop`)
- Complete hook input/output types for new events
- Missing fields in existing hook types (tool_use_id, notification_type, etc.)
- Enhanced permission system with toolUseID, agentID, blockedPath, decisionReason
- Hook timeout support for hook callback matching

The Go SDK lacks these capabilities, preventing Go users from building sophisticated permission workflows and hook-based customization that TypeScript users enjoy. This creates significant parity issues.

## What Changes

### New Hook Events
- Add `PermissionRequest` hook event
- Add `SubagentStart` hook event (while Go has SubagentStop, it's missing SubagentStart)

### New Hook Types
- Add `PermissionRequestHookInput` type
- Add `PermissionRequestHookOutput` type with decision allow/deny behavior
- Add `SubagentStartHookInput` type
- Add `SubagentStartHookOutput` type

### Enhanced Hook Input Types
- Add `ToolUseID` field to `PreToolUseHookInput`
- Add `ToolUseID` field to `PostToolUseHookInput`
- Add `NotificationType` field to `NotificationHookInput`
- Add missing fields to `SubagentStopHookInput`: `AgentID`, `AgentTranscriptPath`

### Enhanced Hook Output Types
- Add `UpdatedInput` field to `PreToolUseHookOutput`
- Add `UpdatedMCPToolOutput` field to `PostToolUseHookOutput`

### Permission System Enhancements
- Add `ToolUseID` field to both `PermissionAllow` and `PermissionDeny` results
- Enhance `CanUseToolFunc` callback with additional parameters: `toolUseID`, `agentID`, `blockedPath`, `decisionReason`

### Hook Callback Enhancements
- Add `Timeout` field to `HookCallbackMatcher` for hook execution timeouts

## Impact

- **Affected specs**:
  - `hook-events` (enhanced capability)
  - `hook-inputs` (enhanced capability)
  - `hook-outputs` (enhanced capability)
  - `permission-system` (enhanced capability)
- **Affected code**:
  - `pkg/claude/hooks_events.go` - Hook event constants
  - `pkg/claude/hooks.go` - Hook input/output types, HookCallbackMatcher
  - `pkg/claude/types.go` - Permission system types
- **Breaking changes**: None - additions are backward compatible
- **TypeScript SDK parity**: Achieves full parity with TypeScript SDK hook and permission systems
- **Dependencies**: None - independent of other proposals

