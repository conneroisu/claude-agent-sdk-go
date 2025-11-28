# Hook Inputs Specification

## Purpose

TODO: Add purpose description

## Requirements

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

