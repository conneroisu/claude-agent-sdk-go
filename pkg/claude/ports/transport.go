package ports

import "context"

// Transport defines what the domain needs from a transport layer.
//
// This port interface abstracts the communication mechanism with Claude CLI.
// Implementations handle the specifics of stdio, network, or other protocols.
//
// The domain layer depends on this interface, allowing different transport
// implementations to be plugged in without changing domain logic.
//
// Example implementation: CLI stdio transport connects to Claude Code process.
type Transport interface {
	// Connect establishes the transport connection.
	//
	// Must be called before any other operations.
	// Returns an error if the connection cannot be established.
	//
	// Example:
	//
	//	if err := transport.Connect(ctx); err != nil {
	//	    return fmt.Errorf("failed to connect: %w", err)
	//	}
	Connect(ctx context.Context) error

	// Write sends data to the transport.
	//
	// The data should be a complete message (typically JSON).
	// Returns an error if the write fails.
	//
	// Example:
	//
	//	msg := `{"type":"control_request","request_id":"req_1"}`
	//	if err := transport.Write(ctx, msg); err != nil {
	//	    return fmt.Errorf("write failed: %w", err)
	//	}
	Write(ctx context.Context, data string) error

	// ReadMessages continuously reads messages from the transport.
	//
	// Returns two channels:
	//   - Message channel: Receives parsed JSON messages as map[string]any
	//   - Error channel: Receives any errors during reading
	//
	// The channels remain open until the transport closes or an error occurs.
	// Callers should monitor both channels and handle accordingly.
	//
	// Example:
	//
	//	msgCh, errCh := transport.ReadMessages(ctx)
	//	for {
	//	    select {
	//	    case msg := <-msgCh:
	//	        handleMessage(msg)
	//	    case err := <-errCh:
	//	        return fmt.Errorf("read error: %w", err)
	//	    case <-ctx.Done():
	//	        return ctx.Err()
	//	    }
	//	}
	ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error)

	// EndInput signals end of input to the transport.
	//
	// For stdin-based transports, this closes the write side while keeping
	// the read side open to receive final responses.
	//
	// Returns an error if ending input fails.
	//
	// Example:
	//
	//	if err := transport.EndInput(); err != nil {
	//	    log.Printf("Warning: failed to end input: %v", err)
	//	}
	EndInput() error

	// Close releases all transport resources.
	//
	// Should be called when done with the transport.
	// After Close, the transport cannot be reused.
	//
	// Example:
	//
	//	defer transport.Close()
	Close() error

	// IsReady returns whether the transport is ready for use.
	//
	// Returns true if connected and ready to read/write.
	// Returns false if not connected or closed.
	//
	// Example:
	//
	//	if !transport.IsReady() {
	//	    return errors.New("transport not ready")
	//	}
	IsReady() bool
}
