// Package ports defines interfaces that the domain needs from infrastructure.
// These are "ports" in hexagonal architecture - contracts defined by
// domain needs, not by external systems.
package ports

import "context"

// Transport defines what the domain needs from a transport layer.
// This abstracts stdio communication with the Claude CLI process.
type Transport interface {
	// Connect establishes connection to the Claude CLI
	Connect(ctx context.Context) error

	// Write sends a message to the CLI
	Write(ctx context.Context, data string) error

	// ReadMessages returns channels for incoming messages and errors
	ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error)

	// EndInput signals end of input stream
	EndInput() error

	// Close terminates the connection
	Close() error

	// IsReady checks if transport is ready to send/receive
	IsReady() bool
}
