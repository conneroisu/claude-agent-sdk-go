## Implementation Phases

This document outlines the dependency-ordered implementation phases with clear acceptance gates, staffing requirements, risk mitigation strategies, and explicit file decomposition tactics to meet linting constraints.

### Phase 1: Foundation
**Duration:** 1-2 weeks
**Dependencies:** None
**Staffing:** 1 Backend Go Developer (lead), 1 Backend Go Developer (support)

**Work Items:**
- Core domain models (messages, options, errors packages)
- Port interfaces (Transport, ProtocolHandler, MessageParser, MCPServer)
- Basic type definitions and discriminated unions
- File decomposition per PLAN/14 (175-line limit, 25-line function limit)

**Deliverables:**
- `pkg/claude/messages/` - All message types and content blocks (8 files, see decomposition)
- `pkg/claude/options/` - AgentOptions and configuration types (3 files)
- `pkg/claude/ports/` - All port interfaces defined (4 files)
- `pkg/claude/errors.go` - SDK error types (1 file)

**File Decomposition Strategy (ref: PLAN/14):**
- `messages/` split into 8 files to stay under 175-line limit:
  - `messages.go` (interfaces, 50 lines)
  - `user.go`, `assistant.go`, `system.go`, `result.go`, `stream.go`, `content.go`, `usage.go`
- Function extraction for parsing logic (25-line limit per function)

**Checkpoints:**
- **Week 1, Day 3:** Port interfaces compile, types defined
- **Week 1, Day 5:** Message types implemented with discriminated unions
- **Week 2, Day 2:** Options package complete
- **Week 2, Day 4:** All files pass linting (175-line check)

**Risk Mitigation:**
- **Risk:** Message type complexity exceeds function line limits
  - **Mitigation:** Extract sub-parsers per message type (see PLAN/14 Pattern B)
- **Risk:** Discriminated union handling causes deep nesting (>3 levels)
  - **Mitigation:** Use early returns and type switch extraction patterns
- **Risk:** Circular imports between messages and options
  - **Mitigation:** Enforce depguard rules, domain packages cannot import each other

**Acceptance Gates:**
- ✅ All port interfaces compile without implementation
- ✅ Message types correctly handle discriminated unions
- ✅ Options package supports all MCP server config types
- ✅ golangci-lint passes with zero violations (all files <175 lines, functions <25 lines)
- ✅ Package structure follows hexagonal architecture (no circular imports)
- ✅ 15% comment density achieved per PLAN/14

### Phase 2: Core Implementation
**Duration:** 2-3 weeks
**Dependencies:** Phase 1 (ports and domain models)
**Staffing:** 2 Backend Go Developers (can work in parallel on separate services)

**Work Items:**
- Domain services (querying, streaming, hooks, permissions)
- File decomposition per PLAN/14 linting constraints (175-line limit)
- Control protocol logic in domain layer
- Service initialization and dependency injection
- Mock port implementations for testing

**Deliverables:**
- `pkg/claude/querying/` - Querying service decomposed into 5 files
- `pkg/claude/streaming/` - Streaming service decomposed into 6 files
- `pkg/claude/hooking/` - Hooking service with matcher logic (4 files)
- `pkg/claude/permissions/` - Permissions service with callback support (3 files)
- `pkg/claude/mocks/` - Mock implementations of all ports for testing

**File Decomposition Strategy (ref: PLAN/14):**
- `querying/`: `service.go` (60L), `execute.go` (80L), `routing.go` (70L), `errors.go` (50L), `state.go` (40L)
- `streaming/`: `service.go` (50L), `connect.go` (70L), `send.go` (60L), `receive.go` (80L), `lifecycle.go` (50L), `state.go` (40L)
- Extract validation functions to stay under 25-line function limit
- Use handler maps instead of large switch statements to reduce cyclomatic complexity

**Checkpoints:**
- **Week 1, Day 2:** Mock port implementations complete
- **Week 1, Day 5:** Querying service implemented and tested with mocks
- **Week 2, Day 3:** Streaming service implemented and tested with mocks
- **Week 2, Day 5:** Hooking and permissions services complete
- **Week 3, Day 2:** All unit tests passing (>70% coverage)
- **Week 3, Day 4:** Lint compliance verified (all files <175 lines)

**Risk Mitigation:**
- **Risk:** Phase 3 adapters not ready blocks integration testing
  - **Mitigation:** Mock port implementations allow Phase 2 to proceed independently and test in isolation
