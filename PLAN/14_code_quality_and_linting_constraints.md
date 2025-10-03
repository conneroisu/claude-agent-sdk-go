# Code Quality & Linting Constraints

## Overview

This SDK enforces **exceptionally strict** code quality standards via golangci-lint with 30+ linters enabled and comprehensive revive rules. These constraints are not suggestions - they are **enforced at CI/CD level** and will block merges on violations.

**Key Philosophy**: Code quality constraints drive architectural decisions. The 175-line file limit and 25-line function limit fundamentally shape how we structure the codebase.

The golangci-lint configuration already exists and is maintained in this repository.

---

## Critical Linting Rules

### Size Constraints

| Constraint | Limit | Impact |
|------------|-------|--------|
| **File Length** | 175 lines (excl. comments/blanks) | Forces file decomposition |
| **Function Length** | 25 lines | Requires aggressive function extraction |
| **Line Length** | 80 characters | Affects API design, requires line breaks |

### Complexity Constraints

| Constraint | Limit | Impact |
|------------|-------|--------|
| **Cyclomatic Complexity** | 15 | Limits branching logic per function |
| **Cognitive Complexity** | 20 | Limits overall function complexity |
| **Max Control Nesting** | 3 levels | Prevents deeply nested if/for/switch |

### Structural Constraints

| Constraint | Limit | Impact |
|------------|-------|--------|
| **Function Parameters** | 4 max | Requires option structs or builders |
| **Function Return Values** | 3 max | Affects error handling patterns |
| **Public Structs per Package** | 19 max | May require sub-packages |

### Code Quality Requirements

| Requirement | Target | Impact |
|-------------|--------|--------|
| **Comments Density** | 15% minimum | Every file needs substantial documentation |
| **Test Coverage** | 80%+ | Comprehensive test suites required |

---

## Enabled Linters

### Core Linters (Always Running)

- **asasalint** - Prevent passing []any to append
- **bidichk** - Dangerous Unicode detection
- **bodyclose** - HTTP response body close checking
- **copyloopvar** - Loop variable capture issues
- **errname** - Error naming conventions
- **exhaustive** - Exhaustive enum switch checking
- **gocritic** - Comprehensive code checker
- **godot** - Comment punctuation
- **gocheckcompilerdirectives** - Compiler directive validation
- **govet** - Official Go vet checks
- **intrange** - Integer range loop checking
- **makezero** - Slice initialization checking
- **misspell** - Spelling errors
- **nlreturn** - Newline before return
- **revive** - **90+ rules enabled** (see below)
- **staticcheck** - **100+ rules enabled** (see below)
- **unconvert** - Unnecessary conversions
- **usestdlibvars** - Use stdlib constants
- **wastedassign** - Wasted assignments

### Revive Rules (Highlights)

**Critical Rules:**
- `argument-limit: 4` - Max 4 function parameters
- `function-length: [25, 0]` - Max 25 lines per function
- `file-length-limit: {max: 175}` - Max 175 lines per file
- `line-length-limit: 80` - Max 80 chars per line
- `cognitive-complexity: 20` - Max cognitive complexity
- `cyclomatic: 15` - Max cyclomatic complexity
- `max-control-nesting: 3` - Max 3 nesting levels
- `max-public-structs: 19` - Max 19 public structs per package
- `comments-density: 15` - Min 15% comments

**Naming & Style:**
- `var-naming` - Variable naming conventions (ID allowed)
- `receiver-naming` - Max 2 char receiver names
- `error-naming` - Errors must start with Err/err
- `error-strings` - Error strings lowercase, no punctuation
- `exported` - All exported items need documentation

**Code Quality:**
- `early-return` - Prefer early returns
- `indent-error-flow` - Error flow indentation
- `superfluous-else` - Eliminate unnecessary else
- `if-return` - Simplify if-return patterns
- `empty-block` - No empty blocks
- `unused-parameter` - Flag unused params (allow `_` prefix)

---

## Architectural Implications

### 1. File Decomposition Strategy

**Problem**: Many planned files exceed 175 lines.

**Solution**: Logical file splitting pattern

#### Pattern A: Split by Responsibility

