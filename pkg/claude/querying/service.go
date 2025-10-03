// Package querying implements the domain service for executing
// one-shot queries to Claude.
//
// This is a DOMAIN service containing only business logic, with
// no infrastructure concerns. It depends on port interfaces for
// transport, protocol, and parsing.
package querying

import (
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Service handles query execution.
// This is a DOMAIN service - it contains only business logic,
// no infrastructure concerns like protocol state management.
type Service struct {
	transport   ports.Transport
	protocol    ports.ProtocolHandler
	parser      ports.MessageParser
	hooks       *hooking.Service
	permissions *permissions.Service
	mcpServers  map[string]ports.MCPServer
}

// NewService creates a new querying service with dependencies.
//nolint:revive // argument-limit: all parameters required for service
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
