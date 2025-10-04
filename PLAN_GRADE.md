# Claude Agent SDK Go Plan – Grading

## Overall Takeaways
- Plan is impressively thorough but often copies final code rather than outlining approach, making the "plan" brittle and hard to adapt.
- Several sections contradict repository reality (e.g., dependency claims, MCP in-memory transport, hook outputs) and need correction before execution.
- Cross-cutting concerns such as sequencing, risks, timelines, and validation criteria are unevenly covered across documents.

## Document Grades
| File | Grade | Key Notes |
| --- | --- | --- |
| `PLAN/00_preamble.md` | B | Strong orientation but misses delivery framing and repeats the "zero dependencies" claim that contradicts go.mod. |
| `PLAN/01_executive_summary.md` | B | Clear scope/metrics, yet slips in inaccurate "zero external dependencies" messaging. |
| `PLAN/02_architecture_overview.md` | C | Helpful structure diagrams, but several factual errors (dependency list, in-memory MCP flow) and no enforcement mechanics. |
| `PLAN/03_phase_1_core_domain_ports.md` | C | Useful model inventory, yet mostly verbatim code with little guidance on sequencing, testing, or open questions. |
| `PLAN/04_phase_2_domain_services.md` | C | Rich detail on flows, but again almost all implementation, sparse acceptance criteria, and no risk discussion. |
| `PLAN/05_phase_3_adapters_infrastructure.md` | C | Walks through adapters, though heavy copy/paste and missing operational constraints (Windows, CLI discovery failures). |
| `PLAN/06_phase_4_public_api_facade.md` | C | Describes wiring but omits API surface decisions (error types, blocking semantics) and assumes helper functions that do not exist. |
| `PLAN/07a_phase_5_hooks.md` | C- | Defines types that are absent in the Go code (e.g., `HookJSONOutput`), and mandates file splits that diverge from the repo. |
| `PLAN/07b_phase_5_mcp_servers.md` | C | Captures desired UX, yet the described in-memory transport is not implemented and will mislead engineers. |
| `PLAN/07c_phase_5_permissions.md` | B- | Solid description of ask/allow/deny flow, but lacks clarity on persistence of permission updates and integration tests. |
| `PLAN/07d_phase_5_integration_summary.md` | B- | Good end-to-end picture; still assumes hook/permission APIs that differ from current types. |
| `PLAN/08_phase_6_testing_documentation.md` | C | Reasonable structure but includes broken snippets (e.g., `ttt.setupMock`) and no coverage strategy for CLI-dependent tests. |
| `PLAN/09_phase_7_publishing_cicd.md` | C- | Action list is handy, yet relies on unreleased Go 1.25 and omits artifact signing or module validation. |
| `PLAN/10_implementation_phases.md` | D+ | Too terse to guide execution—no milestones, owners, or dependencies. |
| `PLAN/11_key_design_decisions.md` | B | Concise and accurate, though could call out deviations that already exist in code (e.g., permissions service coupling). |
| `PLAN/12_success_criteria.md` | C | Lists outcomes but lacks measurable, verifiable checkpoints beyond coverage. |
| `PLAN/13_hexagonal_architecture_summary.md` | B | Clear restatement of boundaries, but repeats the inaccurate MCP adapter behaviour. |
| `PLAN/14_code_quality_and_linting_constraints.md` | B- | Mirrors golangci config well; would benefit from guidance on handling necessary exceptions.

## Detailed Feedback & Improvements

### `PLAN/00_preamble.md` (B)
- **Positives:** Sets context, architecture, and navigation clearly (`PLAN/00_preamble.md:1-140`).
- **Gaps:** No delivery cadence, staffing, or explicit acceptance gates; repeats "Zero Dependencies" despite mandatory MCP libraries (`PLAN/00_preamble.md:34-41`).
- **Improve:** Add project timeline & risk framing, and clarify "minimal" vs. "zero" dependencies.

