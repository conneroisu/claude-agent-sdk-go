# GRADE

**Grade:** Needs significant revision

**What Looks Good**
- Provides high-level architecture and hexagonal layering guidance.
- Identifies major domains (querying, streaming, hooks, permissions) and corresponding adapters/tests.

**Major Issues**
1. `jsonrpc.NewAdapter` is defined to require transport, permissions, and hook callback wiring, but the plan's public API glue passes only the transport, so the example facade code would not compile or function (docs/PLAN.md:1822-1833, docs/PLAN.md:1384-1431).
2. CLI transport references `AgentOptions.InputFormat`, yet that field is never defined in the options struct, leaving the transport logic impossible to implement as written (docs/PLAN.md:1222-1225, docs/PLAN.md:201-239).
3. `CreateSDKMCPServer` returns an `SDKServerConfig` with an `Instance` field even though the earlier type definition intentionally omits that field to avoid circular dependencies, so the documented construction path is contradictory and unimplementable (docs/PLAN.md:2088-2092, docs/PLAN.md:296-305).
4. Several critical method bodies (for example `querying.Service.Execute`, `streaming.Service.Connect`, `hooking.Service.Execute`, `permissions.Service.CheckToolUse`) are left as comments saying “Domain logic ...” with no concrete plan for steps, so the implementation roadmap is incomplete for the highest-risk components (docs/PLAN.md:462-481, docs/PLAN.md:626-665, docs/PLAN.md:820-838, docs/PLAN.md:967-973).

**Recommendations**
- Correct the adapter wiring examples so the constructor signatures line up and the facade demonstrates the intended dependency graph.
- Either add the missing fields (e.g., `InputFormat`) to `AgentOptions` or adjust the transport logic to work with the fields that actually exist.
- Reconcile the MCP server configuration story so the config type matches the stated constraints and shows how instances are registered.
- Flesh out the TODO/comment-only method sections with concrete acceptance criteria or step-by-step plans so future implementers know what behavior is required.
