package options

import "fmt"

// BuiltinTool provides type safety for Claude built-in tools.
// Using a custom type instead of string prevents typos and enables
// compile-time validation of tool names.
type BuiltinTool string

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

	// ToolGlob finds files matching patterns.
	ToolGlob BuiltinTool = "Glob"

	// ToolGrep searches file contents.
	ToolGrep BuiltinTool = "Grep"

	// ToolTask manages task lists.
	ToolTask BuiltinTool = "Task"

	// ToolExitPlanMode exits planning mode.
	ToolExitPlanMode BuiltinTool = "ExitPlanMode"

	// ToolWebFetch fetches web content.
	ToolWebFetch BuiltinTool = "WebFetch"

	// ToolWebSearch searches the web.
	ToolWebSearch BuiltinTool = "WebSearch"

	// ToolListMcpResources lists MCP server resources.
	ToolListMcpResources BuiltinTool = "ListMcpResources"

	// ToolReadMcpResource reads an MCP resource.
	ToolReadMcpResource BuiltinTool = "ReadMcpResource"

	// ToolMcp invokes MCP tools.
	ToolMcp BuiltinTool = "Mcp"

	// ToolNotebookEdit edits Jupyter notebooks.
	ToolNotebookEdit BuiltinTool = "NotebookEdit"

	// ToolTodoWrite manages todo lists.
	ToolTodoWrite BuiltinTool = "TodoWrite"

	// ToolSlashCommand executes slash commands.
	ToolSlashCommand BuiltinTool = "SlashCommand"
)

// WithMatcher creates a pattern-based permission rule.
// This enables fine-grained permissions like "Read(*.go)" to allow reading
// only Go source files.
func (t BuiltinTool) WithMatcher(pattern string) string {
	return fmt.Sprintf("%s(%s)", t, pattern)
}

// String converts the tool to its string representation.
func (t BuiltinTool) String() string {
	return string(t)
}