### `PLAN/01_executive_summary.md` (B)
- **Positives:** Articulates scope, differentiators, and success metrics (`PLAN/01_executive_summary.md:5-71`).
- **Gaps:** Claim of "Zero external dependencies" conflicts with go.mod reality (`PLAN/01_executive_summary.md:36`). No mention of delivery risks or support commitments.
- **Improve:** Qualify dependency statement and outline top risks/mitigations.

### `PLAN/02_architecture_overview.md` (C)
- **Positives:** Package map and diagrams aid orientation (`PLAN/02_architecture_overview.md:34-118`).
- **Issues:** Contradictory dependency messaging (`PLAN/02_architecture_overview.md:10-15`); inaccurate description of SDK MCP servers using in-memory transports (`PLAN/02_architecture_overview.md:408-412`); truncations/typos in diagrams (`PLAN/02_architecture_overview.md:89-90`).
- **Improve:** Reconcile dependency statements, align MCP server behaviour with current adapters, and specify enforcement tooling (e.g., depguard rules).

### `PLAN/03_phase_1_core_domain_ports.md` (C)
- **Positives:** Clarifies when to use typed structs vs. `map[string]any` and lists core models (`PLAN/03_phase_1_core_domain_ports.md:12-205`).
- **Issues:** Primarily dumps finished code, leaving no sequencing guidance or acceptance criteria (`PLAN/03_phase_1_core_domain_ports.md:62-381`); omits error taxonomy despite being listed in TOC.
- **Improve:** Replace code blocks with implementation steps, call out unresolved schema questions, and define validation/tests per model.

### `PLAN/04_phase_2_domain_services.md` (C)
- **Positives:** Documents control flows and responsibilities (`PLAN/04_phase_2_domain_services.md:7-175`).
- **Issues:** Again mostly final code; lacks test strategy per service and doesn't discuss failure handling (e.g., transport reconnect). Hook callback ID scheme deviates from actual implementation (`PLAN/04_phase_2_domain_services.md:238-267`).
- **Improve:** Add task breakdown (state machines, error surfaces), identify integration risks, and align ID formats with existing code.

### `PLAN/05_phase_3_adapters_infrastructure.md` (C)
- **Positives:** Walks through transport command composition (`PLAN/05_phase_3_adapters_infrastructure.md:33-164`).
- **Issues:** No guidance on Windows support or CLI installation detection beyond static paths; lacks failure-mode plan for stderr handling; code-style prescriptive without rationale.
- **Improve:** Document platform matrix, subprocess lifecycle risks, and metrics/logging expectations.

### `PLAN/06_phase_4_public_api_facade.md` (C)
- **Positives:** Explains how public API wires services (`PLAN/06_phase_4_public_api_facade.md:10-140`).
- **Issues:** Assumes helper like `initializeMCPServers`/`permissions.NewService` already exist without defining responsibilities or error surfaces. No plan for ergonomics (sync vs async APIs, context cancellation semantics).
- **Improve:** Enumerate API design decisions still open, add acceptance tests, and specify documentation deliverables for ergonomics.

### `PLAN/07a_phase_5_hooks.md` (C-)
- **Positives:** Captures desired hook lifecycle and registration flow (`PLAN/07a_phase_5_hooks.md:79-154`).
- **Issues:** Introduces `HookJSONOutput` type not present in Go implementation (`PLAN/07a_phase_5_hooks.md:42-46`); mandates file splits that differ from repository reality (`PLAN/07a_phase_5_hooks.md:383-409`); glosses over concurrency/error propagation policies.
- **Improve:** Align with existing Go types, document hook output contract precisely, and add guidance on timeout/error handling.

### `PLAN/07b_phase_5_mcp_servers.md` (C)
- **Positives:** Provides comprehensive UX narrative and examples (`PLAN/07b_phase_5_mcp_servers.md:17-201`).
- **Issues:** Claims SDK creates in-memory transports and channel-based routing (`PLAN/07b_phase_5_mcp_servers.md:40-44`, `PLAN/07b_phase_5_mcp_servers.md:665-680`), which is not implemented—current `ServerAdapter` is a thin wrapper returning nil. Could misdirect engineers.
- **Improve:** Update flow to match actual adapter behaviour, clarify limitations (e.g., SDK servers not auto-run), and specify required tests.

