// Package mcp provides adapters for SDK-managed MCP servers.
// This file contains tests for the MCP adapter implementation.
package mcp

import (
	"context"
	"errors"
	"testing"
)

// mockMCPServer is a mock MCP server for testing.
// It implements the MessageHandler interface with a configurable handler.
type mockMCPServer struct {
	// handleFunc is the function called when HandleMessage is invoked
	handleFunc func(ctx context.Context, message []byte) ([]byte, error)
}

// HandleMessage implements the MessageHandler interface for testing.
func (m *mockMCPServer) HandleMessage(
	ctx context.Context,
	message []byte,
) ([]byte, error) {
	return m.handleFunc(ctx, message)
}

// TestNewAdapter verifies that the adapter can be created with valid
// instances and rejects invalid ones that don't implement MessageHandler.
func TestNewAdapter(t *testing.T) {
	t.Run("creates adapter with valid instance", func(t *testing.T) {
		// Create a mock server that echoes messages
		mock := &mockMCPServer{
			handleFunc: func(_ context.Context, msg []byte) ([]byte, error) {
				return msg, nil
			},
		}

		// Create adapter and verify no error
		adapter, err := NewAdapter("test-server", mock)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify adapter name is set correctly
		if adapter.Name() != "test-server" {
			t.Errorf(
				"expected name 'test-server', got '%s'",
				adapter.Name(),
			)
		}
	})

	t.Run("returns error for invalid instance", func(t *testing.T) {
		// Try to create adapter with invalid instance (string)
		_, err := NewAdapter("test-server", "invalid")
		if err == nil {
			t.Fatal("expected error for invalid instance, got nil")
		}
	})
}

// TestAdapter_HandleMessage verifies that the adapter correctly proxies
// messages to the underlying handler and propagates errors appropriately.
func TestAdapter_HandleMessage(t *testing.T) {
	t.Run("proxies message to handler", func(t *testing.T) {
		// Setup expected request and response
		expectedMsg := []byte(`{"method":"test"}`)
		expectedResp := []byte(`{"result":"success"}`)

		// Create mock that verifies request and returns response
		mock := &mockMCPServer{
			handleFunc: func(_ context.Context, msg []byte) ([]byte, error) {
				// Verify the message is forwarded correctly
				if string(msg) != string(expectedMsg) {
					t.Errorf(
						"expected message %s, got %s",
						string(expectedMsg),
						string(msg),
					)
				}

				return expectedResp, nil
			},
		}

		// Create adapter with the mock server
		adapter, err := NewAdapter("test-server", mock)
		if err != nil {
			t.Fatalf("unexpected error creating adapter: %v", err)
		}

		// Send message through adapter
		ctx := context.Background()
		resp, err := adapter.HandleMessage(ctx, expectedMsg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify response matches expected
		if string(resp) != string(expectedResp) {
			t.Errorf(
				"expected response %s, got %s",
				string(expectedResp),
				string(resp),
			)
		}
	})

	t.Run("propagates handler errors", func(t *testing.T) {
		// Setup expected error
		expectedErr := errors.New("handler error")

		// Create mock that returns error
		mock := &mockMCPServer{
			handleFunc: func(_ context.Context, _ []byte) ([]byte, error) {
				return nil, expectedErr
			},
		}

		// Create adapter with error-returning mock
		adapter, err := NewAdapter("test-server", mock)
		if err != nil {
			t.Fatalf("unexpected error creating adapter: %v", err)
		}

		// Send message and verify error propagates
		ctx := context.Background()
		_, err = adapter.HandleMessage(ctx, []byte("test"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Verify error message matches expected
		if err.Error() != expectedErr.Error() {
			t.Errorf("expected error '%v', got '%v'", expectedErr, err)
		}
	})
}
