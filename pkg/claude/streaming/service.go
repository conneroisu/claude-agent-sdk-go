// Package streaming implements the domain service for
// bidirectional streaming conversations with Claude.
//
// This is a DOMAIN service containing only business logic. Like
// the querying service, control protocol state management is
// delegated to the protocol adapter.
package streaming

import (
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Service handles streaming conversations.
// This is a DOMAIN service - pure business logic for managing
// conversations.
type Service struct {
	transport   ports.Transport
	protocol    ports.ProtocolHandler
	parser      ports.MessageParser
	hooks       *hooking.Service
	permissions *permissions.Service
	mcpServers  map[string]ports.MCPServer

	// Message routing channels (internal to service)
	msgCh chan map[string]any
	errCh chan error
}

// NewService creates a new streaming service with dependencies.
//nolint:revive // argument-limit: all parameters required for DI
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
		msgCh:       make(chan map[string]any),
		errCh:       make(chan error, 1),
	}
}
