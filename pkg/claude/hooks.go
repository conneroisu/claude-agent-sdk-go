package claude

import (
	"fmt"
	"strings"

	"github.com/conneroisu/claude/pkg/claude/hooking"
)

// Re-export domain hook types for public API.
type HookEvent = hooking.HookEvent
type HookContext = hooking.HookContext
type HookCallback = hooking.HookCallback
type HookMatcher = hooking.HookMatcher

// Re-export hook event constants.
const (
	// HookEventPreToolUse is the event for pre-tool use hooks.
	HookEventPreToolUse = hooking.HookEventPreToolUse
	// HookEventPostToolUse is the event for post-tool use hooks.
	HookEventPostToolUse = hooking.HookEventPostToolUse
	// HookEventUserPromptSubmit is the event for user prompt submit hooks.
	HookEventUserPromptSubmit = hooking.HookEventUserPromptSubmit
	// HookEventStop is the event for stop hooks.
	HookEventStop = hooking.HookEventStop
	// HookEventSubagentStop is the event for subagent stop hooks.
	HookEventSubagentStop = hooking.HookEventSubagentStop
	// HookEventPreCompact is the event for pre-compact hooks.
	HookEventPreCompact = hooking.HookEventPreCompact
)

// HookJSONOutput represents the JSON output structure for hooks.
type HookJSONOutput struct {
	Decision           *string        `json:"decision,omitempty"` // "block"
	SystemMessage      *string        `json:"systemMessage,omitempty"`
	HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"`
}

// BlockBashPatternHook returns a hook callback that blocks bash commands containing forbidden patterns.
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
