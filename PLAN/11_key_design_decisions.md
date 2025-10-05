## Key Design Decisions
### Hexagonal Architecture Principles
1. Domain Independence: Core domain packages (`querying`, `streaming`, `hooking`, `permissions`) never import adapters
2. Ports Define Contracts: Interfaces in `ports/` package are defined by domain needs, not external systems
3. Adapters Implement Ports: All infrastructure code in `adapters/` implements port interfaces
4. Dependency Direction: Always flows inward (adapters → domain), never outward (domain → adapters)
5. Package Naming: Named for what they provide (`querying`, `streaming`) not what they contain (`models`, `handlers`)
### Go Idioms
6. Channels vs Iterators: Use channels for async message streaming (idiomatic Go)
7. Context Integration: Full context.Context support throughout
8. Error Handling: Return errors explicitly, use error wrapping
9. Interface Compliance: Use `var _ ports.Transport = (*Adapter)(nil)` pattern to verify at compile time
10. Async Model: Goroutines + channels (Go's native async)
11. JSON Handling: Use encoding/json with struct tags
12. Testing Strategy: Table-driven tests, interface mocks, integration tests
### Architectural Benefits
- Testability: Domain logic testable without infrastructure dependencies
- Flexibility: Easy to swap adapters (e.g., different transport mechanisms)
- Clarity: Clear separation between business logic and technical details
- Maintainability: Changes to infrastructure don't affect domain
- Discoverability: Package names describe purpose at a glance

---

## Known Architectural Deviations & Exceptions

This section documents deliberate violations of standard rules and architectural boundaries, providing rationale and enforcement guidance for reviewers.

### 1. Infrastructure Concerns in Adapters (Not Domain)

**Deviation**: Control protocol timeout handling and request ID generation

**Location**: `adapters/jsonrpc/protocol.go` (PLAN/02:301-307)

**Rationale**:
- 60-second timeout protection for CLI control requests belongs in the adapter layer per hexagonal architecture
- Domain services pass `context.Context` but remain unaware of specific timeout values
- Request ID generation (`req_{counter}_{randomHex}`) is a protocol implementation detail, not business logic

**Status**: **Permanent** - This is the correct architectural placement

**Enforcement**: Domain services in `querying/` and `streaming/` packages MUST NOT:
- Import time package for timeout logic
- Generate request IDs directly
- Implement retry or timeout logic

**Verification**:
```bash
# Domain packages should not import time for timeout logic
grep -r "time\." querying/ streaming/ | grep -i timeout
# Should return empty
```

### 2. External Dependencies (Minimal, Not Zero)

**Deviation**: "Zero dependencies" claim contradicted by actual module manifest

**Location**: Multiple plan documents (PLAN/00:40, PLAN/01:39, PLAN/02:10)

**Actual State**: `go.mod:5-7` shows two direct dependencies:
- `github.com/mark3labs/mcp-go v0.41.1`
- `github.com/modelcontextprotocol/go-sdk v1.0.0`

**Rationale**:
- MCP protocol integration requires official SDK for compatibility
- Alternative would be reimplementing MCP spec, increasing maintenance burden
- Trade-off: dependency cost vs. implementation/maintenance cost

**Status**: **Permanent** - Accepted architectural dependency

**Corrected Messaging**: "Minimal external dependencies" or "Only essential MCP SDK dependencies"

**Enforcement**: New dependencies require:
1. Architecture review justifying necessity
2. Evaluation of alternatives (stdlib, vendoring)
3. Update to dependency rationale documentation

### 3. Linting Exceptions

#### 3A. File Length Exceptions for Generated/Schema Code

**Deviation**: 175-line limit may not apply to:
- MCP protocol schema definitions (if auto-generated)
- Large enumeration types from MCP spec
- Comprehensive test fixture files

**Rationale**: Generated code and exhaustive spec enumerations are not subject to maintainability concerns

**Status**: **Conditional** - Only for truly generated or spec-mandated code

**Enforcement**:
- Mark files with `//go:generate` directive or `// Code generated` comment
- Review required to justify exception
- Preference: Split even generated code when feasible

**Example Exception**:
```go
// Code generated from MCP JSON Schema. DO NOT EDIT.
// Source: https://spec.modelcontextprotocol.io/specification/2024-11-05/schema/
//
//nolint:lll,file-length-limit // Generated MCP schema types
package mcpschema
```

#### 3B. Line Length Exceptions for Error Messages

**Deviation**: 80-character limit difficult for descriptive error messages

**Rationale**: Errors should be clear and self-contained; breaking them reduces readability

**Status**: **Temporary workaround** - Use error variables when messages exceed 80 chars

**Enforcement**:
```go
// BAD: Inline long error
return fmt.Errorf("failed to parse control protocol response: invalid message type 'foo', expected one of: permission_request, tool_use, set_permission_mode")

// GOOD: Error variable with nolint
//nolint:lll // Error message clarity prioritized
var errInvalidMessageType = "failed to parse control protocol response: invalid message type '%s', expected one of: permission_request, tool_use, set_permission_mode"

func validate(msgType string) error {
    return fmt.Errorf(errInvalidMessageType, msgType)
}
```

#### 3C. Function Parameter Exceptions for Builders

**Deviation**: Some internal constructors may exceed 4 parameters during refactoring

**Rationale**: Intermediate state during migration from monolithic to builder pattern

**Status**: **Temporary** - Must be resolved before release

**Enforcement**:
- Mark with `//nolint:argument-limit // TODO(issue-123): Migrate to builder pattern`
- Create tracking issue in repository
- Maximum 6 parameters even with exception
- No exceptions in public API

**Validation Gate**: CI blocks release branches if `argument-limit` exceptions exist in public packages

### 4. Control Protocol State Management

**Deviation**: Initially planned in domain services, corrected to adapters

**Original Plan**: Phase 2 domain services included request tracking (PLAN/04:120)

**Corrected Placement**: `adapters/jsonrpc/protocol.go` manages:
- Pending request tracking
- Request ID generation
- Response routing
- Timeout handling

**Rationale**: State management is infrastructure concern, not business logic

**Status**: **Resolved** - Plan updated, implementation must follow corrected architecture

**Enforcement**: Code review checklist:
- [ ] Domain services do not maintain request maps
- [ ] Domain services do not track pending operations
- [ ] All protocol state in `adapters/jsonrpc/` package

### 5. MCP In-Memory Transport Limitations

**Deviation**: Plan promises channel-based in-memory MCP transport, but upstream SDK may not support it

**Location**: PLAN/07b:40, PLAN/02:408

**Current Reality**: Python SDK manually routes JSON-RPC without transport abstraction (reference: `claude-agent-sdk-python/src/claude_agent_sdk/_internal/query.py:326`)

**Rationale**: Upstream Go MCP SDK may require similar manual routing initially

**Status**: **Acknowledged limitation** - Implementation may require interim workaround

**Enforcement**:
- Document actual transport mechanism in Phase 5
- Create enhancement issue for upstream SDK if needed
- Tests must cover actual implementation (manual routing if needed)
- Update plan with "future enhancement" caveat

**Interim Approach**: If upstream doesn't provide transport:
```go
// Temporary manual routing until mcp-go provides transport abstraction
// TODO(upstream): Replace with sdk.InMemoryTransport when available
func (a *MCPAdapter) routeRequest(ctx context.Context, req JSONRPCRequest) (JSONRPCResponse, error) {
    // Manual message routing similar to Python SDK
}
```

### 6. Hook Type System Differences

**Deviation**: Plan lists 9 hook events (PLAN/12:13) vs Python SDK's 6 events

**Rationale**: Either Go SDK extends hook types OR plan overstates parity

**Status**: **Under Investigation** - Requires alignment decision

**Enforcement Options**:
1. **Strict Parity**: Match Python SDK exactly (6 hooks)
2. **Justified Extension**: Document 3 additional hooks with use cases
3. **Staged Rollout**: Start with 6, add extensions in v2

**Decision Required**: Architecture review before Phase 5 implementation

**Validation**: Hook types must match documented count and have corresponding test coverage

---

## Exception Request Process

For new exceptions not covered above:

1. **Document Justification**: Why is the exception necessary?
2. **Explore Alternatives**: What approaches were considered and rejected?
3. **Define Scope**: Exactly which files/functions require the exception?
4. **Mark Code**: Use `//nolint` directives with explanatory comments
5. **Time-box Temporary**: Set resolution deadline for temporary exceptions
6. **Review Gate**: Require architecture review approval

**Example Documentation**:
```go
// Package mcpextensions provides MCP server type definitions.
//
// ARCHITECTURAL EXCEPTION: This package exceeds the 19 public struct limit
// due to comprehensive MCP tool schema types defined in the spec.
// Approved: 2024-01-15 (ref: architecture-review-2024-01)
// Alternative considered: Sub-packages rejected due to circular dependency issues.
//
//nolint:max-public-structs // MCP spec requires 27 tool schema types
package mcpextensions
```

---

## Deviation Review Schedule

**Monthly**: Review temporary exceptions and track progress toward resolution
**Per-Release**: Audit all `//nolint` directives and validate justifications still apply
**Quarterly**: Architecture review of permanent exceptions for potential refactoring

**Metrics to Track**:
- Number of exceptions by type (file-length, argument-limit, etc.)
- Age of temporary exceptions
- Public API vs internal exceptions ratio

This ensures architectural discipline while acknowledging pragmatic constraints.