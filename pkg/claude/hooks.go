package claude

import (
	"fmt"
	"strings"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
)

// Re-export domain hook types for public API
type (
	// HookEvent represents a hook event.
	HookEvent = hooking.HookEvent
	// HookContext is the context for a hook.
	HookContext = hooking.HookContext
	// HookCallback is a function that is called when a hook is triggered.
	HookCallback = hooking.HookCallback
	// HookMatcher is a function that matches a hook event.
	HookMatcher = hooking.HookMatcher
)

// Re-export hook event constants
const (
	HookEventPreToolUse       = hooking.HookEventPreToolUse
	HookEventPostToolUse      = hooking.HookEventPostToolUse
	HookEventUserPromptSubmit = hooking.HookEventUserPromptSubmit
	HookEventStop             = hooking.HookEventStop
	HookEventSubagentStop     = hooking.HookEventSubagentStop
	HookEventPreCompact       = hooking.HookEventPreCompact
)

// Re-export permissions types for public API
type (
	// PermissionsConfig represents the permissions configuration.
	PermissionsConfig = permissions.PermissionsConfig
	// PermissionResult represents the result of a permission check.
	PermissionResult = permissions.PermissionResult
	// CanUseToolFunc represents a function that checks if a tool can be
	// used.
	CanUseToolFunc = permissions.CanUseToolFunc
)

// HookJSONOutput represents the JSON output format for hooks
type HookJSONOutput struct {
	Decision           *string        `json:"decision,omitempty"` // "block"
	SystemMessage      *string        `json:"systemMessage,omitempty"`
	HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"`
}

// BlockBashPatternHook creates a hook that blocks bash commands matching patterns
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
