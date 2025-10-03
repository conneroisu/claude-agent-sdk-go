// Package ports defines domain-defined interfaces (hexagonal architecture).
//
// Ports are contracts defined by the domain's needs, not by external systems.
// Infrastructure adapters implement these interfaces to provide functionality.
//
// This package contains:
//   - Transport: CLI subprocess communication
//   - ProtocolHandler: Control protocol (JSON-RPC) handling
//   - MessageParser: Raw JSON to typed message conversion
//   - MCPServer: In-process MCP server proxying
package ports

import "context"

// Transport defines what the domain needs from a transport layer.
//
// The transport layer handles low-level communication with the Claude CLI
// subprocess. It manages stdin/stdout, lifecycle, and message streaming.
type Transport interface {
	// Connect establishes connection to the Claude CLI subprocess.
	Connect(ctx context.Context) error

	// Write sends a string to the CLI's stdin.
	Write(ctx context.Context, data string) error

	// ReadMessages returns channels for streaming messages and errors.
	// The message channel provides raw JSON messages as map[string]any.
	// The error channel reports any errors during reading.
	ReadMessages(
		ctx context.Context,
	) (<-chan map[string]any, <-chan error)

	// EndInput signals end of input (sends EOF to CLI).
	EndInput() error

	// Close terminates the CLI subprocess and releases resources.
	Close() error

	// IsReady returns true if the transport is connected and ready.
	IsReady() bool
}
