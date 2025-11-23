# hook-outputs Specification

## ADDED Requirements

### Requirement: PermissionRequestHookOutput Type
The system SHALL provide `PermissionRequestHookOutput` for permission decision responses.

#### Scenario: Allow permission with unchanged input
- **GIVEN** a PermissionRequest hook
- **WHEN** the hook handler decides to allow the tool
- **THEN** the output has: hookEventName="PermissionRequest", decision.behavior="allow"
- **AND** decision.updatedInput is nil
- **AND** the tool proceeds with original input

#### Scenario: Allow permission with modified input
- **GIVEN** a PermissionRequest hook
- **WHEN** the hook handler allows the tool but wants to modify input
- **THEN** the output has: hookEventName="PermissionRequest", decision.behavior="allow"
- **AND** decision.updatedInput contains the modified arguments
- **AND** the tool proceeds with updated input

#### Scenario: Deny permission
- **GIVEN** a PermissionRequest hook
- **WHEN** the hook handler denies the tool
- **THEN** the output has: hookEventName="PermissionRequest", decision.behavior="deny"
- **AND** decision.message describes why
- **AND** decision.interrupt indicates if session should be interrupted
- **AND** the tool is not executed

#### Scenario: Deny with interrupt
- **GIVEN** a PermissionRequest hook denying with interrupt
- **WHEN** the hook output is processed
- **THEN** the query is interrupted
- **AND** user is presented with the message
- **AND** tool execution is prevented

---

### Requirement: SubagentStartHookOutput Type
The system SHALL provide `SubagentStartHookOutput` for subagent start hook responses.

#### Scenario: Provide additional context to subagent
- **GIVEN** a SubagentStart hook
- **WHEN** the hook handler wants to add context
- **THEN** the output has: hookEventName="SubagentStart"
- **AND** additionalContext contains the context to provide
- **AND** the subagent receives this context

#### Scenario: No additional context
- **GIVEN** a SubagentStart hook
- **WHEN** the hook handler doesn't provide additional context
- **THEN** additionalContext is nil
- **AND** the subagent starts normally

---

## MODIFIED Requirements

### Requirement: PreToolUseHookOutput Enhancement
The PreToolUseHookOutput type SHALL include an updatedInput field for modifying tool inputs.

#### Scenario: Modify tool input before execution
- **GIVEN** a PreToolUse hook completes
- **WHEN** updatedInput is provided
- **THEN** the tool receives the modified input instead of original
- **AND** execution proceeds with updated arguments

#### Scenario: No input modification
- **GIVEN** a PreToolUse hook completes
- **WHEN** updatedInput is nil
- **THEN** the tool receives the original input
- **AND** no modification occurs

---

### Requirement: PostToolUseHookOutput Enhancement
The PostToolUseHookOutput type SHALL include an updatedMCPToolOutput field for modifying tool outputs.

#### Scenario: Modify tool output after execution
- **GIVEN** a PostToolUse hook completes
- **WHEN** updatedMCPToolOutput is provided
- **THEN** the modified output is used instead of the tool's actual output
- **AND** subsequent processing uses the modified value

#### Scenario: No output modification
- **GIVEN** a PostToolUse hook completes
- **WHEN** updatedMCPToolOutput is nil
- **THEN** the tool's actual output is used
- **AND** no modification occurs

---

