# Query Interface Specification

## Purpose

TODO: Add purpose description

## Requirements

### Requirement: SetMaxThinkingTokens Method
The Query interface SHALL provide a `SetMaxThinkingTokens` method that allows dynamic adjustment of the maximum thinking token budget during query execution.

#### Scenario: Set thinking tokens to positive value
- **WHEN** `SetMaxThinkingTokens(maxTokens *int)` is called with a positive integer pointer
- **THEN** the method sends a control request to the Claude CLI
- **AND** the thinking token budget is updated for subsequent operations
- **AND** the method returns nil error on success

#### Scenario: Clear thinking token limit
- **WHEN** `SetMaxThinkingTokens(nil)` is called
- **THEN** the thinking token limit is cleared (reverts to default)
- **AND** the method returns nil error on success

#### Scenario: Query is no longer active
- **WHEN** `SetMaxThinkingTokens` is called after the query has completed
- **THEN** the method returns an error indicating the query is closed
- **AND** the control request is not sent

---

### Requirement: AccountInfo Method
The Query interface SHALL provide an `AccountInfo` method that retrieves current account information including email, organization, subscription type, and API key source.

#### Scenario: Retrieve account info successfully
- **GIVEN** the query is active
- **WHEN** `AccountInfo(ctx context.Context)` is called
- **THEN** a control request is sent to retrieve account information
- **AND** the method returns a populated `AccountInfo` struct
- **AND** the error is nil on success

#### Scenario: Account info with partial data
- **GIVEN** the query is active
- **WHEN** `AccountInfo(ctx context.Context)` is called
- **AND** some fields (e.g., organization) are not available
- **THEN** the method returns an `AccountInfo` struct
- **AND** only populated fields contain values
- **AND** unpopulated fields are zero values

#### Scenario: Context cancellation
- **GIVEN** `AccountInfo` is called with a cancelled context
- **WHEN** the context is already cancelled
- **THEN** the method returns `context.Cancelled` error
- **AND** no control request is sent to the CLI

#### Scenario: Query is closed
- **WHEN** `AccountInfo` is called after the query has completed
- **THEN** the method returns an error indicating the query is closed

---

