# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0-alpha] - 2025-10-04

### Added
- Initial release with Query and Client APIs
- Full control protocol support via JSON-RPC
- Hooks system with 9 event types (on_connect, on_message, on_error, etc.)
- Permissions system with 5 modes (unrestricted, audit, confirm, restricted, isolated)
- MCP client support (stdio, HTTP, SSE transports)
- MCP server hosting capability
- 18 built-in tools (Bash, Read, Write, Edit, Grep, Glob, etc.)
- Hexagonal architecture with strict domain/adapter separation
- CLI transport adapter for Claude CLI integration
- JSON-RPC protocol handler
- Message parser for all Claude message types
- Comprehensive error handling and graceful shutdown
- One-shot query functionality
- Streaming conversation client
- Working quickstart example

### Technical Details
- 56 Go files totaling 3,159 lines of code
- All files under 175-line limit
- All functions under 25-line limit
- Godoc coverage for all exported types and functions
- Zero compilation errors
- Compatible with Go 1.22 and 1.23

[Unreleased]: https://github.com/conneroisu/claude-agent-sdk-go/compare/v0.1.0-alpha...HEAD
[0.1.0-alpha]: https://github.com/conneroisu/claude-agent-sdk-go/releases/tag/v0.1.0-alpha