```
# Instead of single messages.go (500+ lines):
messages/
├── messages.go          # Core interfaces (~50 lines)
├── user.go             # UserMessage (~40 lines)
├── assistant.go        # AssistantMessage (~60 lines)
├── system.go           # SystemMessage variants (~80 lines)
├── result.go           # ResultMessage variants (~90 lines)
├── stream.go           # StreamEvent (~30 lines)
├── content.go          # ContentBlock types (~70 lines)
└── usage.go            # Usage statistics (~40 lines)
```

#### Pattern B: Split by Domain Concept

```
# Instead of single service.go (400+ lines):
querying/
├── service.go          # Service struct + core methods (~60 lines)
├── execute.go          # Execute implementation (~80 lines)
├── routing.go          # Message routing logic (~70 lines)
├── errors.go           # Error handling helpers (~50 lines)
└── options.go          # Option processing (~40 lines)
```

### 2. Function Extraction Patterns

**Problem**: 25-line function limit for complex operations.

**Solutions**:

#### Pattern A: Extract Validation
```go
// BAD: 40+ line function
func (s *Service) Execute(ctx context.Context,
    prompt string, opts *options.AgentOptions) error {
    // 10 lines of validation
    // 15 lines of initialization
    // 15 lines of execution
}

// GOOD: Extracted helpers
func (s *Service) Execute(ctx context.Context,
    prompt string, opts *options.AgentOptions) error {
    if err := s.validateInput(prompt, opts); err != nil {
        return err
    }
    state, err := s.initializeState(ctx, opts)
    if err != nil {
        return err
    }
    return s.executeWithState(ctx, prompt, state)
}

func (s *Service) validateInput(prompt string,
    opts *options.AgentOptions) error {
    // 10 lines
}

func (s *Service) initializeState(ctx context.Context,
    opts *options.AgentOptions) (*executionState, error) {
    // 15 lines
}

func (s *Service) executeWithState(ctx context.Context,
    prompt string, state *executionState) error {
    // 15 lines
}
```

#### Pattern B: Extract Complex Logic
```go
// BAD: 35-line parsing function
func parseMessage(raw map[string]any) (Message, error) {
    // Type checking: 5 lines
    // Field extraction: 10 lines
    // Content parsing: 15 lines
    // Validation: 5 lines
}

// GOOD: Extracted sub-parsers
func parseMessage(raw map[string]any) (Message, error) {
    msgType, err := extractMessageType(raw)
    if err != nil {
        return nil, err
    }

    switch msgType {
    case "assistant":
        return parseAssistantMessage(raw)
    case "system":
        return parseSystemMessage(raw)
    default:
        return nil, fmt.Errorf("unknown type: %s", msgType)
    }
}

func extractMessageType(raw map[string]any) (string, error) {
    // 8 lines
}

func parseAssistantMessage(raw map[string]any) (Message, error) {
    // 18 lines
}

func parseSystemMessage(raw map[string]any) (Message, error) {
    // 20 lines
}
```

### 3. Parameter Reduction Patterns

**Problem**: 4-parameter limit for functions.

**Solutions**:

#### Pattern A: Config Structs
```go
// BAD: 6 parameters
func NewService(
    transport Transport,
    protocol Protocol,
    parser Parser,
    logger Logger,
    timeout time.Duration,
    maxRetries int,
) *Service

// GOOD: Config struct (3 parameters)
type ServiceConfig struct {
    Timeout    time.Duration
    MaxRetries int
    Logger     Logger
}

func NewService(
    transport Transport,
    protocol Protocol,
    cfg ServiceConfig,
) *Service
```

#### Pattern B: Option Functions
```go
// GOOD: Variadic options (1 parameter + options)
type ServiceOption func(*Service)

func WithTimeout(d time.Duration) ServiceOption {
    return func(s *Service) { s.timeout = d }
}

func WithLogger(l Logger) ServiceOption {
    return func(s *Service) { s.logger = l }
}

func NewService(
    transport Transport,
    opts ...ServiceOption,
) *Service
```

### 4. Return Value Optimization

**Problem**: 3-return value limit.

**Solutions**:

#### Pattern A: Result Structs
```go
// BAD: 4 return values
func Execute(ctx context.Context) (
    *Response,
    *Metadata,
    int,
    error,
)

// GOOD: Result struct (2 return values)
type ExecutionResult struct {
    Response *Response
    Metadata *Metadata
    Count    int
}

func Execute(ctx context.Context) (*ExecutionResult, error)
```

