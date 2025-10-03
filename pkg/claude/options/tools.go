// Built-in tool type definitions for Claude Agent.
package options

import "fmt"

// BuiltinTool represents a Claude built-in tool name.
//
// Provides type safety and IDE autocomplete for tool selection.
// All 18 Claude Code built-in tools are defined as constants.
type BuiltinTool string

// Execution tools
const (
	// ToolBash executes bash commands in a persistent shell.
	ToolBash BuiltinTool = "Bash"
	// ToolBashOutput retrieves output from background bash shells.
	ToolBashOutput BuiltinTool = "BashOutput"
	// ToolKillShell kills a running background bash shell.
	ToolKillShell BuiltinTool = "KillShell"
)

// File operation tools
const (
	// ToolRead reads files from the filesystem.
	ToolRead BuiltinTool = "Read"
	// ToolWrite writes files to the filesystem.
	ToolWrite BuiltinTool = "Write"
	// ToolEdit performs exact string replacements in files.
	ToolEdit BuiltinTool = "Edit"
	// ToolGlob finds files using glob patterns.
	ToolGlob BuiltinTool = "Glob"
	// ToolGrep searches file contents using regex.
	ToolGrep BuiltinTool = "Grep"
)

// Agent tools
const (
	// ToolTask launches specialized subagents for complex tasks.
	ToolTask BuiltinTool = "Task"
	// ToolExitPlanMode exits plan mode and begins execution.
	ToolExitPlanMode BuiltinTool = "ExitPlanMode"
)

// Web tools
const (
	// ToolWebFetch fetches and analyzes web content.
	ToolWebFetch BuiltinTool = "WebFetch"
	// ToolWebSearch performs web searches.
	ToolWebSearch BuiltinTool = "WebSearch"
)

// MCP tools
const (
	// ToolListMcpResources lists available MCP resources.
	ToolListMcpResources BuiltinTool = "ListMcpResources"
	// ToolReadMcpResource reads an MCP resource.
	ToolReadMcpResource BuiltinTool = "ReadMcpResource"
	// ToolMcp calls MCP server tools.
	ToolMcp BuiltinTool = "Mcp"
)

// Other tools
const (
	// ToolNotebookEdit edits Jupyter notebook cells.
	ToolNotebookEdit BuiltinTool = "NotebookEdit"
	// ToolTodoWrite manages task lists.
	ToolTodoWrite BuiltinTool = "TodoWrite"
	// ToolSlashCommand executes slash commands.
	ToolSlashCommand BuiltinTool = "SlashCommand"
)

// WithMatcher creates a tool matcher pattern (e.g., "Bash(git:*)").
//
// Used for fine-grained tool permissions. Allows restricting a tool
// to specific use cases or patterns.
func (t BuiltinTool) WithMatcher(matcher string) string {
	return fmt.Sprintf("%s(%s)", t, matcher)
}

// String returns the tool name as a string.
func (t BuiltinTool) String() string {
	return string(t)
}
