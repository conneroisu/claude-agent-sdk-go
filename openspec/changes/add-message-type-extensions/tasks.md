# Implementation Tasks

## 1. Type Definitions
- [x] 1.1 Add `SDKStatus` type to `pkg/claude/types.go` (type SDKStatus string with possible values: "compacting" or "")
- [x] 1.2 Add `SDKToolProgressMessage` struct to `pkg/claude/messages.go` with fields:
  - Type: string = "tool_progress"
  - ToolUseID: string
  - ToolName: string
  - ParentToolUseID: *string (nullable)
  - ElapsedTimeSeconds: float64
  - UUID: UUID
  - SessionID: string
- [x] 1.3 Add `SDKAuthStatusMessage` struct to `pkg/claude/messages.go` with fields:
  - Type: string = "auth_status"
  - IsAuthenticating: bool
  - Output: []string
  - Error: *string (optional)
  - UUID: UUID
  - SessionID: string
- [x] 1.4 Add `SDKStatusMessage` struct to `pkg/claude/messages.go` with fields:
  - Type: string = "system"
  - Subtype: string = "status"
  - Status: SDKStatus
  - UUID: UUID
  - SessionID: string
- [x] 1.5 Add `SDKHookResponseMessage` struct to `pkg/claude/messages.go` with fields:
  - Type: string = "system"
  - Subtype: string = "hook_response"
  - HookName: string
  - HookEvent: string
  - Stdout: string
  - Stderr: string
  - ExitCode: *int (optional)
  - UUID: UUID
  - SessionID: string
- [x] 1.6 Add `SDKUserMessageReplay` struct to `pkg/claude/messages.go` extending SDKUserMessageContent with:
  - UUID: UUID
  - SessionID: string
  - IsReplay: bool = true
- [x] 1.7 Add godoc comments for all new types

## 2. SDKMessage Union Update
- [x] 2.1 Update `SDKMessage` type definition to include all 5 new message types
- [x] 2.2 Ensure message type is properly tagged for JSON unmarshaling
- [x] 2.3 Verify union includes: Assistant, User, UserReplay, Result, System, PartialAssistant, CompactBoundary, Status, HookResponse, ToolProgress, AuthStatus

## 3. JSON Marshaling
- [x] 3.1 Add JSON struct tags to all new message types
- [x] 3.2 Verify JSON field names match TypeScript SDK (camelCase for TypeScript fields)
- [x] 3.3 Test JSON marshaling/unmarshaling for all new types
- [x] 3.4 Verify type discrimination works in union unmarshaling

## 4. Testing
- [x] 4.1 Write unit tests for SDKToolProgressMessage marshaling
- [x] 4.2 Write unit tests for SDKAuthStatusMessage marshaling
- [x] 4.3 Write unit tests for SDKStatusMessage marshaling
- [x] 4.4 Write unit tests for SDKHookResponseMessage marshaling
- [x] 4.5 Write unit tests for SDKUserMessageReplay marshaling
- [x] 4.6 Test JSON unmarshaling for each message type
- [x] 4.7 Test SDKMessage union unmarshaling with all types

## 5. Documentation & Comments
- [x] 5.1 Document SDKToolProgressMessage: when received, progress tracking use cases
- [x] 5.2 Document SDKAuthStatusMessage: authentication flow, error scenarios
- [x] 5.3 Document SDKStatusMessage: system state tracking, compacting status
- [x] 5.4 Document SDKHookResponseMessage: hook execution feedback
- [x] 5.5 Document SDKUserMessageReplay: replay context and usage
- [x] 5.6 Document SDKStatus enum values and meaning

## 6. Integration & Validation
- [x] 6.1 Run `go test ./...` to verify all tests pass
- [x] 6.2 Run `golangci-lint run` to verify code quality
- [x] 6.3 Cross-reference with TypeScript SDK message types
- [x] 6.4 Verify JSON serialization format matches TypeScript SDK
- [x] 6.5 Manual testing: process messages of each new type