#### Pattern B: Error Wrapping
```go
// BAD: Multiple error types as separate returns
func Process(data []byte) (Result, ValidationError, IOError)

// GOOD: Wrapped errors (2 return values)
func Process(data []byte) (Result, error) {
    // Return wrapped errors with context
    if err := validate(data); err != nil {
        return Result{}, fmt.Errorf("validation: %w", err)
    }
    // ...
}
```

### 5. Nesting Reduction Patterns

**Problem**: 3-level max control nesting.

**Solutions**:

#### Pattern A: Early Returns
```go
// BAD: Deep nesting (4 levels)
func Process(data []byte) error {
    if len(data) > 0 {
        if valid := validate(data); valid {
            if result, err := parse(data); err == nil {
                if err := save(result); err != nil {
                    return err
                }
            }
        }
    }
    return nil
}

// GOOD: Early returns (1-2 levels max)
func Process(data []byte) error {
    if len(data) == 0 {
        return nil
    }
    if !validate(data) {
        return ErrInvalidData
    }
    result, err := parse(data)
    if err != nil {
        return fmt.Errorf("parse: %w", err)
    }
    return save(result)
}
```

#### Pattern B: Extract Nested Logic
```go
// BAD: Nested loops and conditions (4 levels)
func ProcessAll(items []Item) error {
    for _, item := range items {
        if item.Valid {
            for _, child := range item.Children {
                if child.Active {
                    // process
                }
            }
        }
    }
}

// GOOD: Extracted functions (2 levels max)
func ProcessAll(items []Item) error {
    for _, item := range items {
        if err := processItem(item); err != nil {
            return err
        }
    }
    return nil
}

func processItem(item Item) error {
    if !item.Valid {
        return nil
    }
    return processChildren(item.Children)
}

func processChildren(children []Child) error {
    for _, child := range children {
        if child.Active {
            // process
        }
    }
    return nil
}
```

---

## Implementation Patterns for Compliance

### Pattern 1: Table-Driven Tests (File Size Management)

**Problem**: Test files easily exceed 175 lines.

**Solution**: Table-driven tests with shared fixtures

```go
// tests/fixtures.go (~60 lines)
package tests

var MessageFixtures = map[string]map[string]any{
    "assistant_simple": { /* ... */ },
    "system_init": { /* ... */ },
    // ... 10-15 fixtures
}

// parse_test.go (~90 lines - under limit!)
package parse_test

func TestParseMessage(t *testing.T) {
    tests := []struct {
        name     string
        fixture  string
        wantType string
        wantErr  bool
    }{
        {"assistant", "assistant_simple", "AssistantMessage", false},
        {"system", "system_init", "SystemMessage", false},
        // ... 20 test cases in compact form
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            input := tests.MessageFixtures[tt.fixture]
            got, err := parse.ParseMessage(input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if !tt.wantErr && got.Type() != tt.wantType {
                t.Errorf("type = %v, want %v", got.Type(), tt.wantType)
            }
        })
    }
}
```

### Pattern 2: Builder for Complex Initialization

**Problem**: Complex struct initialization exceeds function/nesting limits.

**Solution**: Fluent builder pattern

```go
// builder.go (~70 lines)
type ClientBuilder struct {
    transport Transport
    protocol  Protocol
    opts      []Option
}

func NewClientBuilder() *ClientBuilder {
    return &ClientBuilder{}
}

func (b *ClientBuilder) WithTransport(t Transport) *ClientBuilder {
    b.transport = t
    return b
}

func (b *ClientBuilder) WithProtocol(p Protocol) *ClientBuilder {
    b.protocol = p
    return b
}

func (b *ClientBuilder) WithOptions(opts ...Option) *ClientBuilder {
    b.opts = append(b.opts, opts...)
    return b
}

func (b *ClientBuilder) Build() (*Client, error) {
    if err := b.validate(); err != nil {
        return nil, err
    }
    return b.construct()
}

func (b *ClientBuilder) validate() error {
    // 10 lines validation
}

func (b *ClientBuilder) construct() (*Client, error) {
    // 15 lines construction
}
```

