# Implementation Tasks

## 1. Hook Events
- [ ] 1.1 Add `PermissionRequest` hook event constant to `pkg/claude/hooks_events.go` with value "PermissionRequest"
- [ ] 1.2 Add `SubagentStart` hook event constant to `pkg/claude/hooks_events.go` with value "SubagentStart"
- [ ] 1.3 Update HOOK_EVENTS list to include both new events
- [ ] 1.4 Document when each hook event is triggered

## 2. New Hook Input Types
- [ ] 2.1 Add `PermissionRequestHookInput` struct to `pkg/claude/hooks.go` with fields:
  - HookEventName: string = "PermissionRequest"
  - ToolName: string
  - ToolInput: interface{}
  - Plus inherited BaseHookInput fields (context, sessionID, uuid)
- [ ] 2.2 Add `SubagentStartHookInput` struct to `pkg/claude/hooks.go` with fields:
  - HookEventName: string = "SubagentStart"
  - AgentID: string
  - AgentType: string
  - Plus inherited BaseHookInput fields
- [ ] 2.3 Implement UnmarshalJSON for both types to handle JSON deserialization
- [ ] 2.4 Add godoc comments for both types

## 3. New Hook Output Types
- [ ] 3.1 Add `PermissionRequestHookOutput` struct to `pkg/claude/hooks.go` with fields:
  - HookEventName: string = "PermissionRequest"
  - Decision: PermissionRequestDecision (interface supporting allow/deny with fields)
- [ ] 3.2 Create decision types for PermissionRequest:
  - Allow decision: behavior: "allow", updatedInput: *map[string]interface{}
  - Deny decision: behavior: "deny", message: *string, interrupt: *bool
- [ ] 3.3 Add `SubagentStartHookOutput` struct to `pkg/claude/hooks.go` with fields:
  - HookEventName: string = "SubagentStart"
  - AdditionalContext: *string
- [ ] 3.4 Implement MarshalJSON for both types
- [ ] 3.5 Add godoc comments for both types

## 4. Hook Input Enhancements
- [ ] 4.1 Add `ToolUseID` field to `PreToolUseHookInput` (string)
- [ ] 4.2 Add `ToolUseID` field to `PostToolUseHookInput` (string)
- [ ] 4.3 Add `NotificationType` field to `NotificationHookInput` (string)
- [ ] 4.4 Add `AgentID` field to `SubagentStopHookInput` (string)
- [ ] 4.5 Add `AgentTranscriptPath` field to `SubagentStopHookInput` (string)
- [ ] 4.6 Update JSON struct tags to use camelCase for TypeScript compatibility

## 5. Hook Output Enhancements
- [ ] 5.1 Add `UpdatedInput` field to `PreToolUseHookOutput` (*map[string]interface{})
- [ ] 5.2 Add `UpdatedMCPToolOutput` field to `PostToolUseHookOutput` (interface{})
- [ ] 5.3 Update JSON struct tags for new fields

## 6. Permission System Enhancements
- [ ] 6.1 Add `ToolUseID` field to `PermissionAllow` struct (*string)
- [ ] 6.2 Add `ToolUseID` field to `PermissionDeny` struct (*string)
- [ ] 6.3 Update `CanUseToolFunc` callback signature to include additional parameters:
  - toolUseID: string
  - agentID: *string
  - blockedPath: *string
  - decisionReason: *string
- [ ] 6.4 Update Options struct to support CanUseToolFunc with new signature
- [ ] 6.5 Document each new parameter and its meaning

## 7. Hook Callback Enhancements
- [ ] 7.1 Add `Timeout` field to `HookCallbackMatcher` struct (*time.Duration or *int for milliseconds)
- [ ] 7.2 Update JSON struct tag for timeout (camelCase)
- [ ] 7.3 Document timeout behavior and units

## 8. Hook Type Union Updates
- [ ] 8.1 Update HookInput union type to include PermissionRequestHookInput and SubagentStartHookInput
- [ ] 8.2 Update HookOutput union type to include PermissionRequestHookOutput and SubagentStartHookOutput
- [ ] 8.3 Verify JSON unmarshaling correctly discriminates between types

## 9. Testing
- [ ] 9.1 Write unit tests for PermissionRequestHookInput marshaling
- [ ] 9.2 Write unit tests for PermissionRequestHookOutput (allow decision)
- [ ] 9.3 Write unit tests for PermissionRequestHookOutput (deny decision)
- [ ] 9.4 Write unit tests for SubagentStartHookInput/Output marshaling
- [ ] 9.5 Test enhanced hook inputs with new fields
- [ ] 9.6 Test enhanced hook outputs with new fields
- [ ] 9.7 Test permission system with new callback parameters
- [ ] 9.8 Test HookCallbackMatcher with timeout field
- [ ] 9.9 Test hook type union marshaling/unmarshaling

## 10. Documentation & Comments
- [ ] 10.1 Document PermissionRequest hook: when triggered, how to respond
- [ ] 10.2 Document SubagentStart hook: lifecycle, agent context
- [ ] 10.3 Document new hook fields and their purposes
- [ ] 10.4 Document permission decision types and behavior
- [ ] 10.5 Document timeout semantics and default behavior
- [ ] 10.6 Add examples of complete hook workflows

## 11. Integration & Validation
- [ ] 11.1 Run `go test ./...` to verify all tests pass
- [ ] 11.2 Run `golangci-lint run` to verify code quality
- [ ] 11.3 Cross-reference with TypeScript SDK hook types
- [ ] 11.4 Verify JSON serialization matches TypeScript format
- [ ] 11.5 Manual testing: test complete hook workflows with new types

