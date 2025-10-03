# Contributing to Claude Agent SDK for Go

Thank you for your interest in contributing! This guide will help you get started with development.

## Development Setup

### Prerequisites

- Go 1.22 or later
- golangci-lint (for code quality)
- Claude CLI (for integration tests)

### Getting Started

1. Clone the repository:
```bash
git clone https://github.com/conneroisu/claude-agent-sdk-go.git
cd claude-agent-sdk-go
```

2. Install dependencies:
```bash
go mod download
```

3. Install development tools:
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install gofumpt for formatting
go install mvdan.cc/gofumpt@latest
```

## Development Workflow

### Running Tests

```bash
# Run all unit tests
go test -v -race ./...

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run integration tests (requires Claude CLI and API key)
go test -v -tags=integration ./tests/integration/...
```

### Code Quality

```bash
# Format code
go fmt ./...
gofumpt -l -w .

# Run linter
golangci-lint run --timeout=5m

# Run go vet
go vet ./...
```

### Building Examples

```bash
# Build all examples
go build -o bin/quickstart ./cmd/examples/quickstart
go build -o bin/streaming ./cmd/examples/streaming
go build -o bin/hooks ./cmd/examples/hooks
go build -o bin/tools ./cmd/examples/tools

# Run examples
./bin/quickstart
./bin/streaming
```

## Code Style & Quality Standards

### Architecture Constraints

This project follows hexagonal architecture (ports and adapters):

- **Domain layer** (`pkg/claude/messages`, `pkg/claude/domain/ports`): Core business logic, no external dependencies
- **Service layer** (`pkg/claude/permissions`, `pkg/claude/hooking`): Domain services
- **Adapter layer** (`pkg/claude/adapters/*`): Infrastructure implementations
- **Public API** (`pkg/claude/*.go`): Simple facade over domain

### Code Quality Rules

**File Structure:**
- Maximum 175 lines per file
- Maximum 25 lines per function
- Maximum 80 characters per line

**Go Best Practices:**
- Use interfaces for all external dependencies
- Prefer composition over inheritance
- Write table-driven tests
- Document all exported functions and types
- Use meaningful variable names
- Avoid global state

**Linting:**
All code must pass `golangci-lint` with our strict configuration:
- gofmt, gofumpt, goimports
- govet, staticcheck, gosimple
- errcheck, ineffassign, misspell
- godot, godox, gocognit
- funlen, lll, gocyclo

### Testing Requirements

**Unit Tests:**
- Test all public functions
- Use table-driven tests
- Mock all external dependencies
- Aim for >80% coverage

**Integration Tests:**
- Test real Claude CLI integration
- Use build tag `//go:build integration`
- Document any required setup

**Test Structure:**
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   Input
        want    Output
        wantErr bool
    }{
        // test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Pull Request Process

### Before Submitting

1. Run all quality checks:
```bash
go fmt ./...
gofumpt -l -w .
golangci-lint run --timeout=5m
go test -v -race ./...
```

2. Update documentation if needed:
   - Update `docs/README.md` for API changes
   - Update `docs/CHANGELOG.md` with your changes
   - Add godoc comments for new exports

3. Add tests for new functionality

### PR Guidelines

1. **Title:** Use conventional commits format
   - `feat: add new feature`
   - `fix: resolve bug`
   - `docs: update documentation`
   - `test: add tests`
   - `refactor: restructure code`

2. **Description:**
   - Explain what changed and why
   - Reference any related issues
   - Include examples if adding features

3. **Commits:**
   - Keep commits focused and atomic
   - Write clear commit messages
   - Squash WIP commits before merging

4. **CI Checks:**
   All PRs must pass:
   - Linting (golangci-lint)
   - Unit tests (all Go versions)
   - Integration tests (if applicable)

## Release Process

### Version Numbering

We follow semantic versioning (SemVer):
- **Major (x.0.0):** Breaking API changes
- **Minor (0.x.0):** New features, backwards compatible
- **Patch (0.0.x):** Bug fixes, backwards compatible

### Creating a Release

1. Update `docs/CHANGELOG.md`:
```markdown
## [0.2.0] - 2025-01-15

### Added
- New feature description

### Changed
- Modified behavior description

### Fixed
- Bug fix description
```

2. Create and push a version tag:
```bash
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

3. GitHub Actions will automatically:
   - Run all tests
   - Extract release notes from CHANGELOG.md
   - Create a GitHub release

## Getting Help

- **Documentation:** See `docs/README.md`
- **Examples:** Check `cmd/examples/`
- **Issues:** Open an issue for bugs or feature requests
- **Discussions:** Use GitHub Discussions for questions

## Code of Conduct

- Be respectful and inclusive
- Provide constructive feedback
- Focus on the code, not the person
- Help newcomers learn and grow

Thank you for contributing! 🎉
