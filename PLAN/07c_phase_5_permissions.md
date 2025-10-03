## Phase 5c: Permission Callbacks

### Priority: Medium

### Overview
Permission callbacks allow users to implement custom authorization logic for tool usage. These callbacks receive context about the tool being used and can allow, deny, or modify the request.

**Hexagonal Architecture Alignment:**
- Permissions are a distinct domain boundary (authorization/access control)
- Located in `pkg/claude/permissions/` package as a domain service (consistent with Phase 2)
- Public API surface matches Phase 2's `permissions.Service` definition
- Core agent/tool logic depends on the permissions service through dependency injection

### Package Structure

The permissions service is defined in Phase 2 as `permissions.Service`. This phase provides implementation guidance and usage examples. The service is located at:

```
pkg/claude/permissions/
  └── service.go     # Service type with CheckToolUse, UpdateMode methods (defined in Phase 2)
```

---

### Implementation Reference

The `permissions.Service` type is fully defined in **Phase 2: Domain Services** (section 2.4). Key components include:

- `PermissionResult` interface with `PermissionResultAllow` and `PermissionResultDeny` implementations
- `PermissionUpdate` struct for permission modifications
- `ToolPermissionContext` for passing suggestion data
- `CanUseToolFunc` callback signature
- `PermissionsConfig` for service configuration
- `Service` struct with `CheckToolUse()` and `UpdateMode()` methods

Refer to Phase 2 for the complete implementation.

---

## Implementation Notes

### Hexagonal Architecture Benefits

**Clear Domain Boundaries:**
- Permissions isolated as a domain service
- No coupling between permissions and other domain services
- Clean dependency injection through constructor

**Dependency Injection:**
- Other services receive `*permissions.Service` through their constructors
- Callback function (`CanUseToolFunc`) passed via `PermissionsConfig`
- Easy to test with mock callbacks

### File Size Requirements

**pkg/claude/permissions/ package:**
- See Phase 2 for linting compliance notes
- Service implementation defined in Phase 2 follows all file size constraints

### Integration with Domain Services

The permissions service is injected into querying and streaming services as shown in Phase 2:

```go
// Example from Phase 2
queryService := querying.NewService(
	transport,
	protocol,
	parser,
	hooks,
	permissionsService,  // Injected here
	mcpServers,
)
```

The JSON-RPC adapter calls the permissions service when handling `can_use_tool` control requests (see Phase 3 for details).

### Integration Points


**Dependency Flow:**
```
Domain Services (querying, streaming)
    ↓ depend on
permissions.Service
    ↓ uses
CanUseToolFunc callback (provided by user)
```

**Key Points:**
- Domain services receive `*permissions.Service` via constructor
- Service is optional (can be nil)
- Callback function customizes permission logic
- See Phase 2 and Phase 3 for complete integration details

### Checklist

**Architecture:**
- [ ] `permissions.Service` implemented as defined in Phase 2
- [ ] Service properly integrated into domain services via constructor injection
- [ ] Permission types match Phase 2 definitions

**Functionality:**
- [ ] Permission callback properly invoked via `CheckToolUse()` method
- [ ] Permission results (allow/deny) correctly handled
- [ ] Permission updates applied when present
- [ ] Tool inputs modified when permissions return updates

**Testing:**
- [ ] Unit tests for `Service` with mock callbacks
- [ ] Integration tests with querying and streaming services
- [ ] Test various permission modes and scenarios

---

## Related Files
- [Phase 5a: Hooks Support](./07a_phase_5_hooks.md)
- [Phase 5b: MCP Server Support](./07b_phase_5_mcp_servers.md)
- [Phase 5: Integration Summary](./07d_phase_5_integration_summary.md)
