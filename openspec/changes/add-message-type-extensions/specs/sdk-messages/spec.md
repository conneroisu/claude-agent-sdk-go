# sdk-messages Specification

## ADDED Requirements

### Requirement: Tool Progress Message
The system SHALL provide `SDKToolProgressMessage` for tracking real-time tool execution progress with timing information.

#### Scenario: Tool execution progress update
- **GIVEN** a tool is executing during a query
- **WHEN** progress is available
- **THEN** an SDKToolProgressMessage is sent
- **AND** it contains: tool_use_id, tool_name, elapsed_time_seconds
- **AND** JSON includes `"type": "tool_progress"`

#### Scenario: Nested tool progress with parent reference
- **GIVEN** a tool calls another tool (nested execution)
- **WHEN** progress is reported
- **THEN** the parent_tool_use_id field identifies the parent
- **AND** the message can be used to build execution trees

---

### Requirement: Auth Status Message
The system SHALL provide `SDKAuthStatusMessage` for authentication status notifications.

#### Scenario: Authentication in progress
- **GIVEN** authentication is occurring
- **WHEN** status updates are available
- **THEN** an SDKAuthStatusMessage is sent
- **AND** isAuthenticating is true
- **AND** output array contains status messages

#### Scenario: Authentication error
- **GIVEN** authentication fails
- **WHEN** the error occurs
- **THEN** an SDKAuthStatusMessage is sent
- **AND** the error field contains the error message
- **AND** isAuthenticating is false

#### Scenario: Authentication complete
- **GIVEN** authentication succeeds
- **WHEN** completion is confirmed
- **THEN** an SDKAuthStatusMessage is sent
- **AND** isAuthenticating is false
- **AND** error is nil

---

### Requirement: Status Message
The system SHALL provide `SDKStatusMessage` for system-level status notifications.

#### Scenario: Compaction status
- **GIVEN** the system is compacting messages
- **WHEN** status changes occur
- **THEN** an SDKStatusMessage is sent
- **AND** subtype is "status"
- **AND** status field contains "compacting" or similar

#### Scenario: Status message JSON structure
- **WHEN** an SDKStatusMessage is serialized
- **THEN** JSON includes: `"type": "system"`, `"subtype": "status"`, `"status": "..."`

---

### Requirement: Hook Response Message
The system SHALL provide `SDKHookResponseMessage` for hook execution feedback.

#### Scenario: Hook execution feedback
- **GIVEN** a hook executes (e.g., PreToolUse)
- **WHEN** the hook completes
- **THEN** an SDKHookResponseMessage is sent
- **AND** it contains: hook_name, hook_event, stdout, stderr
- **AND** exit_code is populated if applicable

#### Scenario: Successful hook execution
- **GIVEN** a hook completes successfully
- **WHEN** the message is sent
- **THEN** exit_code is 0 or nil
- **AND** stderr is empty
- **AND** stdout contains hook output

#### Scenario: Failed hook execution
- **GIVEN** a hook fails
- **WHEN** the message is sent
- **THEN** exit_code is non-zero
- **AND** stderr contains error output
- **AND** the hook_event identifies which hook

---

### Requirement: User Message Replay
The system SHALL support `SDKUserMessageReplay` for representing replayed user messages.

#### Scenario: User message in replay context
- **GIVEN** a message was previously sent in this session
- **WHEN** it's included again for context
- **THEN** an SDKUserMessageReplay is sent
- **AND** isReplay is true
- **AND** it includes all user message content fields

#### Scenario: Replay message distinguishability
- **GIVEN** message handling code
- **WHEN** processing an SDKUserMessageReplay
- **THEN** it can be distinguished from regular SDKUserMessage by isReplay flag
- **AND** the UUID and SessionID match the original message

---

### Requirement: Updated SDKMessage Union
The SDKMessage union type SHALL include all message types.

#### Scenario: Process any SDK message
- **GIVEN** a message received from Claude
- **WHEN** it's unmarshaled into SDKMessage
- **THEN** the message's actual type is preserved
- **AND** type switches can handle: Assistant, User, UserReplay, Result, System, PartialAssistant, CompactBoundary, Status, HookResponse, ToolProgress, AuthStatus
- **AND** new message types don't break existing code

---

