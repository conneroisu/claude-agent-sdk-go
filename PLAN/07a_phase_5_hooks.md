## Phase 5a: Hooks Support

### Priority: Critical

### Overview
The facade re-exports domain hook types from the `hooking` package for public API convenience. This allows users to register callbacks that execute at specific points in the agent lifecycle.

### Public API (hooks.go)

```go
package claude

import (
	"fmt"
	"strings"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
)

// Re-export domain hook types for public API
type HookEvent = hooking.HookEvent
type HookContext = hooking.HookContext
type HookCallback = hooking.HookCallback
type HookMatcher = hooking.HookMatcher

// Re-export hook event constants
const (
	// HookEventPreToolUse is the event for pre-tool use hooks.
	HookEventPreToolUse       = hooking.HookEventPreToolUse
	// HookEventPostToolUse is the event for post-tool use hooks.
	HookEventPostToolUse      = hooking.HookEventPostToolUse
	// HookEventUserPromptSubmit is the event for user prompt submit hooks.
	HookEventUserPromptSubmit = hooking.HookEventUserPromptSubmit
	// HookEventStop is the event for stop hooks.
	HookEventStop             = hooking.HookEventStop
	// HookEventSubagentStop is the event for subagent stop hooks.
	HookEventSubagentStop     = hooking.HookEventSubagentStop
	// HookEventPreCompact is the event for pre-compact hooks.
	HookEventPreCompact       = hooking.HookEventPreCompact
)

type HookJSONOutput struct {
	Decision           *string        `json:"decision,omitempty"`           // "block"
	SystemMessage      *string        `json:"systemMessage,omitempty"`
	HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"`
}

// BlockBashPatternHook returns a hook callback that blocks bash commands containing forbidden patterns
func BlockBashPatternHook(patterns []string) HookCallback {
	return func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
		toolName, _ := input["tool_name"].(string)
		if toolName != "Bash" {
			return map[string]any{}, nil
		}
		toolInput, _ := input["tool_input"].(map[string]any)
		command, _ := toolInput["command"].(string)
		for _, pattern := range patterns {
			if strings.Contains(command, pattern) {
				return map[string]any{
					"hookSpecificOutput": map[string]any{
						"hookEventName":            "PreToolUse",
						"permissionDecision":       "deny",
						"permissionDecisionReason": fmt.Sprintf("Command contains forbidden pattern: %s", pattern),
					},
				}, nil
			}
		}
		return map[string]any{}, nil
	}
}
```

---

## Implementation Notes

### File Size Requirements (175 line limit)

**hooking/ package must be split into 4 files:**
- `service.go` - Service struct + constructor (~50 lines)
- `execute.go` - Execution logic (~80 lines)
- `registry.go` - Hook registry management (~60 lines)
- `types.go` - Hook input/output types (~60 lines)

❌ Do NOT create a single `service.go` (would be 250+ lines)

### Complexity Hotspots

**Hook execution logic requires extraction:**
- Type switching for hook inputs → Extract per-type handlers
- Callback invocation → Extract wrapper function
- Error handling → Extract error wrapper

**Recommended patterns:**
- Use handler map/registry instead of large type switches
- Create callback wrapper functions to reduce boilerplate
- Extract error formatting into dedicated functions

### Checklist

- [ ] Hook execution functions under 25 lines each
- [ ] Type switching extracted to handler map/registry
- [ ] Callback handling simplified with wrappers
- [ ] All hooking/ files under 175 lines

---

## Related Files
- [Phase 5b: MCP Server Support](./07b_phase_5_mcp_servers.md)
- [Phase 5c: Permission Callbacks](./07c_phase_5_permissions.md)
- [Phase 5: Integration Summary](./07d_phase_5_integration_summary.md)
