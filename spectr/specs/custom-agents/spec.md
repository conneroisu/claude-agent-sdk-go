# Custom Agents Specification

## Purpose

Define the structure and behavior of custom agent definitions in the Go SDK, enabling users to create specialized agents with configurable tools, prompts, and model selection that integrate seamlessly with the Claude Code CLI.

## Requirements

### Requirement: Agent Definition Structure

The SDK SHALL provide an `AgentDefinition` type that supports all fields available in the Claude Code CLI's custom agent definitions, maintaining parity with the TypeScript SDK.

#### Scenario: Complete agent definition with all fields

- **WHEN** a user creates an `AgentDefinition` with all supported fields
- **THEN** the definition SHALL include: `Description`, `Prompt`, `Tools`, `DisallowedTools`, and `Model` fields
- **AND** all fields SHALL serialize correctly to JSON for CLI consumption

#### Scenario: Agent definition with minimal required fields

- **WHEN** a user creates an `AgentDefinition` with only required fields
- **THEN** the definition SHALL require only `Description` and `Prompt` fields
- **AND** optional fields (`Tools`, `DisallowedTools`, `Model`) SHALL be omitted or null in JSON output

### Requirement: Tool Exclusion via DisallowedTools

The SDK SHALL support explicit tool blocking through the `DisallowedTools` field in agent definitions, allowing users to prevent specific tools from being available to a custom agent.

#### Scenario: Agent with disallowed tools

- **WHEN** an agent definition specifies `DisallowedTools` with tool names `["WebSearch", "Bash"]`
- **THEN** the SDK SHALL serialize this field as `"disallowedTools": ["WebSearch", "Bash"]`
- **AND** the Claude Code CLI SHALL receive and enforce these exclusions

#### Scenario: Agent with both tools and disallowedTools

- **WHEN** an agent definition specifies both `Tools` and `DisallowedTools`
- **THEN** both fields SHALL be included in the serialized output
- **AND** the CLI SHALL determine tool availability using both lists (with appropriate precedence rules)

#### Scenario: Empty or nil disallowedTools

- **WHEN** `DisallowedTools` is nil or an empty slice
- **THEN** the JSON output SHALL omit the field (due to `omitempty` tag)
- **AND** no tool exclusions SHALL be applied to the agent

### Requirement: TypeScript SDK Parity

The Go SDK's `AgentDefinition` type SHALL maintain structural parity with the TypeScript SDK's `AgentDefinition` interface, ensuring consistent behavior across language implementations.

#### Scenario: Field names and types match TypeScript SDK

- **WHEN** comparing Go `AgentDefinition` to TypeScript `AgentDefinition`
- **THEN** all field names SHALL match (accounting for Go's PascalCase convention)
- **AND** field types SHALL be semantically equivalent
- **AND** JSON serialization SHALL produce compatible output

#### Scenario: Optional fields handling

- **WHEN** optional fields are not provided
- **THEN** Go struct tags SHALL use `omitempty` for `Tools`, `DisallowedTools`, and `Model`
- **AND** serialization behavior SHALL match TypeScript SDK's optional field handling

### Requirement: JSON Serialization

The `AgentDefinition` type SHALL serialize to JSON using standard Go conventions while producing output compatible with the Claude Code CLI's expectations.

#### Scenario: CamelCase JSON field names

- **WHEN** an `AgentDefinition` is marshaled to JSON
- **THEN** Go field `DisallowedTools` SHALL serialize as `"disallowedTools"`
- **AND** all fields SHALL use camelCase naming in JSON output
- **AND** field names SHALL match TypeScript SDK's serialization

#### Scenario: Omitempty behavior

- **WHEN** optional fields are empty or nil
- **THEN** they SHALL be omitted from JSON output
- **AND** only fields with non-zero values SHALL appear in the serialized JSON

