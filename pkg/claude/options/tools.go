package options

import "fmt"

// BuiltinTool represents a Claude built-in tool name.
// This provides type safety and IDE autocomplete for tool selection.
type BuiltinTool string

// Built-in tool constants - all 18 Claude Code tools.
const (
	// ToolBash executes bash commands.
	ToolBash BuiltinTool = "Bash"

	// ToolBashOutput reads output from background bash shells.
	ToolBashOutput BuiltinTool = "BashOutput"

	// ToolKillShell kills a background bash shell.
	ToolKillShell BuiltinTool = "KillShell"

	// ToolRead reads files from the filesystem.
	ToolRead BuiltinTool = "Read"

	// ToolWrite writes files to the filesystem.
	ToolWrite BuiltinTool = "Write"

	// ToolEdit edits existing files.
	ToolEdit BuiltinTool = "Edit"

	// ToolGlob finds files using glob patterns.
	ToolGlob BuiltinTool = "Glob"

	// ToolGrep searches file contents using regex.
	ToolGrep BuiltinTool = "Grep"

	// ToolTask launches specialized subagents.
	ToolTask BuiltinTool = "Task"

	// ToolExitPlanMode exits plan mode.
	ToolExitPlanMode BuiltinTool = "ExitPlanMode"

	// ToolWebFetch fetches web content.
	ToolWebFetch BuiltinTool = "WebFetch"

	// ToolWebSearch performs web searches.
	ToolWebSearch BuiltinTool = "WebSearch"

	// ToolListMcpResources lists MCP server resources.
	ToolListMcpResources BuiltinTool = "ListMcpResources"

	// ToolReadMcpResource reads a specific MCP resource.
	ToolReadMcpResource BuiltinTool = "ReadMcpResource"

	// ToolMcp invokes MCP server tools.
	ToolMcp BuiltinTool = "Mcp"

	// ToolNotebookEdit edits Jupyter notebooks.
	ToolNotebookEdit BuiltinTool = "NotebookEdit"

	// ToolTodoWrite manages TODO lists.
	ToolTodoWrite BuiltinTool = "TodoWrite"

	// ToolSlashCommand executes slash commands.
	ToolSlashCommand BuiltinTool = "SlashCommand"
)

// WithMatcher creates a tool matcher pattern (e.g., "Bash(git:*)").
// This is used for fine-grained tool permissions.
func (t BuiltinTool) WithMatcher(matcher string) string {
	return fmt.Sprintf("%s(%s)", t, matcher)
}

// String returns the tool name as a string.
func (t BuiltinTool) String() string {
	return string(t)
}