### Pattern 3: File Organization Template

**Every file follows this structure to maximize clarity and stay under limits:**

```go
// package_file.go (~150 lines typical)

// Package comment (5-10 lines documentation)
package packagename

// Imports (10-20 lines, grouped: stdlib, external, internal)
import (
    "context"
    "fmt"

    "github.com/external/lib"

    "github.com/user/repo/internal"
)

// Constants and vars (10-20 lines)
const (
    DefaultTimeout = 30 * time.Second
)

// Type definitions (30-50 lines)
// Each type with godoc comment
type Service struct {
    // ...
}

// Constructor (15-20 lines)
func NewService(...) *Service {
    // ...
}

// Methods (20 lines each, 3-5 methods max per file)
func (s *Service) Method1() {
    // ...
}

// Helper functions (15 lines each, 2-3 max per file)
func helperFunc() {
    // ...
}

// File must have 15%+ comments = ~25 comment lines minimum
```

---

## Phase-by-Phase Compliance Checklists

### Phase 1: Core Domain & Ports

**File Size Estimates & Decomposition Plan:**

- ❌ `messages/messages.go` - **Planned: 500 lines** → Split into 8 files
  - ✅ `messages.go` - Interfaces only (50 lines)
  - ✅ `user.go` - UserMessage (40 lines)
  - ✅ `assistant.go` - AssistantMessage (60 lines)
  - ✅ `system.go` - SystemMessage types (80 lines)
  - ✅ `result.go` - ResultMessage types (90 lines)
  - ✅ `stream.go` - StreamEvent (30 lines)
  - ✅ `content.go` - ContentBlock types (70 lines)
  - ✅ `usage.go` - Usage stats (40 lines)

- ✅ `options/domain.go` - **OK: 80 lines**
- ✅ `options/transport.go` - **OK: 90 lines**
- ✅ `options/mcp.go` - **OK: 70 lines**
- ✅ `ports/transport.go` - **OK: 40 lines**
- ✅ `ports/protocol.go` - **OK: 60 lines**
- ✅ `ports/parser.go` - **OK: 25 lines**
- ✅ `ports/mcp.go` - **OK: 30 lines**

**Complexity Hotspots:**
- Message parsing logic → **Extract sub-parsers**
- Content block type switching → **Extract per-type parsers**

**Checklist:**
- [ ] All files under 175 lines
- [ ] All functions under 25 lines
- [ ] All lines under 80 chars
- [ ] 15%+ comments per file
- [ ] Max 4 params per function
- [ ] Max 3 returns per function

### Phase 2: Domain Services

**File Size Estimates & Decomposition Plan:**

- ❌ `querying/service.go` - **Planned: 300 lines** → Split into 5 files
  - ✅ `service.go` - Service struct + New (60 lines)
  - ✅ `execute.go` - Execute method (80 lines)
  - ✅ `routing.go` - Message routing (70 lines)
  - ✅ `errors.go` - Error handling (50 lines)
  - ✅ `state.go` - Execution state (40 lines)

- ❌ `streaming/service.go` - **Planned: 350 lines** → Split into 6 files
  - ✅ `service.go` - Service struct + New (50 lines)
  - ✅ `connect.go` - Connection logic (70 lines)
  - ✅ `send.go` - SendMessage (60 lines)
  - ✅ `receive.go` - ReceiveMessages (80 lines)
  - ✅ `lifecycle.go` - Lifecycle methods (50 lines)
  - ✅ `state.go` - State management (40 lines)

**Complexity Hotspots:**
- Message routing switch statements → **Extract handler map**
- Control protocol handling → **Extract per-subtype handlers**
- Hook execution → **Extract hook executor helper**

**Checklist:**
- [ ] Cyclomatic complexity ≤ 15
- [ ] Cognitive complexity ≤ 20
- [ ] Max nesting depth ≤ 3
- [ ] Use early returns
- [ ] Extract validation functions

### Phase 3: Adapters (Infrastructure)

**File Size Estimates & Decomposition Plan:**