- **Risk:** Control protocol handling increases complexity beyond limits (cyclomatic >15)
  - **Mitigation:** Extract per-subtype handlers into separate functions (ref: PLAN/14 Pattern B)
- **Risk:** Message routing logic causes deep nesting (>3 levels)
  - **Mitigation:** Use handler registry pattern and early returns (ref: PLAN/14 nesting reduction)
- **Risk:** Function line limits prevent complete implementation
  - **Mitigation:** Aggressive function extraction - validation, initialization, execution as separate helpers

**Parallelization Opportunities:**
- Querying and streaming services can be developed simultaneously by different developers
- Hooking and permissions services can start once service patterns are established

**Acceptance Gates:**
- ✅ All domain services compile and inject dependencies correctly
- ✅ No domain service imports infrastructure packages (enforced by depguard)
- ✅ All files under 175 lines (verified via golangci-lint file-length-limit)
- ✅ All functions under 25 lines with cyclomatic complexity ≤15
- ✅ Max nesting depth ≤3 (enforced by max-control-nesting rule)
- ✅ Unit tests for each service using mock ports (>70% coverage)
- ✅ Mock implementations allow downstream phases to begin

### Phase 3: Infrastructure Adapters
**Duration:** 2-3 weeks
**Dependencies:** Phase 2 (domain services need ports to inject), but can start early using port interfaces
**Staffing:** 1 Backend Go Developer (lead), 1 Backend Go Developer (MCP focus), 1 DevOps Engineer (CLI/process management)

**Work Items:**
- CLI subprocess transport adapter with process lifecycle management
- JSON-RPC protocol adapter with control protocol handling
- Message parser adapter with type discrimination
- MCP client adapter (stdio/HTTP/SSE)
- MCP SDK server adapter (manual routing until Go SDK supports in-memory transport)

**Deliverables:**
- `pkg/claude/adapters/cli/` - CLI transport (7 files, see decomposition)
- `pkg/claude/adapters/jsonrpc/` - Protocol handler (5 files)
- `pkg/claude/adapters/parse/` - Message parser (3 files)
- `pkg/claude/adapters/mcp/client.go` - External MCP client wrapper
- `pkg/claude/adapters/mcp/sdk_server.go` - SDK MCP server wrapper (manual routing)

**File Decomposition Strategy (ref: PLAN/14):**
- `cli/`: `transport.go` (60L), `connect.go` (70L), `command.go` (80L), `io.go` (90L), `discovery.go` (50L), `process.go` (60L), `errors.go` (40L)
- `jsonrpc/`: `protocol.go` (50L), `control.go` (80L), `routing.go` (90L), `handlers.go` (80L), `state.go` (50L)
- Use builder pattern for CLI command assembly (ref: PLAN/14 Pattern A)
- Extract I/O readers/writers to separate helpers

**Checkpoints:**
- **Week 1, Day 3:** CLI transport connects and spawns subprocess
- **Week 1, Day 5:** JSON-RPC protocol sends/receives basic messages
- **Week 2, Day 2:** Message parser handles all message types
- **Week 2, Day 4:** MCP adapters route to external servers
- **Week 3, Day 2:** All adapters pass integration tests
- **Week 3, Day 4:** Lint compliance verified

**Risk Mitigation:**
- **Risk:** CLI discovery fails on different OS/environments
  - **Mitigation:** Fallback chain (env var → PATH search → common locations), extensive testing on Linux/macOS/Windows
- **Risk:** Process I/O handling complexity exceeds line/complexity limits
  - **Mitigation:** Extract reader/writer goroutines to separate files, use handler registry for routing
- **Risk:** MCP Go SDK lacks in-memory transport (like Python)
  - **Mitigation:** Manual JSON-RPC routing similar to Python SDK workaround (ref: claude-agent-sdk-python query.py:330)
- **Risk:** JSON-RPC control protocol increases cyclomatic complexity
  - **Mitigation:** Handler map pattern with per-subtype functions (ref: PLAN/14 Pattern B)

**Parallelization Opportunities:**
- CLI and JSON-RPC adapters can start immediately using port interfaces from Phase 1
- Parser adapter can be developed in parallel
- MCP adapters can start once protocol adapter establishes routing patterns

