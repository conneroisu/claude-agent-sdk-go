# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial implementation of Claude Agent SDK for Go
- Hexagonal architecture with ports and adapters pattern
- Core domain models (messages, content blocks, options)
- Transport layer (CLI adapter)
- Protocol layer (JSON-RPC adapter)
- Message parsing adapter
- MCP client/server adapters
- Domain services:
  - Querying service for one-shot queries
  - Streaming service for bidirectional conversations
  - Hooking service for lifecycle hooks
  - Permissions service for tool access control
- Public API facade with convenience functions
- Comprehensive test suite:
  - Parser tests (5 tests)
  - Hooking service tests (3 tests)
  - Permissions service tests (3 tests)
  - Querying service tests (2 tests)
  - Streaming service tests (3 tests)
- Test utilities and mocks in `internal/testutil`
- Four example programs:
  - quickstart - Simple one-shot query
  - streaming - Bidirectional conversation
  - hooks - Pre/post tool use hooks
  - tools - Tool filtering
- Documentation:
  - README.md with usage examples
  - ARCHITECTURE.md explaining design
  - CONTRIBUTING.md with guidelines
  - This CHANGELOG.md

### Code Quality

- Maximum 175 lines per file enforced
- Maximum 25 lines per function enforced
- Maximum 80 characters per line
- Maximum cognitive complexity 20
- Maximum nesting depth 3
- Minimum 15% comment density
- 100% golangci-lint compliance
- All tests passing

## [0.1.0] - TBD

### Added

- Initial release

[Unreleased]: https://github.com/conneroisu/claude-agent-sdk-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/conneroisu/claude-agent-sdk-go/releases/tag/v0.1.0