### `PLAN/07c_phase_5_permissions.md` (B-)
- **Positives:** Good depiction of permission request lifecycle (`PLAN/07c_phase_5_permissions.md:35-138`).
- **Issues:** Leaves persistence of `PermissionUpdate` undefined; does not cover multi-session behaviour or how mode changes propagate.
- **Improve:** Specify storage/update strategy, add requirements for logging/auditing, and reference integration tests.

### `PLAN/07d_phase_5_integration_summary.md` (B-)
- **Positives:** Helpful combined example (`PLAN/07d_phase_5_integration_summary.md:1-210`).
- **Issues:** Example relies on hook API returning `hooking.HookMatcher{Hooks: []HookCallback{...}}` signatures that differ from current types; lacks verification steps.
- **Improve:** Sync example with actual API, add expected outputs, and tie back to acceptance criteria.

### `PLAN/08_phase_6_testing_documentation.md` (C)
- **Positives:** Enumerates test categories and doc deliverables (`PLAN/08_phase_6_testing_documentation.md:1-550`).
- **Issues:** Contains typographical bug (`ttt.setupMock`) that would break tests (`PLAN/08_phase_6_testing_documentation.md:57-59`); no plan for CLI-integration tests requiring Claude binary/API key; optional use of `testify` conflicts with "zero dependencies" stance.
- **Improve:** Correct snippets, define strategy for sandboxed CLI tests, and decide on third-party testing libs explicitly.

### `PLAN/09_phase_7_publishing_cicd.md` (C-)
- **Positives:** Action-oriented checklist and sample workflows (`PLAN/09_phase_7_publishing_cicd.md:6-206`).
- **Issues:** Requires Go 1.25+ which is unreleased/stable risk (`PLAN/09_phase_7_publishing_cicd.md:14-21`); release workflow assumes binary artifacts though SDK is a library; no supply-chain measures (checksums/signing).
- **Improve:** Target current stable Go version, outline module publishing verification, and document security controls.

### `PLAN/10_implementation_phases.md` (D+)
- **Issues:** Too skeletal to drive execution (`PLAN/10_implementation_phases.md:1-14`); lacks milestones, owners, or dependencies.
- **Improve:** Expand into phase roadmap with entry/exit criteria, timeline, and parallelisation guidance.

### `PLAN/11_key_design_decisions.md` (B)
- **Positives:** Summarises guiding principles succinctly (`PLAN/11_key_design_decisions.md:1-45`).
- **Gaps:** Could flag where existing code diverges (e.g., permissions service currently couples to options); lacks discussion of trade-offs.
- **Improve:** Note known deviations and rationale to keep plan actionable.

### `PLAN/12_success_criteria.md` (C)
- **Issues:** High-level statements without measurement guidance apart from coverage (`PLAN/12_success_criteria.md:3-9`).
- **Improve:** Add quantifiable metrics (latency budgets, startup time, lint debt targets) and validation owners.

### `PLAN/13_hexagonal_architecture_summary.md` (B)
- **Positives:** Communicates dependency rules and benefits clearly (`PLAN/13_hexagonal_architecture_summary.md:1-117`).
- **Issues:** Repeats inaccurate assumption that `adapters/mcp` handles full JSON-RPC routing for SDK servers (`PLAN/13_hexagonal_architecture_summary.md:46-58`).
- **Improve:** Clarify current limitations and document future enhancements separately.

### `PLAN/14_code_quality_and_linting_constraints.md` (B-)
- **Positives:** Mirrors golangci config faithfully, including file/function limits (`PLAN/14_code_quality_and_linting_constraints.md:1-176`).
- **Gaps:** No guidance on how to request justified exceptions or manage generated code.
- **Improve:** Document waiver process and tooling (e.g., build tags, generated directories).

