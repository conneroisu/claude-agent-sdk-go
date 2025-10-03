# PLAN Markdown Evaluation

## Major Issues
- `PLAN/00_preamble.md:92` links to `07_phase_5_advanced_features.md`, but the folder only contains the split files `07a`–`07d`; following the plan as written would produce a broken reference.
- `PLAN/04_phase_2_domain_services.md:635` instantiates the permissions service with `options.PermissionModeAsk`, yet Phase 1 only defines `Default`, `AcceptEdits`, `Plan`, and `BypassPermissions`; the missing constant would prevent the code from compiling.
- `PLAN/04_phase_2_domain_services.md:305` defines hook event constants for six events, but subsequent hook inputs (e.g. Notification, SessionStart, SessionEnd at lines 348–408) rely on additional event names that are never declared; consumers would receive events the service cannot reference.
- `PLAN/06_phase_4_public_api_facade.md:79` stores `*PermissionsConfig` on the facade, but that type is not defined or imported in the claude package—Phase 2 exposes `permissions.PermissionsConfig`; the sample code as written will not compile.
- `PLAN/06_phase_4_public_api_facade.md:163` calls `server.Close()` while the port in `PLAN/03_phase_1_core_domain_ports.md:458` only exposes `Name` and `HandleMessage`; either the port or the facade must change, otherwise cleanup cannot compile.
- `PLAN/07b_phase_5_mcp_servers.md:46` contradicts Phase 1 by saying `options.SDKServerConfig` will hold a `*mcp.Server`, while the earlier plan intentionally avoided storing instances to prevent circular dependencies; those two documents cannot both be correct.
- `PLAN/07c_phase_5_permissions.md:16-91` moves permissions into `internal/permissions` and imports `github.com/anthropics/claude-agent-sdk-go`, conflicting with the established module path `github.com/conneroisu/claude` and the Phase 2 location `pkg/claude/permissions`; the plan would produce an incoherent package layout.
- `PLAN/07c_phase_5_permissions.md:304-337` presents public API usage (`claude.NewClient("api-key")`, `client.NewAgent`, `WithPermissionCallback`) that do not exist anywhere else in the plan, making the guidance misleading for SDK users.

## Moderate Issues
- `PLAN/08_phase_6_testing_documentation.md:58` has a typo (`ttt.setupMock`) that would not compile; the same snippet also calls `querying.NewService(transport, protocol)` (line 61) even though the service in Phase 2 requires parser, hooks, permissions, and MCP arguments.
- `PLAN/08_phase_6_testing_documentation.md:74` imports `github.com/conneroisu/claude/pkg/claude/internal/parse`, but the architecture places the parser under `adapters/parse`; following the snippet would import a non-existent package.
- `PLAN/08_phase_6_testing_documentation.md:190-194` uses `exec.LookPath` and `fmt.Println` without importing `exec` or `fmt`; the integration test example does not compile as written.
- `PLAN/09_phase_7_publishing_cicd.md:29` pins CI to Go 1.23 even though `go.mod` declares Go 1.25; running CI with the older toolchain will diverge from local builds.

## Minor Observations
- `PLAN/06_phase_4_public_api_facade.md:210-214` already notes the need for `Close()` on the port; Phase 1 should be updated in tandem to keep the documents synchronized.
