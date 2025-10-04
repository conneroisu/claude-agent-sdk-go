package options

import "fmt"

// BuiltinTool represents a Claude Code built-in tool name.
// This type provides compile-time safety and IDE autocomplete,
// preventing typos and making tool configuration maintainable.
type BuiltinTool string

// Execution tools for running commands and managing processes.
const (
	// ToolBash executes bash commands in a persistent shell session.
	ToolBash BuiltinTool = "Bash"
	// ToolBashOutput retrieves output from a running background shell.
	ToolBashOutput BuiltinTool = "BashOutput"
	// ToolKillShell terminates a running background shell by ID.
	ToolKillShell BuiltinTool = "KillShell"
)

// File operation tools for reading, writing, and searching files.
const (
	// ToolRead reads file contents with optional line range selection.
	ToolRead BuiltinTool = "Read"
	// ToolWrite writes or overwrites file contents.
	ToolWrite BuiltinTool = "Write"
	// ToolEdit performs exact string replacements in files.
	ToolEdit BuiltinTool = "Edit"
	// ToolGlob finds files matching glob patterns.
	ToolGlob BuiltinTool = "Glob"
	// ToolGrep searches file contents using regex patterns.
	ToolGrep BuiltinTool = "Grep"
)

// Agent coordination tools for delegating tasks and managing planning mode.
const (
	// ToolTask delegates work to a subagent with specific capabilities.
	ToolTask BuiltinTool = "Task"
	// ToolExitPlanMode exits planning mode and returns to normal execution.
	ToolExitPlanMode BuiltinTool = "ExitPlanMode"
)

// Web interaction tools for fetching and searching online content.
const (
	// ToolWebFetch retrieves and processes content from URLs.
	ToolWebFetch BuiltinTool = "WebFetch"
	// ToolWebSearch performs web searches and returns results.
	ToolWebSearch BuiltinTool = "WebSearch"
)

// MCP (Model Context Protocol) tools for interacting with MCP servers.
const (
	// ToolListMcpResources lists available resources from MCP servers.
	ToolListMcpResources BuiltinTool = "ListMcpResources"
	// ToolReadMcpResource reads a specific resource from an MCP server.
	ToolReadMcpResource BuiltinTool = "ReadMcpResource"
	// ToolMcp sends raw JSON-RPC messages to MCP servers.
	ToolMcp BuiltinTool = "Mcp"
)

// Specialized tools for notebooks, task management, and slash commands.
const (
	// ToolNotebookEdit edits cells in Jupyter notebooks.
	ToolNotebookEdit BuiltinTool = "NotebookEdit"
	// ToolTodoWrite creates and manages task lists.
	ToolTodoWrite BuiltinTool = "TodoWrite"
	// ToolSlashCommand executes custom slash commands.
	ToolSlashCommand BuiltinTool = "SlashCommand"
)

// WithMatcher creates a tool matcher pattern for fine-grained permissions.
// Matchers allow restricting tool usage to specific patterns, such as limiting
// Bash to only git commands: ToolBash.WithMatcher("git:*")
//
// Returns a formatted string like "Bash(git:*)" for use in permission rules.
func (t BuiltinTool) WithMatcher(matcher string) string {
	return fmt.Sprintf("%s(%s)", t, matcher)
}

// String returns the tool name as a string.
// This method enables seamless conversion to string for CLI flag construction.
func (t BuiltinTool) String() string {
	return string(t)
}
