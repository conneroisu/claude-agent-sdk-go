# client-options Specification

## ADDED Requirements

### Requirement: MaxBudgetUsd Option
The ClientOptions struct SHALL support a `MaxBudgetUsd` field that enforces a maximum spending limit on API calls for the query session.

#### Scenario: Query respects budget limit
- **GIVEN** a ClientOptions with `MaxBudgetUsd: 5.00`
- **WHEN** the query makes API calls
- **THEN** operations stop when the budget would be exceeded
- **AND** an error is returned indicating budget exceeded

#### Scenario: Budget in USD cents precision
- **GIVEN** a ClientOptions with `MaxBudgetUsd: 0.01`
- **WHEN** the query is executed
- **THEN** the minimum USD amount (penny) is enforced
- **AND** precision to two decimal places is maintained

#### Scenario: No budget limit
- **GIVEN** a ClientOptions with `MaxBudgetUsd: 0` or omitted
- **WHEN** the query executes
- **THEN** no budget enforcement occurs
- **AND** operations proceed without cost constraints

---

### Requirement: OutputFormat Option
The ClientOptions struct SHALL support an `OutputFormat` field that specifies the desired format for structured outputs, with support for JSON Schema format specification.

#### Scenario: Specify JSON Schema output format
- **GIVEN** a ClientOptions with:
  ```
  OutputFormat: JsonSchemaOutputFormat{
    Type: "json_schema",
    Schema: {
      "type": "object",
      "properties": {...}
    }
  }
  ```
- **WHEN** the query is executed
- **THEN** the output format is sent to Claude
- **AND** responses are formatted according to the schema
- **AND** the schema is validated for correctness

#### Scenario: No output format specified
- **GIVEN** a ClientOptions with `OutputFormat: nil`
- **WHEN** the query executes
- **THEN** default text output format is used
- **AND** no schema constraints are applied

---

### Requirement: AllowDangerouslySkipPermissions Option
The ClientOptions struct SHALL support an `AllowDangerouslySkipPermissions` field that allows bypassing permission checks when explicitly enabled, with clear security implications.

#### Scenario: Skip permission prompts
- **GIVEN** a ClientOptions with `AllowDangerouslySkipPermissions: true`
- **WHEN** the query executes and encounters tool use
- **THEN** permission checks are bypassed
- **AND** tools are allowed to execute without prompts

#### Scenario: Default to permission checks
- **GIVEN** a ClientOptions with `AllowDangerouslySkipPermissions: false` or omitted
- **WHEN** the query executes
- **THEN** standard permission prompts are shown
- **AND** user approval is required for tool execution

#### Scenario: Documentation warns of security implications
- **WHEN** reviewing the `AllowDangerouslySkipPermissions` field documentation
- **THEN** it clearly states this is a security risk
- **AND** it explains the implications of disabling permission checks
- **AND** it recommends only using in controlled environments

---

### Requirement: Plugins Option
The ClientOptions struct SHALL support a `Plugins` field that allows configuration of SDK plugins for extending functionality.

#### Scenario: Configure local plugin
- **GIVEN** a ClientOptions with:
  ```
  Plugins: []SdkPluginConfig{
    {Type: "local", Path: "/path/to/plugin"}
  }
  ```
- **WHEN** the query is initialized
- **THEN** the plugin configuration is passed to Claude
- **AND** the plugin at the specified path is loaded
- **AND** plugin functionality becomes available

#### Scenario: Multiple plugins configured
- **GIVEN** a ClientOptions with multiple SdkPluginConfig entries
- **WHEN** the query is initialized
- **THEN** all plugins are loaded in order
- **AND** plugin interactions are handled correctly
- **AND** no plugin loading errors are masked

#### Scenario: Invalid plugin path
- **GIVEN** a ClientOptions with a plugin path that doesn't exist
- **WHEN** the query initializes
- **THEN** an error is returned indicating the plugin path is invalid
- **AND** the query fails to initialize

#### Scenario: No plugins specified
- **GIVEN** a ClientOptions with `Plugins: nil` or empty
- **WHEN** the query executes
- **THEN** no plugins are loaded
- **AND** operations proceed normally

---

### Requirement: OutputFormat Type Definition
The system SHALL define `OutputFormat` related types to support structured output configuration.

#### Scenario: JsonSchemaOutputFormat with valid schema
- **GIVEN** a JsonSchemaOutputFormat struct
- **WHEN** marshaled to JSON
- **THEN** it produces:
  ```json
  {
    "type": "json_schema",
    "schema": {...}
  }
  ```

#### Scenario: BaseOutputFormat as base type
- **GIVEN** BaseOutputFormat as the base for output formats
- **WHEN** different format types extend it
- **THEN** all formats have a consistent `type` field
- **AND** format type is always present in JSON serialization

---

### Requirement: SdkPluginConfig Type Definition
The system SHALL define `SdkPluginConfig` type for plugin configuration.

#### Scenario: Local plugin configuration
- **GIVEN** a SdkPluginConfig with Type="local" and Path set
- **WHEN** marshaled to JSON
- **THEN** it produces:
  ```json
  {
    "type": "local",
    "path": "/path/to/plugin"
  }
  ```

#### Scenario: Plugin config validation
- **WHEN** creating a SdkPluginConfig
- **THEN** the Type field must be "local" (only supported type currently)
- **AND** the Path field is validated to be a non-empty string

---

