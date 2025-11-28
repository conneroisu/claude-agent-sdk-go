# hook-inputs Specification

## ADDED Requirements

### Requirement: PermissionRequestHookInput Type
The system SHALL provide `PermissionRequestHookInput` for permission request hook payloads.

#### Scenario: PermissionRequest hook input structure
- **GIVEN** a PermissionRequest hook fires
- **WHEN** the hook input is provided
- **THEN** it contains: hook_event_name="PermissionRequest", tool_name, tool_input
- **AND** it includes inherited BaseHookInput fields (context, sessionID, uuid)

#### Scenario: Tool input for decision making
- **WHEN** examining the hook input
- **THEN** tool_input contains the arguments to the tool
- **AND** tool_name identifies which tool is being used
- **AND** these can be inspected to make permission decisions

---

### Requirement: SubagentStartHookInput Type
The system SHALL provide `SubagentStartHookInput` for subagent start hook payloads.

#### Scenario: SubagentStart hook input structure
- **GIVEN** a SubagentStart hook fires
- **WHEN** the hook input is provided
- **THEN** it contains: hook_event_name="SubagentStart", agent_id, agent_type
- **AND** it includes inherited BaseHookInput fields

---

## ADDED Requirements

### Requirement: PreToolUseHookInput Enhancement
The PreToolUseHookInput type SHALL include a tool_use_id field for tracking specific tool invocations.

#### Scenario: Tool use identification
- **GIVEN** a PreToolUse hook fires
- **WHEN** the hook input is received
- **THEN** tool_use_id uniquely identifies this specific tool invocation
- **AND** it can be correlated with PostToolUse hooks

---

### Requirement: PostToolUseHookInput Enhancement
The PostToolUseHookInput type SHALL include a tool_use_id field for tracking specific tool invocations.

#### Scenario: Correlation with PreToolUse
- **GIVEN** both PreToolUse and PostToolUse hooks
- **WHEN** they fire for the same tool execution
- **THEN** tool_use_id is the same in both hook inputs
- **AND** can be used to correlate execution lifecycle

---

### Requirement: NotificationHookInput Enhancement
The NotificationHookInput type SHALL include a notification_type field describing the notification category.

#### Scenario: Notification type specification
- **GIVEN** a Notification hook fires
- **WHEN** the hook input is received
- **THEN** notification_type identifies the category (e.g., "info", "warning", "error")
- **AND** can be used to filter or prioritize notifications

---

### Requirement: SubagentStopHookInput Enhancement
The SubagentStopHookInput type SHALL include agent_id and agent_transcript_path fields.

#### Scenario: Agent identification and transcript access
- **GIVEN** a SubagentStop hook fires
- **WHEN** the hook input is received
- **THEN** agent_id identifies which subagent completed
- **AND** agent_transcript_path provides access to the execution transcript
- **AND** both can be used for logging or analysis

---

