package streaming

import (
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Service handles streaming conversations.
// This is a DOMAIN service - pure business logic for managing conversations.
// The service coordinates bidirectional communication with Claude CLI,
// delegating protocol concerns to the protocol adapter layer.
type Service struct {
	transport   ports.Transport
	protocol    ports.ProtocolHandler
	parser      ports.MessageParser
	hooks       *hooking.Service
	permissions *permissions.Service
	mcpServers  map[string]ports.MCPServer
	msgCh       chan map[string]any
	errCh       chan error
}

// NewService creates a new streaming conversation service.
// MCP servers are initialized by the public API layer before creating
// this service. The service receives already-connected adapters (both
// client and SDK types). When control protocol receives mcp_message
// requests, it uses this map for routing.
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

// Close terminates the streaming service and cleans up resources.
// It closes the transport connection and releases internal channels.
func (s *Service) Close() error {
	if s.transport != nil {
		return s.transport.Close()
	}

	return nil
}
