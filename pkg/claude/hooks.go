// Package claude provides hook functionality for the Claude SDK,
// including hook types, event constants, permission configurations,
// and built-in hook implementations. It re-exports core hooking and
// permissions types from internal packages and provides utility hooks
// such as BlockBashPatternHook for common use cases.
package claude

import (
	"fmt"
	"strings"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
)

// Re-export domain hook types for public API

// HookEvent represents a hook event type.
type HookEvent = hooking.HookEvent

// HookContext represents a hook context type.
type HookContext = hooking.HookContext

// HookCallback represents a hook callback type.
type HookCallback = hooking.HookCallback

// HookMatcher represents a hook matcher type.
type HookMatcher = hooking.HookMatcher

// Re-export hook event constants.
const (
	// HookEventPreToolUse is triggered before a tool is used.
	HookEventPreToolUse = hooking.HookEventPreToolUse
	// HookEventPostToolUse is triggered after a tool is used.
	HookEventPostToolUse = hooking.HookEventPostToolUse
	// HookEventUserPromptSubmit is triggered when the user submits a prompt.
	HookEventUserPromptSubmit = hooking.HookEventUserPromptSubmit
	// HookEventStop is triggered when the agent is stopped.
	HookEventStop = hooking.HookEventStop
	// HookEventSubagentStop is triggered when a subagent is stopped.
	HookEventSubagentStop = hooking.HookEventSubagentStop
	// HookEventPreCompact is triggered before compaction.
	HookEventPreCompact = hooking.HookEventPreCompact
)

// Re-export permissions types for public API

// PermissionsConfig represents the configuration for permissions.
type PermissionsConfig = permissions.PermissionsConfig

// PermissionResult represents the result of a permission check.
type PermissionResult = permissions.PermissionResult

// CanUseToolFunc represents a function that checks if a tool can be used.
type CanUseToolFunc = permissions.CanUseToolFunc

// HookJSONOutput represents the output structure for hooks.
type HookJSONOutput struct {
	Decision           *string        `json:"decision,omitempty"` // "block"
	SystemMessage      *string        `json:"systemMessage,omitempty"`
	HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"`
}

// BlockBashPatternHook creates a hook that blocks bash commands
// containing certain patterns. This is an example hook implementation
// showing how to use the hook system.
func BlockBashPatternHook(patterns []string) HookCallback {
	return func(
		input map[string]any,
		_ *string,
		_ HookContext,
	) (map[string]any, error) {
		toolName, _ := input["tool_name"].(string)
		if toolName != "Bash" {
			return make(map[string]any), nil
		}

		toolInput, _ := input["tool_input"].(map[string]any)
		command, _ := toolInput["command"].(string)

		for _, pattern := range patterns {
			if strings.Contains(command, pattern) {
				return map[string]any{
					"hookSpecificOutput": map[string]any{
						"hookEventName":      "PreToolUse",
						"permissionDecision": "deny",
						"permissionDecisionReason": fmt.Sprintf(
							"Command contains forbidden pattern: %s",
							pattern,
						),
					},
				}, nil
			}
		}

		return make(map[string]any), nil
	}
}
