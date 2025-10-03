# Review of PLAN/*.md

## Summary
- Walked through `PLAN/00_preamble.md` through `PLAN/14_code_quality_and_linting_constraints.md`.
- Multiple documents align with the Python reference, but several contain factual errors, internal inconsistencies, or code samples that would not compile as written.
- Key risks come from conflicting architectural guidance (especially around permissions), inaccessible struct fields, and incorrect assumptions about linting capabilities.

## File-by-File Findings

### PLAN/00_preamble.md
- The dependency arrow `Public API → Adapters → Ports → Domain Services` conflicts with the later Go samples where domain services import the `ports` package; the arrow implies the opposite (ports depending on the domain services). Needs correction so the direction reflects reality (domain services depend on ports, adapters depend on ports, etc.).

### PLAN/01_executive_summary.md
- No substantive correctness issues found.

### PLAN/02_architecture_overview.md
- Package layout keeps permissions under `pkg/claude/permissions`, but Phase 5c later moves the same responsibility into `internal/permissions/`. The docs need to converge on a single location to avoid contradictory guidance.

### PLAN/03_phase_1_core_domain_ports.md
- Imports and type layouts generally track the Python reference. No blocking correctness issues identified in this file.

### PLAN/04_phase_2_domain_services.md
- `permissions.Service` section defines the concrete type, but Phase 5c later renames the same concept to `permissions.Handler` and relocates it. The cross-document mismatch should be resolved.

### PLAN/05_phase_3_adapters_infrastructure.md
- The CLI adapter accesses `a.options._isStreaming`, but `_isStreaming` is an unexported field on `options.AgentOptions`. Code outside the `options` package cannot read or write it, so this sample would not compile. Either expose the flag or pass the information some other way.

### PLAN/06_phase_4_public_api_facade.md
- Missing `fmt` import even though `fmt.Errorf` is used.
- `Client` stores `*PermissionsConfig`, but that type is never declared or aliased in this file. The only available definition lives in `permissions` package, so either alias it (`type PermissionsConfig = permissions.PermissionsConfig`) or use the fully qualified name.

### PLAN/07a_phase_5_hooks.md
- The sample imports `permissions` but never uses it, which would fail `go build` under default settings. Remove the import or use it.

### PLAN/07b_phase_5_mcp_servers.md
- States that "`options.SDKServerConfig` will be updated to hold the `*mcp.Server` instance", but earlier phases define `AgentOptions.MCPServers` as `map[string]MCPServerConfig` (configuration, not runtime server). The two plans contradict each other; decide which structure is authoritative.

### PLAN/07c_phase_5_permissions.md
- Import path switches to `github.com/anthropics/claude-agent-sdk-go`, disagreeing with `go.mod` (`github.com/conneroisu/claude`) and every other file.
- Recommends moving permissions into `internal/permissions`, contradicting prior phases that place them in `pkg/claude/permissions`.
- Public API example (`claude.NewClient("api-key")`, `WithPermissionCallback`, `client.NewAgent`) does not match the API described in Phase 4. These calls do not exist in the surrounding plan.
- Later examples reference `permissions.Handler`, but Phase 2 defines the working type as `permissions.Service`.

### PLAN/07d_phase_5_integration_summary.md
- No additional correctness issues beyond the cross-references already called out.

### PLAN/08_phase_6_testing_documentation.md
- Table-driven test example calls `ttt.setupMock` (typo); should be `tt.setupMock`.
- Sample `querying.NewService(transport, protocol)` omits required parameters (parser, hooks, permissions, MCP servers) from the Phase 2 constructor signature.
- `claude.Query(ctx, "What is 2 + 2?", nil)` ignores the `hooks` argument introduced in Phase 4.

### PLAN/09_phase_7_publishing_cicd.md
- Workflow snippet assumes Go `1.23`; ensure the selected version actually exists when the pipeline is set up.

### PLAN/10_implementation_phases.md
- No correctness issues noted.

### PLAN/11_key_design_decisions.md
- No correctness issues noted.

### PLAN/12_success_criteria.md
- No correctness issues noted.

### PLAN/13_hexagonal_architecture_summary.md
- Restates the same dependency arrow issue called out in the preamble; make sure the diagram matches import directions in the code samples.

## Recommendations
- Reconcile the conflicting guidance around permissions (package location, type names, API surface) before implementation starts.
- Fix compile-time issues in the public API, hook, and adapter samples so the snippets reflect buildable Go code.
- Update linting guidance to reference actual `golangci-lint` capabilities and adjust constraints accordingly.
- Once the above corrections are made, re-check cross references so the plan presents a single coherent architecture.
