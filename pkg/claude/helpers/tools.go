// Package helpers provides convenience utilities for common
// operations with Claude.
package helpers

import (
	"strings"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// ToolsToString converts a slice of BuiltinTools to CLI format.
// Example: [ToolRead, ToolWrite] -> "Read,Write".
func ToolsToString(tools []options.BuiltinTool) string {
	strs := make([]string, len(tools))
	for i, t := range tools {
		strs[i] = string(t)
	}

	return strings.Join(strs, ",")
}

// AllowTools creates an allowed tools specification for CLI.
// Supports both simple names and matcher patterns.
func AllowTools(specs ...string) string {
	return strings.Join(specs, ",")
}

// DenyTools creates a denied tools specification for CLI.
func DenyTools(specs ...string) string {
	return AllowTools(specs...)
}

// AllToolsExcept returns all builtin tools except the specified ones.
func AllToolsExcept(exclude ...options.BuiltinTool) []options.BuiltinTool {
	excludeMap := make(map[options.BuiltinTool]bool)
	for _, t := range exclude {
		excludeMap[t] = true
	}

	allTools := []options.BuiltinTool{
		options.ToolBash, options.ToolBashOutput, options.ToolKillShell,
		options.ToolRead, options.ToolWrite, options.ToolEdit,
		options.ToolGlob, options.ToolGrep,
		options.ToolTask, options.ToolExitPlanMode,
		options.ToolWebFetch, options.ToolWebSearch,
		options.ToolListMcpResources,
		options.ToolReadMcpResource,
		options.ToolMcp,
		options.ToolNotebookEdit,
		options.ToolTodoWrite,
		options.ToolSlashCommand,
	}

	result := make([]options.BuiltinTool, 0, len(allTools))
	for _, t := range allTools {
		if !excludeMap[t] {
			result = append(result, t)
		}
	}

	return result
}
