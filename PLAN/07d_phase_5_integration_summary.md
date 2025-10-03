## Phase 5: Integrations Summary

### Overview
Phase 5 adds three critical integration capabilities to the Claude Agent SDK, enabling users to customize agent behavior and add custom tools.

### Integration Components

#### **5a. Hooks Support** (Critical Priority)
[📄 Full Documentation](./07a_phase_5_hooks.md)

Lifecycle hooks that execute at key points in agent execution:
- `PreToolUse` - Before tool execution (can block/modify)
- `PostToolUse` - After tool execution
- `UserPromptSubmit` - When user submits a prompt
- `Stop` - When agent stops
- `SubagentStop` - When subagent stops
- `PreCompact` - Before context compaction

**Key file:** `hooks.go` in public API

#### **5b. MCP Server Support** (Critical Priority)
[📄 Full Documentation](./07b_phase_5_mcp_servers.md)

In-process user-defined tools via Model Context Protocol:
- Wraps official `github.com/modelcontextprotocol/go-sdk`
- Provides `NewMCPServer()` and `AddTool()` convenience functions
- SDK manages internal adapter for message proxying

**Key file:** `mcp.go` in public API

#### **5c. Permission Callbacks** (Medium Priority)
[📄 Full Documentation](./07c_phase_5_permissions.md)

Custom authorization logic for tool usage:
- `CanUseToolFunc` callback interface
- Can allow, deny, or modify tool requests
- Supports permission updates and suggestions

**Key file:** `permissions.go` in public API

---

## Cross-Cutting Implementation Guidance

### Integration Points
- **Hooks** integrate with tool execution pipeline in agent core
- **MCP servers** integrate via adapter pattern in `adapters/mcp/`
- **Permissions** integrate with tool authorization in agent core

### Shared Patterns
- All three use callback/interface patterns for user extensibility
- All expose simple public APIs that wrap complex internal machinery
- All follow the SDK's adapter pattern for protocol translation

### Dependencies
- Hooks may trigger permission checks
- Permission callbacks may generate hook events
- MCP tools participate in both hooks and permissions

---

## Overall Checklist

### Code Organization
- [ ] All files under 175 lines (linting requirement)
- [ ] Functions under 25 lines where possible
- [ ] Complex logic extracted to helper functions
- [ ] Type switching replaced with handler maps

### Testing
- [ ] Unit tests for all hook types
- [ ] Integration tests for MCP server communication
- [ ] Permission callback authorization tests
- [ ] Cross-feature integration tests (hooks + permissions)

### Documentation
- [ ] Public API fully documented with examples
- [ ] Integration patterns documented
- [ ] Error handling documented
- [ ] Migration guide for users (if applicable)

### Performance
- [ ] Hook execution optimized (minimal overhead)
- [ ] MCP message routing efficient
- [ ] Permission checks don't block unnecessarily

---

## Implementation Order

**Recommended sequence:**
1. **Permissions** (simplest, foundational)
2. **Hooks** (depends on permissions)
3. **MCP** (can leverage hook/permission infrastructure)

This order minimizes rework and allows testing of each integration independently before combining them.
