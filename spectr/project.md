# Project Context

## Purpose
Official Go SDK for Claude Agent, providing programmatic access to Claude's agent capabilities. This SDK enables Go developers to interact with Claude Code CLI programmatically, supporting features like:
- Real-time message streaming
- MCP (Model Context Protocol) server integration
- Custom SDK tool definitions
- Fine-grained permission management
- Full context.Context integration following Go best practices

## Tech Stack
- **Go**: 1.23+ (using latest language features)
- **Primary Dependencies**:
  - `github.com/google/uuid` v1.6.0 - UUID generation for session/request tracking
- **Development Tools**:
  - `golangci-lint` - Comprehensive linting with multiple enabled rules
  - Go's native testing framework
  - Nix/direnv for reproducible development environments
- **External Requirements**:
  - Claude Code CLI (must be installed and in PATH)
  - ANTHROPIC_API_KEY environment variable (optional)

## Project Conventions

### Code Style
- **Idiomatic Go**: Follow standard Go conventions and best practices
- **Comprehensive Linting**: Using `.golangci.yaml` with extensive rules:
  - `revive` with all rules enabled (severity: error)
  - `staticcheck`, `govet`, `gocritic` for code quality
  - `godot` for documentation formatting
  - `nlreturn` for return statement formatting
  - `errname`, `bodyclose`, `copyloopvar` for common mistakes
  - Examples and test directories have relaxed linting
- **Documentation**:
  - All exported types, functions, and methods must have godoc comments
  - Include design notes for complex patterns (especially TypeScript SDK differences)
  - Use code examples in documentation where helpful
- **Error Handling**: Dedicated `pkg/clauderrs` package for typed errors
- **Naming Conventions**:
  - Constants in camelCase (e.g., `msgChanBufferSize`, `messageTypeUser`)
  - Exported types/functions in PascalCase
  - Package names are lowercase, single-word when possible

### Architecture Patterns
- **Package Structure**:
  - `pkg/claude/` - Public SDK API and types
  - `pkg/clauderrs/` - Error types and handling
  - `internal/transport/` - Communication layer with Claude CLI
  - `examples/` - Complete working examples (basic, streaming, mcp, permissions, etc.)
  - `test/unit/` - Unit tests
  - `test/integration/` - Integration tests
  - `test/e2e/` - End-to-end tests

- **Query Architecture** (Recent Refactoring):
  - Split monolithic `query.go` into focused files:
    - `query.go` - Interface definitions
    - `query_control.go` - Control protocol implementation
    - `query_commands.go` - Command handling
    - `query_messages.go` - Message processing
    - `query_models.go` - Model management
    - `query_operations.go` - Core operations
    - `query_info.go` - Query metadata
    - `query_control_handlers.go` - Control request handlers

- **Communication Patterns**:
  - Channel-based message streaming
  - Context support throughout (context.Context)
  - Buffered channels for message and control requests
  - Mutex-protected state management
  - Graceful shutdown and cleanup

- **TypeScript SDK Parity**:
  - Match TypeScript SDK functionality while maintaining Go idioms
  - Document differences in design patterns (e.g., AsyncGenerator vs Next() method)

### Testing Strategy
- **Test Types**:
  - **Unit Tests**: `go test ./...` - Fast, isolated tests
  - **Integration Tests**: `go test -tags=integration ./test/integration/...` - Test component interactions
  - **E2E Tests**: `go test -tags=e2e ./test/e2e/...` - Require ANTHROPIC_API_KEY, test full workflows

- **Coverage**: Run with `go test -cover ./...`

- **Test Requirements**:
  - All tests must verify they actually work (per CLAUDE.md)
  - Test happy paths, failure paths, and edge cases
  - Use table-driven tests where appropriate

- **Example Verification**:
  - `examples/basic/` is marked as **working** and fully tested
  - New examples should be thoroughly tested before marking as working

### Git Workflow
- **Main Branch**: `main`
- **Feature Branches**: Descriptive names (e.g., `split-query` for query refactoring)
- **Commit Message Style**:
  - Recent commits use descriptive, enthusiastic style
  - Examples: "ARCHITECTURE: query_control.go - The Control Protocol Nexus Emerges!"
  - Include scope prefixes: ARCHITECTURE, FEATURE, REFACTOR, etc.
- **Pre-commit**: Ensure linting passes before committing
- **Testing**: Run relevant tests before pushing

## Domain Context

### Claude Agent Integration
- This SDK wraps the Claude Code CLI binary
- Communication happens via stdin/stdout JSON messages
- Protocol types:
  - `user` - User queries
  - `control_request` - Control protocol commands
  - `control_response` - Responses to control requests
  - `hook_callback` - Hook event responses

### MCP (Model Context Protocol)
- Support for MCP server integration
- Custom SDK tools can be defined
- Tools can be allowed/disallowed via configuration

### Query Lifecycle
1. Client initialization with options
2. Query submission (user message)
3. Message streaming (assistant responses, tool uses, etc.)
4. Control operations (interrupt, model switching, etc.)
5. Query completion with result metadata
6. Cleanup and resource disposal

### Permission System
- Multiple permission modes supported
- Custom permission prompts via hooks
- Tool-level permission controls

## Important Constraints

### Technical
- **Minimum Go Version**: 1.23 (uses latest language features)
- **External Binary Dependency**: Requires Claude Code CLI in PATH
- **Channel Buffering**: Fixed buffer sizes for message channels (100) and control requests (10)
- **API Key**: Optional ANTHROPIC_API_KEY environment variable
- **Context Cancellation**: All operations must respect context.Context cancellation

### Development
- **Linting**: Must pass all golangci-lint rules before merge
- **Documentation**: All exported symbols must have godoc comments
- **Breaking Changes**: Coordinate with TypeScript SDK for API parity
- **Examples**: Must be working and tested before being marked as such

### Performance
- **Streaming**: Real-time message delivery via channels
- **Resource Management**: Proper cleanup of goroutines and file handles
- **Error Recovery**: Graceful degradation and meaningful error messages

## External Dependencies

### Required
- **Claude Code CLI**: Official Claude Code command-line interface
  - Must be installed and available in PATH
  - Provides the actual Claude agent capabilities

### Optional
- **ANTHROPIC_API_KEY**: Environment variable for API authentication
  - Required for e2e tests
  - May be optional depending on Claude CLI configuration

### Development
- **Nix**: Reproducible development environment (via flake.nix)
- **direnv**: Automatic environment loading (.envrc)
- **golangci-lint**: Comprehensive linting tool

### Related Projects
- [Claude Code TypeScript SDK](https://github.com/anthropics/anthropic-sdk-typescript) - Reference implementation
- [Claude Code Python SDK](https://github.com/anthropics/anthropic-sdk-python) - Python counterpart
- Claude Code documentation - Official docs for CLI and protocol
