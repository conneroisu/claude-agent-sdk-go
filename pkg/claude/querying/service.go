// Package querying handles one-shot query execution.
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

// Config holds configuration for creating a querying service.
type Config struct {
	Transport   ports.Transport
	Protocol    ports.ProtocolHandler
	Parser      ports.MessageParser
	Hooks       *hooking.Service
	Permissions *permissions.Service
	MCPServers  map[string]ports.MCPServer
}

// NewService creates a new querying service.
func NewService(cfg *Config) *Service {
	return &Service{
		transport:   cfg.Transport,
		protocol:    cfg.Protocol,
		parser:      cfg.Parser,
		hooks:       cfg.Hooks,
		permissions: cfg.Permissions,
		mcpServers:  cfg.MCPServers,
	}
}
