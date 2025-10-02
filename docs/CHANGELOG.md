# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Core Domain & Ports (Phase 1)
- **Domain Models** (`pkg/claude/messages/`)
  - `Message` interface with discriminated union types
  - `UserMessage`, `AssistantMessage`, `SystemMessage` message types
  - `ResultMessageSuccess` and `ResultMessageError` for query completion
  - `StreamEvent` for raw API streaming events
  - Content blocks: `TextBlock`, `ThinkingBlock`, `ToolUseBlock`, `ToolResultBlock`
  - `UsageStats`, `ModelUsage`, `PermissionDenial` types
  - `SystemMessageInit` and `SystemMessageCompactBoundary` subtypes

- **Options** (`pkg/claude/options/`)
  - `PermissionMode` enum for permission handling
  - `SettingSource` enum for configuration sources
  - `AgentDefinition` for subagent configuration
  - `SystemPromptConfig` interface with string and preset variants
  - `AgentOptions` combining domain and infrastructure configuration
  - MCP server configs: `StdioServerConfig`, `SSEServerConfig`, `HTTPServerConfig`, `SDKServerConfig`

- **Ports** (`pkg/claude/ports/`)
  - `Transport` interface for CLI communication
  - `ProtocolHandler` interface for control protocol
  - `MessageParser` interface for message parsing
  - `MCPServer` interface for MCP server integration
  - `ControlDependencies` structure for dependency injection
  - `PermissionChecker` and `HookCallback` minimal interfaces

- **Error Types** (`pkg/claude/errors.go`)
  - Standard errors: `ErrNotConnected`, `ErrCLINotFound`, `ErrCLIConnection`, etc.
  - `CLINotFoundError`, `ProcessError`, `JSONDecodeError` custom error types

#### Domain Services (Phase 2)
- **Hooking Service** (`pkg/claude/hooking/`)
  - Hook event types: `PreToolUse`, `PostToolUse`, `UserPromptSubmit`, `Stop`, `SubagentStop`, `PreCompact`
  - Hook input types with discriminated union pattern
  - `HookContext` for cancellation support
  - `HookCallback` function type
  - `HookMatcher` for pattern-based hook execution
  - `Service` for hook registration and execution
  - Pattern matching logic for tool-specific hooks

- **Permissions Service** (`pkg/claude/permissions/`)
  - `PermissionResult` interface with `Allow` and `Deny` variants
  - `PermissionUpdate` for permission changes
  - `PermissionRuleValue` for permission rules
  - `PermissionBehavior` enum: allow, deny, ask
  - `PermissionUpdateDestination` enum for settings scope
  - `ToolPermissionContext` with suggestions
  - `Service` for permission checking and mode updates
  - Permission update parsing from raw data

- **Querying Service** (`pkg/claude/querying/`)
  - One-shot query execution
  - Transport connection management
  - Hook callback map building
  - Message routing setup
  - Prompt formatting and sending
  - Message streaming with parsing

- **Streaming Service** (`pkg/claude/streaming/`)
  - Bidirectional conversation management
  - Connection establishment with optional initial prompt
  - Message sending capability
  - Message receiving with channels
  - Graceful connection closing

#### Adapters (Phase 3)
- **CLI Transport Adapter** (`pkg/claude/adapters/cli/`)
  - Subprocess-based CLI transport
  - CLI binary discovery in PATH and common locations
  - Command building with all AgentOptions
  - Stdin/stdout/stderr pipe management
  - Buffered JSON message reading with configurable buffer size
  - One-shot vs streaming mode detection
  - Stderr callback support
  - Process lifecycle management

- **JSON-RPC Protocol Adapter** (`pkg/claude/adapters/jsonrpc/`)
  - Control protocol state management (pending requests, request IDs)
  - Unique request ID generation with counter and random hex
  - 60-second timeout for control requests
  - Message routing by type: control_response, control_request, control_cancel_request
  - Control request handlers: `can_use_tool`, `hook_callback`, `mcp_message`
  - Permission suggestion parsing
  - Hook callback execution
  - MCP message proxying with error handling
  - Async control request handling

- **Message Parser Adapter** (`pkg/claude/adapters/parse/`)
  - Message type detection and routing
  - User message parsing with string and block content
  - Assistant message parsing with all block types
  - System message parsing with flexible data field
  - Result message parsing (success and error subtypes)
  - Stream event parsing
  - Content block parsing: text, thinking, tool_use, tool_result
  - Usage statistics parsing
  - Model usage parsing with costs
  - Permission denial parsing
  - Helper functions for optional string pointers

#### Public API (Phase 4)
- **Query Function** (`pkg/claude/query.go`)
  - One-shot query entry point
  - Automatic adapter wiring
  - Hook and permission service creation
  - IsStreaming flag management

- **Client** (`pkg/claude/client.go`)
  - Bidirectional streaming client
  - Connection management
  - Message sending and receiving
  - Thread-safe operations with mutex
  - Graceful cleanup

- **Hooks Re-exports** (`pkg/claude/hooks.go`)
  - Public hook types and constants
  - Permission types re-export
  - `HookJSONOutput` structure
  - `BlockBashPatternHook` example implementation

#### Examples
- **Quickstart** (`cmd/examples/quickstart/`)
  - Simple one-shot query example
  - Message handling demonstration
  - Usage statistics display

- **Streaming** (`cmd/examples/streaming/`)
  - Interactive bidirectional conversation
  - User input loop
  - Concurrent message handling

- **Hooks** (`cmd/examples/hooks/`)
  - Bash command blocking example
  - Tool use logging hook
  - Permission denial tracking

#### Documentation
- **README** (`docs/README.md`)
  - Feature overview
  - Installation instructions
  - Quick start examples
  - Architecture diagrams
  - Configuration guide
  - Message type reference
  - Development guide

- **CHANGELOG** (`docs/CHANGELOG.md`)
  - Comprehensive change tracking
  - Semantic versioning commitment

### Architecture
- Implemented hexagonal architecture (ports and adapters pattern)
- Strict dependency direction: adapters → domain
- Domain independence from infrastructure
- Port interfaces defined by domain needs
- Context-based package naming

### Technical Details
- Zero external dependencies (stdlib only)
- Full `context.Context` integration
- Goroutines and channels for concurrency
- Strong typing with generics where appropriate
- Compile-time interface verification
- Configurable buffer sizes for large responses
- Proper error wrapping and unwrapping

## [0.1.0] - Initial Implementation

This is the initial implementation of the Claude Agent SDK for Go, based on the Python SDK reference implementation.

### Goals Achieved
- ✅ One-shot query support
- ✅ Bidirectional streaming conversations
- ✅ Lifecycle hooks system
- ✅ Permission system
- ✅ Hexagonal architecture
- ✅ Type-safe messages
- ✅ Example applications
- ✅ Documentation

### Planned Features
- ⏳ MCP server integration
- ⏳ Comprehensive test suite
- ⏳ Performance benchmarks
- ⏳ Additional examples
- ⏳ API documentation generation
