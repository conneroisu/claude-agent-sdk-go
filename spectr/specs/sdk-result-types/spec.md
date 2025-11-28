# Sdk Result Types Specification

## Purpose

TODO: Add purpose description

## Requirements

### Requirement: Error Subtype for Budget Exceeded
The system SHALL support an error subtype for maximum budget USD limit exceeded.

#### Scenario: Budget limit exceeded
- **GIVEN** a ClientOptions with maxBudgetUsd set to a limit
- **WHEN** the query operations exceed the budget
- **THEN** the result has subtype "error_max_budget_usd"
- **AND** the Errors field contains details about the budget overage
- **AND** the error is clearly distinguishable from other error types

#### Scenario: Budget error in result message
- **GIVEN** a result message with subtype "error_max_budget_usd"
- **WHEN** processed by the SDK
- **THEN** ResultSubtypeErrorMaxBudgetUsd constant matches the subtype value
- **AND** callers can check result.Subtype == ResultSubtypeErrorMaxBudgetUsd

---

### Requirement: Error Subtype for Structured Output Retries Exceeded
The system SHALL support an error subtype for maximum structured output retries exceeded.

#### Scenario: Structured output retry limit exceeded
- **GIVEN** a query with OutputFormat configured
- **AND** the system has attempted the maximum number of retries
- **WHEN** structured output validation still fails
- **THEN** the result has subtype "error_max_structured_output_retries"
- **AND** the Errors field contains details about each failed retry
- **AND** the error is clearly distinguishable from other error types

#### Scenario: Structured output retries error in result message
- **GIVEN** a result message with subtype "error_max_structured_output_retries"
- **WHEN** processed by the SDK
- **THEN** ResultSubtypeErrorMaxStructuredOutputRetries constant matches the subtype value
- **AND** callers can check result.Subtype == ResultSubtypeErrorMaxStructuredOutputRetries

---

