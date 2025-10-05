// Package querying provides one-shot query execution services.
package querying

import (
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Service orchestrates one-shot query execution with automatic lifecycle
// management. It coordinates transport connection, message formatting,
// and cleanup on completion or error.
type Service struct {
	transport  ports.Transport
	protocol   ports.ProtocolHandler
	parser     ports.MessageParser
	hooks      *hooking.Service
	perms      *permissions.Service
	mcpServers map[string]ports.MCPServer
}

// NewService creates a new querying service with the provided dependencies.
// All port dependencies are required. Hooks, permissions, and MCP servers
// are optional and can be nil/empty.
//
//nolint:revive // 6 params acceptable for service constructor
func NewService(
	transport ports.Transport,
	protocol ports.ProtocolHandler,
	parser ports.MessageParser,
	hooks *hooking.Service,
	perms *permissions.Service,
	mcpServers map[string]ports.MCPServer,
) *Service {
	return &Service{
		transport:  transport,
		protocol:   protocol,
		parser:     parser,
		hooks:      hooks,
		perms:      perms,
		mcpServers: mcpServers,
	}
}
