# Contributing to Claude Agent SDK for Go

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Claude CLI installed (`npm install -g @anthropic-ai/claude-code`)
- golangci-lint for linting
- Nix (optional, for formatting)

### Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/claude-agent-sdk-go
   cd claude-agent-sdk-go
   ```

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Run tests:
   ```bash
   go test ./pkg/claude/...
   ```

## Code Quality Standards

This project enforces **strict code quality constraints**:

### File Constraints

- **Maximum 175 lines per file**
  - Split large files into focused modules
  - One responsibility per file

- **Maximum 25 lines per function**
  - Extract helper functions
  - Use table-driven tests
  - Keep functions focused

- **Maximum 80 characters per line**
  - Break long lines at logical points
  - Use intermediate variables

### Complexity Constraints

- **Maximum cognitive complexity: 20**
  - Reduce nesting
  - Extract conditions to functions
  - Use early returns

- **Maximum nesting depth: 3**
  - Flatten nested conditions
  - Extract nested logic to functions
  - Use guard clauses

### Style Requirements

- **Minimum 15% comment density**
  - Add godoc comments to all exported types
  - Document complex logic
  - Explain non-obvious decisions

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes

Follow the hexagonal architecture pattern:

- **Domain changes**: Modify `pkg/claude/messages/`, `options/`, `ports/`
- **Service changes**: Modify service packages (`querying/`, `streaming/`, etc.)
- **Adapter changes**: Modify adapter packages (`adapters/cli/`, etc.)
- **Public API**: Modify `pkg/claude/client.go` or related files

### 3. Write Tests

All new code **must** have tests:

```go
// Example test structure
func TestYourFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   any
        want    any
        wantErr bool
    }{
        // Test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 4. Run Quality Checks

Before committing, ensure all checks pass:

```bash
# Format code
nix fmt
# OR
gofmt -w .

# Run linter
golangci-lint run ./...

# Run tests
go test ./pkg/claude/...

# Run tests with coverage
go test -cover ./pkg/claude/...

# Build examples
go build ./cmd/examples/...
```

### 5. Commit Changes

Use conventional commit messages:

```
feat: add streaming support for tool results
fix: correct message parsing for thinking blocks
docs: update architecture documentation
test: add tests for permission service
refactor: extract command building to separate file
```

### 6. Push and Create PR

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub.

## Testing Guidelines

### Unit Tests

- **Use mocks** from `internal/testutil/mocks.go`
- **Use fixtures** from `internal/testutil/fixtures.go`
- **Table-driven tests** for multiple cases
- **Test both success and error paths**

Example:
```go
func TestServiceMethod(t *testing.T) {
    transport := &testutil.MockTransport{
        ConnectFunc: func(_ context.Context) error {
            return nil
        },
    }

    // Test implementation
}
```

### Integration Tests

- Tag with `//go:build integration`
- Require Claude CLI installed
- Test with real subprocess
- Clean up resources

### Test Coverage Goals

- **Aim for 80%+ coverage**
- **100% coverage for critical paths**
- **All error paths tested**

## Code Review Process

1. **Automated checks must pass**:
   - Build succeeds
   - All tests pass
   - Linting clean
   - Code formatted

2. **Manual review** will check:
   - Architecture compliance
   - Code clarity
   - Test quality
   - Documentation

3. **Changes requested**:
   - Address feedback
   - Push updates
   - Re-request review

## Common Patterns

### Discriminated Unions

Use marker methods for type-safe unions:

```go
type Message interface {
    message()  // Marker method
}

type UserMessage struct {
    Content string
}

func (UserMessage) message() {}  // Implements marker
```

### Config Structs

Use Config structs to avoid parameter explosion:

```go
type ServiceConfig struct {
    Transport   ports.Transport
    Protocol    ports.ProtocolHandler
    Parser      ports.MessageParser
    // ... more dependencies
}

func NewService(cfg *ServiceConfig) *Service {
    return &Service{
        transport: cfg.Transport,
        // ...
    }
}
```

### Channel-based Communication

Use channels for async operations:

```go
func (s *Service) Execute(ctx context.Context) (<-chan Message, <-chan error) {
    msgCh := make(chan Message)
    errCh := make(chan error, 1)

    go func() {
        defer close(msgCh)
        defer close(errCh)
        // Implementation
    }()

    return msgCh, errCh
}
```

## Documentation

### Godoc Comments

All exported types must have godoc comments:

```go
// Service handles query execution.
// It coordinates the transport, protocol, and parsing layers.
type Service struct {
    // ...
}

// Execute runs a one-shot query against Claude.
// It returns channels for messages and errors.
func (s *Service) Execute(ctx context.Context, prompt string) (<-chan Message, <-chan error) {
    // ...
}
```

### Examples

Add examples to `cmd/examples/` for new features.

### Architecture Docs

Update `docs/ARCHITECTURE.md` for significant changes.

## Getting Help

- **GitHub Issues**: Report bugs or request features
- **Discussions**: Ask questions or propose ideas
- **Code Review**: Request feedback on approach

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