**Acceptance Gates:**
- ✅ All adapters implement their respective port interfaces
- ✅ CLI transport successfully spawns subprocess and pipes stdin/stdout
- ✅ CLI discovery works across Linux/macOS/Windows
- ✅ JSON-RPC adapter handles all control request types with timeout (60s default)
- ✅ Parser correctly deserializes all message types (discriminated unions work)
- ✅ MCP adapters route messages to/from MCP SDK correctly
- ✅ Integration tests validate adapter contracts against real CLI
- ✅ All files <175 lines, functions <25 lines, complexity ≤15

### Phase 4: Public API & Wiring
**Duration:** 1 week
**Dependencies:** Phase 3 (needs adapters to wire)
**Staffing:** 1 Backend Go Developer (API design focus), 1 Technical Writer (documentation)

**Work Items:**
- Public API facade (Query function, Client type)
- MCP initialization helpers with config validation
- Tool selection helpers
- System prompt builders
- Error handling and resource cleanup
- Dependency injection wiring

**Deliverables:**
- `pkg/claude/query.go` - One-shot query function (80 lines)
- `pkg/claude/client.go` - Bidirectional client (120 lines, may split if needed)
- `pkg/claude/mcp_init.go` - MCP server initialization (90 lines)
- `pkg/claude/helpers/` - Tool and prompt helpers (3 files, ~60 lines each)
- `pkg/claude/wiring.go` - Dependency injection helpers (70 lines)

**File Decomposition Strategy:**
- If `client.go` exceeds 175 lines, split into: `client.go` (struct/New), `client_connect.go`, `client_send.go`, `client_receive.go`
- Use config structs to stay under 4-parameter limit (ref: PLAN/14 Pattern A)
- Builder pattern for complex Client initialization

**Checkpoints:**
- **Day 2:** Query() function complete with end-to-end test
- **Day 3:** Client type implemented with lifecycle methods
- **Day 4:** MCP initialization supports all config types
- **Day 5:** Helpers implemented, all APIs documented

**Risk Mitigation:**
- **Risk:** Query() complexity exceeds function limits
  - **Mitigation:** Extract validation, initialization, execution into separate helpers
- **Risk:** Client lifecycle management causes deep nesting
  - **Mitigation:** State machine pattern with early returns, extract state transition helpers
- **Risk:** MCP config validation increases parameter count
  - **Mitigation:** Use MCPConfig struct to group related parameters
- **Risk:** Resource cleanup complexity (channels, goroutines, processes)
  - **Mitigation:** Defer pattern with cleanup helper functions, context cancellation

**Parallelization Opportunities:**
- MCP initialization can be developed while Client is being finalized
- Helpers can be developed in parallel by separate developer
- Documentation can start as soon as API signatures are stable

**Acceptance Gates:**
- ✅ Query() successfully executes end-to-end with real CLI adapter
- ✅ Client Connect/Send/Receive cycle works correctly
- ✅ MCP initialization handles all config types (stdio, HTTP, SSE, SDK with manual routing)
- ✅ Helper utilities tested with table-driven tests
- ✅ Public API fully documented with godoc examples (every exported symbol)
- ✅ Resource cleanup verified (no goroutine leaks, processes terminated)
- ✅ All files <175 lines, <4 params per function

### Phase 5: Advanced Features Integration
**Duration:** 1-2 weeks
**Dependencies:** Phase 4 (needs public API)
**Staffing:** 1 Backend Go Developer (integration focus), 1 QA Engineer (edge case testing)

**Work Items:**
- Hook lifecycle wiring and callback execution
- Permission flow integration (can_use_tool, set_permission_mode)
- MCP message proxying validation
- Control protocol edge cases (timeouts, cancellation)
- Concurrent safety for hooks/permissions

**Deliverables:**
- `pkg/claude/integration/` - Hook and permission integration logic (3 files)
- Hook integration tests with real CLI
- Permission callback examples (allow/deny/suggest modes)
- MCP proxy validation tests
- Control protocol timeout and cancellation handling
- Concurrency safety tests

**File Decomposition Strategy:**
- `integration/hooks.go` - Hook execution and lifecycle (80 lines)
- `integration/permissions.go` - Permission callback handling (90 lines)
- `integration/mcp_proxy.go` - MCP message routing (70 lines)

**Checkpoints:**
- **Week 1, Day 2:** Hook wiring complete, lifecycle tests passing
- **Week 1, Day 4:** Permission flow implemented with all modes
- **Week 1, Day 5:** MCP proxying validated with multiple servers
- **Week 2, Day 2:** Timeout/cancellation edge cases handled
- **Week 2, Day 3:** Concurrency tests passing

