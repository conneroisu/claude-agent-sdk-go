# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-10-03

### Added

#### Core API
- `Query()` function for one-shot queries
- `NewClient()` function for streaming conversations
- Channel-based message handling for idiomatic Go patterns
- Context-aware API with proper cancellation support

#### Message Types
- `AssistantMessage` with typed content blocks
- `UserMessage` for user input
- `SystemMessage` for system events
- `ResultMessageSuccess` and `ResultMessageError` for query results
- Content blocks: `TextBlock`, `ToolUseBlock`, `ToolResultBlock`, `ThinkingBlock`

#### Configuration
- `AgentOptions` for comprehensive query configuration
- Support for model selection, max turns, max tokens
- Thinking budget and output format configuration
- Tool filtering via `AllowedTools` and `BlockedTools`

#### Lifecycle Hooks
- 9 hook events: PreToolUse, PostToolUse, UserPromptSubmit, Stop, SubagentStop, PreCompact, Notification, SessionStart, SessionEnd
- Pattern matching for selective hook execution
- `BlockBashPatternHook` helper for security patterns
- Hook aggregation and result merging

#### Permission System
- `PermissionsConfig` with customizable callbacks
- 5 permission modes: Default, BypassPermissions, Ask, AcceptEdits, Plan
- `CanUseTool` callback for fine-grained control
- Permission results: Allow, Deny, Ask with optional input modification

#### MCP Integration
- `NewMCPServer()` helper for server creation
- `AddTool()` generic wrapper for type-safe tool registration
- Support for custom MCP server adapters
- Integration with Model Context Protocol Go SDK

#### Architecture
- Hexagonal (ports and adapters) architecture
- Domain services: Querying, Hooking, Permissions
- Port interfaces: Transport, ProtocolHandler, MessageParser, PermissionService, MCPServer
- Adapters: CLI Transport, JSON-RPC Protocol, Message Parser

#### Testing
- Comprehensive unit tests for all domain services
- Mock implementations of all port interfaces
- Test fixtures for common message types
- Integration tests with Claude CLI
- Table-driven test patterns

#### Examples
- `quickstart` - Basic query usage
- `streaming` - Multi-turn conversations
- `hooks` - Custom lifecycle hooks
- `mcp` - MCP server integration
- `permissions` - Permission callbacks
- `tools` - Tool filtering

#### Documentation
- Comprehensive README with architecture diagrams
- API documentation with godoc comments
- Hook development guide
- MCP integration guide
- Example programs for all major features

### Changed

Nothing (initial release)

### Fixed

Nothing (initial release)

### Security

- Built-in hook system for blocking dangerous bash commands
- Permission system for tool usage control
- No direct execution of user input

## [Unreleased]

### Planned Features

- HTTP transport adapter for remote Claude instances
- WebSocket support for real-time streaming
- Token usage tracking and budgeting
- Conversation history management
- Rate limiting and retry logic
- Prometheus metrics integration
- OpenTelemetry tracing support
