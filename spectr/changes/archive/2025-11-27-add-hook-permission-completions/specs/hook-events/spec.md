# hook-events Specification

## ADDED Requirements

### Requirement: PermissionRequest Hook Event
The system SHALL support a `PermissionRequest` hook event that fires when the agent needs permission to use a tool.

#### Scenario: Permission request hook triggered
- **GIVEN** a tool use is about to occur
- **WHEN** permission checking is enabled
- **THEN** the PermissionRequest hook is triggered
- **AND** it provides tool_name and tool_input for decision making

#### Scenario: Hook event value
- **GIVEN** hook event processing code
- **WHEN** receiving a PermissionRequest event
- **THEN** the event name is exactly "PermissionRequest"
- **AND** it can be matched against constant

---

### Requirement: SubagentStart Hook Event
The system SHALL support a `SubagentStart` hook event that fires when a subagent begins execution.

#### Scenario: SubagentStart hook triggered
- **GIVEN** a subagent is about to be created and started
- **WHEN** subagent initialization occurs
- **THEN** the SubagentStart hook is triggered
- **AND** it provides agent_id and agent_type for context

#### Scenario: SubagentStart and SubagentStop pair
- **GIVEN** a subagent lifecycle
- **WHEN** subagent starts and stops
- **THEN** SubagentStart is triggered on start
- **AND** SubagentStop is triggered on completion
- **AND** both events can be correlated via agent_id

#### Scenario: Hook event value
- **GIVEN** hook event processing code
- **WHEN** receiving a SubagentStart event
- **THEN** the event name is exactly "SubagentStart"
- **AND** it can be matched against constant

---

