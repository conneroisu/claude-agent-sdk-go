# Particulars: Claude Agent SDK Python Reference

## Repository Shape
- `src/claude_agent_sdk/` carries the public surface (`client.py`, `query.py`) and typed configuration in `types.py`; implementation hides in `_internal/`.
- `_internal/query.py` orchestrates the control protocol, while `_internal/transport/subprocess_cli.py` wraps the Claude Code CLI subprocess and streams newline-delimited JSON.
- Tests under `tests/` exercise streaming control flow, hook callbacks, and tool permissions, offering runnable examples of the protocol expectations.

## Transport & Message Stream
- `SubprocessCLITransport` discovers the CLI binary, builds the command-line (system prompt, tool allow/deny lists, MCP config, etc.), and launches it with AnyIO’s `open_process`.
- Stdout is read via `TextReceiveStream`; the transport accumulates partial JSON fragments, enforces a configurable buffer limit, and yields parsed dicts to consumers.
- Stdin writes require the transport to be marked ready; errors on the process (non-zero exit, broken pipe) are captured and promoted to SDK-specific exceptions.

## Control Protocol & JSON-RPC Flow
- `Query.start()` spins up an AnyIO task group that continuously reads transport messages. Messages are partitioned into:
  - `control_response`: matched to pending requests via `request_id` and an `anyio.Event` in `pending_control_responses`.
  - `control_request`: delegated to `_handle_control_request` for inbound commands from the CLI (permissions, hooks, MCP proxying).
  - `control_cancel_request`: currently ignored with a `TODO`.
  - Everything else: forwarded to the public message stream.
- `Query.initialize()` prepares hook registrations by allocating callback IDs, then sends a control request with subtype `initialize`. Responses cache supported command metadata for later use.
- `_send_control_request()` serialises control payloads, writes them to the transport, and awaits completion with a 60s timeout. Results are stored in `pending_control_results`; errors bubble up as exceptions.

### Handling Incoming Control Requests
- `can_use_tool`: wraps the SDK callback and returns `{"allow": True/False}` plus optional input rewrites. It mirrors TypeScript semantics by translating `PermissionResultAllow/Deny` dataclasses from `types.py` into protocol-friendly dicts.
- `hook_callback`: looks up the hook function by the callback ID assigned during initialization, executes it, and returns the hook’s JSON output.
- `mcp_message`: bridges JSON-RPC messages from the CLI to in-process MCP servers. `_handle_sdk_mcp_request()` inspects the `method` and routes to the appropriate handler on the `mcp.server` instance.

### JSON-RPC Bridging Details
- Supported methods include `initialize`, `tools/list`, `tools/call`, and `notifications/initialized`; each returns a JSON-RPC envelope (`{"jsonrpc": "2.0", "id": ..., "result": ...}`) or an error object with MCP-standard codes.
- The bridge adapts MCP dataclasses (e.g., `CallToolRequestParams`, `ListToolsRequest`) into JSON-RPC responses, flattening Pydantic models via `model_dump()` when available.
- Unsupported methods return `-32601 Method not found`, while unexpected errors surface as `-32603 Internal error`, ensuring the CLI receives protocol-compliant responses.

## Hooks & Permissions Wiring
- Public `HookMatcher`/`HookCallback` definitions live in `types.py`; both `InternalClient` and `ClaudeSDKClient` convert them into a serialisable form, mapping each hook to a generated callback ID registered with `Query`.
- Permission callbacks (`can_use_tool`) are validated up front: they require streaming mode and force `permission_prompt_tool_name="stdio"` to align with the control protocol’s expectations.
- Hook output follows the documented JSON shape (decision/systemMessage/hookSpecificOutput), and the Python SDK forwards the hook’s dict verbatim back to the CLI via control responses.

## Message Parsing Layer
- `_internal/message_parser.py` normalises CLI output into rich dataclasses (`UserMessage`, `AssistantMessage`, `ToolUseBlock`, etc.), preserving tool-use IDs, thinking blocks, and result metadata.
- Errors at this layer raise `MessageParseError`, allowing consumers to differentiate parsing failures from transport or protocol issues.

## Notable Behaviours to Mirror in Go
- Maintain separation between the transport (raw IO) and the Query/controller (protocol logic) to support alternative transports and easier testing.
- Replicate the pending-response bookkeeping for control requests, including timeout semantics and error propagation.
- Emulate JSON-RPC bridging behaviour for MCP servers: method coverage, error codes, and transformation of MCP SDK results into JSON-compatible structures.
- Preserve hook and permission invariants (streaming requirement, mutual exclusivity, callback ID registration) so cross-language parity holds.
- Implement buffering safeguards similar to `_max_buffer_size` to avoid truncated JSON when reading from the CLI.

## Suggested Inspection References
- Control flow: `claude-agent-sdk-python/src/claude_agent_sdk/_internal/query.py` (entire file)
- Transport details: `claude-agent-sdk-python/src/claude_agent_sdk/_internal/transport/subprocess_cli.py`
- Types and callbacks: `claude-agent-sdk-python/src/claude_agent_sdk/types.py`
- Message parsing: `claude-agent-sdk-python/src/claude_agent_sdk/_internal/message_parser.py`