- ❌ `adapters/cli/transport.go` - **Planned: 400 lines** → Split into 7 files
  - ✅ `transport.go` - Adapter struct (60 lines)
  - ✅ `connect.go` - Connection logic (70 lines)
  - ✅ `command.go` - Command building (80 lines)
  - ✅ `io.go` - I/O handling (90 lines)
  - ✅ `discovery.go` - CLI discovery (50 lines)
  - ✅ `process.go` - Process management (60 lines)
  - ✅ `errors.go` - Error types (40 lines)

- ❌ `adapters/jsonrpc/protocol.go` - **Planned: 350 lines** → Split into 5 files
  - ✅ `protocol.go` - Handler struct (50 lines)
  - ✅ `control.go` - Control requests (80 lines)
  - ✅ `routing.go` - Message routing (90 lines)
  - ✅ `handlers.go` - Request handlers (80 lines)
  - ✅ `state.go` - State tracking (50 lines)

**Complexity Hotspots:**
- CLI argument building → **Use builder pattern**
- Process I/O handling → **Extract reader/writer helpers**
- JSON-RPC routing → **Use handler registry**

**Checklist:**
- [ ] Error handling optimized for line count
- [ ] I/O operations extracted to helpers
- [ ] Process management in separate file

### Phase 4: Public API

**File Size Estimates:**

- ✅ `client.go` - **Planned: 120 lines** → OK (extract if needed)
- ✅ `query.go` - **Planned: 80 lines** → OK
- ✅ `errors.go` - **Planned: 60 lines** → OK

**Checklist:**
- [ ] Public API surface documented
- [ ] Example code in godoc
- [ ] Builder pattern for complex init

### Phase 5: Advanced Features

**File Size Estimates & Decomposition Plan:**

- ❌ `hooking/service.go` - **Planned: 250 lines** → Split into 4 files
  - ✅ `service.go` - Service struct (50 lines)
  - ✅ `execute.go` - Execution logic (80 lines)
  - ✅ `registry.go` - Hook registry (60 lines)
  - ✅ `types.go` - Hook types (60 lines)

**Checklist:**
- [ ] Hook execution under 25 lines
- [ ] Type switching extracted
- [ ] Callback handling simplified

### Phase 6: Testing

**File Size Strategy:**

- Use **table-driven tests** extensively
- Create **shared fixtures** file per package
- Keep **test helpers** in separate files
- Aim for **~100 lines per test file**

**Test File Template:**
```go
// service_test.go (~100 lines)
// - Table test: 40 lines
// - Helper setup: 30 lines
// - Mock setup: 30 lines
```

**Checklist:**
- [ ] fixtures.go for shared test data
- [ ] helpers.go for test utilities
- [ ] Each test file under 175 lines
- [ ] Table-driven where possible

---

## Enforcement & CI Integration

### Pre-Commit Hooks

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Run linter on staged files
golangci-lint run --new-from-rev=HEAD~1

if [ $? -ne 0 ]; then
    echo "❌ Linting failed. Fix issues before committing."
    exit 1
fi
```

### CI Pipeline (GitHub Actions Example)

```yaml
# .github/workflows/lint.yml
name: Lint
on: [push, pull_request]

jobs:
  golangci:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
          args: --timeout=5m
```

### Local Development Workflow

```bash
# Before committing
golangci-lint run

# Auto-fix some issues
golangci-lint run --fix

# Check specific file
golangci-lint run path/to/file.go

# Generate report
golangci-lint run --out-format=html > lint-report.html
```

---

## Quick Reference: Common Violations & Fixes

| Violation | Fix |
|-----------|-----|
| File too long (>175 lines) | Split by responsibility or domain |
| Function too long (>25 lines) | Extract helpers/validators |
| Too many params (>4) | Use config struct or options |
| Too many returns (>3) | Use result struct |
| Nesting too deep (>3) | Early returns, extract functions |
| Cyclomatic too high (>15) | Split complex logic |
| Missing comments (<15%) | Add godoc for all exports |
| Line too long (>80 chars) | Break into multiple lines |

---

## Success Criteria

✅ **Zero linting violations** in CI
✅ **All files** under 175 lines
✅ **All functions** under 25 lines
✅ **All packages** have ≥15% comments
✅ **Test coverage** ≥80%
✅ **No complexity violations**

The linting rules are **non-negotiable** - they enforce the quality bar for this SDK.
