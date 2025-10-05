// Package streaming provides persistent bidirectional conversation services.
package streaming

import (
	"sync"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Service manages persistent bidirectional streaming conversations.
// It establishes a connection, allows multiple message exchanges,
// and requires explicit cleanup.
type Service struct {
	transport  ports.Transport
	protocol   ports.ProtocolHandler
	parser     ports.MessageParser
	hooks      *hooking.Service
	perms      *permissions.Service
	mcpServers map[string]ports.MCPServer

	mu        sync.Mutex
	connected bool
	msgCh     chan map[string]any
	errCh     chan error
}

// NewService creates a new streaming service with the provided dependencies.
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
