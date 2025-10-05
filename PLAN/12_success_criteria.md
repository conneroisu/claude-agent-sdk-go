## Success Criteria

This document defines measurable success criteria with explicit verification methods for each requirement.

### 1. Functional Parity with Python SDK

**Criterion:** 100% feature parity with Python SDK reference implementation

**Verification Methods:**
- ✅ Cross-reference checklist comparing Python SDK features to Go SDK implementation
  - Query API (`query()` function exists and works)
  - Streaming Client (`Client` class with `connect/send/receive` methods)
  - Hooks support (all 6 hook events matching Python SDK: PreToolUse, PostToolUse, UserPromptSubmit, Stop, SubagentStop, PreCompact)
  - Permissions (`can_use_tool` callback, `set_permission_mode`, permission suggestions)
  - MCP integration (stdio, HTTP, SSE client connections + SDK server wrapping via `create_sdk_mcp_server`)
  - Builtin tools (Bash, Read, Write, Edit, Glob, Grep, etc. - provided by CLI, SDK enables permission callbacks)

- ✅ Integration test suite that validates:
  - One-shot query completes successfully
  - Bidirectional streaming conversation works
  - Hooks execute at correct lifecycle points
  - Permission callbacks correctly allow/deny tool use
  - MCP messages route to correct servers
  - Control protocol handles all request/response types

- ✅ Example programs demonstrating each major feature:
  - `cmd/examples/query/` - Simple one-shot query
  - `cmd/examples/streaming/` - Bidirectional conversation
  - `cmd/examples/hooks/` - Hook registration and execution
  - `cmd/examples/permissions/` - Permission callback usage
  - `cmd/examples/mcp/calculator/` - SDK MCP server
  - `cmd/examples/mcp/client/` - External MCP client

### 2. Clean, Idiomatic Go API

**Criterion:** API follows Go conventions and best practices

**Verification Methods:**
- ✅ `go vet` passes with zero warnings
- ✅ `golangci-lint run` passes with all strictness linters enabled
- ✅ API uses standard Go patterns:
  - Context-aware APIs (all blocking operations accept `context.Context`)
  - Channels for streaming (`<-chan messages.Message`, `<-chan error`)
  - Error handling via explicit returns (no exceptions)
  - Option structs instead of long parameter lists
  - Discriminated unions via interfaces

- ✅ Code review checklist:
  - Public APIs are documented with godoc comments
  - Examples are runnable and included in docs
  - Naming follows Go conventions (PascalCase for exports, camelCase for private)
  - No naked returns in functions
  - No global mutable state

### 3. Efficient Resource Usage

**Criterion:** SDK uses resources efficiently without leaks

**Verification Methods:**
- ✅ Memory leak detection:
  - `go test -race ./...` passes with zero data races
  - Goroutine leak detector shows zero leaked goroutines after tests
  - Benchmarks show constant memory usage over time
  - `defer Close()` used correctly for resource cleanup

- ✅ Performance benchmarks with instrumentation:
  - Control protocol round-trip <100ms for local subprocess
    - Instrumentation: `go test -bench=BenchmarkControlProtocolRoundTrip -benchtime=10s`
    - Task: `internal/protocol/protocol_bench_test.go` measuring SendRequest→ReceiveResponse latency
    - Target: p50 <50ms, p95 <100ms, p99 <200ms
  - Message parsing throughput >10,000 messages/sec
    - Instrumentation: `go test -bench=BenchmarkMessageParsing -benchtime=100000x`
    - Task: `internal/messages/parser_bench_test.go` parsing realistic message batches
    - Target: >10,000 ops/sec sustained over 100k iterations
  - Subprocess cleanup completes within 5 seconds
    - Instrumentation: `go test -run=TestTransportCleanup -timeout=10s`
    - Task: `internal/transport/subprocess_test.go` measuring Close() to process termination
    - Target: cleanup completes <5s even under load
  - No blocking operations in hot paths
    - Instrumentation: `go test -race -run=TestStreamingFlow`
    - Task: Verify all channel operations have timeout/context cancellation

- ✅ Resource cleanup validation:
  - All examples properly close client connections
  - Transport Close() terminates subprocess
  - MCP adapters close sessions correctly
  - Context cancellation propagates and cleans up

### 4. Comprehensive Documentation

**Criterion:** Complete documentation for all public APIs and usage patterns

**Verification Methods:**
- ✅ godoc coverage:
  - Every exported type has godoc comment
  - Every exported function has godoc comment with example
  - Package-level documentation in doc.go files
  - Documentation renders correctly on pkg.go.dev

- ✅ README.md completeness:
  - Quick start guide (<5 minutes to first query)
  - Feature overview with code snippets
  - Installation instructions
  - Link to full documentation
  - Link to examples directory

- ✅ Usage guides (if needed):
  - How to use hooks
  - How to implement permission callbacks
  - How to create SDK MCP servers
  - How to connect to external MCP servers
  - Migration guide from Python SDK

### 5. Test Coverage >80%

**Criterion:** Minimum 80% test coverage across all packages

**Verification Methods:**
- ✅ Coverage measurement:
  - `go test -cover ./...` shows ≥80% overall coverage
  - Critical paths have ≥90% coverage:
    - Domain services (querying, streaming, hooks, permissions)
    - Protocol adapter (control request routing)
    - Message parser (all message types)
    - MCP adapters (client and SDK server)

- ✅ Test quality validation:
  - Table-driven tests for comprehensive case coverage
  - Unit tests use mock ports (no subprocess/network I/O)
  - Integration tests validate end-to-end flows
  - Edge cases tested (timeouts, cancellation, errors)

- ✅ CI enforcement:
  - GitHub Actions workflow fails if coverage drops below 80%
  - Coverage reports uploaded to codecov.io
  - Pull requests show coverage diff

### 6. Automated CI/CD

**Criterion:** Full CI/CD pipeline with automated quality checks

**Verification Methods:**
- ✅ GitHub Actions workflows exist and run:
  - `.github/workflows/lint.yml` - Runs golangci-lint on every push/PR
  - `.github/workflows/test.yml` - Runs tests with coverage on every push/PR
  - `.github/workflows/release.yml` - Creates GitHub releases on version tags

- ✅ CI enforcement:
  - Branch protection requires status checks to pass
  - Linting failures block merges
  - Test failures block merges
  - Coverage regressions block merges

- ✅ Release automation:
  - `scripts/release.sh` automates tagging and releasing
  - GitHub releases created automatically from tags
  - CHANGELOG.md updated with each release
  - Module published and accessible via `go get`

### 7. Ease of Use

**Criterion:** SDK is easy to install, learn, and use

**Verification Methods:**
- ✅ Installation simplicity:
  - Single `go get` command installs SDK
  - No external dependencies beyond Go stdlib + MCP SDK
  - Works on Linux, macOS, Windows (Go cross-compilation)

- ✅ Learning curve validation:
  - First query example runs in <5 minutes
  - Examples cover all major use cases
  - Error messages are clear and actionable
  - API surface is minimal and focused

- ✅ User feedback (post-release):
  - GitHub issues labeled "documentation" or "usability" tracked
  - API pain points identified and addressed in patch releases
  - Community contributions welcomed and merged

---

## Verification Timeline

**Phase 6 (Testing & Documentation):**
- Run all verification methods
- Fix gaps identified by verification
- Achieve 100% pass rate before proceeding to Phase 7

**Phase 7 (Publishing):**
- Final verification sweep before v0.1.0 release
- Automated CI enforces criteria on every PR

**Post-Release:**
- Monitor criteria compliance in CI
- Address regressions immediately
- Track user feedback for ease-of-use improvements
