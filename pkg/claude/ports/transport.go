// Package ports defines interfaces that the domain requires from adapters.
// Following hexagonal architecture, these interfaces are defined BY the
// domain's needs, not by external systems. Adapters implement these ports.
package ports

import "context"

// Transport abstracts connection to the Claude CLI process.
// This port defines what the domain needs for stdin/stdout communication,
// without coupling to specific process management implementations.
//
// Channel ownership: ReadMessages returns channels that are owned by the
// transport implementation. The caller must consume from these channels
// until they close. The transport closes both channels when the connection
// terminates.
//
// Concurrency: All methods are safe for concurrent use except Close,
// which should only be called once.
type Transport interface {
	// Connect establishes connection to the CLI process.
	// Returns an error if the CLI cannot be started or is unreachable.
	Connect(ctx context.Context) error

	// Write sends data to CLI stdin.
	// The data should be newline-delimited JSON messages.
	Write(ctx context.Context, data string) error

	// ReadMessages streams raw JSON messages from CLI stdout.
	// Returns two channels: one for messages, one for errors.
	// Both channels are closed when the connection terminates.
	// Messages are returned as unmarshaled maps for flexibility.
	ReadMessages(ctx context.Context) (
		<-chan map[string]any,
		<-chan error,
	)

	// EndInput signals end of input (EOF) to the CLI.
	// This tells the CLI that no more input will be sent.
	EndInput() error

	// Close terminates the connection and cleans up resources.
	// Should be called exactly once, typically in a defer.
	Close() error

	// IsReady checks if the transport is connected and ready.
	// Returns false if not connected or if connection is broken.
	IsReady() bool
}
