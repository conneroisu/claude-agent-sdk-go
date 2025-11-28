# Result Message Specification

## Purpose

TODO: Add purpose description

## Requirements

### Requirement: SDKResultMessage Structure
The SDKResultMessage type SHALL support structured output results and error collections.

#### Scenario: Result with structured output
- **GIVEN** a result from a query with structured output configuration
- **WHEN** the result message is received
- **THEN** the StructuredOutput field contains the structured data
- **AND** the data format matches the requested JSON Schema
- **AND** JSON serialization includes the `structured_output` field

#### Scenario: Success result with no structured output
- **GIVEN** a successful result from a query without structured output
- **WHEN** the result message is received
- **THEN** the StructuredOutput field is nil
- **AND** the field is omitted from JSON serialization

#### Scenario: Error result with error list
- **GIVEN** an error result (subtype: error_max_budget_usd or similar)
- **WHEN** the result message is received
- **THEN** the Errors field contains an array of error messages
- **AND** each error provides context about the failure
- **AND** the field is included in JSON serialization

#### Scenario: Success result without errors
- **GIVEN** a successful result
- **WHEN** the result message is processed
- **THEN** the Errors field is nil or empty
- **AND** the field is omitted from JSON serialization for success results

---

