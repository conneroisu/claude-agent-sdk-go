// Package helpers provides utility functions for SDK configuration.
package helpers

import (
	"strings"
)

// AllowTools joins tool specs into a comma-separated string for CLI flags.
func AllowTools(specs ...string) string {
	return strings.Join(specs, ",")
}

// DenyTools is an alias for AllowTools for semantic clarity.
func DenyTools(specs ...string) string {
	return AllowTools(specs...)
}

// AllToolsExcept returns all builtin tools except the specified exclusions.
func AllToolsExcept(exclude ...string) []string {
	allTools := []string{
		"Bash",
		"BashOutput",
		"KillShell",
		"Read",
		"Write",
		"Edit",
		"Glob",
		"Grep",
		"Task",
		"ExitPlanMode",
		"WebFetch",
		"WebSearch",
		"ListMcpResources",
		"ReadMcpResource",
		"Mcp",
		"NotebookEdit",
		"TodoWrite",
		"SlashCommand",
	}

	if len(exclude) == 0 {
		return allTools
	}

	excludeMap := make(map[string]bool)
	for _, tool := range exclude {
		excludeMap[tool] = true
	}

	result := make([]string, 0, len(allTools))
	for _, tool := range allTools {
		if !excludeMap[tool] {
			result = append(result, tool)
		}
	}

	return result
}
