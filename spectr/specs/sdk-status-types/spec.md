# Sdk Status Types Specification

## Purpose

TODO: Add purpose description

## Requirements

### Requirement: SDKStatus Type
The system SHALL define `SDKStatus` type to represent system status values.

#### Scenario: Compacting status
- **GIVEN** the system is performing message compaction
- **WHEN** status is reported
- **THEN** SDKStatus = "compacting"
- **AND** it can be used in SDKStatusMessage.Status

#### Scenario: Null/empty status
- **GIVEN** no status is being reported
- **WHEN** status is checked
- **THEN** SDKStatus = "" or null
- **AND** it represents idle/normal state

#### Scenario: Status values are constants
- **WHEN** code references status values
- **THEN** they match TypeScript SDK values exactly
- **AND** new status values can be added without breaking existing code

---