**Risk Mitigation:**
- **Risk:** Hook execution timeout/panic handling adds complexity
  - **Mitigation:** Extract timeout wrapper and panic recovery to separate helper (ref: PLAN/07a_phase_5_hooks.md)
- **Risk:** Permission callbacks block message flow
  - **Mitigation:** Async callback execution with timeout, default deny on timeout
- **Risk:** MCP routing to wrong server causes data leakage
  - **Mitigation:** Explicit server-ID validation before routing, integration tests with multiple servers
- **Risk:** Context cancellation doesn't propagate to all goroutines
  - **Mitigation:** Context passed to all async operations, defer cleanup pattern

**Parallelization Opportunities:**
- Hooks and permissions can be integrated in parallel (different control flows)
- MCP proxy validation can run concurrently with hook testing
- Edge case testing (timeouts, cancellation) can be parallelized across features

**Overlap with Phase 6:**
- Integration tests from Phase 5 feed directly into Phase 6 test suite
- Example programs can start being written as features stabilize

**Acceptance Gates:**
- ✅ Hooks execute at correct lifecycle points (9 hook events per PLAN/12)
- ✅ Hook timeout/panic handling works correctly (default 30s timeout)
- ✅ Permission callbacks correctly allow/deny tool use
- ✅ Permission suggestions applied in allow mode
- ✅ Permission mode switching persists across control requests
- ✅ MCP messages routed to correct server based on server-ID
- ✅ Control requests timeout after 60 seconds (configurable)
- ✅ Context cancellation propagates to all goroutines (no leaks)
- ✅ Concurrent hook/permission execution is thread-safe
- ✅ All files <175 lines, complexity ≤15

### Phase 6: Testing & Documentation
**Duration:** 1-2 weeks
**Dependencies:** Phase 5 (needs complete implementation), overlaps with Phase 5
**Staffing:** 1 QA Engineer (testing lead), 1 Technical Writer (documentation), 1 Backend Go Developer (example programs)

**Work Items:**
- Comprehensive unit tests for all packages (table-driven)
- Integration tests for end-to-end flows
- Example programs (query, streaming, hooks, permissions, MCP)
- README and usage documentation
- godoc comments for all public APIs
- Test fixtures and shared test utilities

**Deliverables:**
- `*_test.go` files achieving >80% coverage (using table-driven tests, ref: PLAN/14)
- `internal/testutil/` - Shared test fixtures and utilities
- `cmd/examples/` - 5+ working examples (query, streaming, hooks, permissions, MCP)
- `README.md` - Quick start and feature overview
- godoc examples for all public APIs

**File Decomposition Strategy (ref: PLAN/14):**
- Test files kept under 175 lines using table-driven tests
- Shared fixtures in `testutil/fixtures.go` (separate from test logic)
- Test helpers in `testutil/helpers.go` (mock setup, assertions)
- Each test file: ~40 lines table test + ~30 lines setup + ~30 lines helpers = ~100 lines

**Checkpoints:**
- **Week 1, Day 2:** Unit tests for Phase 1-2 packages complete (>70% coverage)
- **Week 1, Day 4:** Unit tests for Phase 3-4 packages complete (>75% coverage)
- **Week 1, Day 5:** Integration tests for all end-to-end flows passing
- **Week 2, Day 2:** Example programs written and tested
- **Week 2, Day 3:** Documentation complete (README + godoc)
- **Week 2, Day 4:** Coverage >80%, all lint violations fixed

**Risk Mitigation:**
- **Risk:** Test files exceed 175-line limit
  - **Mitigation:** Table-driven tests with shared fixtures pattern (ref: PLAN/14 Pattern 1)
- **Risk:** Integration tests fail due to CLI availability
  - **Mitigation:** Hermetic testing with record/replay transport, fallback to mock transport
- **Risk:** Coverage <80% due to error paths
  - **Mitigation:** Error injection in mocks, edge case table tests
- **Risk:** Examples break due to API changes
  - **Mitigation:** Run examples as part of CI, treat as integration tests

**Parallelization Opportunities:**
- Unit tests can be written in parallel across packages
- Example programs can be developed while tests are being written
- Documentation can be written as soon as APIs are stable (during Phase 5)

**Overlap with Phase 5:**
- Start unit tests for stable packages while Phase 5 features are being integrated
- Begin README and godoc while APIs are being finalized
- Example programs can validate Phase 5 integration work

