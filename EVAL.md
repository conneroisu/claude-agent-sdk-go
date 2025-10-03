# PLAN Folder Evaluation

## PLAN/00_preamble.md
- The Table of Contents links Phase 5 to `07_phase_5_advanced_features.md`, but the folder only provides the split files `07a`–`07d`, so the link at line 92 is broken.

## PLAN/03_phase_1_core_domain_ports.md
- `ports.ProtocolHandler.Initialize` is defined to take `config map[string]any` at line 414, but the adapter implementation later in Phase 3 uses `config any`, so the signatures no longer match.
- `ports.MCPServer` (line 458) does not define a `Close() error` method, yet later phases call `server.Close()` on these adapters (see Phase 4, lines 148–165), so the interface needs to be updated for consistency.

## PLAN/04_phase_2_domain_services.md
- `permissions.NewService` defaults to `options.PermissionModeAsk` at line 635, but that constant is never defined in Phase 1 (only `Default`, `AcceptEdits`, `Plan`, and `BypassPermissions` exist).

## PLAN/05_phase_3_adapters_infrastructure.md
- `jsonrpc.Adapter.Initialize` uses the signature `Initialize(ctx context.Context, config any)` at line 377, which no longer satisfies the Phase 1 port contract that requires `map[string]any`.
- `handleHookCallback` attempts to read `tool_use_id` as a `*string` at line 675, but JSON decoding into `map[string]any` only produces plain `string` values, so the ID will always be `nil`.

## PLAN/06_phase_4_public_api_facade.md
- The import block (lines 11–20) omits `fmt`, yet the file calls `fmt.Errorf` multiple times (e.g., line 46), so the snippet would not compile.
- `Query` returns a `nil` message channel on MCP initialization failure (lines 45–48). Returning a nil channel causes callers that select on the result to block indefinitely.
- The signature `hooks map[HookEvent][]HookMatcher` at line 23 uses unqualified types that are not defined in this file or imported; callers must currently refer to `hooking.HookEvent` and `hooking.HookMatcher`.
- `Client` stores `permissions *PermissionsConfig` at lines 78–79, but no `PermissionsConfig` type is declared in the public API; the available configuration type is `permissions.PermissionsConfig`.
- `Client.Close` (lines 163–165) and MCP initialisation cleanup (line 211) both call `server.Close()`, which is missing from the Phase 1 `ports.MCPServer` interface.

## PLAN/07b_phase_5_mcp_servers.md
- The text at line 46 states that `options.SDKServerConfig` will hold a `*mcp.Server`, contradicting the Phase 1 definition that intentionally keeps only configuration data to avoid circular dependencies.

## PLAN/07c_phase_5_permissions.md
- Imports reference the module as `github.com/anthropics/claude-agent-sdk-go` (lines 35 and 310), but the repository’s `go.mod` declares `github.com/conneroisu/claude`, so the paths do not resolve.
- The public API example calls `claude.NewClient("api-key")` at line 329, which does not match the Phase 4 constructor that expects `AgentOptions`, hooks, and permission config parameters.
- The plan introduces an `internal/permissions/` package (lines 96–170), but earlier architecture sections place permissions under `pkg/claude/permissions`; the discrepancy should be reconciled.

## PLAN/08_phase_6_testing_documentation.md
- The unit-test example calls `ttt.setupMock` at line 58 (typo), so the snippet fails to compile.
- `querying.NewService` is invoked with only two arguments at line 61, omitting the required protocol/parser/hook/permission parameters defined in Phase 2.
- The parser tests import `github.com/conneroisu/claude/pkg/claude/internal/parse` and call `parse.ParseMessage` at lines 74 and 108, but the plan places the parser in `adapters/parse` with a method on the adapter instead of a package-level function.
- Several examples assert `msg.(*messages.AssistantMessage)` (lines 218 and 341), but the parser returns value types, so these assertions will always fail.

## PLAN/09_phase_7_publishing_cicd.md
- The GitHub Actions example pins Go to `1.23` at line 29, but the repository’s `go.mod` targets Go `1.25.0`, so CI would build with the wrong toolchain.

## Additional Observations
- `PLAN/05_phase_3_adapters_infrastructure.md` and `PLAN/06_phase_4_public_api_facade.md` both rely on an updated `ports.MCPServer` interface that includes `Close() error`; ensure Phase 1 is revised accordingly.
- Where snippets intentionally omit details (e.g., TODO comments), the missing pieces are acceptable as placeholders; only concrete mismatches or broken references are noted above.
