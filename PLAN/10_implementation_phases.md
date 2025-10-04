## Implementation Phases

This document outlines the dependency-ordered implementation phases with clear acceptance gates.

### Phase 1: Foundation
**Duration:** 1-2 weeks
**Dependencies:** None

**Work Items:**
- Core domain models (messages, options, errors packages)
- Port interfaces (Transport, ProtocolHandler, MessageParser, MCPServer)
- Basic type definitions and discriminated unions

**Deliverables:**
- `pkg/claude/messages/` - All message types and content blocks
- `pkg/claude/options/` - AgentOptions and configuration types
- `pkg/claude/ports/` - All port interfaces defined
- `pkg/claude/errors.go` - SDK error types

**Acceptance Gates:**
- ✅ All port interfaces compile without implementation
- ✅ Message types correctly handle discriminated unions
- ✅ Options package supports all MCP server config types
- ✅ golangci-lint passes with zero violations
- ✅ Package structure follows hexagonal architecture (no circular imports)

### Phase 2: Core Implementation
**Duration:** 2-3 weeks
**Dependencies:** Phase 1 (ports and domain models)

**Work Items:**
- Domain services (querying, streaming, hooks, permissions)
- File decomposition per linting constraints (175-line limit)
- Control protocol logic in domain layer
- Service initialization and dependency injection

**Deliverables:**
- `pkg/claude/querying/` - Querying service decomposed into 5 files
- `pkg/claude/streaming/` - Streaming service decomposed into 6 files
- `pkg/claude/hooking/` - Hooking service with matcher logic
- `pkg/claude/permissions/` - Permissions service with callback support

**Acceptance Gates:**
- ✅ All domain services compile and inject dependencies correctly
- ✅ No domain service imports infrastructure packages (enforced by depguard)
- ✅ All files under 175 lines
- ✅ All functions under 25 lines with cyclomatic complexity ≤15
- ✅ Unit tests for each service using mock ports (>70% coverage)

### Phase 3: Infrastructure Adapters
**Duration:** 2-3 weeks
**Dependencies:** Phase 2 (domain services need ports to inject)

**Work Items:**
- CLI subprocess transport adapter
- JSON-RPC protocol adapter with control protocol handling
- Message parser adapter
- MCP client adapter (stdio/HTTP/SSE)
- MCP SDK server adapter (in-memory transport)

**Deliverables:**
- `pkg/claude/adapters/cli/` - CLI transport implementation
- `pkg/claude/adapters/jsonrpc/` - Protocol handler with request/response routing
- `pkg/claude/adapters/parse/` - Message parser implementation
- `pkg/claude/adapters/mcp/client.go` - External MCP client wrapper
- `pkg/claude/adapters/mcp/sdk_server.go` - SDK MCP server wrapper

**Acceptance Gates:**
- ✅ All adapters implement their respective port interfaces
- ✅ CLI transport successfully spawns subprocess and pipes stdin/stdout
- ✅ JSON-RPC adapter handles all control request types
- ✅ Parser correctly deserializes all message types
- ✅ MCP adapters route messages to/from MCP SDK correctly
- ✅ Integration tests validate adapter contracts

### Phase 4: Public API & Wiring
**Duration:** 1 week
**Dependencies:** Phase 3 (needs adapters to wire)

**Work Items:**
- Public API facade (Query function, Client type)
- MCP initialization helpers
- Tool selection helpers
- System prompt builders
- Error handling and resource cleanup

**Deliverables:**
- `pkg/claude/query.go` - One-shot query function
- `pkg/claude/client.go` - Bidirectional client
- `pkg/claude/mcp_init.go` - MCP server initialization
- `pkg/claude/helpers/` - Tool and prompt helpers

**Acceptance Gates:**
- ✅ Query() successfully executes end-to-end with mock CLI
- ✅ Client Connect/Send/Receive cycle works correctly
- ✅ MCP initialization handles all config types (stdio, HTTP, SSE, SDK)
- ✅ Helper utilities tested and documented
- ✅ Public API fully documented with godoc examples

### Phase 5: Advanced Features Integration
**Duration:** 1-2 weeks
**Dependencies:** Phase 4 (needs public API)

**Work Items:**
- Hook lifecycle wiring and callback execution
- Permission flow (can_use_tool, set_permission_mode)
- MCP message proxying
- Control protocol edge cases (timeouts, cancellation)

**Deliverables:**
- Hook integration tests
- Permission callback examples
- MCP proxy validation
- Control protocol timeout handling

**Acceptance Gates:**
- ✅ Hooks execute at correct lifecycle points
- ✅ Permission callbacks correctly allow/deny tool use
- ✅ Permission suggestions applied in allow mode
- ✅ MCP messages routed to correct server
- ✅ Control requests timeout after 60 seconds
- ✅ Context cancellation propagates correctly

### Phase 6: Testing & Documentation
**Duration:** 1-2 weeks
**Dependencies:** Phase 5 (needs complete implementation)

**Work Items:**
- Comprehensive unit tests for all packages
- Integration tests for end-to-end flows
- Example programs (query, streaming, hooks, permissions, MCP)
- README and usage documentation
- godoc comments for all public APIs

**Deliverables:**
- `*_test.go` files achieving >80% coverage
- `cmd/examples/` - 5+ working examples
- `README.md` - Quick start and feature overview
- `docs/` - Detailed usage guides (optional, only if needed)

**Acceptance Gates:**
- ✅ Test coverage >80% overall
- ✅ All examples run successfully against real CLI
- ✅ Zero golangci-lint violations
- ✅ Documentation builds correctly on pkg.go.dev
- ✅ All public APIs have godoc with examples

### Phase 7: Publishing & CI/CD
**Duration:** 3-5 days
**Dependencies:** Phase 6 (needs tests and docs)

**Work Items:**
- GitHub Actions workflows (lint, test, release)
- Release automation script
- CHANGELOG.md and VERSION file
- Branch protection and CI enforcement
- Initial v0.1.0 release

**Deliverables:**
- `.github/workflows/` - Lint, test, and release workflows
- `scripts/release.sh` - Automated release script
- `CHANGELOG.md` - Keep a Changelog format
- GitHub release with v0.1.0 tag

**Acceptance Gates:**
- ✅ CI runs on every push/PR
- ✅ Linting failures block merges
- ✅ Test coverage tracked and enforced
- ✅ Release workflow creates GitHub releases
- ✅ Module published and accessible via go get
- ✅ pkg.go.dev documentation renders correctly

---

## Phase Dependencies Diagram

```
Phase 1 (Foundation)
    ↓
Phase 2 (Domain Services)
    ↓
Phase 3 (Infrastructure Adapters)
    ↓
Phase 4 (Public API)
    ↓
Phase 5 (Advanced Features)
    ↓
Phase 6 (Testing & Docs)
    ↓
Phase 7 (Publishing)
```

**Critical Path:** Phases 1-4 must be completed in order. Phases 5-7 can overlap slightly once Phase 4 is stable.

**Risk Mitigation:**
- Mock implementations allow Phase 2 to proceed before Phase 3 completion
- Integration tests in Phase 6 validate all phase deliverables work together
- CI setup in Phase 7 prevents regressions in future work