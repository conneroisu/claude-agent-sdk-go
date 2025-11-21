# permission-system Specification

## MODIFIED Requirements

### Requirement: CanUseToolFunc Callback Signature
The CanUseToolFunc callback SHALL be enhanced with additional parameters for fine-grained permission control.

#### Scenario: Tool use identification
- **GIVEN** a CanUseToolFunc callback
- **WHEN** invoked
- **THEN** it receives toolUseID identifying the specific tool invocation
- **AND** it can use this for detailed logging or permission tracking

#### Scenario: Agent context in permissions
- **GIVEN** a CanUseToolFunc callback invoked from a subagent
- **WHEN** the callback is executed
- **THEN** it receives agentID identifying the subagent
- **AND** can apply agent-specific permission policies

#### Scenario: Blocked path indication
- **GIVEN** a CanUseToolFunc callback
- **WHEN** invoked for a tool that accesses filesystem
- **THEN** it receives blockedPath if the path was blocked
- **AND** can provide context about the block

#### Scenario: Decision reason context
- **GIVEN** a CanUseToolFunc callback
- **WHEN** previous permission denied occurred
- **THEN** it receives decisionReason explaining the prior decision
- **AND** can use this for context-aware permission logic

#### Scenario: Backward compatibility
- **GIVEN** existing code with old CanUseToolFunc signature
- **WHEN** new signature is adopted
- **THEN** all parameters are provided (some may be nil/empty)
- **AND** code can safely ignore new parameters

---

### Requirement: PermissionAllow Result Enhancement
The PermissionAllow result type SHALL include a toolUseID field for tracking.

#### Scenario: Allow with tool tracking
- **GIVEN** a permission decision allows a tool
- **WHEN** the PermissionAllow result is created
- **THEN** it includes toolUseID matching the request
- **AND** can be used for auditing tool executions

---

### Requirement: PermissionDeny Result Enhancement
The PermissionDeny result type SHALL include a toolUseID field for tracking.

#### Scenario: Deny with tool tracking
- **GIVEN** a permission decision denies a tool
- **WHEN** the PermissionDeny result is created
- **THEN** it includes toolUseID matching the request
- **AND** can be used for auditing blocked tools

---

### Requirement: HookCallbackMatcher Timeout Support
The HookCallbackMatcher type SHALL support timeout configuration for hook execution.

#### Scenario: Hook execution timeout
- **GIVEN** a HookCallbackMatcher with timeout configured
- **WHEN** the hook executes
- **THEN** if execution exceeds the timeout
- **AND** the hook is terminated
- **AND** the query handles the timeout appropriately

#### Scenario: No timeout specified
- **GIVEN** a HookCallbackMatcher without timeout
- **WHEN** the hook executes
- **THEN** it uses the default timeout
- **AND** execution can proceed without artificial limits

#### Scenario: Timeout in milliseconds or Duration
- **GIVEN** the Timeout field
- **WHEN** set
- **THEN** it specifies the maximum time for hook execution
- **AND** is enforced by the hook execution framework

---

