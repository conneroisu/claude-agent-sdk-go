# Tool Inputs Specification

## Purpose

TODO: Add purpose description

## Requirements

### Requirement: AgentInput Type Complete Definition
The AgentInput type SHALL include model selection and resume functionality.

#### Scenario: Create subagent with specific model
- **GIVEN** an AgentInput for creating a subagent
- **WHEN** the Model field is set to "opus"
- **THEN** the subagent uses Claude Opus
- **AND** JSON serialization includes `"model": "opus"`

#### Scenario: Inherit model from parent
- **GIVEN** an AgentInput with Model="inherit"
- **WHEN** the subagent is created
- **THEN** it inherits the parent agent's model
- **AND** no explicit model override occurs

#### Scenario: Resume subagent execution
- **GIVEN** an AgentInput with Resume field set
- **WHEN** the subagent is invoked
- **THEN** it resumes from the specified checkpoint
- **AND** execution state is restored

#### Scenario: Default model
- **GIVEN** an AgentInput without Model specified
- **WHEN** marshaled to JSON
- **THEN** the model field is omitted
- **AND** the default model is used

---

### Requirement: BashInput Type with Sandbox Control
The BashInput type SHALL support disabling sandbox restrictions.

#### Scenario: Bash with sandbox enabled (default)
- **GIVEN** a BashInput without DangerouslyDisableSandbox
- **WHEN** the command executes
- **THEN** standard sandbox restrictions apply
- **AND** access is limited to safe directories

#### Scenario: Bash with sandbox disabled
- **GIVEN** a BashInput with DangerouslyDisableSandbox=true
- **WHEN** the command executes
- **THEN** sandbox restrictions are bypassed
- **AND** full system access is possible
- **AND** documentation warns of security implications

#### Scenario: Security warning in documentation
- **WHEN** reviewing BashInput documentation
- **THEN** it clearly warns about security implications
- **AND** recommends only using in controlled environments
- **AND** notes that this bypasses important safety measures

---

