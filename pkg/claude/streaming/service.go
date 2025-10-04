// Package streaming handles bidirectional streaming conversations.
package streaming

import (
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Service handles streaming conversations.
// This is a DOMAIN service - pure business logic for managing conversations.
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

// Config holds configuration for creating a streaming service.
type Config struct {
	Transport   ports.Transport
	Protocol    ports.ProtocolHandler
	Parser      ports.MessageParser
	Hooks       *hooking.Service
	Permissions *permissions.Service
	MCPServers  map[string]ports.MCPServer
}

// NewService creates a new streaming service.
func NewService(cfg *Config) *Service {
	return &Service{
		transport:   cfg.Transport,
		protocol:    cfg.Protocol,
		parser:      cfg.Parser,
		hooks:       cfg.Hooks,
		permissions: cfg.Permissions,
		mcpServers:  cfg.MCPServers,
		msgCh:       make(chan map[string]any),
		errCh:       make(chan error, 1),
	}
}

// Close terminates the streaming connection.
func (s *Service) Close() error {
	if s.transport != nil {
		return s.transport.Close()
	}

	return nil
}
