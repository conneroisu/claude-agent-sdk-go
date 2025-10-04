package querying

import (
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Service handles query execution.
// This is a DOMAIN service - it contains only business logic,
// no infrastructure concerns like protocol state management.
// The service coordinates transport, protocol, parsing, hooks,
// and permissions to execute one-shot queries against Claude CLI.
type Service struct {
	transport   ports.Transport
	protocol    ports.ProtocolHandler
	parser      ports.MessageParser
	hooks       *hooking.Service
	permissions *permissions.Service
	mcpServers  map[string]ports.MCPServer
}

// NewService creates a new query execution service.
// The mcpServers map contains ports.MCPServer implementations:
// - For SDK servers: ServerAdapter instances wrapping user's *mcp.Server
// - For client servers: ClientAdapter instances with active sessions
// The protocol adapter uses this map to route control protocol
// mcp_message requests.
func NewService(
	transport ports.Transport,
	protocol ports.ProtocolHandler,
	parser ports.MessageParser,
	hooks *hooking.Service,
	perms *permissions.Service,
	mcpServers map[string]ports.MCPServer,
) *Service {
	return &Service{
		transport:   transport,
		protocol:    protocol,
		parser:      parser,
		hooks:       hooks,
		permissions: perms,
		mcpServers:  mcpServers,
	}
}
