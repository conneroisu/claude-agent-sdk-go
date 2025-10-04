// Package helpers provides convenience utilities for common SDK operations.
// Inspired by the TypeScript SDK's lib/ directory, it offers helper functions
// for tool selection, system prompt building, and configuration.
package helpers

import (
	"strings"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// ToolsToString converts a slice of BuiltinTools to CLI format.
// It returns a comma-separated string suitable for CLI flags.
//
// Example:
//
//	tools := []options.BuiltinTool{options.ToolRead, options.ToolWrite}
//	result := helpers.ToolsToString(tools)
//	// Returns: "Read,Write"
func ToolsToString(tools []options.BuiltinTool) string {
	strs := make([]string, len(tools))
	for i, t := range tools {
		strs[i] = string(t)
	}

	return strings.Join(strs, ",")
}

// AllowTools creates an allowed tools specification for CLI.
// It supports both simple tool names and matcher patterns.
//
// Example:
//
//	spec := helpers.AllowTools("Read", "Bash(git:*)", "Grep")
//	// Returns: "Read,Bash(git:*),Grep"
func AllowTools(specs ...string) string {
	return strings.Join(specs, ",")
}

// DenyTools creates a denied tools specification for CLI.
// Functionally identical to AllowTools, provided for semantic clarity.
//
// Example:
//
//	spec := helpers.DenyTools("Bash", "WebFetch")
//	// Returns: "Bash,WebFetch"
func DenyTools(specs ...string) string {
	return AllowTools(specs...)
}

// AllToolsExcept returns all builtin tools except the specified ones.
// Useful for creating allowlists by exclusion rather than inclusion.
//
// Example:
//
//	tools := helpers.AllToolsExcept(
//		options.ToolBash,     // Exclude shell access
//		options.ToolWebFetch, // Exclude web access
//	)
//	// Returns all tools except Bash and WebFetch
func AllToolsExcept(exclude ...options.BuiltinTool) []options.BuiltinTool {
	excludeMap := make(map[options.BuiltinTool]bool)
	for _, t := range exclude {
		excludeMap[t] = true
	}

	allTools := getAllTools()
	result := make([]options.BuiltinTool, 0, len(allTools))
	for _, t := range allTools {
		if !excludeMap[t] {
			result = append(result, t)
		}
	}

	return result
}

// getAllTools returns the complete list of all builtin tools.
// This is an internal helper for AllToolsExcept.
func getAllTools() []options.BuiltinTool {
	return []options.BuiltinTool{
		options.ToolBash, options.ToolBashOutput, options.ToolKillShell,
		options.ToolRead, options.ToolWrite, options.ToolEdit,
		options.ToolGlob, options.ToolGrep,
		options.ToolTask, options.ToolExitPlanMode,
		options.ToolWebFetch, options.ToolWebSearch,
		options.ToolListMcpResources, options.ToolReadMcpResource,
		options.ToolMcp, options.ToolNotebookEdit,
		options.ToolTodoWrite, options.ToolSlashCommand,
	}
}
