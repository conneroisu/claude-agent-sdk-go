// Package ports defines interfaces that the domain layer needs
// from infrastructure.
//
// These interfaces are defined BY the domain's needs, not by
// external systems. This is the "ports" in hexagonal architecture.
package ports

import "context"

// Transport defines what the domain needs from a transport layer.
// This interface abstracts subprocess communication, allowing the
// domain to remain independent of CLI implementation details.
type Transport interface {
	// Connect establishes the transport connection
	Connect(ctx context.Context) error

	// Write sends data through the transport
	Write(ctx context.Context, data string) error

	// ReadMessages returns channels for receiving messages and errors
	ReadMessages(
		ctx context.Context,
	) (<-chan map[string]any, <-chan error)

	// EndInput signals no more input will be sent
	EndInput() error

	// Close terminates the transport connection
	Close() error

	// IsReady checks if transport is ready for communication
	IsReady() bool
}
