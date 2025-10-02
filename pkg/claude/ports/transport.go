package ports

import "context"

// Transport defines what the domain needs from a transport layer
type Transport interface {
	// Connect establishes a connection to the transport layer
	Connect(ctx context.Context) error
	// Write sends data through the transport
	Write(ctx context.Context, data string) error
	// ReadMessages returns channels for receiving messages and errors
	// from the transport
	ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error)
	// EndInput signals that no more input will be sent
	EndInput() error
	// Close terminates the transport connection
	Close() error
	// IsReady returns true if the transport is ready to accept operations
	IsReady() bool
}