**Acceptance Gates:**
- ✅ Test coverage >80% overall (verified via go test -cover)
- ✅ All test files under 175 lines (using table-driven pattern)
- ✅ Shared test fixtures eliminate duplication
- ✅ All examples run successfully against real CLI
- ✅ Examples demonstrate: query, streaming, hooks, permissions, MCP servers
- ✅ Zero golangci-lint violations across entire codebase
- ✅ Documentation builds correctly on pkg.go.dev (verified locally)
- ✅ All public APIs have godoc with runnable examples
- ✅ README includes quick start, installation, and feature overview

### Phase 7: Publishing & CI/CD
**Duration:** 3-5 days
**Dependencies:** Phase 6 (needs tests and docs)
**Staffing:** 1 DevOps Engineer (CI/CD lead), 1 Backend Go Developer (release validation)

**Work Items:**
- GitHub Actions workflows (lint, test, release)
- Release automation script with safeguards
- CHANGELOG.md and VERSION file
- Branch protection and CI enforcement
- Initial v0.1.0 release
- pkg.go.dev documentation verification

**Deliverables:**
- `.github/workflows/lint.yml` - golangci-lint on every push/PR
- `.github/workflows/test.yml` - Tests with coverage reporting
- `.github/workflows/release.yml` - Automated release on tag push
- `scripts/release.sh` - Release script with validation checks
- `CHANGELOG.md` - Keep a Changelog format (semantic versioning)
- GitHub release with v0.1.0 tag

**Checkpoints:**
- **Day 1:** Lint and test workflows configured and passing
- **Day 2:** Branch protection rules enabled (require passing CI)
- **Day 3:** Release automation tested with dry-run
- **Day 4:** CHANGELOG complete, v0.1.0 release created
- **Day 5:** pkg.go.dev documentation verified and indexed

**Risk Mitigation:**
- **Risk:** Release script pushes to main without validation
  - **Mitigation:** Dry-run mode, require clean working tree, tag validation before push
- **Risk:** CI fails on legitimate code due to overly strict linting
  - **Mitigation:** Linting plan (PLAN/14) already validated during Phase 6, escalation path for exceptions
- **Risk:** Test coverage drops below 80% without detection
  - **Mitigation:** Coverage enforcement in CI, fail PR if coverage decreases
- **Risk:** pkg.go.dev doesn't index module correctly
  - **Mitigation:** Local godoc verification in Phase 6, test with pkg.go.dev proxy before release

**Parallelization Opportunities:**
- Lint and test workflows can be configured in parallel
- CHANGELOG can be drafted while CI is being set up
- Release script development can happen alongside workflow configuration

**Overlap with Future Work:**
- CI/CD foundation enables rapid iteration post-v0.1.0
- Branch protection prevents regression from future contributions
- Release automation streamlines future version bumps

**Acceptance Gates:**
- ✅ CI runs on every push/PR (lint + test workflows)
- ✅ Linting failures block merges (branch protection enabled)
- ✅ Test coverage tracked and enforced (>80% required)
- ✅ Multi-Go-version testing (1.22, 1.23, 1.24+)
- ✅ Release workflow creates GitHub releases with artifacts
- ✅ Release script validates: clean tree, passing tests, version bump
- ✅ Module published and accessible via go get github.com/user/claude-agent-sdk-go
- ✅ pkg.go.dev documentation renders correctly with examples
- ✅ CHANGELOG follows Keep a Changelog format with v0.1.0 entry

---

## Phase Dependencies & Parallelization Strategy

### Critical Path (Sequential Dependencies)
```
Phase 1 (Foundation - Weeks 1-2)
    ↓
Phase 2 (Domain Services - Weeks 3-5)
    ↓ (Mocks enable parallel work)
Phase 3 (Adapters - Weeks 4-7) ← Can start Week 4 using port interfaces
    ↓
Phase 4 (Public API - Week 8)
    ↓ (Overlaps with Phase 6 testing)
Phase 5 (Advanced Features - Weeks 9-10)
    ↓ (Overlaps with Phase 6 docs)
Phase 6 (Testing & Docs - Weeks 10-11)
    ↓
Phase 7 (CI/CD & Release - Week 12)
```

### Parallel Work Opportunities

**Weeks 1-2 (Phase 1):**
- Lead dev: Port interfaces and domain models
- Support dev: Message types and options (can work in parallel on separate packages)

