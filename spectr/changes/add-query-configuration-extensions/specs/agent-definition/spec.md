# agent-definition Specification

## ADDED Requirements

### Requirement: AgentDefinition Model Field Type Constraint
The `AgentDefinition.Model` field SHALL be a string constrained to specific valid values: 'sonnet', 'opus', 'haiku', or 'inherit'.

#### Scenario: Valid model value 'sonnet'
- **GIVEN** an AgentDefinition with `Model: "sonnet"`
- **WHEN** the agent is created
- **THEN** the model is set to Claude Sonnet
- **AND** JSON serialization produces `"model": "sonnet"`

#### Scenario: Valid model value 'opus'
- **GIVEN** an AgentDefinition with `Model: "opus"`
- **WHEN** the agent is created
- **THEN** the model is set to Claude Opus
- **AND** JSON serialization produces `"model": "opus"`

#### Scenario: Valid model value 'haiku'
- **GIVEN** an AgentDefinition with `Model: "haiku"`
- **WHEN** the agent is created
- **THEN** the model is set to Claude Haiku
- **AND** JSON serialization produces `"model": "haiku"`

#### Scenario: Model inheritance
- **GIVEN** an AgentDefinition with `Model: "inherit"`
- **WHEN** the agent is created
- **THEN** the model inherits from parent agent context
- **AND** JSON serialization produces `"model": "inherit"`

#### Scenario: Default model
- **GIVEN** an AgentDefinition with `Model: ""` or omitted
- **WHEN** the agent is created
- **THEN** the default model (likely Sonnet) is used
- **AND** JSON serialization omits the model field

#### Scenario: Invalid model value
- **GIVEN** an AgentDefinition with `Model: "invalid"`
- **WHEN** the agent is used in a query
- **THEN** the CLI rejects the agent configuration
- **AND** an error is returned indicating invalid model value
- **AND** documentation recommends valid values

#### Scenario: Documentation of Model field
- **WHEN** reviewing AgentDefinition documentation
- **THEN** it clearly lists all valid model values
- **AND** it explains when to use 'inherit'
- **AND** it provides examples of valid configurations

---