**Weeks 3-5 (Phase 2):**
- Dev 1: Querying and streaming services
- Dev 2: Hooking and permissions services (start Week 4)
- Dev 2: Mock implementations (Week 3) to unblock Phase 3

**Weeks 4-7 (Phase 3 - Overlaps with Phase 2):**
- Dev 1: CLI transport and process management
- Dev 2: JSON-RPC protocol and parser
- Dev 3 (DevOps): MCP adapters and integration tests
- **Key**: Phase 3 can start Week 4 using port interfaces from Phase 1

**Week 8 (Phase 4):**
- Dev 1: Query/Client API
- Dev 2: MCP initialization helpers
- Tech Writer: Begin API documentation drafts

**Weeks 9-10 (Phase 5 - Overlaps with Phase 6):**
- Dev 1: Hook and permission integration
- QA: Edge case testing in parallel
- Phase 6 unit tests can begin for stable packages

**Weeks 10-11 (Phase 6 - Overlaps with Phase 5):**
- QA: Unit and integration tests
- Dev: Example programs
- Tech Writer: README and godoc (can start Week 9 once APIs stabilize)

**Week 12 (Phase 7):**
- DevOps: CI/CD workflows
- Dev: Release validation and final testing

### Staffing Summary by Phase

| Phase | Backend Go Devs | DevOps Eng | QA Eng | Tech Writer | Total FTE |
|-------|-----------------|------------|--------|-------------|-----------|
| 1     | 2               | -          | -      | -           | 2.0       |
| 2     | 2               | -          | -      | -           | 2.0       |
| 3     | 2               | 1          | -      | -           | 3.0       |
| 4     | 1               | -          | -      | 1           | 2.0       |
| 5     | 1               | -          | 1      | -           | 2.0       |
| 6     | 1               | -          | 1      | 1           | 3.0       |
| 7     | 1               | 1          | -      | -           | 2.0       |

**Peak Staffing:** Week 6-7 (Phase 3) requires 3 FTE (2 backend + 1 DevOps)
**Average Staffing:** ~2.3 FTE across all phases

### Risk Mitigation Summary

**Cross-Phase Risks:**

1. **Dependency Blocking Risk**
   - **Issue:** Phase 3 blocked waiting for Phase 2 domain services
   - **Mitigation:** Mock port implementations in Phase 2 allow Phase 3 to start Week 4
   - **Owner:** Phase 2 lead developer

2. **Linting Compliance Risk**
   - **Issue:** 175-line limit and 25-line function limit may be discovered late
   - **Mitigation:** File decomposition planned upfront in PLAN/14, lint checks at every phase checkpoint
   - **Owner:** All developers (enforced via CI from Phase 7)

3. **Integration Testing Risk**
   - **Issue:** Components work in isolation but fail when integrated
   - **Mitigation:** Integration tests start in Phase 3 and expand in Phase 5-6, mocks enable early testing
   - **Owner:** QA engineer (Phase 5-6)

4. **CLI Availability Risk**
   - **Issue:** CI environments may not have Claude CLI installed
   - **Mitigation:** Hermetic testing with record/replay transport, fallback to mocks in CI
   - **Owner:** DevOps engineer (Phase 3, 7)

5. **API Stability Risk**
   - **Issue:** Late API changes break examples and documentation
   - **Mitigation:** API design frozen after Phase 4, examples treated as integration tests in CI
   - **Owner:** Tech writer + Dev lead

6. **Coverage Regression Risk**
   - **Issue:** Coverage drops below 80% in later phases
   - **Mitigation:** Coverage tracking in CI (Phase 7), fail PR if coverage decreases
   - **Owner:** QA engineer (Phase 6) + DevOps engineer (Phase 7)

### Key Success Enablers

1. **Mock Implementations (Phase 2):** Unblock Phase 3 development
2. **Port Interfaces (Phase 1):** Enable parallel Phase 2-3 work
3. **Table-Driven Tests (Phase 6):** Maintain <175 line test files
4. **File Decomposition Strategy (PLAN/14):** Prevent late-phase linting failures
5. **Incremental Integration Tests (Phases 3-6):** Catch integration issues early
6. **CI Enforcement (Phase 7):** Prevent future regressions

### Total Timeline: 12 Weeks (3 Months)

- **Optimistic (with full parallelization):** 10 weeks
- **Realistic (with some blocking):** 12 weeks
- **Conservative (with significant blocking):** 14-16 weeks

**Critical Path Items:** Phase 1 port interfaces, Phase 2 mock implementations, Phase 4 API freeze